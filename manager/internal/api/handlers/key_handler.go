package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"myflowhub/manager/internal/client"
	"myflowhub/pkg/protocol"

	"github.com/google/uuid"
)

type KeyHandler struct{ hubClient *client.HubClient }

func NewKeyHandler(hc *client.HubClient) *KeyHandler { return &KeyHandler{hubClient: hc} }

func (h *KeyHandler) HandleListKeys(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	// 透传简化：去掉前缀
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	req := protocol.BaseMessage{ID: uuid.New().String(), Type: "key_list", Payload: map[string]interface{}{"token": token}}
	resp, err := h.hubClient.SendRequest(req, 5*time.Second)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, resp.Payload)
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
	body["token"] = token
	req := protocol.BaseMessage{ID: uuid.New().String(), Type: "key_create", Payload: body}
	resp, err := h.hubClient.SendRequest(req, 5*time.Second)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, resp.Payload)
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
	body["token"] = token
	req := protocol.BaseMessage{ID: uuid.New().String(), Type: "key_update", Payload: body}
	resp, err := h.hubClient.SendRequest(req, 5*time.Second)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, resp.Payload)
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
	req := protocol.BaseMessage{ID: uuid.New().String(), Type: "key_delete", Payload: map[string]interface{}{"token": token, "id": body.ID}}
	resp, err := h.hubClient.SendRequest(req, 5*time.Second)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, resp.Payload)
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
