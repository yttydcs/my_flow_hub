package hub

import (
	"myflowhub/pkg/protocol"

	"github.com/rs/zerolog/log"
)

// HandlerFunc 定义了处理特定消息类型的函数的签名
type HandlerFunc func(s *Server, client *Client, msg protocol.BaseMessage)

// Router 结构体用于管理消息类型到处理函数的映射
type Router struct {
	routes map[string]HandlerFunc
}

// NewRouter 创建一个新的 Router 实例
func NewRouter() *Router {
	return &Router{
		routes: make(map[string]HandlerFunc),
	}
}

// HandleFunc 注册一个消息类型和对应的处理函数
func (r *Router) HandleFunc(msgType string, handler HandlerFunc) {
	if _, exists := r.routes[msgType]; exists {
		log.Warn().Str("type", msgType).Msg("处理函数已被覆盖")
	}
	r.routes[msgType] = handler
}

// Serve 查找并执行与消息类型关联的处理函数
func (r *Router) Serve(s *Server, client *Client, msg protocol.BaseMessage) {
	if handler, ok := r.routes[msg.Type]; ok {
		handler(s, client, msg)
	} else {
		log.Warn().Str("type", msg.Type).Msg("收到未知的消息类型")
	}
}
