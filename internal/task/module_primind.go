//go:build !gcloud

package task

import (
	"context"
	"log/slog"

	"github.com/KasumiMercury/primind-central-backend/internal/task/config"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/taskqueue"
)

func NewRemindQueue(_ context.Context, cfg *config.TaskQueueConfig) (taskqueue.RemindQueue, error) {
	if cfg.PrimindTasksURL == "" {
		slog.Warn("PRIMIND_TASKS_URL is not set; remind queue will be disabled")

		return taskqueue.NewNoopRemindQueue(), nil
	}

	return taskqueue.NewPrimindTasksClient(
		cfg.PrimindTasksURL,
		cfg.QueueName,
		cfg.MaxRetries,
	), nil
}
