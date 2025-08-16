package handlers

import (
	"encoding/json"
	"myflowhub/manager/internal/client"
	"net/http"
	"time"

	binproto "myflowhub/pkg/protocol/binproto"
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
	// binary first
	if pld, err := h.hubClient.SendBinaryRequest(binproto.TypeSystemLogListReq, binproto.TypeSystemLogListResp,
		binproto.EncodeSystemLogListReq(token, body.Level, body.Source, body.Keyword, valueOrZero(body.StartAt), valueOrZero(body.EndAt), int32(body.Page), int32(body.PageSize)), 5*time.Second); err == nil {
		if reqID, total, page, size, items, derr := binproto.DecodeSystemLogListResp(pld); derr == nil {
			// map to JSON structure similar to controller output
			arr := make([]map[string]any, 0, len(items))
			for _, it := range items {
				arr = append(arr, map[string]any{"level": it.Level, "source": it.Source, "message": it.Message, "details": it.Details, "at": it.At})
			}
			h.writeJSON(w, map[string]any{"success": true, "original_id": reqID, "data": map[string]any{"items": arr, "total": total, "page": page, "size": size}})
			return
		}
	}
	h.writeError(w, http.StatusBadGateway, "hub error or timeout")
}

func valueOrZero(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
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
