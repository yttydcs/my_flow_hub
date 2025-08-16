package hub

import (
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// connectToParent establishes and maintains the connection to the parent server.
func (s *Server) connectToParent() {
	u, err := url.Parse(s.ParentAddr)
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
		if _, _, err := conn.ReadMessage(); err != nil {
			log.Error().Err(err).Msg("从上级读取消息失败")
			return
		}
		log.Info().Msg("收到来自上级的消息(二进制未实现，忽略)")
		// Here we could handle messages from parent, e.g., route them to our children
	}
}

// writePumpToParent handles writing messages to the parent.
func (s *Server) writePumpToParent(conn *websocket.Conn, done chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-s.ParentSend:
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
				log.Error().Err(err).Msg("向上级写入消息失败")
				return
			}
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
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
	log.Info().Msg("二进制父链路认证未实现，跳过")
}
