package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"myflowhub/manager/internal/services"

	"github.com/rs/zerolog/log"
)

// DeviceHandler 设备处理器
type DeviceHandler struct {
	deviceService *services.DeviceService
}

// NewDeviceHandler 创建设备处理器实例
func NewDeviceHandler(deviceService *services.DeviceService) *DeviceHandler {
	return &DeviceHandler{
		deviceService: deviceService,
	}
}

// HandleGetDevices 处理获取设备列表
func (h *DeviceHandler) HandleGetDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := h.deviceService.GetAllDevices()
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Failed to get devices: "+err.Error())
		return
	}

	h.writeJSON(w, map[string]interface{}{
		"success": true,
		"data":    devices,
	})
}

// HandleCreateDevice 处理创建设备
func (h *DeviceHandler) HandleCreateDevice(w http.ResponseWriter, r *http.Request) {
	var req services.CreateDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	device, err := h.deviceService.CreateDevice(req)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.writeJSON(w, map[string]interface{}{
		"success": true,
		"message": "Device created successfully",
		"data":    device,
	})
}

// HandleUpdateDevice 处理更新设备
func (h *DeviceHandler) HandleUpdateDevice(w http.ResponseWriter, r *http.Request) {
	var req services.UpdateDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	device, err := h.deviceService.UpdateDevice(req)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.writeJSON(w, map[string]interface{}{
		"success": true,
		"message": "Device updated successfully",
		"data":    device,
	})
}

// HandleDeleteDevice 处理删除设备
func (h *DeviceHandler) HandleDeleteDevice(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID uint64 `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.deviceService.DeleteDevice(req.ID); err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.writeJSON(w, map[string]interface{}{
		"success": true,
		"message": "Device deleted successfully",
	})
}

// HandleGetDeviceByID 处理根据ID获取设备
func (h *DeviceHandler) HandleGetDeviceByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		h.writeError(w, http.StatusBadRequest, "Device ID is required")
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid device ID")
		return
	}

	device, err := h.deviceService.GetDeviceByID(id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "Device not found")
		return
	}

	h.writeJSON(w, map[string]interface{}{
		"success": true,
		"data":    device,
	})
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
