package protocol

import "time"

// BaseMessage 定义了所有消息共有的基础结构
type BaseMessage struct {
	ID        string    `json:"id"`
	Source    uint64    `json:"source,omitempty"`
	Target    uint64    `json:"target,omitempty"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Payload   any       `json:"payload"`
}

// AuthRequestPayload 是 auth_request 消息的载荷
type AuthRequestPayload struct {
	DeviceID  uint64 `json:"deviceId"`
	SecretKey string `json:"secretKey"`
}

// VarsQueryPayload is the payload for a vars_query request.
type VarsQueryPayload struct {
	// A list of queries, e.g., ["deviceA.online", "deviceB.temperature", "deviceC"]
	Queries []string `json:"queries"`
}

// RegisterRequestPayload 是 register_request 消息的载荷
type RegisterRequestPayload struct {
	HardwareID string `json:"hardwareId"`
	Role       string `json:"role,omitempty"`  // e.g., "manager"
	Token      string `json:"token,omitempty"` // For privileged roles
}

// ResponsePayload 是通用响应的载荷
type ResponsePayload struct {
	Success    bool   `json:"success"`
	OriginalID string `json:"original_id"`
	Data       any    `json:"data,omitempty"`
	Message    string `json:"message,omitempty"`
}
