package main

import (
	"encoding/json"
	"fmt"
	"myflowhub/poc/protocol"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// HubMessage is a message sent from a client to the hub.
type HubMessage struct {
	client  *Client
	message []byte
}

// Server 结构体代表一个服务端实例，并作为客户端连接的中心枢纽
type Server struct {
	upgrader   websocket.Upgrader
	parentAddr string
	listenAddr string
	hardwareID string
	deviceID   uint64
	secretKey  string

	clients    map[*Client]bool
	broadcast  chan *HubMessage
	register   chan *Client
	unregister chan *Client
}

// NewServer 创建一个新的服务端实例
func NewServer(parentAddr, listenAddr, hardwareID string) *Server {
	return &Server{
		parentAddr: parentAddr,
		listenAddr: listenAddr,
		hardwareID: hardwareID,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		broadcast:  make(chan *HubMessage),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

// run 启动 hub 的主循环
func (s *Server) run() {
	for {
		select {
		case client := <-s.register:
			s.clients[client] = true
			log.Info().Int("total_clients", len(s.clients)).Msg("客户端已连接到 Hub")
		case client := <-s.unregister:
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				close(client.send)
				log.Info().Uint64("clientID", client.deviceID).Int("total_clients", len(s.clients)).Msg("客户端已从 Hub 注销")
			}
		case hubMessage := <-s.broadcast:
			s.routeMessage(hubMessage)
		}
	}
}

// routeMessage 解析并路由来自客户端的消息
func (s *Server) routeMessage(hubMessage *HubMessage) {
	var msg protocol.BaseMessage
	if err := json.Unmarshal(hubMessage.message, &msg); err != nil {
		log.Warn().Err(err).Msg("无法解析JSON消息")
		return
	}

	sourceClient := hubMessage.client

	if msg.Type == "auth_request" {
		if s.handleAuth(sourceClient, msg) {
			log.Info().Uint64("clientID", sourceClient.deviceID).Msg("客户端在 Hub 中认证成功")
		}
		return
	}
	if msg.Type == "register_request" {
		if s.handleRegister(sourceClient, msg) {
			log.Info().Uint64("clientID", sourceClient.deviceID).Msg("客户端在 Hub 中注册成功")
		}
		return
	}

	if sourceClient.deviceID == 0 {
		log.Warn().Msg("匿名客户端尝试发送非认证/注册消息")
		return
	}

	msg.Source = sourceClient.deviceID

	if msg.Target == 0 { // Broadcast
		for client := range s.clients {
			if client.deviceID != sourceClient.deviceID {
				select {
				case client.send <- hubMessage.message:
				default:
					close(client.send)
					delete(s.clients, client)
				}
			}
		}
	} else { // Point-to-point
		for client := range s.clients {
			if client.deviceID == msg.Target {
				select {
				case client.send <- hubMessage.message:
				default:
					close(client.send)
					delete(s.clients, client)
				}
				return
			}
		}
		log.Warn().Uint64("targetID", msg.Target).Msg("点对点消息目标未找到")
	}
}

// Start 启动服务
func (s *Server) Start() {
	s.bootstrap() // Ensure the server has a DB record for itself
	go s.run()

	if s.parentAddr != "" {
		go s.connectToParent()
	}

	http.HandleFunc("/ws", s.handleSubordinateConnection)
	log.Info().Str("address", s.listenAddr).Msg("服务端启动，监听下级连接")
	if err := http.ListenAndServe(s.listenAddr, nil); err != nil {
		log.Fatal().Err(err).Msg("无法启动监听服务")
	}
}

// handleAuth and handleRegister are now part of the hub logic
func (s *Server) handleAuth(client *Client, msg protocol.BaseMessage) bool {
	var payload protocol.AuthRequestPayload
	jsonPayload, _ := json.Marshal(msg.Payload)
	json.Unmarshal(jsonPayload, &payload)

	var device Device
	if err := DB.Where("device_uid = ?", payload.DeviceID).First(&device).Error; err != nil {
		log.Warn().Uint64("deviceID", payload.DeviceID).Err(err).Msg("认证失败：设备不存在")
		return false
	}

	if err := bcrypt.CompareHashAndPassword([]byte(device.SecretKeyHash), []byte(payload.SecretKey)); err != nil {
		log.Warn().Uint64("deviceID", payload.DeviceID).Msg("认证失败：密钥不正确")
		return false
	}

	client.deviceID = device.DeviceUID
	return true
}

func (s *Server) handleRegister(client *Client, msg protocol.BaseMessage) bool {
	var payload protocol.RegisterRequestPayload
	jsonPayload, _ := json.Marshal(msg.Payload)
	json.Unmarshal(jsonPayload, &payload)

	if payload.HardwareID == "" {
		return false
	}

	var device Device
	err := DB.Where("hardware_id = ?", payload.HardwareID).First(&device).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return false
	}
	if err == nil {
		return false
	}

	hashedSecret, _ := bcrypt.GenerateFromPassword([]byte("default-secret"), bcrypt.DefaultCost)
	newDevice := Device{
		HardwareID:    payload.HardwareID,
		SecretKeyHash: string(hashedSecret),
		Role:          RoleNode,
	}

	if err := DB.Create(&newDevice).Error; err != nil {
		return false
	}

	client.deviceID = newDevice.DeviceUID

	response := protocol.BaseMessage{
		Type: "register_response",
		Payload: map[string]interface{}{
			"success":   true,
			"deviceId":  newDevice.DeviceUID,
			"secretKey": "default-secret",
		},
	}
	responseBytes, _ := json.Marshal(response)
	client.send <- responseBytes

	return true
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	dbHost := "localhost"
	dbUser := "postgres"
	dbPassword := "123456"
	dbName := "myflowhub"
	dbPort := "5432"

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		dbHost, dbUser, dbPassword, dbName, dbPort)

	postgresDsn := fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%s sslmode=disable",
		dbHost, dbUser, dbPassword, dbPort)

	InitDatabase(dsn, postgresDsn, dbName)

	hub := NewServer("", ":8080", "hub-hardware-id")
	go hub.Start()

	if len(os.Args) > 1 && os.Args[1] == "relay" {
		relay := NewServer("ws://localhost:8080/ws", ":8081", "relay-hardware-id-001")
		go relay.Start()
	}

	select {}
}
