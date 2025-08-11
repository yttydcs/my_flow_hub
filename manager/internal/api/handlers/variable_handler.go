package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"myflowhub/manager/internal/services"

	"github.com/rs/zerolog/log"
)

// VariableHandler 变量处理器
type VariableHandler struct {
	variableService *services.VariableService
}

// NewVariableHandler 创建变量处理器实例
func NewVariableHandler(variableService *services.VariableService) *VariableHandler {
	return &VariableHandler{
		variableService: variableService,
	}
}

// HandleGetVariables 处理获取变量列表
func (h *VariableHandler) HandleGetVariables(w http.ResponseWriter, r *http.Request) {
	deviceIDStr := r.URL.Query().Get("deviceId")

	var variables []interface{}

	if deviceIDStr != "" {
		deviceID, parseErr := strconv.ParseUint(deviceIDStr, 10, 64)
		if parseErr != nil {
			h.writeError(w, http.StatusBadRequest, "Invalid device ID")
			return
		}

		deviceVariables, getErr := h.variableService.GetVariablesByDeviceID(deviceID)
		if getErr != nil {
			h.writeError(w, http.StatusInternalServerError, "Failed to get variables: "+getErr.Error())
			return
		}

		for _, v := range deviceVariables {
			variables = append(variables, v)
		}
	} else {
		allVariables, getErr := h.variableService.GetAllVariables()
		if getErr != nil {
			h.writeError(w, http.StatusInternalServerError, "Failed to get variables: "+getErr.Error())
			return
		}

		for _, v := range allVariables {
			variables = append(variables, v)
		}
	}

	h.writeJSON(w, map[string]interface{}{
		"success": true,
		"data":    variables,
	})
}

// HandleCreateVariable 处理创建变量
func (h *VariableHandler) HandleCreateVariable(w http.ResponseWriter, r *http.Request) {
	var req services.CreateVariableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	variable, err := h.variableService.CreateVariable(req)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.writeJSON(w, map[string]interface{}{
		"success": true,
		"message": "Variable created successfully",
		"data":    variable,
	})
}

// HandleUpdateVariable 处理更新变量
func (h *VariableHandler) HandleUpdateVariable(w http.ResponseWriter, r *http.Request) {
	var req services.UpdateVariableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	variable, err := h.variableService.UpdateVariable(req)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.writeJSON(w, map[string]interface{}{
		"success": true,
		"message": "Variable updated successfully",
		"data":    variable,
	})
}

// HandleDeleteVariable 处理删除变量
func (h *VariableHandler) HandleDeleteVariable(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID uint64 `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.variableService.DeleteVariable(req.ID); err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.writeJSON(w, map[string]interface{}{
		"success": true,
		"message": "Variable deleted successfully",
	})
}

// HandleGetVariableByID 处理根据ID获取变量
func (h *VariableHandler) HandleGetVariableByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		h.writeError(w, http.StatusBadRequest, "Variable ID is required")
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid variable ID")
		return
	}

	variable, err := h.variableService.GetVariableByID(id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "Variable not found")
		return
	}

	h.writeJSON(w, map[string]interface{}{
		"success": true,
		"data":    variable,
	})
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
