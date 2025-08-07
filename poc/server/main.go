package main

import (
	"encoding/json"
	"fmt"
	"myflowhub/poc/protocol"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// HubMessage is a message sent from a client to the hub.
type HubMessage struct {
	client  *Client
	message []byte
}

// Server 结构体代表一个服务端实例
type Server struct {
	upgrader   websocket.Upgrader
	parentAddr string
	listenAddr string
	hardwareID string
	deviceID   uint64
	secretKey  string

	clients    map[uint64]*Client
	parentSend chan []byte
	broadcast  chan *HubMessage
	register   chan *Client
	unregister chan *Client
}

// isValidVarName 检查变量名是否有效
var isValidVarName = regexp.MustCompile(`^[\p{Han}A-Za-z0-9_]+$`).MatchString

// NewServer 创建一个新的服务端实例
func NewServer(parentAddr, listenAddr, hardwareID string) *Server {
	return &Server{
		parentAddr: parentAddr,
		listenAddr: listenAddr,
		hardwareID: hardwareID,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		clients:    make(map[uint64]*Client),
		parentSend: make(chan []byte, 256),
		broadcast:  make(chan *HubMessage, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// run 启动 hub 的主循环
func (s *Server) run() {
	for {
		select {
		case <-s.register:
			log.Info().Msg("一个新客户端已连接，等待认证...")
		case client := <-s.unregister:
			if client.deviceID != 0 {
				if _, ok := s.clients[client.deviceID]; ok {
					delete(s.clients, client.deviceID)
					close(client.send)
					log.Info().Uint64("clientID", client.deviceID).Int("total_clients", len(s.clients)).Msg("客户端已从 Hub 注销")
				}
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
			s.clients[sourceClient.deviceID] = sourceClient
			log.Info().Uint64("clientID", sourceClient.deviceID).Msg("客户端在 Hub 中认证成功并注册")
			go s.syncVarsOnLogin(sourceClient) // 上线后同步变量
		}
		return
	}
	if msg.Type == "register_request" {
		if s.handleRegister(sourceClient, msg) {
			s.clients[sourceClient.deviceID] = sourceClient
			log.Info().Uint64("clientID", sourceClient.deviceID).Msg("客户端在 Hub 中注册成功并注册")
		}
		return
	}

	if sourceClient.deviceID == 0 {
		log.Warn().Msg("匿名客户端尝试发送非认证/注册消息")
		return
	}
	msg.Source = sourceClient.deviceID

	switch msg.Type {
	case "var_update":
		s.handleVarUpdate(sourceClient, msg)
	case "vars_query":
		s.handleVarsQuery(sourceClient, msg)
	case "msg_send":
		s.routeGenericMessage(sourceClient, msg)
	default:
		log.Warn().Str("type", msg.Type).Msg("收到未知的消息类型")
	}
}

// routeGenericMessage 处理通用的点对点或广播消息
func (s *Server) routeGenericMessage(sourceClient *Client, msg protocol.BaseMessage) {
	messageBytes := mustMarshal(msg)

	if client, ok := s.clients[msg.Target]; ok {
		select {
		case client.send <- messageBytes:
		default:
			log.Warn().Uint64("clientID", msg.Target).Msg("客户端发送缓冲区已满，消息被丢弃")
		}
		return
	}

	if msg.Target == s.deviceID {
		log.Info().Interface("msg", msg).Msg("消息被本地处理")
		return
	}

	if msg.Target == 0 {
		log.Info().Msg("正在处理广播消息...")
		for id, client := range s.clients {
			if id != sourceClient.deviceID {
				select {
				case client.send <- messageBytes:
				default:
					log.Warn().Uint64("clientID", id).Msg("客户端发送缓冲区已满，消息被丢弃")
				}
			}
		}
		if s.parentAddr != "" {
			s.parentSend <- messageBytes
		}
		return
	}

	if s.parentAddr != "" {
		log.Info().Uint64("target", msg.Target).Msg("目标不在本地，向上级转发")
		s.parentSend <- messageBytes
	} else {
		log.Warn().Uint64("target", msg.Target).Msg("目标未找到，且无上级可转发")
	}
}

// Start 启动服务
func (s *Server) Start() {
	s.bootstrap()
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
		Name:          payload.HardwareID,
	}

	if err := DB.Create(&newDevice).Error; err != nil {
		return false
	}

	client.deviceID = newDevice.DeviceUID

	response := protocol.BaseMessage{
		ID:   msg.ID,
		Type: "register_response",
		Payload: map[string]interface{}{
			"success":   true,
			"deviceId":  newDevice.DeviceUID,
			"secretKey": "default-secret",
		},
	}
	client.send <- mustMarshal(response)
	return true
}

func (s *Server) handleVarUpdate(client *Client, msg protocol.BaseMessage) {
	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return
	}

	variables, ok := payload["variables"].(map[string]interface{})
	if !ok {
		return
	}

	updatedVarsByDevice := make(map[uint64]map[string]interface{})

	for fqdn, value := range variables {
		var deviceIdentifier, varName string
		if strings.Contains(fqdn, ".") {
			parts := strings.SplitN(fqdn, ".", 2)
			deviceIdentifier = parts[0]
			varName = parts[1]
		} else {
			deviceIdentifier = fmt.Sprintf("[%d]", client.deviceID)
			varName = fqdn
		}

		if !isValidVarName(varName) {
			log.Warn().Str("varName", varName).Msg("无效的变量名，已跳过")
			continue
		}

		var targetDevice Device
		var err error
		if strings.HasPrefix(deviceIdentifier, "[") && strings.HasSuffix(deviceIdentifier, "]") {
			err = DB.Where("device_uid = ?", strings.Trim(deviceIdentifier, "[]")).First(&targetDevice).Error
		} else {
			err = DB.Where("name = ?", strings.Trim(deviceIdentifier, "()")).First(&targetDevice).Error
		}
		if err != nil {
			continue
		}

		jsonValue, _ := json.Marshal(value)
		variable := DeviceVariable{
			OwnerDeviceID: targetDevice.ID,
			VariableName:  varName,
			Value:         datatypes.JSON(jsonValue),
		}
		if DB.Where("owner_device_id = ? AND variable_name = ?", targetDevice.ID, varName).Assign(DeviceVariable{Value: datatypes.JSON(jsonValue)}).FirstOrCreate(&variable).Error == nil {
			if _, ok := updatedVarsByDevice[targetDevice.DeviceUID]; !ok {
				updatedVarsByDevice[targetDevice.DeviceUID] = make(map[string]interface{})
			}
			updatedVarsByDevice[targetDevice.DeviceUID][varName] = value
		}
	}

	// 发送下行通知
	for deviceID, vars := range updatedVarsByDevice {
		s.notifyVarChange(deviceID, msg.Source, vars)
	}
}

func (s *Server) handleVarsQuery(client *Client, msg protocol.BaseMessage) {
	var payload protocol.VarsQueryPayload
	jsonPayload, _ := json.Marshal(msg.Payload)
	json.Unmarshal(jsonPayload, &payload)

	results := make([]interface{}, len(payload.Queries))

	for i, query := range payload.Queries {
		var deviceIdentifier, varName string
		if strings.Contains(query, ".") {
			parts := strings.SplitN(query, ".", 2)
			deviceIdentifier = parts[0]
			varName = parts[1]
		} else {
			deviceIdentifier = fmt.Sprintf("[%d]", client.deviceID)
			varName = query
		}

		var targetDevice Device
		var err error
		if strings.HasPrefix(deviceIdentifier, "[") && strings.HasSuffix(deviceIdentifier, "]") {
			err = DB.Where("device_uid = ?", strings.Trim(deviceIdentifier, "[]")).First(&targetDevice).Error
		} else {
			err = DB.Where("name = ?", strings.Trim(deviceIdentifier, "()")).First(&targetDevice).Error
		}
		if err != nil {
			results[i] = nil
			continue
		}

		var variable DeviceVariable
		if err := DB.Where("owner_device_id = ? AND variable_name = ?", targetDevice.ID, varName).First(&variable).Error; err != nil {
			results[i] = nil
		} else {
			var val interface{}
			json.Unmarshal(variable.Value, &val)
			results[i] = val
		}
	}

	response := protocol.BaseMessage{
		ID:        uuid.New().String(),
		Source:    s.deviceID,
		Target:    client.deviceID,
		Type:      "response",
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"success":     true,
			"original_id": msg.ID,
			"data": map[string]interface{}{
				"results": results,
			},
		},
	}
	client.send <- mustMarshal(response)
}

func (s *Server) syncVarsOnLogin(client *Client) {
	var variables []DeviceVariable
	DB.Where("owner_device_id = ?", client.deviceID).Find(&variables)

	if len(variables) == 0 {
		return
	}

	varsMap := make(map[string]interface{})
	for _, v := range variables {
		var val interface{}
		json.Unmarshal(v.Value, &val)
		varsMap[v.VariableName] = val
	}

	s.notifyVarChange(client.deviceID, s.deviceID, varsMap)
	log.Info().Uint64("clientID", client.deviceID).Msg("已完成上线变量同步")
}

func (s *Server) notifyVarChange(targetDeviceID, sourceDeviceID uint64, variables map[string]interface{}) {
	if targetClient, ok := s.clients[targetDeviceID]; ok {
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
		targetClient.send <- mustMarshal(notification)
	}
}

func mustMarshal(msg protocol.BaseMessage) []byte {
	bytes, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return bytes
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	dbHost := "localhost"
	dbUser := "postgres"
	dbPassword := "your_postgres_password"
	dbName := "myflowhub"
	dbPort := "5432"

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		dbHost, dbUser, dbPassword, dbName, dbPort)

	postgresDsn := fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%s sslmode=disable",
		dbHost, dbUser, dbPassword, dbPort)

	InitDatabase(dsn, postgresDsn, dbName)

	hub := NewServer("", ":8080", "hub-001")
	go hub.Start()

	if len(os.Args) > 1 && os.Args[1] == "relay" {
		relay := NewServer("ws://localhost:8080/ws", ":8081", "relay-001")
		go relay.Start()
	}

	select {}
}
