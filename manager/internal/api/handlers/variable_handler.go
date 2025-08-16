package handlers

import (
	"encoding/json"
	"fmt"
	"myflowhub/manager/internal/client"
	binproto "myflowhub/pkg/protocol/binproto"
	"net/http"
	"strconv"
	"time"

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
	// 二进制优先：VarListReq
	if h.hubClient != nil && h.hubClient.IsConnected() {
		var devUIDPtr *uint64
		if deviceIDStr != "" {
			if v, err := strconv.ParseUint(deviceIDStr, 10, 64); err == nil {
				devUIDPtr = &v
			}
		}
		payload := binproto.EncodeVarListReq(token, devUIDPtr)
		if resp, err := h.hubClient.SendBinaryRequest(binproto.TypeVarListReq, binproto.TypeVarListResp, payload, 5*time.Second); err == nil {
			if _, items, e2 := binproto.DecodeVarListResp(resp); e2 == nil {
				// 兼容前端期望的结构：[]DeviceVariable
				out := make([]map[string]any, 0, len(items))
				for _, it := range items {
					out = append(out, map[string]any{
						"ID":            it.ID,
						"OwnerDeviceID": it.OwnerDeviceID,
						"VariableName":  it.Name,
						"Value":         json.RawMessage(append([]byte(nil), it.Value...)),
						"CreatedAt":     time.Unix(it.CreatedAtSec, 0).Format(time.RFC3339),
						"UpdatedAt":     time.Unix(it.UpdatedAtSec, 0).Format(time.RFC3339),
						// 提供最小 Device 对象以兼容前端 selectedDeviceVariables 依赖 Device.DeviceUID
						"Device": map[string]any{"DeviceUID": it.OwnerDeviceUID},
						// Device 预留，JSON 路径会预载；如需可额外查询
					})
				}
				h.writeJSON(w, map[string]any{"success": true, "data": out})
				return
			}
		} else if err != client.ErrTimeout {
			h.writeError(w, http.StatusBadGateway, "hub error: "+err.Error())
			return
		}
	}
	h.writeError(w, http.StatusBadGateway, "hub error or timeout")
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
	// 二进制优先：构建 VarUpdateReq（把 reqBody 的键值展平为 items）
	if h.hubClient != nil && h.hubClient.IsConnected() {
		items := make([]binproto.VarUpdateItem, 0, len(reqBody))
		for fqdn, val := range reqBody {
			// 解析形如 [UID].name 或 (Name).var
			// 仅支持 [UID].name 形式
			var uid uint64
			var name string
			if n, err := parseVarFQDN(fqdn); err == nil {
				uid = n.deviceUID
				name = n.name
			} else {
				continue
			}
			vb, _ := json.Marshal(val)
			items = append(items, binproto.VarUpdateItem{DeviceUID: uid, Name: name, Value: vb})
		}
		payload := binproto.EncodeVarUpdateReq(token, items)
		if resp, err := h.hubClient.SendBinaryRequest(binproto.TypeVarUpdateReq, binproto.TypeOKResp, payload, 5*time.Second); err == nil {
			if _, code, msg, e2 := binproto.DecodeOKResp(resp); e2 == nil {
				if code == 0 {
					h.writeJSON(w, map[string]any{"success": true})
					return
				}
				h.writeError(w, http.StatusForbidden, string(msg))
				return
			}
		} else if err != client.ErrTimeout {
			h.writeError(w, http.StatusBadGateway, "hub error: "+err.Error())
			return
		}
	}
	h.writeError(w, http.StatusBadGateway, "hub error or timeout")
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
	if h.hubClient != nil && h.hubClient.IsConnected() {
		items := make([]binproto.VarDeleteItem, 0, len(reqBody.Variables))
		for _, fqdn := range reqBody.Variables {
			if n, err := parseVarFQDN(fqdn); err == nil {
				items = append(items, binproto.VarDeleteItem{DeviceUID: n.deviceUID, Name: n.name})
			}
		}
		payload := binproto.EncodeVarDeleteReq(token, items)
		if resp, err := h.hubClient.SendBinaryRequest(binproto.TypeVarDeleteReq, binproto.TypeOKResp, payload, 5*time.Second); err == nil {
			if _, code, msg, e2 := binproto.DecodeOKResp(resp); e2 == nil {
				if code == 0 {
					h.writeJSON(w, map[string]any{"success": true})
					return
				}
				h.writeError(w, http.StatusForbidden, string(msg))
				return
			}
		} else if err != client.ErrTimeout {
			h.writeError(w, http.StatusBadGateway, "hub error: "+err.Error())
			return
		}
	}
	h.writeError(w, http.StatusBadGateway, "hub error or timeout")
}

// parseVarFQDN 解析形如 "[12345].temp" 的变量标识
type varAddr struct {
	deviceUID uint64
	name      string
}

func parseVarFQDN(fqdn string) (varAddr, error) {
	// 简易解析：[UID].name
	if len(fqdn) < 4 || fqdn[0] != '[' {
		return varAddr{}, fmt.Errorf("bad fqdn")
	}
	i := 1
	for i < len(fqdn) && fqdn[i] != ']' {
		i++
	}
	if i >= len(fqdn) || i+2 >= len(fqdn) || fqdn[i] != ']' || fqdn[i+1] != '.' {
		return varAddr{}, fmt.Errorf("bad fqdn")
	}
	uidStr := fqdn[1:i]
	name := fqdn[i+2:]
	v, err := strconv.ParseUint(uidStr, 10, 64)
	if err != nil || name == "" {
		return varAddr{}, fmt.Errorf("bad fqdn")
	}
	return varAddr{deviceUID: v, name: name}, nil
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
