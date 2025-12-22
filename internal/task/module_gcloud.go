//go:build gcloud

package task

import (
	"context"
	"fmt"

	"github.com/KasumiMercury/primind-central-backend/internal/task/config"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/remindcancel"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/remindregister"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/taskqueue"
)

func NewRemindQueues(ctx context.Context, cfg *config.TaskQueueConfig) (remindregister.Queue, remindcancel.Queue, taskqueue.Client, error) {
	client, err := taskqueue.NewCloudTasksClient(ctx, taskqueue.CloudTasksClientConfig{
		MaxRetries: cfg.MaxRetries,
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create cloud tasks client: %w", err)
	}

	remindAdapter := remindregister.NewCloudTasksAdapter(remindregister.CloudTasksAdapterConfig{
		Client:     client,
		ProjectID:  cfg.GCloudProjectID,
		LocationID: cfg.GCloudLocationID,
		QueueID:    cfg.GCloudRemindRegisterQueueID,
		TargetURL:  cfg.GCloudRemindTargetURL,
	})

	cancelRemindAdapter := remindcancel.NewCloudTasksAdapter(remindcancel.CloudTasksAdapterConfig{
		Client:     client,
		ProjectID:  cfg.GCloudProjectID,
		LocationID: cfg.GCloudLocationID,
		QueueID:    cfg.GCloudRemindCancelQueueID,
		TargetURL:  cfg.GCloudRemindTargetURL,
	})

	return remindAdapter, cancelRemindAdapter, client, nil
}
