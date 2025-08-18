package hub

import (
	bin "myflowhub/pkg/protocol/binproto"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// HubMessage is a message sent from a client to the hub.
type HubMessage struct {
	Client   *Client
	Message  []byte
	IsBinary bool
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
	binRoutes  map[uint16]func(s *Server, c *Client, h bin.HeaderV1, payload []byte)
	Syslog     interface {
		Info(source, message string, details any) error
		Error(source, message string, details any) error
	} // updated interface to include Error method

	// nonce cache for future ParentAuth anti-replay (reserved)
	nonces   map[string]int64
	noncesMu sync.Mutex
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
		binRoutes:  make(map[uint16]func(*Server, *Client, bin.HeaderV1, []byte)),
		nonces:     make(map[string]int64),
	}
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
	sourceClient := hubMessage.Client
	if hubMessage.IsBinary {
		// 二进制路径
		h, payload, err := bin.DecodeFrame(hubMessage.Message)
		if err != nil {
			log.Warn().Err(err).Str("remoteAddr", sourceClient.RemoteAddr).Msg("无法解析二进制帧")
			return
		}
		log.Debug().Uint16("typeID", h.TypeID).Uint64("msgID", h.MsgID).Uint64("source", h.Source).Uint64("target", h.Target).Msg("收到二进制帧")
		if handler, ok := s.binRoutes[h.TypeID]; ok {
			handler(s, sourceClient, h, payload)
			return
		}
		switch h.TypeID {
		case bin.TypeManagerAuthReq:
			log.Warn().Msg("未注册 ManagerAuth 二进制处理器")
		case bin.TypeParentAuthReq:
			log.Warn().Msg("未注册 ParentAuth 二进制处理器（应由 binroutes 注册）")
		case bin.TypeMsgSend:
			// 透传：当 Target ≠ Hub
			if h.Target != s.DeviceID && h.Target != 0 {
				// 发往目标或上级，不解析 payload
				if client, ok := s.Clients[h.Target]; ok {
					// 直接转发原始帧
					select {
					case client.Send <- hubMessage.Message:
						log.Debug().Uint64("target", h.Target).Msg("消息已放入目标客户端 channel")
					default:
						log.Warn().Uint64("target", h.Target).Msg("目标客户端 channel 已满，消息被丢弃")
					}
				} else if s.ParentAddr != "" {
					s.ParentSend <- hubMessage.Message
				} else {
					log.Warn().Uint64("target", h.Target).Msg("目标未找到，且无上级可转发")
				}
			} else if h.Target == 0 {
				// 广播
				for id, c := range s.Clients {
					if id != sourceClient.DeviceID {
						select {
						case c.Send <- hubMessage.Message:
							log.Debug().Uint64("target", id).Msg("广播消息已放入目标客户端 channel")
						default:
							log.Warn().Uint64("target", id).Msg("目标客户端 channel 已满，广播消息被丢弃")
						}
					}
				}
				if s.ParentAddr != "" {
					s.ParentSend <- hubMessage.Message
				}
			} else {
				log.Info().Msg("MSG_SEND 发往 Hub，自行处理 payload（后续实现）")
			}
		default:
			log.Warn().Uint16("typeID", h.TypeID).Msg("未知 TypeID")
		}
		return
	}

	// JSON 路径已移除
}

// RegisterBinRoute registers a binary TypeID handler
func (s *Server) RegisterBinRoute(typeID uint16, handler func(s *Server, c *Client, h bin.HeaderV1, payload []byte)) {
	s.binRoutes[typeID] = handler
}

// SendBin sends a binary frame to client
func (s *Server) SendBin(c *Client, typeID uint16, msgID uint64, target uint64, payload []byte) {
	h := bin.HeaderV1{TypeID: typeID, MsgID: msgID, Source: s.DeviceID, Target: target, Timestamp: time.Now().UnixMilli()}
	frame, err := bin.EncodeFrame(h, payload)
	if err != nil {
		log.Error().Err(err).Msg("EncodeFrame failed")
		return
	}
	select {
	case c.Send <- frame:
	default:
	}
}

// JSON 路由与兼容占位符已彻底移除

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

// JSON 回复/通知相关方法已删除（二进制专用）
