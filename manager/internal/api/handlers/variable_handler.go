package handlers

import (
	"encoding/json"
	"myflowhub/manager/internal/client"
	"myflowhub/pkg/protocol"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// VariableHandler 变量处理器
type VariableHandler struct {
	hubClient *client.HubClient
}

// NewVariableHandler 创建变量处理器实例
func NewVariableHandler(hubClient *client.HubClient) *VariableHandler {
	return &VariableHandler{
		hubClient: hubClient,
	}
}

// HandleGetVariables 处理获取变量列表
func (h *VariableHandler) HandleGetVariables(w http.ResponseWriter, r *http.Request) {
	deviceIDStr := r.URL.Query().Get("deviceId")
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	payload := make(map[string]interface{})
	if deviceIDStr != "" {
		payload["deviceId"] = deviceIDStr
	}
	payload["userKey"] = token

	req := protocol.BaseMessage{
		ID:      uuid.New().String(),
		Type:    "query_variables",
		Payload: payload,
	}

	response, err := h.hubClient.SendRequest(req, 5*time.Second)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Failed to get variables from hub: "+err.Error())
		return
	}

	h.writeJSON(w, response.Payload)
}

// HandleCreateVariable 处理创建变量
func (h *VariableHandler) HandleCreateVariable(w http.ResponseWriter, r *http.Request) {
	h.HandleUpdateVariable(w, r)
}

// HandleUpdateVariable 处理更新变量
func (h *VariableHandler) HandleUpdateVariable(w http.ResponseWriter, r *http.Request) {
	var reqBody map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	req := protocol.BaseMessage{
		ID:   uuid.New().String(),
		Type: "var_update",
		Payload: map[string]interface{}{
			"variables": reqBody,
			"userKey":   token,
		},
	}

	if err := h.hubClient.SendMessage(req); err != nil {
		h.writeError(w, http.StatusInternalServerError, "Failed to send update to hub: "+err.Error())
		return
	}

	h.writeJSON(w, map[string]interface{}{
		"success": true,
		"message": "Variable update sent to hub",
	})
}

// HandleDeleteVariable 处理删除变量
func (h *VariableHandler) HandleDeleteVariable(w http.ResponseWriter, r *http.Request) {
	var reqBody struct {
		Variables []string `json:"variables"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	req := protocol.BaseMessage{
		ID:   uuid.New().String(),
		Type: "var_delete",
		Payload: map[string]interface{}{
			"variables": reqBody.Variables,
			"userKey":   token,
		},
	}

	if err := h.hubClient.SendMessage(req); err != nil {
		h.writeError(w, http.StatusInternalServerError, "Failed to send delete to hub: "+err.Error())
		return
	}

	h.writeJSON(w, map[string]interface{}{
		"success": true,
		"message": "Variable delete sent to hub",
	})
}

// HandleGetVariableByID 处理根据ID获取变量
func (h *VariableHandler) HandleGetVariableByID(w http.ResponseWriter, r *http.Request) {
	h.writeError(w, http.StatusNotImplemented, "Get variable by ID not implemented")
}

// writeJSON 写入JSON响应
func (h *VariableHandler) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// writeError 写入错误响应
func (h *VariableHandler) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"message": message,
	})
	log.Error().Int("status", statusCode).Str("error", message).Msg("Variable API error")
}
