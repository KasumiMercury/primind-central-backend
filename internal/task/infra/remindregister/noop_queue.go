package remindregister

import (
	"context"
	"time"
)

// NoopQueue is a no-operation queue that does nothing.
// It is used when the remind queue is disabled.
type NoopQueue struct{}

// NewNoopQueue creates a new NoopQueue.
func NewNoopQueue() *NoopQueue {
	return &NoopQueue{}
}

// RegisterRemind returns a dummy response without actually registering a remind.
func (q *NoopQueue) RegisterRemind(_ context.Context, _ *CreateRemindRequest) (*RemindResponse, error) {
	return &RemindResponse{
		Name:       "noop",
		CreateTime: time.Time{},
	}, nil
}
