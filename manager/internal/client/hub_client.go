package client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	"myflowhub/pkg/protocol"

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
	sendCh     chan []byte
	responseCh chan protocol.BaseMessage

	// 连接状态
	connected bool
	mu        sync.RWMutex

	// 停止信号
	stopCh chan struct{}
}

// NewHubClient 创建新的Hub客户端
func NewHubClient(serverAddr, managerToken string) *HubClient {
	return &HubClient{
		serverAddr:   serverAddr,
		managerToken: managerToken,
		sendCh:       make(chan []byte, 256),
		responseCh:   make(chan protocol.BaseMessage, 256),
		stopCh:       make(chan struct{}),
	}
}

// Connect 连接到中枢/中继服务器
func (c *HubClient) Connect() error {
	u, err := url.Parse(c.serverAddr)
	if err != nil {
		return err
	}

	log.Info().Str("addr", c.serverAddr).Msg("正在连接到中枢/中继服务器...")

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}

	c.conn = conn
	c.setConnected(true)

	// 启动读写协程
	go c.readLoop()
	go c.writeLoop()

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
	authMsg := protocol.BaseMessage{
		ID:   "auth-" + time.Now().Format("20060102150405"),
		Type: "manager_auth",
		Payload: map[string]interface{}{
			"token": c.managerToken,
		},
		Timestamp: time.Now(),
	}

	return c.SendMessage(authMsg)
}

// readLoop 读取消息循环
func (c *HubClient) readLoop() {
	defer c.setConnected(false)

	for {
		select {
		case <-c.stopCh:
			return
		default:
			var msg protocol.BaseMessage
			if err := c.conn.ReadJSON(&msg); err != nil {
				log.Error().Err(err).Msg("读取消息失败")
				return
			}

			log.Debug().Interface("msg", msg).Msg("收到消息")

			// 处理认证响应
			if msg.Type == "auth_response" || msg.Type == "manager_auth_response" {
				c.handleAuthResponse(msg)
				continue // 认证响应不发送到响应通道
			}

			// 将非认证消息发送到响应通道
			select {
			case c.responseCh <- msg:
			default:
				log.Warn().Msg("响应通道已满，丢弃消息")
			}
		}
	}
}

// writeLoop 写入消息循环
func (c *HubClient) writeLoop() {
	for {
		select {
		case <-c.stopCh:
			return
		case data := <-c.sendCh:
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Error().Err(err).Msg("发送消息失败")
				return
			}
		}
	}
}

// handleAuthResponse 处理认证响应
func (c *HubClient) handleAuthResponse(msg protocol.BaseMessage) {
	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		log.Error().Msg("无效的认证响应格式")
		return
	}

	success, _ := payload["success"].(bool)
	if !success {
		log.Error().Msg("管理员认证失败")
		return
	}

	if deviceID, ok := payload["deviceId"].(float64); ok {
		c.deviceID = uint64(deviceID)
		log.Info().Uint64("deviceID", c.deviceID).Msg("管理员认证成功")
	}
}

// SendMessage 发送消息
func (c *HubClient) SendMessage(msg protocol.BaseMessage) error {
	if !c.IsConnected() {
		return ErrNotConnected
	}

	// 设置源ID
	msg.Source = c.deviceID

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	select {
	case c.sendCh <- data:
		return nil
	default:
		return ErrSendChannelFull
	}
}

// GetResponse 获取响应消息（带超时）
func (c *HubClient) GetResponse(timeout time.Duration) (*protocol.BaseMessage, error) {
	select {
	case msg := <-c.responseCh:
		return &msg, nil
	case <-time.After(timeout):
		return nil, ErrTimeout
	case <-c.stopCh:
		return nil, ErrClientClosed
	}
}

// SendRequest 发送请求并等待响应
func (c *HubClient) SendRequest(req protocol.BaseMessage, timeout time.Duration) (*protocol.BaseMessage, error) {
	if err := c.SendMessage(req); err != nil {
		return nil, err
	}

	// 循环接收消息，直到找到匹配的响应
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case resp := <-c.responseCh:
			payload, ok := resp.Payload.(map[string]interface{})
			if !ok {
				continue
			}
			if originalID, ok := payload["original_id"].(string); ok && originalID == req.ID {
				return &resp, nil
			}
		case <-timer.C:
			return nil, ErrTimeout
		case <-c.stopCh:
			return nil, ErrClientClosed
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
	ErrNotConnected    = fmt.Errorf("not connected to hub")
	ErrSendChannelFull = fmt.Errorf("send channel is full")
	ErrTimeout         = fmt.Errorf("operation timeout")
	ErrClientClosed    = fmt.Errorf("client is closed")
)
