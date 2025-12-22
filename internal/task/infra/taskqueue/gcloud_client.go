//go:build gcloud

package taskqueue

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	taskspb "cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
)

type CloudTasksClient struct {
	client     *cloudtasks.Client
	maxRetries int
}

type CloudTasksClientConfig struct {
	MaxRetries int
}

func NewCloudTasksClient(ctx context.Context, cfg CloudTasksClientConfig) (*CloudTasksClient, error) {
	client, err := cloudtasks.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloud tasks client: %w", err)
	}

	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	return &CloudTasksClient{
		client:     client,
		maxRetries: maxRetries,
	}, nil
}

// CreateTask creates a task in Google Cloud Tasks.
func (c *CloudTasksClient) CreateTask(ctx context.Context, req CreateTaskRequest) (*TaskResponse, error) {
	cloudTask := &taskspb.Task{
		MessageType: &taskspb.Task_HttpRequest{
			HttpRequest: &taskspb.HttpRequest{
				HttpMethod: taskspb.HttpMethod_POST,
				Url:        req.TargetURL,
				Headers:    req.Headers,
				Body:       req.Payload,
			},
		},
	}

	taskReq := &taskspb.CreateTaskRequest{
		Parent: req.QueuePath,
		Task:   cloudTask,
	}

	var lastErr error

	for attempt := 0; attempt < c.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * 100 * time.Millisecond
			slog.Debug("retrying task creation",
				slog.Int("attempt", attempt+1),
				slog.Duration("backoff", backoff),
			)

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, err := c.doCreateTask(ctx, taskReq)
		if err == nil {
			return resp, nil
		}

		lastErr = err
	}

	slog.Error("all retries exhausted for task creation",
		slog.Int("max_retries", c.maxRetries),
		slog.String("error", lastErr.Error()),
	)

	return nil, fmt.Errorf("failed to create task after %d retries: %w", c.maxRetries, lastErr)
}

func (c *CloudTasksClient) doCreateTask(ctx context.Context, req *taskspb.CreateTaskRequest) (*TaskResponse, error) {
	slog.Debug("creating task in Cloud Tasks",
		slog.String("queue_path", req.Parent),
	)

	createdTask, err := c.client.CreateTask(ctx, req)
	if err != nil {
		slog.Warn("failed to create cloud task",
			slog.String("error", err.Error()),
		)

		return nil, fmt.Errorf("failed to create cloud task: %w", err)
	}

	slog.Debug("task created in Cloud Tasks",
		slog.String("task_name", createdTask.Name),
	)

	var createTime time.Time
	if createdTask.CreateTime != nil {
		createTime = createdTask.CreateTime.AsTime()
	}

	return &TaskResponse{
		Name:       createdTask.Name,
		CreateTime: createTime,
	}, nil
}

// Close closes the Cloud Tasks client.
func (c *CloudTasksClient) Close() error {
	return c.client.Close()
}
