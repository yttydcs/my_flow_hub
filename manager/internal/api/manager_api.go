package api

import (
	"encoding/json"
	"net/http"
	"time"

	"myflowhub/manager/internal/client"
	"myflowhub/pkg/database"
	"myflowhub/pkg/protocol"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// ManagerAPI 管理API结构体
type ManagerAPI struct {
	hubClient *client.HubClient
}

// NewManagerAPI 创建新的管理API实例
func NewManagerAPI(hubClient *client.HubClient) *ManagerAPI {
	return &ManagerAPI{
		hubClient: hubClient,
	}
}

// RegisterRoutes 注册API路由
func (api *ManagerAPI) RegisterRoutes(mux *http.ServeMux) {
	// 启用CORS
	mux.HandleFunc("/api/", api.corsMiddleware(api.handleAPI))
}

// corsMiddleware CORS中间件
func (api *ManagerAPI) corsMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler(w, r)
	}
}

// handleAPI 处理API请求
func (api *ManagerAPI) handleAPI(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[5:] // 去掉 "/api/" 前缀

	switch {
	case path == "nodes" && r.Method == "GET":
		api.handleGetNodes(w, r)
	case path == "nodes" && r.Method == "POST":
		api.handleCreateDevice(w, r)
	case path == "nodes" && r.Method == "PUT":
		api.handleUpdateDevice(w, r)
	case path == "nodes" && r.Method == "DELETE":
		api.handleDeleteDevice(w, r)
	case path == "variables" && r.Method == "GET":
		api.handleGetVariables(w, r)
	case path == "variables" && r.Method == "POST":
		api.handleCreateVariable(w, r)
	case path == "variables" && r.Method == "PUT":
		api.handleUpdateVariable(w, r)
	case path == "variables" && r.Method == "DELETE":
		api.handleDeleteVariable(w, r)
	case path == "message" && r.Method == "POST":
		api.handleSendMessage(w, r)
	case path == "debug/db" && r.Method == "GET":
		api.handleDebugDB(w, r)
	default:
		api.writeError(w, http.StatusNotFound, "API endpoint not found")
	}
}

// handleGetNodes 获取所有节点信息
func (api *ManagerAPI) handleGetNodes(w http.ResponseWriter, r *http.Request) {
	if !api.hubClient.IsConnected() {
		api.writeError(w, http.StatusServiceUnavailable, "Not connected to hub")
		return
	}

	// 通过WebSocket请求节点数据
	msg := protocol.BaseMessage{
		ID:        uuid.New().String(),
		Type:      "query_nodes",
		Timestamp: time.Now(),
		Payload:   map[string]interface{}{},
	}

	if err := api.hubClient.SendMessage(msg); err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to send nodes query")
		return
	}

	// 等待响应
	if response, err := api.hubClient.GetResponse(10 * time.Second); err == nil {
		if payload, ok := response.Payload.(map[string]interface{}); ok {
			if success, _ := payload["success"].(bool); success {
				api.writeJSON(w, map[string]interface{}{
					"success": true,
					"data":    payload["data"],
				})
				return
			}
		}
	}

	api.writeError(w, http.StatusInternalServerError, "Failed to get nodes from hub")
}

// handleGetVariables 获取变量信息
func (api *ManagerAPI) handleGetVariables(w http.ResponseWriter, r *http.Request) {
	if !api.hubClient.IsConnected() {
		api.writeError(w, http.StatusServiceUnavailable, "Not connected to hub")
		return
	}

	deviceIDStr := r.URL.Query().Get("deviceId")

	// 通过WebSocket请求变量数据
	msg := protocol.BaseMessage{
		ID:        uuid.New().String(),
		Type:      "query_variables",
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"deviceId": deviceIDStr,
		},
	}

	if err := api.hubClient.SendMessage(msg); err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to send variables query")
		return
	}

	// 等待响应
	if response, err := api.hubClient.GetResponse(10 * time.Second); err == nil {
		if payload, ok := response.Payload.(map[string]interface{}); ok {
			if success, _ := payload["success"].(bool); success {
				api.writeJSON(w, map[string]interface{}{
					"success": true,
					"data":    payload["data"],
				})
				return
			}
		}
	}

	api.writeError(w, http.StatusInternalServerError, "Failed to get variables from hub")
}

// handleDebugDB 调试数据库连接和表结构
func (api *ManagerAPI) handleDebugDB(w http.ResponseWriter, r *http.Request) {
	// 检查数据库连接
	sqlDB, err := database.DB.DB()
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Database connection error: "+err.Error())
		return
	}

	if err := sqlDB.Ping(); err != nil {
		api.writeError(w, http.StatusInternalServerError, "Database ping failed: "+err.Error())
		return
	}

	// 检查表是否存在
	var tableCount int64
	database.DB.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'device_variables'").Scan(&tableCount)

	// 检查设备数量
	var deviceCount int64
	database.DB.Model(&database.Device{}).Count(&deviceCount)

	// 检查变量数量
	var variableCount int64
	database.DB.Model(&database.DeviceVariable{}).Count(&variableCount)

	// 获取一些样本数据
	var sampleDevices []database.Device
	database.DB.Limit(3).Find(&sampleDevices)

	var sampleVariables []database.DeviceVariable
	database.DB.Preload("Device").Limit(3).Find(&sampleVariables)

	api.writeJSON(w, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"database_connection":           "OK",
			"device_variables_table_exists": tableCount > 0,
			"device_count":                  deviceCount,
			"variable_count":                variableCount,
			"sample_devices":                sampleDevices,
			"sample_variables":              sampleVariables,
		},
	})
}

// UpdateVariableRequest 更新变量请求结构体
type UpdateVariableRequest struct {
	Variables map[string]interface{} `json:"variables"`
}

// handleUpdateVariables 更新变量
func (api *ManagerAPI) handleUpdateVariables(w http.ResponseWriter, r *http.Request) {
	if !api.hubClient.IsConnected() {
		api.writeError(w, http.StatusServiceUnavailable, "Not connected to hub")
		return
	}

	var req UpdateVariableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// 通过WebSocket发送变量更新消息
	msg := protocol.BaseMessage{
		ID:        uuid.New().String(),
		Type:      "var_update",
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"variables": req.Variables,
		},
	}

	if err := api.hubClient.SendMessage(msg); err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to send update message")
		return
	}

	api.writeJSON(w, map[string]interface{}{
		"success": true,
		"message": "Variables update sent",
	})
}

// SendMessageRequest 发送消息请求结构体
type SendMessageRequest struct {
	Target  uint64      `json:"target"`
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// handleSendMessage 发送消息到指定节点
func (api *ManagerAPI) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	if !api.hubClient.IsConnected() {
		api.writeError(w, http.StatusServiceUnavailable, "Not connected to hub")
		return
	}

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// 发送管理指令
	msg := protocol.BaseMessage{
		ID:        uuid.New().String(),
		Type:      req.Type,
		Target:    req.Target,
		Timestamp: time.Now(),
		Payload:   req.Payload,
	}

	if err := api.hubClient.SendMessage(msg); err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to send message")
		return
	}

	// 等待响应（可选）
	if response, err := api.hubClient.GetResponse(5 * time.Second); err == nil {
		api.writeJSON(w, map[string]interface{}{
			"success":  true,
			"message":  "Message sent successfully",
			"response": response,
		})
	} else {
		api.writeJSON(w, map[string]interface{}{
			"success": true,
			"message": "Message sent, no response received",
		})
	}
}

// writeJSON 写入JSON响应
func (api *ManagerAPI) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// writeError 写入错误响应
func (api *ManagerAPI) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   message,
	})
	log.Error().Int("status", statusCode).Str("error", message).Msg("API error")
}

// CreateDeviceRequest 创建设备请求结构体
type CreateDeviceRequest struct {
	HardwareID string  `json:"hardwareId"`
	Name       string  `json:"name"`
	Role       string  `json:"role"`
	ParentID   *uint64 `json:"parentId"`
}

// UpdateDeviceRequest 更新设备请求结构体
type UpdateDeviceRequest struct {
	ID       uint64  `json:"id"`
	Name     string  `json:"name"`
	Role     string  `json:"role"`
	ParentID *uint64 `json:"parentId"`
}

// DeleteDeviceRequest 删除设备请求结构体
type DeleteDeviceRequest struct {
	ID uint64 `json:"id"`
}

// CreateVariableRequest 创建变量请求结构体
type CreateVariableRequest struct {
	Name          string      `json:"name"`
	Value         interface{} `json:"value"`
	OwnerDeviceID uint64      `json:"deviceId"`
}

// UpdateVariableRequest 更新变量请求结构体
type UpdateVariableRequestNew struct {
	ID    uint64      `json:"id"`
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

// DeleteVariableRequest 删除变量请求结构体
type DeleteVariableRequest struct {
	ID uint64 `json:"id"`
}

// handleCreateDevice 创建设备
func (api *ManagerAPI) handleCreateDevice(w http.ResponseWriter, r *http.Request) {
	var req CreateDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// 验证必填字段
	if req.HardwareID == "" || req.Role == "" {
		api.writeError(w, http.StatusBadRequest, "HardwareID and Role are required")
		return
	}

	// 转换角色类型
	var role database.DeviceRole
	switch req.Role {
	case "node":
		role = database.RoleNode
	case "relay":
		role = database.RoleRelay
	case "hub":
		role = database.RoleHub
	case "manager":
		role = database.RoleManager
	default:
		api.writeError(w, http.StatusBadRequest, "Invalid role. Must be one of: node, relay, hub, manager")
		return
	}

	// 创建设备对象
	device := database.Device{
		HardwareID: req.HardwareID,
		Name:       req.Name,
		Role:       role,
		ParentID:   req.ParentID,
	}

	// 保存到数据库
	if err := database.DB.Create(&device).Error; err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to create device: "+err.Error())
		return
	}

	api.writeJSON(w, map[string]interface{}{
		"success": true,
		"message": "Device created successfully",
		"data":    device,
	})
}

// handleUpdateDevice 更新设备
func (api *ManagerAPI) handleUpdateDevice(w http.ResponseWriter, r *http.Request) {
	var req UpdateDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// 查找设备
	var device database.Device
	if err := database.DB.First(&device, req.ID).Error; err != nil {
		api.writeError(w, http.StatusNotFound, "Device not found")
		return
	}

	// 转换角色类型
	var role database.DeviceRole
	switch req.Role {
	case "node":
		role = database.RoleNode
	case "relay":
		role = database.RoleRelay
	case "hub":
		role = database.RoleHub
	case "manager":
		role = database.RoleManager
	default:
		api.writeError(w, http.StatusBadRequest, "Invalid role. Must be one of: node, relay, hub, manager")
		return
	}

	// 更新字段
	device.Name = req.Name
	device.Role = role
	device.ParentID = req.ParentID

	// 保存更改
	if err := database.DB.Save(&device).Error; err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to update device: "+err.Error())
		return
	}

	api.writeJSON(w, map[string]interface{}{
		"success": true,
		"message": "Device updated successfully",
		"data":    device,
	})
}

// handleDeleteDevice 删除设备
func (api *ManagerAPI) handleDeleteDevice(w http.ResponseWriter, r *http.Request) {
	var req DeleteDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// 检查设备是否存在
	var device database.Device
	if err := database.DB.First(&device, req.ID).Error; err != nil {
		api.writeError(w, http.StatusNotFound, "Device not found")
		return
	}

	// 删除设备
	if err := database.DB.Delete(&device).Error; err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to delete device: "+err.Error())
		return
	}

	api.writeJSON(w, map[string]interface{}{
		"success": true,
		"message": "Device deleted successfully",
	})
}

// handleCreateVariable 创建变量
func (api *ManagerAPI) handleCreateVariable(w http.ResponseWriter, r *http.Request) {
	var req CreateVariableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// 验证必填字段
	if req.Name == "" {
		api.writeError(w, http.StatusBadRequest, "Variable name is required")
		return
	}

	// 检查设备是否存在
	var device database.Device
	if err := database.DB.First(&device, req.OwnerDeviceID).Error; err != nil {
		api.writeError(w, http.StatusNotFound, "Device not found")
		return
	}

	// 转换值为JSON格式
	valueJSON, err := json.Marshal(req.Value)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid variable value")
		return
	}

	// 创建变量对象
	variable := database.DeviceVariable{
		VariableName:  req.Name,
		Value:         valueJSON,
		OwnerDeviceID: req.OwnerDeviceID,
	}

	// 保存到数据库
	if err := database.DB.Create(&variable).Error; err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to create variable: "+err.Error())
		return
	}

	// 预加载设备信息
	database.DB.Preload("Device").First(&variable, variable.ID)

	api.writeJSON(w, map[string]interface{}{
		"success": true,
		"message": "Variable created successfully",
		"data":    variable,
	})
}

// handleUpdateVariable 更新变量
func (api *ManagerAPI) handleUpdateVariable(w http.ResponseWriter, r *http.Request) {
	var req UpdateVariableRequestNew
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// 查找变量
	var variable database.DeviceVariable
	if err := database.DB.First(&variable, req.ID).Error; err != nil {
		api.writeError(w, http.StatusNotFound, "Variable not found")
		return
	}

	// 转换值为JSON格式
	valueJSON, err := json.Marshal(req.Value)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid variable value")
		return
	}

	// 更新字段
	variable.VariableName = req.Name
	variable.Value = valueJSON

	// 保存更改
	if err := database.DB.Save(&variable).Error; err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to update variable: "+err.Error())
		return
	}

	// 预加载设备信息
	database.DB.Preload("Device").First(&variable, variable.ID)

	api.writeJSON(w, map[string]interface{}{
		"success": true,
		"message": "Variable updated successfully",
		"data":    variable,
	})
}

// handleDeleteVariable 删除变量
func (api *ManagerAPI) handleDeleteVariable(w http.ResponseWriter, r *http.Request) {
	var req DeleteVariableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// 检查变量是否存在
	var variable database.DeviceVariable
	if err := database.DB.First(&variable, req.ID).Error; err != nil {
		api.writeError(w, http.StatusNotFound, "Variable not found")
		return
	}

	// 删除变量
	if err := database.DB.Delete(&variable).Error; err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to delete variable: "+err.Error())
		return
	}

	api.writeJSON(w, map[string]interface{}{
		"success": true,
		"message": "Variable deleted successfully",
	})
}
