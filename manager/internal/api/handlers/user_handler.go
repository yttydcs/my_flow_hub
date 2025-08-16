package handlers

import (
	"encoding/json"
	"myflowhub/manager/internal/client"
	"net/http"
)

type UserHandler struct{ hubClient *client.HubClient }

func NewUserHandler(hc *client.HubClient) *UserHandler { return &UserHandler{hubClient: hc} }

func (h *UserHandler) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	h.writeError(w, http.StatusNotImplemented, "users management via binary not implemented")
}

func (h *UserHandler) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	h.writeError(w, http.StatusNotImplemented, "users management via binary not implemented")
}

func (h *UserHandler) HandleUpdateUser(w http.ResponseWriter, r *http.Request) {
	h.writeError(w, http.StatusNotImplemented, "users management via binary not implemented")
}

func (h *UserHandler) HandleDeleteUser(w http.ResponseWriter, r *http.Request) {
	h.writeError(w, http.StatusNotImplemented, "users management via binary not implemented")
}

// 列出用户权限
func (h *UserHandler) HandleListUserPerms(w http.ResponseWriter, r *http.Request) {
	h.writeError(w, http.StatusNotImplemented, "users management via binary not implemented")
}

// 添加用户权限
func (h *UserHandler) HandleAddUserPerm(w http.ResponseWriter, r *http.Request) {
	h.writeError(w, http.StatusNotImplemented, "users management via binary not implemented")
}

// 移除用户权限
func (h *UserHandler) HandleRemoveUserPerm(w http.ResponseWriter, r *http.Request) {
	h.writeError(w, http.StatusNotImplemented, "users management via binary not implemented")
}

// 自助更新资料（仅自身）
func (h *UserHandler) HandleSelfUpdate(w http.ResponseWriter, r *http.Request) {
	h.writeError(w, http.StatusNotImplemented, "users management via binary not implemented")
}

// 自助修改密码（校验旧密码）
func (h *UserHandler) HandleSelfPassword(w http.ResponseWriter, r *http.Request) {
	h.writeError(w, http.StatusNotImplemented, "users management via binary not implemented")
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
