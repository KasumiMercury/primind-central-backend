//go:build gcloud

package remindcancel

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/KasumiMercury/primind-central-backend/internal/observability/logging"
	"github.com/KasumiMercury/primind-central-backend/internal/observability/tracing"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/taskqueue"
)

type CloudTasksAdapter struct {
	client     *taskqueue.CloudTasksClient
	projectID  string
	locationID string
	queueID    string
	targetURL  string
}

type CloudTasksAdapterConfig struct {
	Client     *taskqueue.CloudTasksClient
	ProjectID  string
	LocationID string
	QueueID    string
	TargetURL  string
}

func NewCloudTasksAdapter(cfg CloudTasksAdapterConfig) *CloudTasksAdapter {
	return &CloudTasksAdapter{
		client:     cfg.Client,
		projectID:  cfg.ProjectID,
		locationID: cfg.LocationID,
		queueID:    cfg.QueueID,
		targetURL:  cfg.TargetURL,
	}
}

func (a *CloudTasksAdapter) CancelRemind(ctx context.Context, req *CancelRemindRequest) (*CancelRemindResponse, error) {
	ctx = logging.WithModule(ctx, logging.Module("task"))
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cancel remind request: %w", err)
	}

	queuePath := fmt.Sprintf("projects/%s/locations/%s/queues/%s",
		a.projectID, a.locationID, a.queueID)

	headers := map[string]string{
		"Content-Type": "application/json",
		"message_type": "remind.cancel",
	}
	tracing.InjectToMap(ctx, headers)
	reqID := logging.ValidateAndExtractRequestID(logging.RequestIDFromContext(ctx))
	headers["x-request-id"] = reqID

	taskReq := taskqueue.CreateTaskRequest{
		QueuePath: queuePath,
		TargetURL: a.targetURL,
		Payload:   payload,
		Headers:   headers,
	}

	slog.DebugContext(ctx, "sending cancel remind to Cloud Tasks",
		slog.String("queue_path", queuePath),
		slog.String("task_id", req.TaskID),
	)

	resp, err := a.client.CreateTask(ctx, taskReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send cancel remind: %w", err)
	}

	slog.InfoContext(ctx, "cancel remind task sent to Cloud Tasks",
		slog.String("task_name", resp.Name),
		slog.String("task_id", req.TaskID),
	)

	return &CancelRemindResponse{
		Name:       resp.Name,
		CreateTime: resp.CreateTime,
	}, nil
}
