package remindregister

import "time"

type CreateRemindRequest struct {
	Times    []time.Time     `json:"times"`
	UserID   string          `json:"user_id"`
	Devices  []DeviceRequest `json:"devices"`
	TaskID   string          `json:"task_id"`
	TaskType string          `json:"task_type"`
	Color    string          `json:"color"`
}

type DeviceRequest struct {
	DeviceID string `json:"device_id"`
	FCMToken string `json:"fcm_token"`
}

type RemindResponse struct {
	Name       string    `json:"name"`
	CreateTime time.Time `json:"create_time"`
}
