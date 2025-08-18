package client

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	binproto "myflowhub/pkg/protocol/binproto"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// HubClient 表示与中枢/中继的WebSocket连接客户端
type HubClient struct {
	conn         *websocket.Conn
	serverAddr   string
	deviceID     uint64
	secretKey    string
	managerToken string

	// 消息处理
	binary bool
	// binary response multiplexing
	binRespMu  sync.Mutex
	binWaiters map[uint64]chan binproto.HeaderV1
	msgSeq     uint64
	Send       chan []byte
	// 控制帧：用于通过单写协程发送 Pong，避免与业务写并发
	pongCh chan string

	// 连接状态
	connected bool
	mu        sync.RWMutex

	// 每次建立新连接时创建的关闭信号，用于结束与该连接相关的 goroutine
	// 注意：不要在重连之间复用同一个关闭通道，否则会导致新连接的写循环立即退出。
	connDone chan struct{}
	// 全局关闭信号：用于终止重连循环与整体客户端
	quitCh chan struct{}
}

// NewHubClient 创建新的Hub客户端
func NewHubClient(serverAddr, managerToken string) *HubClient {
	return &HubClient{
		serverAddr:   serverAddr,
		managerToken: managerToken,
		binWaiters:   make(map[uint64]chan binproto.HeaderV1),
		quitCh:       make(chan struct{}),
	}
}

// Connect 连接到中枢/中继服务器
func (c *HubClient) Connect() error {
	u, err := url.Parse(c.serverAddr)
	if err != nil {
		return err
	}

	log.Info().Str("addr", c.serverAddr).Msg("正在连接到中枢/中继服务器...")

	dialer := *websocket.DefaultDialer
	dialer.Subprotocols = []string{"myflowhub.bin.v1"}
	conn, _, err := dialer.Dial(u.String(), http.Header{})
	if err != nil {
		return err
	}

	c.conn = conn
	c.binary = conn.Subprotocol() == "myflowhub.bin.v1"
	c.setConnected(true)
	c.Send = make(chan []byte, 256)
	// 为本次连接创建独立的关闭信号
	c.connDone = make(chan struct{})
	c.pongCh = make(chan string, 8)

	// 启动读写协程
	go c.writePump()
	go c.readPump()

	// 进行管理员认证
	if err := c.authenticate(); err != nil {
		c.Close()
		return err
	}

	log.Info().Msg("成功连接并认证到中枢/中继服务器")
	return nil
}

// authenticate 使用管理员令牌进行认证
func (c *HubClient) authenticate() error {
	// 二进制：发送 ManagerAuthReq 帧
	payload := binproto.EncodeManagerAuthReq(c.managerToken)
	h := binproto.HeaderV1{TypeID: binproto.TypeManagerAuthReq, MsgID: c.nextMsgID(), Source: 0, Target: 0, Timestamp: time.Now().UnixMilli()}
	frame, _ := binproto.EncodeFrame(h, payload)
	c.Send <- frame
	return nil
}

// readPump 从 websocket 连接将消息泵送到 hub
func (c *HubClient) readPump() {
	defer func() {
		c.setConnected(false)
		// 先关闭与本连接相关的 goroutine（如 writePump）
		if c.connDone != nil {
			close(c.connDone)
		}
		// 关闭发送通道以结束写入尝试
		close(c.Send)
		// 关闭底层连接
		c.conn.Close()
		go c.reconnect()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		log.Debug().Msg("readPump: 收到 Pong，重置读超时")
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	// 自定义 PingHandler：不直接写 Pong，转交给写协程，避免并发写
	c.conn.SetPingHandler(func(appData string) error {
		log.Debug().Msg("readPump: 收到 Ping")
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		select {
		case c.pongCh <- appData:
		default:
			// 丢弃过多的 ping，防止阻塞
		}
		return nil
	})

	for {
		log.Debug().Msg("readPump: 等待读取消息...")
		mt, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error().Err(err).Msg("Unexpected websocket close")
			}
			log.Error().Err(err).Msg("读取消息失败，将尝试重连...")
			break
		}
		// 每次成功读取都刷新读超时，提升稳健性
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		log.Debug().Int("type", mt).Int("bytes", len(data)).Msg("readPump: 成功读取消息")

		if h, pl, err := binproto.DecodeFrame(data); err == nil {
			c.storeLastPayload(h.MsgID, pl)
			c.binRespMu.Lock()
			if ch, ok := c.binWaiters[h.MsgID]; ok {
				delete(c.binWaiters, h.MsgID)
				c.binRespMu.Unlock()
				select {
				case ch <- h:
				default:
					log.Warn().Uint64("msgID", h.MsgID).Msg("readPump: 响应无人接收，可能已超时")
				}
			} else {
				c.binRespMu.Unlock()
			}
			if h.TypeID == binproto.TypeManagerAuthResp {
				if _, uid, _, e := binproto.DecodeManagerAuthResp(pl); e == nil {
					c.deviceID = uid
					log.Info().Uint64("deviceID", c.deviceID).Msg("管理员(二进制)认证成功")
				}
			}
		}
	}
}

// storeLast holds recent binary payloads by MsgID for short time.
var binPayloadStore = struct {
	mu sync.RWMutex
	m  map[uint64][]byte
}{m: make(map[uint64][]byte)}

func (c *HubClient) storeLastPayload(msgID uint64, payload []byte) {
	binPayloadStore.mu.Lock()
	defer binPayloadStore.mu.Unlock()
	cp := make([]byte, len(payload))
	copy(cp, payload)
	binPayloadStore.m[msgID] = cp
	// Use a single timer to clean up old entries, avoiding goroutine-per-message.
	time.AfterFunc(15*time.Second, func() {
		binPayloadStore.mu.Lock()
		defer binPayloadStore.mu.Unlock()
		delete(binPayloadStore.m, msgID)
	})
}

func (c *HubClient) loadLastPayload(msgID uint64) ([]byte, bool) {
	binPayloadStore.mu.RLock()
	defer binPayloadStore.mu.RUnlock()
	p, ok := binPayloadStore.m[msgID]
	if !ok {
		return nil, false
	}
	// Return a copy to avoid race conditions if the caller modifies the slice.
	cp := make([]byte, len(p))
	copy(cp, p)
	return cp, true
}

// SendBinaryRequest 发送二进制请求并等待指定响应类型
func (c *HubClient) SendBinaryRequest(typeIDReq, typeIDResp uint16, payload []byte, timeout time.Duration) ([]byte, error) {
	if !c.IsConnected() {
		return nil, ErrNotConnected
	}
	msgID := c.nextMsgID()
	h := binproto.HeaderV1{TypeID: typeIDReq, MsgID: msgID, Source: c.deviceID, Target: 0, Timestamp: time.Now().UnixMilli()}
	frame, _ := binproto.EncodeFrame(h, payload)
	ch := make(chan binproto.HeaderV1, 1)
	c.binRespMu.Lock()
	c.binWaiters[msgID] = ch
	c.binRespMu.Unlock()
	start := time.Now()
	log.Debug().Uint64("msgID", msgID).Uint16("typeIDReq", typeIDReq).Dur("timeout", timeout).Msg("SendBinaryRequest: dispatch")
	// 将请求放入发送队列
	select {
	case c.Send <- frame:
	case <-c.connDone:
		return nil, ErrClientClosed
	}
	select {
	case h := <-ch:
		// 显式处理 ERR 帧
		if h.TypeID == binproto.TypeErrResp {
			if p, ok := c.loadLastPayload(h.MsgID); ok {
				_, code, msg, _ := binproto.DecodeErrResp(p)
				if len(msg) == 0 {
					return nil, fmt.Errorf("hub ERR %d", code)
				}
				return nil, fmt.Errorf("hub ERR %d: %s", code, string(msg))
			}
			return nil, fmt.Errorf("hub ERR")
		}
		if h.TypeID != typeIDResp {
			return nil, fmt.Errorf("unexpected resp type: %d", h.TypeID)
		}
		if p, ok := c.loadLastPayload(h.MsgID); ok {
			return p, nil
		}
		return nil, fmt.Errorf("payload missing")
	case <-time.After(timeout):
		c.binRespMu.Lock()
		delete(c.binWaiters, msgID)
		c.binRespMu.Unlock()
		log.Error().Uint64("msgID", msgID).Uint16("typeIDReq", typeIDReq).Dur("elapsed", time.Since(start)).Msg("SendBinaryRequest: timeout")
		return nil, ErrTimeout
	case <-c.connDone:
		return nil, ErrClientClosed
	}
}

// nextMsgID 生成全局唯一的消息ID，减少并发冲突
func (c *HubClient) nextMsgID() uint64 {
	seq := atomic.AddUint64(&c.msgSeq, 1)
	// 组合时间与自增序号，降低碰撞概率
	return (uint64(time.Now().UnixNano()) << 8) ^ seq
}

// reconnect 自动重连
func (c *HubClient) reconnect() {
	// 确保只有一个重连循环在运行
	if c.IsConnected() {
		return
	}
	for {
		select {
		case <-c.quitCh:
			return
		default:
			log.Info().Msg("正在尝试重新连接...")
			if err := c.Connect(); err != nil {
				log.Error().Err(err).Msg("重连失败")
				time.Sleep(5 * time.Second)
				continue
			}
			log.Info().Msg("重新连接成功")
			return
		}
	}
}

// writePump 将消息从 hub 泵送到 websocket 连接。
func (c *HubClient) writePump() {
	defer func() {
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			log.Debug().Int("bytes", len(message)).Msg("writePump: 从 channel 收到消息，准备写入")
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// hub 关闭了通道
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				log.Info().Msg("writePump: channel 已关闭，正常退出")
				return
			}
			if err := c.conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
				log.Error().Err(err).Msg("writePump: 写入二进制消息失败")
				return
			}
			log.Debug().Int("bytes", len(message)).Msg("writePump: 成功写入消息")
		case appData := <-c.pongCh:
			// 通过单写协程发送 Pong 控制帧
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(writeWait)); err != nil {
				log.Error().Err(err).Msg("writePump: 发送 Pong 失败")
				return
			}
		case <-c.connDone:
			return
		}
	}
}

// IsConnected 检查连接状态
func (c *HubClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// setConnected 设置连接状态
func (c *HubClient) setConnected(connected bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connected = connected
}

// GetDeviceID 获取设备ID
func (c *HubClient) GetDeviceID() uint64 {
	return c.deviceID
}

// NextMsgID 暴露下一个消息ID（供外部构造帧时使用）
func (c *HubClient) NextMsgID() uint64 { return c.nextMsgID() }

// ConnWriteBinary 线程安全地写入二进制帧
func (c *HubClient) ConnWriteBinary(frame []byte) error {
	if !c.IsConnected() {
		return ErrNotConnected
	}
	select {
	case c.Send <- frame:
		return nil
	case <-c.connDone:
		return ErrClientClosed
	}
}

// Close 关闭连接
func (c *HubClient) Close() error {
	// 主动关闭当前连接与相关 goroutine，并通知停止重连
	select {
	case <-c.quitCh:
		// 已关闭
	default:
		close(c.quitCh)
	}
	if c.connDone != nil {
		select {
		case <-c.connDone:
			// 已关闭
		default:
			close(c.connDone)
		}
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// 错误和常量定义
var (
	ErrNotConnected = fmt.Errorf("not connected to hub")
	ErrTimeout      = fmt.Errorf("operation timeout")
	ErrClientClosed = fmt.Errorf("client is closed")
)

const (
	writeWait  = 30 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	// 提高读取上限，避免较大二进制帧触发 read limit exceeded 进而导致 1006/EOF
	maxMessageSize = 4 * 1024 * 1024 // 4MB
)
