package taskqueue

import (
	"context"
	"time"
)

type Client interface {
	CreateTask(ctx context.Context, req CreateTaskRequest) (*TaskResponse, error)
	Close() error
}

type CreateTaskRequest struct {
	QueuePath string
	TargetURL string
	Payload   []byte
	Headers   map[string]string
}

type TaskResponse struct {
	Name       string
	CreateTime time.Time
}
