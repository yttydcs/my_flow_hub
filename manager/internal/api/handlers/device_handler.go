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

// DeviceHandler 设备处理器
type DeviceHandler struct {
	hubClient *client.HubClient
}

// NewDeviceHandler 创建设备处理器实例
func NewDeviceHandler(hubClient *client.HubClient) *DeviceHandler {
	return &DeviceHandler{
		hubClient: hubClient,
	}
}

// HandleGetDevices 处理获取设备列表
func (h *DeviceHandler) HandleGetDevices(w http.ResponseWriter, r *http.Request) {
	req := protocol.BaseMessage{
		ID:   uuid.New().String(),
		Type: "query_nodes",
	}

	response, err := h.hubClient.SendRequest(req, 5*time.Second)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Failed to get devices from hub: "+err.Error())
		return
	}

	h.writeJSON(w, response.Payload)
}

// HandleCreateDevice 处理创建设备
func (h *DeviceHandler) HandleCreateDevice(w http.ResponseWriter, r *http.Request) {
	h.writeError(w, http.StatusNotImplemented, "Create device not implemented")
}

// HandleUpdateDevice 处理更新设备
func (h *DeviceHandler) HandleUpdateDevice(w http.ResponseWriter, r *http.Request) {
	h.writeError(w, http.StatusNotImplemented, "Update device not implemented")
}

// HandleDeleteDevice 处理删除设备
func (h *DeviceHandler) HandleDeleteDevice(w http.ResponseWriter, r *http.Request) {
	h.writeError(w, http.StatusNotImplemented, "Delete device not implemented")
}

// HandleGetDeviceByID 处理根据ID获取设备
func (h *DeviceHandler) HandleGetDeviceByID(w http.ResponseWriter, r *http.Request) {
	h.writeError(w, http.StatusNotImplemented, "Get device by ID not implemented")
}

// writeJSON 写入JSON响应
func (h *DeviceHandler) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// writeError 写入错误响应
func (h *DeviceHandler) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"message": message,
	})
	log.Error().Int("status", statusCode).Str("error", message).Msg("Device API error")
}
