package taskqueue

import (
	"context"
	"time"
)

type NoopRemindQueue struct{}

func NewNoopRemindQueue() *NoopRemindQueue {
	return &NoopRemindQueue{}
}

func (q *NoopRemindQueue) RegisterRemind(_ context.Context, req *CreateRemindRequest) (*RemindResponse, error) {
	_ = req

	return &RemindResponse{
		Name:       "noop",
		CreateTime: time.Time{},
	}, nil
}
