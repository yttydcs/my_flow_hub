package hub

import (
	"fmt"
	"myflowhub/pkg/database"
	bin "myflowhub/pkg/protocol/binproto"
	"net/http"
	"regexp"
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
			// 打印十六进制预览与长度，便于定位协议问题
			previewLen := len(hubMessage.Message)
			if previewLen > 64 {
				previewLen = 64
			}
			hex := fmt.Sprintf("% x", hubMessage.Message[:previewLen])
			log.Warn().Err(err).
				Str("remoteAddr", sourceClient.RemoteAddr).
				Int("len", len(hubMessage.Message)).
				Str("hex64", hex).
				Msg("无法解析二进制帧")
			return
		}
		log.Debug().Uint16("typeID", h.TypeID).Uint64("msgID", h.MsgID).Uint64("source", h.Source).Uint64("target", h.Target).Msg("收到二进制帧")
		// 审批门控：认证类请求除外，未审批的连接拒绝后续操作
		switch h.TypeID {
		case bin.TypeManagerAuthReq, bin.TypeParentAuthReq, bin.TypeUserLoginReq, bin.TypeUserMeReq, bin.TypeUserLogoutReq:
			// 认证与自助接口放行
		default:
			if sourceClient.DeviceID != 0 {
				var approved bool
				// 查询一次数据库；也可考虑加入缓存
				var cnt int64
				if err := database.DB.Model(&database.Device{}).
					Where("device_uid = ? AND approved = ?", sourceClient.DeviceID, true).
					Count(&cnt).Error; err == nil {
					approved = cnt > 0
				}
				if !approved {
					// 直接返回 ErrResp（禁止使用任何网络功能，也不能向其他节点发送消息）
					pl := bin.EncodeErrResp(h.MsgID, 403, []byte("device not approved"))
					frame, _ := bin.EncodeFrame(bin.HeaderV1{TypeID: bin.TypeErrResp, MsgID: h.MsgID, Source: s.DeviceID, Target: sourceClient.DeviceID, Timestamp: time.Now().UnixMilli()}, pl)
					select {
					case sourceClient.Send <- frame:
					default:
					}
					return
				}
			} else {
				// 未认证的连接亦禁止访问非认证接口
				pl := bin.EncodeErrResp(h.MsgID, 401, []byte("unauthorized"))
				frame, _ := bin.EncodeFrame(bin.HeaderV1{TypeID: bin.TypeErrResp, MsgID: h.MsgID, Source: s.DeviceID, Target: 0, Timestamp: time.Now().UnixMilli()}, pl)
				select {
				case sourceClient.Send <- frame:
				default:
				}
				return
			}
		}
		if handler, ok := s.binRoutes[h.TypeID]; ok {
			handler(s, sourceClient, h, payload)
			return
		}
		switch h.TypeID {
		case bin.TypeManagerAuthReq:
			log.Warn().Msg("未注册 ManagerAuth 二进制处理器")
		case bin.TypeParentAuthReq:
			log.Warn().Msg("未注册 ParentAuth 二进制处理器（应由 binroutes 注册）")
		case bin.TypeOKResp, bin.TypeErrResp:
			// 某些客户端可能会向 Hub 回传通用响应帧；Hub 端无需处理，静默丢弃以减少噪音
			// 附加诊断：打印头部十六进制，便于对齐问题排查
			if len(hubMessage.Message) >= bin.HeaderSizeV1 {
				hb := hubMessage.Message[:bin.HeaderSizeV1]
				log.Debug().Uint16("typeID", h.TypeID).Uint64("msgID", h.MsgID).Str("hex", fmt.Sprintf("% x", hb)).Msg("收到通用响应帧，已忽略")
			} else {
				log.Debug().Uint16("typeID", h.TypeID).Uint64("msgID", h.MsgID).Int("len", len(hubMessage.Message)).Msg("收到通用响应帧，已忽略")
			}
		case bin.TypeMsgSend:
			// 自回环：当目标就是当前客户端自身 UID，直接回送一份，便于本地回环测试
			if sourceClient.DeviceID != 0 && h.Target == sourceClient.DeviceID {
				select {
				case sourceClient.Send <- hubMessage.Message:
					log.Debug().Uint64("target", h.Target).Msg("MSG_SEND 自回环回送给自身")
				default:
					log.Warn().Uint64("target", h.Target).Msg("自回环回送失败：channel 已满")
				}
				return
			}
			// 透传：当 Target ≠ Hub（自身设备）且 ≠ 广播
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
			if len(hubMessage.Message) >= bin.HeaderSizeV1 {
				hb := hubMessage.Message[:bin.HeaderSizeV1]
				log.Warn().Uint16("typeID", h.TypeID).Str("hex", fmt.Sprintf("% x", hb)).Msg("未知 TypeID")
			} else {
				log.Warn().Uint16("typeID", h.TypeID).Int("len", len(hubMessage.Message)).Msg("未知 TypeID")
			}
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
		log.Debug().Uint64("msgID", msgID).Uint16("typeID", typeID).Uint64("target", target).Msg("frame enqueued to client.Send")
	default:
		// 队列已满，丢弃并记录
		log.Warn().Uint64("msgID", msgID).Uint16("typeID", typeID).Uint64("target", target).Int("queueCap", cap(c.Send)).Msg("client.Send full, drop frame")
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
