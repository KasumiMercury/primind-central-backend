package taskqueue

import (
	"context"
	"time"
)

type NoopClient struct{}

func NewNoopClient() *NoopClient {
	return &NoopClient{}
}

func (c *NoopClient) CreateTask(_ context.Context, _ CreateTaskRequest) (*TaskResponse, error) {
	return &TaskResponse{
		Name:       "noop",
		CreateTime: time.Time{},
	}, nil
}

func (c *NoopClient) Close() error {
	return nil
}
