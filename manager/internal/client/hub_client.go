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
	// 文本帧/JSON 路径已移除
	binary bool
	// binary response multiplexing
	binRespMu  sync.Mutex
	binWaiters map[uint64]chan binproto.HeaderV1
	msgSeq     uint64

	// 连接状态
	connected bool
	mu        sync.RWMutex
	writeMu   sync.Mutex

	// 停止信号
	stopCh chan struct{}
}

// NewHubClient 创建新的Hub客户端
func NewHubClient(serverAddr, managerToken string) *HubClient {
	return &HubClient{
		serverAddr:   serverAddr,
		managerToken: managerToken,
		stopCh:       make(chan struct{}),
		binWaiters:   make(map[uint64]chan binproto.HeaderV1),
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

	// 启动读写协程
	go c.readLoop()
	// 无文本队列写循环，二进制直接写 conn

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
	return c.conn.WriteMessage(websocket.BinaryMessage, frame)
}

// readLoop 读取消息循环
func (c *HubClient) readLoop() {
	defer c.setConnected(false)

	for {
		select {
		case <-c.stopCh:
			return
		default:
			mt, data, err := c.conn.ReadMessage()
			if err != nil {
				log.Error().Err(err).Msg("读取消息失败，将尝试重连...")
				c.conn.Close()
				c.setConnected(false)
				go c.reconnect()
				return
			}
			if mt == websocket.BinaryMessage {
				if h, pl, err := binproto.DecodeFrame(data); err == nil {
					// 先存储 payload，再唤醒等待者，避免竞态
					c.storeLastPayload(h.MsgID, pl)
					c.binRespMu.Lock()
					if ch, ok := c.binWaiters[h.MsgID]; ok {
						delete(c.binWaiters, h.MsgID)
						c.binRespMu.Unlock()
						ch <- h
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
				continue
			}
			// 忽略非二进制帧
		}
	}
}

// storeLast holds recent binary payloads by MsgID for short time.
var binPayloadStore = struct {
	mu sync.Mutex
	m  map[uint64][]byte
}{m: map[uint64][]byte{}}

func (c *HubClient) storeLastPayload(msgID uint64, payload []byte) {
	binPayloadStore.mu.Lock()
	defer binPayloadStore.mu.Unlock()
	cp := make([]byte, len(payload))
	copy(cp, payload)
	binPayloadStore.m[msgID] = cp
	// 简易过期：异步清理
	go func(id uint64) {
		time.Sleep(10 * time.Second)
		binPayloadStore.mu.Lock()
		delete(binPayloadStore.m, id)
		binPayloadStore.mu.Unlock()
	}(msgID)
}
func (c *HubClient) loadLastPayload(msgID uint64) ([]byte, bool) {
	binPayloadStore.mu.Lock()
	defer binPayloadStore.mu.Unlock()
	p, ok := binPayloadStore.m[msgID]
	return p, ok
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
	c.writeMu.Lock()
	err := c.conn.WriteMessage(websocket.BinaryMessage, frame)
	c.writeMu.Unlock()
	if err != nil {
		return nil, err
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
		return nil, ErrTimeout
	case <-c.stopCh:
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
	for {
		select {
		case <-c.stopCh:
			return
		default:
			log.Info().Msg("正在尝试重新连接...")
			if err := c.Connect(); err == nil {
				log.Info().Msg("重新连接成功")
				return
			}
			time.Sleep(5 * time.Second)
		}
	}
}

// writeLoop 写入消息循环
// 文本写循环已移除（仅二进制）。

// handleAuthResponse 处理认证响应
// JSON 认证响应处理已移除。

// SendMessage 发送消息
// JSON 消息发送已移除。

// GetResponse 获取响应消息（带超时）
// JSON 响应等待已移除。

// SendRequest 发送请求并等待响应
// JSON 请求-响应已移除。

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
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteMessage(websocket.BinaryMessage, frame)
}

// Close 关闭连接
func (c *HubClient) Close() error {
	close(c.stopCh)

	if c.conn != nil {
		return c.conn.Close()
	}

	return nil
}

// 错误定义
var (
	ErrNotConnected = fmt.Errorf("not connected to hub")
	ErrTimeout      = fmt.Errorf("operation timeout")
	ErrClientClosed = fmt.Errorf("client is closed")
)
