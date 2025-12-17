//go:build gcloud

package task

import (
	"context"

	"github.com/KasumiMercury/primind-central-backend/internal/task/config"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/taskqueue"
)

func NewRemindQueue(ctx context.Context, cfg *config.TaskQueueConfig) (taskqueue.RemindQueue, error) {
	return taskqueue.NewCloudTasksClient(ctx, taskqueue.CloudTasksConfig{
		ProjectID:  cfg.GCloudProjectID,
		LocationID: cfg.GCloudLocationID,
		QueueID:    cfg.GCloudQueueID,
		TargetURL:  cfg.GCloudTargetURL,
		MaxRetries: cfg.MaxRetries,
	})
}
