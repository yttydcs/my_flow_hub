package main

import (
	"encoding/json"
	"net/url"
	"time"

	"myflowhub/poc/protocol"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// connectToParent establishes and maintains the connection to the parent server.
func (s *Server) connectToParent() {
	u, err := url.Parse(s.parentAddr)
	if err != nil {
		log.Fatal().Err(err).Msg("无效的上级服务器地址")
	}

	for { // Main reconnect loop
		log.Info().Str("address", u.String()).Msg("正在连接到上级服务器...")
		conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			log.Error().Err(err).Msg("连接上级失败，将在5秒后重试")
			time.Sleep(5 * time.Second)
			continue
		}

		// Authenticate with the parent
		s.authenticateWithParent(conn)

		// Start reader and writer pumps
		done := make(chan struct{})
		go s.writePumpToParent(conn, done)
		go s.readPumpFromParent(conn, done)

		<-done // Wait until a pump fails
		log.Warn().Msg("与上级的连接已断开，准备重连...")
		conn.Close()
		time.Sleep(5 * time.Second)
	}
}

// readPumpFromParent handles reading messages from the parent.
func (s *Server) readPumpFromParent(conn *websocket.Conn, done chan struct{}) {
	defer close(done)
	for {
		var msg protocol.BaseMessage
		if err := conn.ReadJSON(&msg); err != nil {
			log.Error().Err(err).Msg("从上级读取消息失败")
			return
		}
		log.Info().Interface("message", msg).Msg("收到来自上级的消息")
		// Here we could handle messages from parent, e.g., route them to our children
	}
}

// writePumpToParent handles writing messages to the parent.
func (s *Server) writePumpToParent(conn *websocket.Conn, done chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-s.parentSend:
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Error().Err(err).Msg("向上级写入消息失败")
				return
			}
		case <-ticker.C:
			pingMsg := protocol.BaseMessage{
				ID:      uuid.New().String(),
				Target:  0,
				Type:    "ping",
				Payload: map[string]string{"status": "ok"},
			}
			pingBytes, _ := json.Marshal(pingMsg)
			if err := conn.WriteMessage(websocket.TextMessage, pingBytes); err != nil {
				log.Error().Err(err).Msg("向上级发送心跳失败")
				return
			}
		case <-done:
			return
		}
	}
}

// authenticateWithParent sends an authentication request to the parent.
func (s *Server) authenticateWithParent(conn *websocket.Conn) {
	log.Info().Uint64("deviceID", s.deviceID).Msg("向上级发送认证请求...")
	authPayload := protocol.AuthRequestPayload{
		DeviceID:  s.deviceID,
		SecretKey: s.secretKey,
	}
	message := protocol.BaseMessage{
		ID:      uuid.New().String(),
		Target:  0,
		Type:    "auth_request",
		Payload: authPayload,
	}
	authBytes, _ := json.Marshal(message)
	// Send directly instead of via channel, as this is the first message.
	if err := conn.WriteMessage(websocket.TextMessage, authBytes); err != nil {
		log.Error().Err(err).Msg("向上级发送认证失败")
	}
}
