package handlers

import (
	"encoding/json"
	"myflowhub/manager/internal/client"
	"myflowhub/pkg/protocol"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type LogHandler struct{ hubClient *client.HubClient }

func NewLogHandler(hc *client.HubClient) *LogHandler { return &LogHandler{hubClient: hc} }

type LogListBody struct {
	Keyword  string `json:"keyword"`
	Level    string `json:"level"`
	Source   string `json:"source"`
	StartAt  *int64 `json:"startAt"`
	EndAt    *int64 `json:"endAt"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
}

func (h *LogHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	var body LogListBody
	if r.Method == http.MethodPost {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	payload := map[string]interface{}{
		"userKey":  token,
		"keyword":  body.Keyword,
		"level":    body.Level,
		"source":   body.Source,
		"startAt":  body.StartAt,
		"endAt":    body.EndAt,
		"page":     body.Page,
		"pageSize": body.PageSize,
	}
	req := protocol.BaseMessage{ID: uuid.New().String(), Type: "systemlog_list", Payload: payload}
	resp, err := h.hubClient.SendRequest(req, 5*time.Second)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, resp.Payload)
}

func (h *LogHandler) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
func (h *LogHandler) writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": msg})
}
