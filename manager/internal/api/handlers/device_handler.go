package handlers

import (
	"encoding/json"
	"myflowhub/manager/internal/client"
	binproto "myflowhub/pkg/protocol/binproto"
	"net/http"
	"time"

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
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	// 优先二进制
	if h.hubClient != nil && h.hubClient.IsConnected() {
		payload := binproto.EncodeQueryNodesReq(token)
		if resp, err := h.hubClient.SendBinaryRequest(binproto.TypeQueryNodesReq, binproto.TypeQueryNodesResp, payload, 5*time.Second); err == nil {
			if _, items, err2 := binproto.DecodeQueryNodesResp(resp); err2 == nil {
				// 直接返回为向后兼容的 JSON 结构：{ success:true, data:[devices] }
				h.writeJSON(w, map[string]any{"success": true, "data": items})
				return
			}
		} else if err != client.ErrTimeout {
			// 非超时：直接报网关错误，避免落入 JSON 回退触发匿名警告
			h.writeError(w, http.StatusBadGateway, "hub error: "+err.Error())
			return
		}
	}
	// 二进制失败即返回
	h.writeError(w, http.StatusBadGateway, "hub error or timeout")
}

// HandleCreateDevice 处理创建设备
func (h *DeviceHandler) HandleCreateDevice(w http.ResponseWriter, r *http.Request) {
	var reqBody map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	// 优先二进制
	if h.hubClient != nil && h.hubClient.IsConnected() {
		// 构建 DeviceItem
		item := binproto.DeviceItem{}
		if v, ok := reqBody["ID"].(float64); ok {
			item.ID = uint64(v)
		}
		if v, ok := reqBody["DeviceUID"].(float64); ok {
			item.DeviceUID = uint64(v)
		}
		if v, ok := reqBody["HardwareID"].(string); ok {
			item.HardwareID = v
		}
		if v, ok := reqBody["Role"].(string); ok {
			item.Role = v
		}
		if v, ok := reqBody["Name"].(string); ok {
			item.Name = v
		}
		if v, ok := reqBody["ParentID"].(float64); ok {
			vv := uint64(v)
			item.ParentID = &vv
		} else if v, ok := reqBody["parentId"].(float64); ok {
			vv := uint64(v)
			item.ParentID = &vv
		}
		if v, ok := reqBody["OwnerUserID"].(float64); ok {
			vv := uint64(v)
			item.OwnerUserID = &vv
		}
		payload := binproto.EncodeCreateDeviceReq(token, item)
		// 期待 OK；若收到 ERR 或解码异常，直接向前端报错，避免回退 JSON 触发匿名警告
		if resp, err := h.hubClient.SendBinaryRequest(binproto.TypeCreateDeviceReq, binproto.TypeOKResp, payload, 5*time.Second); err == nil {
			if _, code, msg, e2 := binproto.DecodeOKResp(resp); e2 == nil {
				if code == 0 {
					h.writeJSON(w, map[string]any{"success": true})
					return
				}
				h.writeError(w, http.StatusForbidden, string(msg))
				return
			}
		} else if err == client.ErrTimeout {
			// 仅超时回退
		} else {
			h.writeError(w, http.StatusBadGateway, "hub error: "+err.Error())
			return
		}
	}
	h.writeError(w, http.StatusBadGateway, "hub error or timeout")
}

// HandleUpdateDevice 处理更新设备
func (h *DeviceHandler) HandleUpdateDevice(w http.ResponseWriter, r *http.Request) {
	var reqBody map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	// 优先二进制
	if h.hubClient != nil && h.hubClient.IsConnected() {
		item := binproto.DeviceItem{}
		if v, ok := reqBody["ID"].(float64); ok {
			item.ID = uint64(v)
		}
		if v, ok := reqBody["DeviceUID"].(float64); ok {
			item.DeviceUID = uint64(v)
		}
		if v, ok := reqBody["HardwareID"].(string); ok {
			item.HardwareID = v
		}
		if v, ok := reqBody["Role"].(string); ok {
			item.Role = v
		}
		if v, ok := reqBody["Name"].(string); ok {
			item.Name = v
		}
		if v, ok := reqBody["ParentID"].(float64); ok {
			vv := uint64(v)
			item.ParentID = &vv
		} else if v, ok := reqBody["parentId"].(float64); ok {
			vv := uint64(v)
			item.ParentID = &vv
		}
		if v, ok := reqBody["OwnerUserID"].(float64); ok {
			vv := uint64(v)
			item.OwnerUserID = &vv
		}
		payload := binproto.EncodeUpdateDeviceReq(token, item)
		if resp, err := h.hubClient.SendBinaryRequest(binproto.TypeUpdateDeviceReq, binproto.TypeOKResp, payload, 5*time.Second); err == nil {
			if _, code, msg, e2 := binproto.DecodeOKResp(resp); e2 == nil {
				if code == 0 {
					h.writeJSON(w, map[string]any{"success": true})
					return
				}
				h.writeError(w, http.StatusForbidden, string(msg))
				return
			}
		} else if err == client.ErrTimeout {
			// 仅超时回退
		} else {
			h.writeError(w, http.StatusBadGateway, "hub error: "+err.Error())
			return
		}
	}
	h.writeError(w, http.StatusBadGateway, "hub error or timeout")
}

// HandleDeleteDevice 处理删除设备
func (h *DeviceHandler) HandleDeleteDevice(w http.ResponseWriter, r *http.Request) {
	var reqBody map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	// 优先二进制
	if h.hubClient != nil && h.hubClient.IsConnected() {
		var id uint64
		if v, ok := reqBody["id"].(float64); ok {
			id = uint64(v)
		}
		payload := binproto.EncodeDeleteDeviceReq(id, token)
		if resp, err := h.hubClient.SendBinaryRequest(binproto.TypeDeleteDeviceReq, binproto.TypeOKResp, payload, 5*time.Second); err == nil {
			if _, code, msg, e2 := binproto.DecodeOKResp(resp); e2 == nil {
				if code == 0 {
					h.writeJSON(w, map[string]any{"success": true})
					return
				}
				h.writeError(w, http.StatusForbidden, string(msg))
				return
			}
		} else if err == client.ErrTimeout {
			// 仅超时回退
		} else {
			h.writeError(w, http.StatusBadGateway, "hub error: "+err.Error())
			return
		}
	}
	h.writeError(w, http.StatusBadGateway, "hub error or timeout")
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
