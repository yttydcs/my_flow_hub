package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"myflowhub/manager/internal/client"
	binproto "myflowhub/pkg/protocol/binproto"
)

type KeyHandler struct{ hubClient *client.HubClient }

func NewKeyHandler(hc *client.HubClient) *KeyHandler { return &KeyHandler{hubClient: hc} }

func (h *KeyHandler) HandleListKeys(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	// 透传简化：去掉前缀
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	// 二进制优先
	if h.hubClient != nil && h.hubClient.IsConnected() {
		if pld, err := h.hubClient.SendBinaryRequest(binproto.TypeKeyListReq, binproto.TypeKeyListResp, binproto.EncodeKeyListReq(token), 5*time.Second); err == nil {
			if _, items, derr := binproto.DecodeKeyListResp(pld); derr == nil {
				// 直接返回为 { success:true, data:items }
				h.writeJSON(w, map[string]any{"success": true, "data": items})
				return
			}
		} else if err != client.ErrTimeout {
			h.writeError(w, http.StatusBadGateway, err.Error())
			return
		}
	}
	h.writeError(w, http.StatusBadGateway, "hub error or timeout")
}

func (h *KeyHandler) HandleCreateKey(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	// 二进制优先
	if h.hubClient != nil && h.hubClient.IsConnected() {
		var bindType *string
		if v, ok := body["bindType"].(string); ok {
			bindType = &v
		}
		var bindID *uint64
		if v, ok := body["bindId"].(float64); ok {
			vv := uint64(v)
			bindID = &vv
		}
		var exp *int64
		if v, ok := body["expiresAt"].(float64); ok {
			vv := int64(v)
			exp = &vv
		}
		var max *int32
		if v, ok := body["maxUses"].(float64); ok {
			vv := int32(v)
			max = &vv
		}
		var meta []byte
		if v, ok := body["meta"].(string); ok {
			meta = []byte(v)
		}
		var nodes []string
		if arr, ok := body["nodes"].([]interface{}); ok {
			for _, n := range arr {
				if s, ok := n.(string); ok {
					nodes = append(nodes, s)
				}
			}
		}
		payload := binproto.EncodeKeyCreateReq(token, bindType, bindID, exp, max, meta, nodes)
		if pld, err := h.hubClient.SendBinaryRequest(binproto.TypeKeyCreateReq, binproto.TypeKeyCreateResp, payload, 5*time.Second); err == nil {
			if _, secret, item, nodes, derr := binproto.DecodeKeyCreateResp(pld); derr == nil {
				h.writeJSON(w, map[string]any{"success": true, "data": item, "secret": secret, "nodes": nodes})
				return
			}
		} else if err != client.ErrTimeout {
			h.writeError(w, http.StatusBadGateway, err.Error())
			return
		}
	}
	h.writeError(w, http.StatusBadGateway, "hub error or timeout")
}

func (h *KeyHandler) HandleUpdateKey(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	// 二进制优先
	if h.hubClient != nil && h.hubClient.IsConnected() {
		var item binproto.KeyItem
		if v, ok := body["id"].(float64); ok {
			item.ID = uint64(v)
		}
		if v, ok := body["ownerUserId"].(float64); ok {
			vv := uint64(v)
			item.OwnerUserID = &vv
		}
		if v, ok := body["bindType"].(string); ok {
			item.BindSubjectType = &v
		}
		if v, ok := body["bindId"].(float64); ok {
			vv := uint64(v)
			item.BindSubjectID = &vv
		}
		if v, ok := body["secretHash"].(string); ok {
			item.SecretHash = v
		}
		if v, ok := body["expiresAt"].(float64); ok {
			vv := int64(v)
			item.ExpiresAtSec = &vv
		}
		if v, ok := body["maxUses"].(float64); ok {
			vv := int32(v)
			item.MaxUses = &vv
		}
		if v, ok := body["remainingUses"].(float64); ok {
			vv := int32(v)
			item.RemainingUses = &vv
		}
		if v, ok := body["revoked"].(bool); ok {
			item.Revoked = v
		}
		if v, ok := body["issuedBy"].(float64); ok {
			vv := uint64(v)
			item.IssuedBy = &vv
		}
		if v, ok := body["issuedAt"].(float64); ok {
			item.IssuedAtSec = int64(v)
		}
		if v, ok := body["meta"].(string); ok {
			item.Meta = []byte(v)
		}
		payload := binproto.EncodeKeyUpdateReq(token, item)
		if pld, err := h.hubClient.SendBinaryRequest(binproto.TypeKeyUpdateReq, binproto.TypeOKResp, payload, 5*time.Second); err == nil {
			if _, code, msg, derr := binproto.DecodeOKResp(pld); derr == nil {
				h.writeJSON(w, map[string]any{"success": code == 0, "message": string(msg)})
				return
			}
		} else if err != client.ErrTimeout {
			h.writeError(w, http.StatusBadGateway, err.Error())
			return
		}
	}
	h.writeError(w, http.StatusBadGateway, "hub error or timeout")
}

func (h *KeyHandler) HandleDeleteKey(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	var body struct {
		ID uint64 `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	// 二进制优先
	if h.hubClient != nil && h.hubClient.IsConnected() {
		payload := binproto.EncodeKeyDeleteReq(token, body.ID)
		if pld, err := h.hubClient.SendBinaryRequest(binproto.TypeKeyDeleteReq, binproto.TypeOKResp, payload, 5*time.Second); err == nil {
			if _, code, msg, derr := binproto.DecodeOKResp(pld); derr == nil {
				h.writeJSON(w, map[string]any{"success": code == 0, "message": string(msg)})
				return
			}
		} else if err != client.ErrTimeout {
			h.writeError(w, http.StatusBadGateway, err.Error())
			return
		}
	}
	h.writeError(w, http.StatusBadGateway, "hub error or timeout")
}

// HandleKeyDevices 列出当前用户在创建密钥时可见的设备
func (h *KeyHandler) HandleKeyDevices(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	if h.hubClient != nil && h.hubClient.IsConnected() {
		if pld, err := h.hubClient.SendBinaryRequest(binproto.TypeKeyDevicesReq, binproto.TypeKeyDevicesResp, binproto.EncodeKeyDevicesReq(token), 5*time.Second); err == nil {
			if _, items, derr := binproto.DecodeKeyDevicesResp(pld); derr == nil {
				h.writeJSON(w, map[string]any{"success": true, "data": items})
				return
			}
		} else if err != client.ErrTimeout {
			h.writeError(w, http.StatusBadGateway, err.Error())
			return
		}
	}
	h.writeError(w, http.StatusBadGateway, "hub error or timeout")
}

func (h *KeyHandler) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
func (h *KeyHandler) writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": msg})
}
