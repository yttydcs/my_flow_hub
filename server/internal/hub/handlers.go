package hub

import (
	"net/http"
	"time"

	"myflowhub/pkg/config"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

const (
	writeWait  = 30 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	// 提高最大消息大小，避免较大二进制帧导致 read limit exceeded → 1006 异常断开
	maxMessageSize = 4 * 1024 * 1024 // 4MB
)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	Hub        *Server
	Conn       *websocket.Conn
	Send       chan []byte
	DeviceID   uint64
	RemoteAddr string
	UserAgent  string
	Binary     bool
	// 控制帧：通过写协程发送 Pong，避免与业务写并发
	pongCh chan string
	// 诊断：记录最近一次成功读取
	lastReadAt time.Time
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Client) readPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		log.Debug().Uint64("clientID", c.DeviceID).Msg("readPump: 收到 Pong，刷新读超时")
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		mt, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error().Err(err).Msg("Unexpected websocket close")
			}
			break
		}
		// 任何成功读取都刷新读超时，提升稳健性
		c.lastReadAt = time.Now()
		c.Conn.SetReadDeadline(c.lastReadAt.Add(pongWait))
		if mt != websocket.BinaryMessage {
			// 二进制专用：拒绝非二进制帧
			continue
		}
		c.Hub.Broadcast <- &HubMessage{Client: c, Message: message, IsBinary: true}
	}
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			log.Debug().Uint64("clientID", c.DeviceID).Int("bytes", len(message)).Msg("writePump: 从 channel 收到消息，准备写入")
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				log.Info().Uint64("clientID", c.DeviceID).Msg("writePump: channel 已关闭，正常退出")
				return
			}
			// 发送队列仅用于透传二进制帧
			if err := c.Conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
				log.Error().Err(err).Uint64("clientID", c.DeviceID).Msg("writePump: 写入二进制消息失败")
				return
			}
			log.Debug().Uint64("clientID", c.DeviceID).Int("bytes", len(message)).Msg("writePump: 成功写入消息")
		case appData := <-c.pongCh:
			// 通过单写协程发送 Pong 控制帧
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(writeWait)); err != nil {
				log.Error().Err(err).
					Uint64("clientID", c.DeviceID).
					Time("lastReadAt", c.lastReadAt).
					Dur("sinceLastRead", time.Since(c.lastReadAt)).
					Msg("writePump: 发送 Pong 失败")
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
				log.Error().Err(err).
					Uint64("clientID", c.DeviceID).
					Time("lastReadAt", c.lastReadAt).
					Dur("sinceLastRead", time.Since(c.lastReadAt)).
					Msg("writePump: 发送 Ping 失败")
				return
			}
		}
	}
}

// HandleSubordinateConnection handles websocket requests from the peer.
func (s *Server) HandleSubordinateConnection(w http.ResponseWriter, r *http.Request) {
	// 协商：优先使用 Sec-WebSocket-Protocol: myflowhub.bin.v1，否则通过 ?bin=1 开启
	var respHdr http.Header
	if r.Header.Get("Sec-WebSocket-Protocol") != "" {
		respHdr = http.Header{}
		for _, p := range websocket.Subprotocols(r) {
			if p == "myflowhub.bin.v1" {
				respHdr.Set("Sec-WebSocket-Protocol", p)
				break
			}
		}
	}
	conn, err := s.Upgrader.Upgrade(w, r, respHdr)
	if err != nil {
		log.Error().Err(err).Msg("Failed to upgrade connection")
		return
	}
	binary := conn.Subprotocol() == "myflowhub.bin.v1" || r.URL.Query().Get("bin") == "1"
	// 发送队列容量从配置读取，默认 256
	qsize := config.AppConfig.WS.SendQueueSize
	if qsize <= 0 {
		qsize = 256
	}
	client := &Client{Hub: s, Conn: conn, Send: make(chan []byte, qsize), DeviceID: 0, RemoteAddr: r.RemoteAddr, UserAgent: r.UserAgent(), Binary: binary, pongCh: make(chan string, 8)}
	s.Register <- client

	go client.writePump()
	go client.readPump()
}
