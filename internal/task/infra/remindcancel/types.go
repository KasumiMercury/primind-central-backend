package remindcancel

import "time"

type CancelRemindRequest struct {
	TaskID string `json:"task_id"`
	UserID string `json:"user_id"`
}

type CancelRemindResponse struct {
	Name       string    `json:"name"`
	CreateTime time.Time `json:"create_time"`
}
