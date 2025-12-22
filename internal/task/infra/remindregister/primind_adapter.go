//go:build !gcloud

package remindregister

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/taskqueue"
)

type PrimindAdapter struct {
	client    *taskqueue.PrimindTasksClient
	queueName string
}

type PrimindAdapterConfig struct {
	Client    *taskqueue.PrimindTasksClient
	QueueName string
}

func NewPrimindAdapter(cfg PrimindAdapterConfig) *PrimindAdapter {
	return &PrimindAdapter{
		client:    cfg.Client,
		queueName: cfg.QueueName,
	}
}

func (a *PrimindAdapter) RegisterRemind(ctx context.Context, req *CreateRemindRequest) (*RemindResponse, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal remind request: %w", err)
	}

	taskReq := taskqueue.CreateTaskRequest{
		QueuePath: a.queueName,
		TargetURL: "",
		Payload:   payload,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	slog.Debug("registering remind to Primind Tasks",
		slog.String("queue_name", a.queueName),
		slog.String("task_id", req.TaskID),
	)

	resp, err := a.client.CreateTask(ctx, taskReq)
	if err != nil {
		return nil, fmt.Errorf("failed to register remind: %w", err)
	}

	slog.Info("remind task registered to Primind Tasks",
		slog.String("task_name", resp.Name),
		slog.String("task_id", req.TaskID),
	)

	return &RemindResponse{
		Name:       resp.Name,
		CreateTime: resp.CreateTime,
	}, nil
}
