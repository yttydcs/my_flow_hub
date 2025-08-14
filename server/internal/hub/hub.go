package hub

import (
	"encoding/json"
	"myflowhub/pkg/protocol"
	"net/http"
	"regexp"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// HubMessage is a message sent from a client to the hub.
type HubMessage struct {
	Client  *Client
	Message []byte
}

// Server 结构体代表一个服务端实例
type Server struct {
	Upgrader   websocket.Upgrader
	ParentAddr string
	ListenAddr string
	HardwareID string
	DeviceID   uint64
	SecretKey  string

	Clients    map[uint64]*Client
	ParentSend chan []byte
	Broadcast  chan *HubMessage
	Register   chan *Client
	Unregister chan *Client
	router     *Router
	Syslog     interface {
		Info(source, message string, details any) error
		Error(source, message string, details any) error
	} // updated interface to include Error method
}

// isValidVarName 检查变量名是否有效
var IsValidVarName = regexp.MustCompile(`^[\p{Han}A-Za-z0-9_]+$`).MatchString

// NewServer 创建一个新的服务端实例
func NewServer(parentAddr, listenAddr, hardwareID string) *Server {
	s := &Server{
		ParentAddr: parentAddr,
		ListenAddr: listenAddr,
		HardwareID: hardwareID,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		Clients:    make(map[uint64]*Client),
		ParentSend: make(chan []byte, 256),
		Broadcast:  make(chan *HubMessage, 256),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		router:     NewRouter(),
	}
	s.registerRoutes()
	return s
}

// Run 启动 hub 的主循环
func (s *Server) Run() {
	for {
		select {
		case c := <-s.Register:
			log.Info().Msg("一个新客户端已连接，等待认证...")
			if s.Syslog != nil {
				_ = s.Syslog.Info("hub", "client connected", map[string]any{"ip": c.RemoteAddr, "ua": c.UserAgent})
			}
		case client := <-s.Unregister:
			if client.DeviceID != 0 {
				if _, ok := s.Clients[client.DeviceID]; ok {
					delete(s.Clients, client.DeviceID)
					close(client.Send)
					log.Info().Uint64("clientID", client.DeviceID).Int("total_clients", len(s.Clients)).Msg("客户端已从 Hub 注销")
					if s.Syslog != nil {
						_ = s.Syslog.Info("hub", "client disconnected", map[string]any{"deviceUID": client.DeviceID, "ip": client.RemoteAddr, "ua": client.UserAgent})
					}
				}
			}
		case hubMessage := <-s.Broadcast:
			s.routeMessage(hubMessage)
		}
	}
}

// routeMessage 解析并路由来自客户端的消息
func (s *Server) routeMessage(hubMessage *HubMessage) {
	var msg protocol.BaseMessage
	if err := json.Unmarshal(hubMessage.Message, &msg); err != nil {
		log.Warn().Err(err).Msg("无法解析JSON消息")
		return
	}

	sourceClient := hubMessage.Client

	// 检查是否为认证或注册消息
	isAuthOrRegister := msg.Type == "auth_request" || msg.Type == "manager_auth" || msg.Type == "register_request"

	// 如果客户端未认证，且消息不是认证/注册类型，则拒绝处理
	if sourceClient.DeviceID == 0 && !isAuthOrRegister {
		log.Warn().Str("type", msg.Type).Msg("匿名客户端尝试发送非认证/注册消息")
		return
	}

	// 为已认证的客户端消息设置来源ID
	if sourceClient.DeviceID != 0 {
		msg.Source = sourceClient.DeviceID
	}

	s.router.Serve(s, sourceClient, msg)
}

// registerRoutes 注册所有消息类型的处理函数
func (s *Server) registerRoutes() {
	// 核心功能
	s.router.HandleFunc("msg_send", routeGenericMessage)
}

// RegisterRoute 注册一个消息类型和对应的处理函数
func (s *Server) RegisterRoute(msgType string, handler HandlerFunc) {
	s.router.HandleFunc(msgType, handler)
}

// routeGenericMessage 处理通用的点对点或广播消息
func routeGenericMessage(s *Server, client *Client, msg protocol.BaseMessage) {
	messageBytes := mustMarshal(msg)

	if client, ok := s.Clients[msg.Target]; ok {
		select {
		case client.Send <- messageBytes:
		default:
			log.Warn().Uint64("clientID", msg.Target).Msg("客户端发送缓冲区已满，消息被丢弃")
		}
		return
	}

	if msg.Target == s.DeviceID {
		log.Info().Interface("msg", msg).Msg("消息被本地处理")
		return
	}

	if msg.Target == 0 {
		log.Info().Msg("正在处理广播消息...")
		for id, c := range s.Clients {
			if id != client.DeviceID {
				select {
				case c.Send <- messageBytes:
				default:
					log.Warn().Uint64("clientID", id).Msg("客户端发送缓冲区已满，消息被丢弃")
				}
			}
		}
		if s.ParentAddr != "" {
			s.ParentSend <- messageBytes
		}
		return
	}

	if s.ParentAddr != "" {
		log.Info().Uint64("target", msg.Target).Msg("目标不在本地，向上级转发")
		s.ParentSend <- messageBytes
	} else {
		log.Warn().Uint64("target", msg.Target).Msg("目标未找到，且无上级可转发")
	}
}

// Start 启动服务
func (s *Server) Start() {
	s.Bootstrap()
	go s.Run()

	if s.ParentAddr != "" {
		go s.connectToParent()
	}

	http.HandleFunc("/ws", s.HandleSubordinateConnection)
	log.Info().Str("address", s.ListenAddr).Msg("服务端启动，监听下级连接")
	if err := http.ListenAndServe(s.ListenAddr, nil); err != nil {
		log.Fatal().Err(err).Msg("无法启动监听服务")
	}
}

func mustMarshal(msg protocol.BaseMessage) []byte {
	bytes, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return bytes
}

// SendResponse 发送一个通用的成功响应
func (s *Server) SendResponse(client *Client, originalID string, payload map[string]interface{}) {
	response := protocol.BaseMessage{
		ID:        uuid.New().String(),
		Source:    s.DeviceID,
		Target:    client.DeviceID,
		Type:      "response",
		Timestamp: time.Now(),
		Payload:   payload,
	}
	payload["original_id"] = originalID
	client.Send <- mustMarshal(response)
}

// SendErrorResponse 发送一个错误响应
func (s *Server) SendErrorResponse(client *Client, originalID, errorMsg string) {
	if s.Syslog != nil {
		_ = s.Syslog.Error("controller", "error response", map[string]any{
			"error":      errorMsg,
			"originalId": originalID,
			"deviceUID":  client.DeviceID,
			"ip":         client.RemoteAddr,
			"ua":         client.UserAgent,
			"stack":      string(debug.Stack()),
		})
	}
	s.SendResponse(client, originalID, map[string]interface{}{
		"success": false,
		"error":   errorMsg,
	})
}

// NotifyVarChange 通知变量变更
func (s *Server) NotifyVarChange(targetDeviceID, sourceDeviceID uint64, variables map[string]interface{}) {
	if targetClient, ok := s.Clients[targetDeviceID]; ok {
		notification := protocol.BaseMessage{
			ID:        uuid.New().String(),
			Source:    sourceDeviceID,
			Target:    targetDeviceID,
			Type:      "var_notify",
			Timestamp: time.Now(),
			Payload: map[string]interface{}{
				"variables": variables,
			},
		}
		targetClient.Send <- mustMarshal(notification)
	}
}
