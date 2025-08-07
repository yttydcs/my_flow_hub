package main

import (
	"net/url"
	"time"

	"myflowhub/poc/protocol"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// connectToParent 作为客户端连接到上级服务器，并维持连接
func (s *Server) connectToParent() {
	u, err := url.Parse(s.parentAddr)
	if err != nil {
		log.Fatal().Err(err).Msg("无效的上级服务器地址")
	}

	for { // 自动重连循环
		log.Info().Str("address", u.String()).Msg("正在连接到上级服务器...")
		conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			log.Error().Err(err).Msg("连接上级失败，将在5秒后重试")
			time.Sleep(5 * time.Second)
			continue
		}

		done := make(chan struct{})
		go s.readFromParent(conn, done)

		// Always authenticate with the bootstrapped identity
		s.authenticateWithParent(conn)

		ticker := time.NewTicker(30 * time.Second)
	loop:
		for {
			select {
			case <-done:
				log.Warn().Msg("与上级的连接已断开，准备重连...")
				break loop
			case <-ticker.C:
				s.sendPingToParent(conn)
			}
		}
		ticker.Stop()
		conn.Close()
		time.Sleep(5 * time.Second)
	}
}

// readFromParent 持续读取来自上级服务器的消息
func (s *Server) readFromParent(conn *websocket.Conn, done chan struct{}) {
	defer close(done)
	for {
		var msg protocol.BaseMessage
		if err := conn.ReadJSON(&msg); err != nil {
			log.Error().Err(err).Msg("从上级读取消息失败")
			return
		}
		log.Info().Interface("message", msg).Msg("收到来自上级的消息")
		// In a real app, we would process these messages (e.g., pong, notifications)
	}
}

// authenticateWithParent 使用已保存的凭证向上级认证
func (s *Server) authenticateWithParent(conn *websocket.Conn) {
	log.Info().Uint64("deviceID", s.deviceID).Msg("向上级发送认证请求...")
	authPayload := protocol.AuthRequestPayload{
		DeviceID:  s.deviceID,
		SecretKey: s.secretKey,
	}
	message := protocol.BaseMessage{
		ID:        uuid.New().String(),
		Target:    0,
		Type:      "auth_request",
		Timestamp: time.Now(),
		Payload:   authPayload,
	}
	if err := conn.WriteJSON(message); err != nil {
		log.Error().Err(err).Msg("向上级发送认证失败")
	}
}

// sendPingToParent 向上级发送心跳
func (s *Server) sendPingToParent(conn *websocket.Conn) {
	pingMsg := protocol.BaseMessage{
		ID:        uuid.New().String(),
		Target:    0,
		Type:      "ping",
		Timestamp: time.Now(),
		Payload:   map[string]string{"status": "ok"},
	}
	if err := conn.WriteJSON(pingMsg); err != nil {
		log.Error().Err(err).Msg("向上级发送心跳失败")
	} else {
		log.Debug().Msg("已向上级发送心跳")
	}
}
