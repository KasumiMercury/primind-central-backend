package taskqueue

import "time"

type CreateRemindRequest struct {
	Times    []time.Time     `json:"times"`
	UserID   string          `json:"user_id"`
	Devices  []DeviceRequest `json:"devices"`
	TaskID   string          `json:"task_id"`
	TaskType string          `json:"task_type"`
}

type DeviceRequest struct {
	DeviceID string `json:"device_id"`
	FCMToken string `json:"fcm_token"`
}

type RemindResponse struct {
	Name       string    `json:"name"`
	CreateTime time.Time `json:"create_time"`
}

type PrimindTaskRequest struct {
	Task PrimindTask `json:"task"`
}

type PrimindTask struct {
	HTTPRequest PrimindHTTPRequest `json:"httpRequest"`
}

type PrimindHTTPRequest struct {
	Body    string            `json:"body"`
	Headers map[string]string `json:"headers,omitempty"`
}

type PrimindTaskResponse struct {
	Name       string `json:"name"`
	CreateTime string `json:"createTime"`
}
