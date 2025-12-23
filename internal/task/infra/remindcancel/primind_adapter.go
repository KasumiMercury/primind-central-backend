//go:build !gcloud

package remindcancel

import (
	"context"
	"fmt"
	"log/slog"

	remindv1 "github.com/KasumiMercury/primind-central-backend/internal/gen/remind/v1"
	pjson "github.com/KasumiMercury/primind-central-backend/internal/proto"
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

func (a *PrimindAdapter) CancelRemind(ctx context.Context, req *CancelRemindRequest) (*CancelRemindResponse, error) {
	protoReq := &remindv1.CancelRemindRequest{
		TaskId: req.TaskID,
		UserId: req.UserID,
	}

	payload, err := pjson.Marshal(protoReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cancel remind request: %w", err)
	}

	taskReq := taskqueue.CreateTaskRequest{
		QueuePath: a.queueName,
		TargetURL: "",
		Payload:   payload,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	slog.Debug("sending cancel remind to Primind Tasks",
		slog.String("queue_name", a.queueName),
		slog.String("task_id", req.TaskID),
	)

	resp, err := a.client.CreateTask(ctx, taskReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send cancel remind: %w", err)
	}

	slog.Info("cancel remind task sent to Primind Tasks",
		slog.String("task_name", resp.Name),
		slog.String("task_id", req.TaskID),
	)

	return &CancelRemindResponse{
		Name:       resp.Name,
		CreateTime: resp.CreateTime,
	}, nil
}
