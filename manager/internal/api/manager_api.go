package api

import (
	"encoding/json"
	"net/http"
	"time"

	"myflowhub/manager/internal/api/handlers"
	"myflowhub/manager/internal/client"
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

	deviceHandler := handlers.NewDeviceHandler(api.hubClient)
	variableHandler := handlers.NewVariableHandler(api.hubClient)
	userHandler := handlers.NewUserHandler(api.hubClient)

	switch {
	// 登录
	case path == "auth/login" && r.Method == "POST":
		api.handleLogin(w, r)
	// 设备相关路由
	case path == "nodes" && r.Method == "GET":
		deviceHandler.HandleGetDevices(w, r)
	case path == "nodes" && r.Method == "POST":
		deviceHandler.HandleCreateDevice(w, r)
	case path == "nodes" && r.Method == "PUT":
		deviceHandler.HandleUpdateDevice(w, r)
	case path == "nodes" && r.Method == "DELETE":
		deviceHandler.HandleDeleteDevice(w, r)

	// 变量相关路由
	case path == "variables" && r.Method == "GET":
		variableHandler.HandleGetVariables(w, r)
	case path == "variables" && r.Method == "POST":
		variableHandler.HandleCreateVariable(w, r)
	case path == "variables" && r.Method == "PUT":
		variableHandler.HandleUpdateVariable(w, r)
	case path == "variables" && r.Method == "DELETE":
		variableHandler.HandleDeleteVariable(w, r)

	// 其他路由
	case path == "message" && r.Method == "POST":
		api.handleSendMessage(w, r)
	case path == "debug/db" && r.Method == "GET":
		api.handleDebugDB(w, r)
	// 用户相关（仅管理员）
	case path == "users" && r.Method == "GET":
		userHandler.HandleListUsers(w, r)
	case path == "users" && r.Method == "POST":
		userHandler.HandleCreateUser(w, r)
	case path == "users" && r.Method == "PUT":
		userHandler.HandleUpdateUser(w, r)
	case path == "users" && r.Method == "DELETE":
		userHandler.HandleDeleteUser(w, r)
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

// handleLogin 将登录请求转发为 hub 的 user_login 消息
func (api *ManagerAPI) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !api.hubClient.IsConnected() {
		api.writeError(w, http.StatusServiceUnavailable, "Not connected to hub")
		return
	}

	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid body")
		return
	}

	msg := protocol.BaseMessage{ID: uuid.New().String(), Type: "user_login", Payload: creds, Timestamp: time.Now()}
	resp, err := api.hubClient.SendRequest(msg, 5*time.Second)
	if err != nil {
		api.writeError(w, http.StatusUnauthorized, "login failed")
		return
	}
	api.writeJSON(w, resp.Payload)
}

// handleDebugDB 调试数据库连接和表结构
func (api *ManagerAPI) handleDebugDB(w http.ResponseWriter, r *http.Request) {
	api.writeJSON(w, map[string]interface{}{
		"success": true,
		"message": "Database connection has been removed from the manager.",
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
