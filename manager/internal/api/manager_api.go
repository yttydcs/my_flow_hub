package api

import (
	"encoding/json"
	"net/http"
	"time"

	"myflowhub/manager/internal/api/handlers"
	"myflowhub/manager/internal/client"
	"myflowhub/manager/internal/repository"
	"myflowhub/manager/internal/services"
	"myflowhub/pkg/database"
	"myflowhub/pkg/protocol"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// ManagerAPI 管理API结构体
type ManagerAPI struct {
	hubClient       *client.HubClient
	deviceHandler   *handlers.DeviceHandler
	variableHandler *handlers.VariableHandler
}

// NewManagerAPI 创建新的管理API实例
func NewManagerAPI(hubClient *client.HubClient) *ManagerAPI {
	// 初始化Repository层
	deviceRepo := repository.NewDeviceRepository(database.DB)
	variableRepo := repository.NewVariableRepository(database.DB)

	// 初始化Service层
	deviceService := services.NewDeviceService(deviceRepo, variableRepo, hubClient)
	variableService := services.NewVariableService(variableRepo, deviceRepo, hubClient)

	// 初始化Handler层
	deviceHandler := handlers.NewDeviceHandler(deviceService)
	variableHandler := handlers.NewVariableHandler(variableService)

	return &ManagerAPI{
		hubClient:       hubClient,
		deviceHandler:   deviceHandler,
		variableHandler: variableHandler,
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
	// 设备相关路由
	case path == "nodes" && r.Method == "GET":
		api.deviceHandler.HandleGetDevices(w, r)
	case path == "nodes" && r.Method == "POST":
		api.deviceHandler.HandleCreateDevice(w, r)
	case path == "nodes" && r.Method == "PUT":
		api.deviceHandler.HandleUpdateDevice(w, r)
	case path == "nodes" && r.Method == "DELETE":
		api.deviceHandler.HandleDeleteDevice(w, r)

	// 变量相关路由
	case path == "variables" && r.Method == "GET":
		api.variableHandler.HandleGetVariables(w, r)
	case path == "variables" && r.Method == "POST":
		api.variableHandler.HandleCreateVariable(w, r)
	case path == "variables" && r.Method == "PUT":
		api.variableHandler.HandleUpdateVariable(w, r)
	case path == "variables" && r.Method == "DELETE":
		api.variableHandler.HandleDeleteVariable(w, r)

	// 其他路由
	case path == "message" && r.Method == "POST":
		api.handleSendMessage(w, r)
	case path == "debug/db" && r.Method == "GET":
		api.handleDebugDB(w, r)
	default:
		api.writeError(w, http.StatusNotFound, "API endpoint not found")
	}
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
		"message": message,
	})
	log.Error().Int("status", statusCode).Str("error", message).Msg("API error")
}
