//go:build !gcloud

package remindregister

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	commonv1 "github.com/KasumiMercury/primind-central-backend/internal/gen/common/v1"
	remindv1 "github.com/KasumiMercury/primind-central-backend/internal/gen/remind/v1"
	"github.com/KasumiMercury/primind-central-backend/internal/observability/logging"
	pjson "github.com/KasumiMercury/primind-central-backend/internal/proto"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/taskqueue"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	ctx = logging.WithModule(ctx, logging.Module("task"))

	protoTimes := make([]*timestamppb.Timestamp, 0, len(req.Times))
	for _, t := range req.Times {
		protoTimes = append(protoTimes, timestamppb.New(t))
	}

	protoDevices := make([]*remindv1.Device, 0, len(req.Devices))
	for _, d := range req.Devices {
		protoDevices = append(protoDevices, &remindv1.Device{
			DeviceId: d.DeviceID,
			FcmToken: d.FCMToken,
		})
	}

	protoReq := &remindv1.CreateRemindRequest{
		Times:    protoTimes,
		UserId:   req.UserID,
		Devices:  protoDevices,
		TaskId:   req.TaskID,
		TaskType: stringToTaskType(req.TaskType),
	}

	payload, err := pjson.Marshal(protoReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal remind request: %w", err)
	}

	taskReq := taskqueue.CreateTaskRequest{
		QueuePath: a.queueName,
		TargetURL: "",
		Payload:   payload,
		Headers: map[string]string{
			"Content-Type": "application/json",
			"message_type": "remind.register",
		},
	}

	slog.DebugContext(ctx, "registering remind to Primind Tasks",
		slog.String("queue_name", a.queueName),
		slog.String("task_id", req.TaskID),
	)

	resp, err := a.client.CreateTask(ctx, taskReq)
	if err != nil {
		return nil, fmt.Errorf("failed to register remind: %w", err)
	}

	slog.InfoContext(ctx, "remind task registered to Primind Tasks",
		slog.String("task_name", resp.Name),
		slog.String("task_id", req.TaskID),
	)

	return &RemindResponse{
		Name:       resp.Name,
		CreateTime: resp.CreateTime,
	}, nil
}

func stringToTaskType(s string) commonv1.TaskType {
	upper := "TASK_TYPE_" + strings.ToUpper(s)
	if v, ok := commonv1.TaskType_value[upper]; ok {
		return commonv1.TaskType(v)
	}

	return commonv1.TaskType_TASK_TYPE_UNSPECIFIED
}
