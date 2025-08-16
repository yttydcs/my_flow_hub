package handlers

import (
	"encoding/json"
	"myflowhub/manager/internal/client"
	"net/http"
	"time"

	binproto "myflowhub/pkg/protocol/binproto"
)

type UserHandler struct{ hubClient *client.HubClient }

func NewUserHandler(hc *client.HubClient) *UserHandler { return &UserHandler{hubClient: hc} }

func (h *UserHandler) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	if h.hubClient != nil && h.hubClient.IsConnected() {
		if pld, err := h.hubClient.SendBinaryRequest(binproto.TypeUserListReq, binproto.TypeUserListResp, binproto.EncodeUserMeReq(token), 5*time.Second); err == nil {
			if _, items, derr := binproto.DecodeUserListResp(pld); derr == nil {
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

func (h *UserHandler) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	var body struct{ Username, DisplayName, Password string }
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if h.hubClient != nil && h.hubClient.IsConnected() {
		payload := binproto.EncodeUserCreateReq(token, body.Username, body.DisplayName, body.Password)
		if pld, err := h.hubClient.SendBinaryRequest(binproto.TypeUserCreateReq, binproto.TypeUserCreateResp, payload, 5*time.Second); err == nil {
			if _, id, derr := binproto.DecodeUserCreateResp(pld); derr == nil {
				h.writeJSON(w, map[string]any{"success": true, "id": id})
				return
			}
		} else if err != client.ErrTimeout {
			h.writeError(w, http.StatusBadGateway, err.Error())
			return
		}
	}
	h.writeError(w, http.StatusBadGateway, "hub error or timeout")
}

func (h *UserHandler) HandleUpdateUser(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	var body struct {
		ID          uint64  `json:"id"`
		DisplayName *string `json:"displayName"`
		Password    *string `json:"password"`
		Disabled    *bool   `json:"disabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if h.hubClient != nil && h.hubClient.IsConnected() {
		payload := binproto.EncodeUserUpdateReq(token, body.ID, body.DisplayName, body.Password, body.Disabled)
		if pld, err := h.hubClient.SendBinaryRequest(binproto.TypeUserUpdateReq, binproto.TypeOKResp, payload, 5*time.Second); err == nil {
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

func (h *UserHandler) HandleDeleteUser(w http.ResponseWriter, r *http.Request) {
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
	if h.hubClient != nil && h.hubClient.IsConnected() {
		payload := binproto.EncodeUserDeleteReq(token, body.ID)
		if pld, err := h.hubClient.SendBinaryRequest(binproto.TypeUserDeleteReq, binproto.TypeOKResp, payload, 5*time.Second); err == nil {
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

// 列出用户权限
func (h *UserHandler) HandleListUserPerms(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	var body struct {
		UserID uint64 `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if h.hubClient != nil && h.hubClient.IsConnected() {
		payload := binproto.EncodeUserPermListReq(token, body.UserID)
		if pld, err := h.hubClient.SendBinaryRequest(binproto.TypeUserPermListReq, binproto.TypeUserPermListResp, payload, 5*time.Second); err == nil {
			if _, items, derr := binproto.DecodeUserPermListResp(pld); derr == nil {
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

// 添加用户权限
func (h *UserHandler) HandleAddUserPerm(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	var body struct {
		UserID uint64 `json:"userId"`
		Node   string `json:"node"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if h.hubClient != nil && h.hubClient.IsConnected() {
		payload := binproto.EncodeUserPermAddReq(token, body.UserID, body.Node)
		if pld, err := h.hubClient.SendBinaryRequest(binproto.TypeUserPermAddReq, binproto.TypeOKResp, payload, 5*time.Second); err == nil {
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

// 移除用户权限
func (h *UserHandler) HandleRemoveUserPerm(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	var body struct {
		UserID uint64 `json:"userId"`
		Node   string `json:"node"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if h.hubClient != nil && h.hubClient.IsConnected() {
		payload := binproto.EncodeUserPermRemoveReq(token, body.UserID, body.Node)
		if pld, err := h.hubClient.SendBinaryRequest(binproto.TypeUserPermRemoveReq, binproto.TypeOKResp, payload, 5*time.Second); err == nil {
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

// 自助更新资料（仅自身）
func (h *UserHandler) HandleSelfUpdate(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	var body struct {
		DisplayName string `json:"displayName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if h.hubClient != nil && h.hubClient.IsConnected() {
		payload := binproto.EncodeUserSelfUpdateReq(token, body.DisplayName)
		if pld, err := h.hubClient.SendBinaryRequest(binproto.TypeUserSelfUpdateReq, binproto.TypeOKResp, payload, 5*time.Second); err == nil {
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

// 自助修改密码（校验旧密码）
func (h *UserHandler) HandleSelfPassword(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	var body struct{ OldPassword, NewPassword string }
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if h.hubClient != nil && h.hubClient.IsConnected() {
		payload := binproto.EncodeUserSelfPasswordReq(token, body.OldPassword, body.NewPassword)
		if pld, err := h.hubClient.SendBinaryRequest(binproto.TypeUserSelfPasswordReq, binproto.TypeOKResp, payload, 5*time.Second); err == nil {
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

func (h *UserHandler) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
func (h *UserHandler) writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": msg})
}
