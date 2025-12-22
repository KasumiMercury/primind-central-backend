//go:build !gcloud

package task

import (
	"context"
	"log/slog"

	"github.com/KasumiMercury/primind-central-backend/internal/task/config"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/remindcancel"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/remindregister"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/taskqueue"
)

func NewRemindQueues(_ context.Context, cfg *config.TaskQueueConfig) (remindregister.Queue, remindcancel.Queue, taskqueue.Client, error) {
	if cfg.PrimindTasksURL == "" {
		slog.Warn("PRIMIND_TASKS_URL is not set; remind queues will be disabled")

		return remindregister.NewNoopQueue(), remindcancel.NewNoopQueue(), taskqueue.NewNoopClient(), nil
	}

	client := taskqueue.NewPrimindTasksClient(cfg.PrimindTasksURL, cfg.MaxRetries)

	remindAdapter := remindregister.NewPrimindAdapter(remindregister.PrimindAdapterConfig{
		Client:    client,
		QueueName: cfg.RemindRegisterQueueName,
	})

	cancelRemindAdapter := remindcancel.NewPrimindAdapter(remindcancel.PrimindAdapterConfig{
		Client:    client,
		QueueName: cfg.RemindCancelQueueName,
	})

	return remindAdapter, cancelRemindAdapter, client, nil
}
