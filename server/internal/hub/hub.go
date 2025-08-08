package hub

import (
	"encoding/json"
	"fmt"
	"myflowhub/pkg/database"
	"myflowhub/pkg/protocol"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
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
}

// isValidVarName 检查变量名是否有效
var IsValidVarName = regexp.MustCompile(`^[\p{Han}A-Za-z0-9_]+$`).MatchString

// NewServer 创建一个新的服务端实例
func NewServer(parentAddr, listenAddr, hardwareID string) *Server {
	return &Server{
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
	}
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

	if msg.Type == "auth_request" {
		if s.handleAuth(sourceClient, msg) {
			s.Clients[sourceClient.DeviceID] = sourceClient
			log.Info().Uint64("clientID", sourceClient.DeviceID).Msg("客户端在 Hub 中认证成功并注册")
			go s.syncVarsOnLogin(sourceClient)
		}
		return
	}
	if msg.Type == "register_request" {
		if s.handleRegister(sourceClient, msg) {
			s.Clients[sourceClient.DeviceID] = sourceClient
			log.Info().Uint64("clientID", sourceClient.DeviceID).Msg("客户端在 Hub 中注册成功并注册")
		}
		return
	}

	if sourceClient.DeviceID == 0 {
		log.Warn().Msg("匿名客户端尝试发送非认证/注册消息")
		return
	}
	msg.Source = sourceClient.DeviceID

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
		for id, client := range s.Clients {
			if id != sourceClient.DeviceID {
				select {
				case client.Send <- messageBytes:
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

func (s *Server) handleAuth(client *Client, msg protocol.BaseMessage) bool {
	var payload protocol.AuthRequestPayload
	jsonPayload, _ := json.Marshal(msg.Payload)
	json.Unmarshal(jsonPayload, &payload)

	var device database.Device
	if err := database.DB.Where("device_uid = ?", payload.DeviceID).First(&device).Error; err != nil {
		log.Warn().Uint64("deviceID", payload.DeviceID).Err(err).Msg("认证失败：设备不存在")
		return false
	}

	if err := bcrypt.CompareHashAndPassword([]byte(device.SecretKeyHash), []byte(payload.SecretKey)); err != nil {
		log.Warn().Uint64("deviceID", payload.DeviceID).Msg("认证失败：密钥不正确")
		return false
	}

	client.DeviceID = device.DeviceUID
	return true
}

func (s *Server) handleRegister(client *Client, msg protocol.BaseMessage) bool {
	var payload protocol.RegisterRequestPayload
	jsonPayload, _ := json.Marshal(msg.Payload)
	json.Unmarshal(jsonPayload, &payload)

	if payload.HardwareID == "" {
		return false
	}

	var device database.Device
	err := database.DB.Where("hardware_id = ?", payload.HardwareID).First(&device).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return false
	}
	if err == nil {
		return false
	}

	hashedSecret, _ := bcrypt.GenerateFromPassword([]byte("default-secret"), bcrypt.DefaultCost)
	newDevice := database.Device{
		HardwareID:    payload.HardwareID,
		SecretKeyHash: string(hashedSecret),
		Role:          database.RoleNode,
		Name:          payload.HardwareID,
	}

	if err := database.DB.Create(&newDevice).Error; err != nil {
		return false
	}

	client.DeviceID = newDevice.DeviceUID

	response := protocol.BaseMessage{
		ID:   msg.ID,
		Type: "register_response",
		Payload: map[string]interface{}{
			"success":   true,
			"deviceId":  newDevice.DeviceUID,
			"secretKey": "default-secret",
		},
	}
	client.Send <- mustMarshal(response)
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

	var updatedCount int
	for fqdn, value := range variables {
		var deviceIdentifier, varName string
		if strings.Contains(fqdn, ".") {
			parts := strings.SplitN(fqdn, ".", 2)
			deviceIdentifier = parts[0]
			varName = parts[1]
		} else {
			deviceIdentifier = fmt.Sprintf("[%d]", client.DeviceID)
			varName = fqdn
		}

		if !IsValidVarName(varName) {
			log.Warn().Str("varName", varName).Msg("无效的变量名，已跳过")
			continue
		}

		var targetDevice database.Device
		var err error
		if strings.HasPrefix(deviceIdentifier, "[") && strings.HasSuffix(deviceIdentifier, "]") {
			err = database.DB.Where("device_uid = ?", strings.Trim(deviceIdentifier, "[]")).First(&targetDevice).Error
		} else {
			err = database.DB.Where("name = ?", strings.Trim(deviceIdentifier, "()")).First(&targetDevice).Error
		}
		if err != nil {
			continue
		}

		jsonValue, _ := json.Marshal(value)
		variable := database.DeviceVariable{
			OwnerDeviceID: targetDevice.ID,
			VariableName:  varName,
			Value:         datatypes.JSON(jsonValue),
		}
		if database.DB.Where("owner_device_id = ? AND variable_name = ?", targetDevice.ID, varName).Assign(database.DeviceVariable{Value: datatypes.JSON(jsonValue)}).FirstOrCreate(&variable).Error == nil {
			updatedCount++
		}
	}
	log.Info().Int("count", updatedCount).Msg("变量已更新")
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
			deviceIdentifier = fmt.Sprintf("[%d]", client.DeviceID)
			varName = query
		}

		var targetDevice database.Device
		var err error
		if strings.HasPrefix(deviceIdentifier, "[") && strings.HasSuffix(deviceIdentifier, "]") {
			err = database.DB.Where("device_uid = ?", strings.Trim(deviceIdentifier, "[]")).First(&targetDevice).Error
		} else {
			err = database.DB.Where("name = ?", strings.Trim(deviceIdentifier, "()")).First(&targetDevice).Error
		}
		if err != nil {
			results[i] = nil
			continue
		}

		var variable database.DeviceVariable
		if err := database.DB.Where("owner_device_id = ? AND variable_name = ?", targetDevice.ID, varName).First(&variable).Error; err != nil {
			results[i] = nil
		} else {
			var val interface{}
			json.Unmarshal(variable.Value, &val)
			results[i] = val
		}
	}

	response := protocol.BaseMessage{
		ID:        uuid.New().String(),
		Source:    s.DeviceID,
		Target:    client.DeviceID,
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
	client.Send <- mustMarshal(response)
}

func (s *Server) syncVarsOnLogin(client *Client) {
	var variables []database.DeviceVariable
	database.DB.Where("owner_device_id = ?", client.DeviceID).Find(&variables)

	if len(variables) == 0 {
		return
	}

	varsMap := make(map[string]interface{})
	for _, v := range variables {
		var val interface{}
		json.Unmarshal(v.Value, &val)
		varsMap[v.VariableName] = val
	}

	s.notifyVarChange(client.DeviceID, s.DeviceID, varsMap)
	log.Info().Uint64("clientID", client.DeviceID).Msg("已完成上线变量同步")
}

func (s *Server) notifyVarChange(targetDeviceID, sourceDeviceID uint64, variables map[string]interface{}) {
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

func mustMarshal(msg protocol.BaseMessage) []byte {
	bytes, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return bytes
}
