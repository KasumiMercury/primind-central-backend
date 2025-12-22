package remindcancel

import (
	"context"
	"time"
)

// NoopQueue is a no-operation queue that does nothing.
// It is used when the cancel remind queue is disabled.
type NoopQueue struct{}

// NewNoopQueue creates a new NoopQueue.
func NewNoopQueue() *NoopQueue {
	return &NoopQueue{}
}

// CancelRemind returns a dummy response without actually canceling a remind.
func (q *NoopQueue) CancelRemind(_ context.Context, _ *CancelRemindRequest) (*CancelRemindResponse, error) {
	return &CancelRemindResponse{
		Name:       "noop",
		CreateTime: time.Time{},
	}, nil
}
