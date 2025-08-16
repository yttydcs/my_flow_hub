package hub

import (
	"bytes"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
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
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Client) readPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		mt, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error().Err(err).Msg("Unexpected websocket close")
			}
			break
		}
		message = bytes.TrimSpace(message)
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
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			// 发送队列仅用于透传二进制帧
			if err := c.Conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
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
	client := &Client{Hub: s, Conn: conn, Send: make(chan []byte, 256), DeviceID: 0, RemoteAddr: r.RemoteAddr, UserAgent: r.UserAgent(), Binary: binary}
	s.Register <- client

	go client.writePump()
	go client.readPump()
}
