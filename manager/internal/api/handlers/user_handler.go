package handlers

import (
	"encoding/json"
	"myflowhub/manager/internal/client"
	"myflowhub/pkg/protocol"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type UserHandler struct{ hubClient *client.HubClient }

func NewUserHandler(hc *client.HubClient) *UserHandler { return &UserHandler{hubClient: hc} }

func (h *UserHandler) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	req := protocol.BaseMessage{ID: uuid.New().String(), Type: "user_list", Payload: map[string]interface{}{"userKey": token}}
	resp, err := h.hubClient.SendRequest(req, 5*time.Second)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, resp.Payload)
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
	req := protocol.BaseMessage{ID: uuid.New().String(), Type: "user_create", Payload: map[string]interface{}{"username": body.Username, "displayName": body.DisplayName, "password": body.Password, "userKey": token}}
	resp, err := h.hubClient.SendRequest(req, 5*time.Second)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, resp.Payload)
}

func (h *UserHandler) HandleUpdateUser(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	var body struct {
		ID          uint64
		DisplayName *string
		Password    *string
		Disabled    *bool
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	req := protocol.BaseMessage{ID: uuid.New().String(), Type: "user_update", Payload: map[string]interface{}{"ID": body.ID, "DisplayName": body.DisplayName, "Password": body.Password, "Disabled": body.Disabled, "userKey": token}}
	resp, err := h.hubClient.SendRequest(req, 5*time.Second)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, resp.Payload)
}

func (h *UserHandler) HandleDeleteUser(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	var body struct{ ID uint64 }
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	req := protocol.BaseMessage{ID: uuid.New().String(), Type: "user_delete", Payload: map[string]interface{}{"ID": body.ID, "userKey": token}}
	resp, err := h.hubClient.SendRequest(req, 5*time.Second)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, resp.Payload)
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
	req := protocol.BaseMessage{ID: uuid.New().String(), Type: "user_perm_list", Payload: map[string]interface{}{"userId": body.UserID, "userKey": token}}
	resp, err := h.hubClient.SendRequest(req, 5*time.Second)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, resp.Payload)
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
	req := protocol.BaseMessage{ID: uuid.New().String(), Type: "user_perm_add", Payload: map[string]interface{}{"userId": body.UserID, "node": body.Node, "userKey": token}}
	resp, err := h.hubClient.SendRequest(req, 5*time.Second)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, resp.Payload)
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
	req := protocol.BaseMessage{ID: uuid.New().String(), Type: "user_perm_remove", Payload: map[string]interface{}{"userId": body.UserID, "node": body.Node, "userKey": token}}
	resp, err := h.hubClient.SendRequest(req, 5*time.Second)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, resp.Payload)
}

// 自助更新资料（仅自身）
func (h *UserHandler) HandleSelfUpdate(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	var body struct{ DisplayName *string }
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	payload := map[string]interface{}{"userKey": token}
	if body.DisplayName != nil {
		payload["displayName"] = *body.DisplayName
	}
	req := protocol.BaseMessage{ID: uuid.New().String(), Type: "user_self_update", Payload: payload}
	resp, err := h.hubClient.SendRequest(req, 5*time.Second)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, resp.Payload)
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
	req := protocol.BaseMessage{ID: uuid.New().String(), Type: "user_self_password", Payload: map[string]interface{}{"userKey": token, "oldPassword": body.OldPassword, "newPassword": body.NewPassword}}
	resp, err := h.hubClient.SendRequest(req, 5*time.Second)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, resp.Payload)
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
