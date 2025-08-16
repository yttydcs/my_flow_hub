package api

import (
	"encoding/json"
	"net/http"
	"time"

	"myflowhub/manager/internal/api/handlers"
	"myflowhub/manager/internal/client"
	binproto "myflowhub/pkg/protocol/binproto"

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
	keyHandler := handlers.NewKeyHandler(api.hubClient)
	logHandler := handlers.NewLogHandler(api.hubClient)

	// 简单鉴权：除登录外的接口都需要 Authorization: Bearer <token>
	if path != "auth/login" {
		authz := r.Header.Get("Authorization")
		if authz == "" {
			api.writeError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		// 这里暂不校验 token 合法性，仅要求存在；后续可与后端校验结合
	}

	switch {
	// 登录
	case path == "auth/login" && r.Method == "POST":
		api.handleLogin(w, r)
	case path == "auth/me" && r.Method == "GET":
		api.handleMe(w, r)
	case path == "auth/logout" && r.Method == "POST":
		api.handleLogout(w, r)
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
	// 用户权限
	case path == "users/perms/list" && r.Method == "POST":
		userHandler.HandleListUserPerms(w, r)
	case path == "users/perms/add" && r.Method == "POST":
		userHandler.HandleAddUserPerm(w, r)
	case path == "users/perms/remove" && r.Method == "POST":
		userHandler.HandleRemoveUserPerm(w, r)
	// 自助资料与密码
	case path == "profile" && r.Method == "PUT":
		userHandler.HandleSelfUpdate(w, r)
	case path == "profile/password" && r.Method == "PUT":
		userHandler.HandleSelfPassword(w, r)
	// 密钥管理
	case path == "keys" && r.Method == "GET":
		keyHandler.HandleListKeys(w, r)
	case path == "keys" && r.Method == "POST":
		keyHandler.HandleCreateKey(w, r)
	case path == "keys" && r.Method == "PUT":
		keyHandler.HandleUpdateKey(w, r)
	case path == "keys" && r.Method == "DELETE":
		keyHandler.HandleDeleteKey(w, r)
	case path == "keys/devices" && r.Method == "GET":
		keyHandler.HandleKeyDevices(w, r)
	// 日志
	case path == "logs" && (r.Method == "GET" || r.Method == "POST"):
		logHandler.HandleList(w, r)
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
	// 当前仅支持二进制协议；透传 MSG_SEND 到目标或广播。
	payloadBytes, _ := json.Marshal(req.Payload)
	h := binproto.HeaderV1{TypeID: binproto.TypeMsgSend, MsgID: api.hubClient.NextMsgID(), Source: api.hubClient.GetDeviceID(), Target: req.Target, Timestamp: time.Now().UnixMilli()}
	frame, _ := binproto.EncodeFrame(h, payloadBytes)
	if err := api.hubClient.ConnWriteBinary(frame); err != nil {
		api.writeError(w, http.StatusBadGateway, "Failed to send to hub: "+err.Error())
		return
	}
	api.writeJSON(w, map[string]any{"success": true})
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

	if api.hubClient.IsConnected() {
		// Prefer binary
		if payload, err := api.hubClient.SendBinaryRequest(binproto.TypeUserLoginReq, binproto.TypeUserLoginResp, binproto.EncodeUserLoginReq(creds.Username, creds.Password), 5*time.Second); err == nil {
			// Decode to JSON-like struct
			reqID, keyID, userID, token, username, displayName, perms, derr := binproto.DecodeUserLoginResp(payload)
			if derr == nil {
				api.writeJSON(w, map[string]any{
					"success":     true,
					"original_id": reqID,
					"token":       token,
					"keyId":       keyID,
					"user":        map[string]any{"id": userID, "username": username, "displayName": displayName},
					"permissions": perms,
				})
				return
			}
		}
	}
	api.writeError(w, http.StatusBadGateway, "binary login failed")
}

// handleMe 根据 Authorization 的 userKey 返回用户与权限
func (api *ManagerAPI) handleMe(w http.ResponseWriter, r *http.Request) {
	if !api.hubClient.IsConnected() {
		api.writeError(w, http.StatusServiceUnavailable, "Not connected to hub")
		return
	}
	authz := r.Header.Get("Authorization")
	if len(authz) < 8 {
		api.writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	token := authz[7:]
	if payload, err := api.hubClient.SendBinaryRequest(binproto.TypeUserMeReq, binproto.TypeUserMeResp, binproto.EncodeUserMeReq(token), 5*time.Second); err == nil {
		if reqID, userID, username, displayName, perms, derr := binproto.DecodeUserMeResp(payload); derr == nil {
			api.writeJSON(w, map[string]any{
				"success":     true,
				"original_id": reqID,
				"user":        map[string]any{"id": userID, "username": username, "displayName": displayName},
				"permissions": perms,
			})
			return
		}
	}
	api.writeError(w, http.StatusBadGateway, "binary me failed")
}

// handleLogout 撤销当前 userKey
func (api *ManagerAPI) handleLogout(w http.ResponseWriter, r *http.Request) {
	if !api.hubClient.IsConnected() {
		api.writeError(w, http.StatusServiceUnavailable, "Not connected to hub")
		return
	}
	authz := r.Header.Get("Authorization")
	if len(authz) < 8 {
		api.writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	token := authz[7:]
	if payload, err := api.hubClient.SendBinaryRequest(binproto.TypeUserLogoutReq, binproto.TypeOKResp, binproto.EncodeUserLogoutReq(token), 5*time.Second); err == nil {
		if _, code, msgb, derr := binproto.DecodeOKResp(payload); derr == nil {
			api.writeJSON(w, map[string]any{"success": code == 0, "message": string(msgb)})
			return
		}
	}
	api.writeError(w, http.StatusBadGateway, "binary logout failed")
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
