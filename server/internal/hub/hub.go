package hub

import (
	"encoding/json"
	"myflowhub/pkg/protocol"
	"net/http"
	"regexp"

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
		case <-s.Register:
			log.Info().Msg("一个新客户端已连接，等待认证...")
		case client := <-s.Unregister:
			if client.DeviceID != 0 {
				if _, ok := s.Clients[client.DeviceID]; ok {
					delete(s.Clients, client.DeviceID)
					close(client.Send)
					log.Info().Uint64("clientID", client.DeviceID).Int("total_clients", len(s.Clients)).Msg("客户端已从 Hub 注销")
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
	// 认证和注册
	s.router.HandleFunc("auth_request", handleAuthRequest)
	s.router.HandleFunc("manager_auth", handleManagerAuthRequest)
	s.router.HandleFunc("register_request", handleRegisterRequest)

	// 核心功能
	s.router.HandleFunc("var_update", handleVarUpdate)
	s.router.HandleFunc("vars_query", handleVarsQuery)
	s.router.HandleFunc("msg_send", routeGenericMessage)

	// 管理功能
	s.router.HandleFunc("query_nodes", handleQueryNodes)
	s.router.HandleFunc("query_variables", handleQueryVariables)
}

// handleAuthRequest 封装了原始的认证逻辑
func handleAuthRequest(s *Server, client *Client, msg protocol.BaseMessage) {
	if s.handleAuth(client, msg) {
		s.Clients[client.DeviceID] = client
		log.Info().Uint64("clientID", client.DeviceID).Msg("客户端在 Hub 中认证成功并注册")
		go s.syncVarsOnLogin(client)
	}
}

// handleManagerAuthRequest 封装了原始的管理员认证逻辑
func handleManagerAuthRequest(s *Server, client *Client, msg protocol.BaseMessage) {
	if s.handleManagerAuth(client, msg) {
		s.Clients[client.DeviceID] = client
		log.Info().Uint64("clientID", client.DeviceID).Msg("管理员在 Hub 中认证成功并注册")
	}
}

// handleRegisterRequest 封装了原始的注册逻辑
func handleRegisterRequest(s *Server, client *Client, msg protocol.BaseMessage) {
	if s.handleRegister(client, msg) {
		s.Clients[client.DeviceID] = client
		log.Info().Uint64("clientID", client.DeviceID).Msg("客户端在 Hub 中注册成功并注册")
	}
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
