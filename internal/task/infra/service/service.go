package task

import (
	"context"
	"errors"
	"log/slog"
	"time"

	connect "connectrpc.com/connect"
	taskv1 "github.com/KasumiMercury/primind-central-backend/internal/gen/task/v1"
	taskv1connect "github.com/KasumiMercury/primind-central-backend/internal/gen/task/v1/taskv1connect"
	apptask "github.com/KasumiMercury/primind-central-backend/internal/task/app/task"
	domaintask "github.com/KasumiMercury/primind-central-backend/internal/task/domain/task"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/interceptor"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	createTask apptask.CreateTaskUseCase
	getTask    apptask.GetTaskUseCase
	logger     *slog.Logger
}

var _ taskv1connect.TaskServiceHandler = (*Service)(nil)

func NewService(
	createTaskUseCase apptask.CreateTaskUseCase,
	getTaskUseCase apptask.GetTaskUseCase,
) *Service {
	return &Service{
		createTask: createTaskUseCase,
		getTask:    getTaskUseCase,
		logger:     slog.Default().WithGroup("task").WithGroup("service"),
	}
}

func (s *Service) CreateTask(
	ctx context.Context,
	req *taskv1.CreateTaskRequest,
) (*taskv1.CreateTaskResponse, error) {
	token := extractSessionTokenFromContext(ctx)
	if token == "" {
		s.logger.Warn("create task called without session token")

		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("session token required"))
	}

	taskType, err := protoTaskTypeToString(req.GetTaskType())
	if err != nil {
		s.logger.Warn("invalid task type", slog.String("error", err.Error()))

		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var scheduledAt *time.Time

	if req.ScheduledAt != nil {
		dt := req.GetScheduledAt().AsTime()
		scheduledAt = &dt
	}

	result, err := s.createTask.CreateTask(ctx, &apptask.CreateTaskRequest{
		TaskID:       req.GetTaskId(),
		SessionToken: token,
		Title:        req.GetTitle(),
		TaskType:     taskType,
		Description:  req.GetDescription(),
		ScheduledAt:  scheduledAt,
		Color:        req.GetColor(),
	})
	if err != nil {
		switch {
		case errors.Is(err, apptask.ErrUnauthorized):
			s.logger.Info("unauthorized create task attempt")

			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		case errors.Is(err, apptask.ErrTitleRequired),
			errors.Is(err, domaintask.ErrTitleTooLong),
			errors.Is(err, domaintask.ErrScheduledAtRequired),
			errors.Is(err, domaintask.ErrScheduledAtNotAllowed),
			errors.Is(err, domaintask.ErrInvalidTaskType),
			errors.Is(err, domaintask.ErrIDInvalidFormat),
			errors.Is(err, domaintask.ErrIDInvalidV7),
			errors.Is(err, domaintask.ErrColorEmpty),
			errors.Is(err, domaintask.ErrColorInvalidFormat):
			s.logger.Warn("invalid create task request", slog.String("error", err.Error()))

			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		case errors.Is(err, apptask.ErrTaskIDAlreadyExists):
			s.logger.Warn("task ID already exists", slog.String("error", err.Error()))

			return nil, connect.NewError(connect.CodeAlreadyExists, err)
		default:
			s.logger.Error("unexpected create task error", slog.String("error", err.Error()))

			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	s.logger.Info("task created", slog.String("task_id", result.TaskID))

	return &taskv1.CreateTaskResponse{
		Task: &taskv1.Task{
			TaskId:      result.TaskID,
			Title:       result.Title,
			TaskType:    stringToProtoTaskType(string(result.TaskType)),
			TaskStatus:  stringToProtoTaskStatus(string(result.TaskStatus)),
			Description: result.Description,
			ScheduledAt: func() *timestamppb.Timestamp {
				if result.ScheduledAt != nil {
					return timestamppb.New(*result.ScheduledAt)
				}

				return nil
			}(),
			CreatedAt: timestamppb.New(result.CreatedAt),
			TargetAt:  timestamppb.New(result.TargetAt),
			Color:     result.Color,
		},
	}, nil
}

func (s *Service) GetTask(
	ctx context.Context,
	req *taskv1.GetTaskRequest,
) (*taskv1.GetTaskResponse, error) {
	token := extractSessionTokenFromContext(ctx)
	if token == "" {
		s.logger.Warn("get task called without session token")

		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("session token required"))
	}

	result, err := s.getTask.GetTask(ctx, &apptask.GetTaskRequest{
		SessionToken: token,
		TaskID:       req.GetTaskId(),
	})
	if err != nil {
		switch {
		case errors.Is(err, apptask.ErrUnauthorized):
			s.logger.Info("unauthorized get task attempt")

			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		case errors.Is(err, apptask.ErrTaskNotFound):
			s.logger.Info("task not found", slog.String("task_id", req.GetTaskId()))

			return nil, connect.NewError(connect.CodeNotFound, err)
		case errors.Is(err, apptask.ErrTaskIDRequired),
			errors.Is(err, domaintask.ErrIDInvalidFormat),
			errors.Is(err, domaintask.ErrIDInvalidV7):
			s.logger.Warn("invalid get task request", slog.String("error", err.Error()))

			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		default:
			s.logger.Error("unexpected get task error", slog.String("error", err.Error()))

			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	protoTaskType := stringToProtoTaskType(string(result.TaskType))
	protoTaskStatus := stringToProtoTaskStatus(string(result.TaskStatus))

	var scheduledAt *timestamppb.Timestamp
	if result.ScheduledAt != nil {
		scheduledAt = timestamppb.New(*result.ScheduledAt)
	}

	response := &taskv1.GetTaskResponse{
		Task: &taskv1.Task{
			TaskId:      result.TaskID,
			Title:       result.Title,
			TaskType:    protoTaskType,
			TaskStatus:  protoTaskStatus,
			Description: result.Description,
			ScheduledAt: scheduledAt,
			CreatedAt:   timestamppb.New(result.CreatedAt),
			TargetAt:    timestamppb.New(result.TargetAt),
			Color:       result.Color,
		},
	}

	s.logger.Info("task retrieved", slog.String("task_id", result.TaskID))

	return response, nil
}

func (s *Service) ListActiveTasks(
	ctx context.Context,
	req *taskv1.ListActiveTasksRequest,
) (*taskv1.ListActiveTasksResponse, error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("not implemented"))
}

func extractSessionTokenFromContext(ctx context.Context) string {
	return interceptor.ExtractSessionToken(ctx)
}

func protoTaskTypeToString(taskType taskv1.TaskType) (domaintask.Type, error) {
	switch taskType {
	case taskv1.TaskType_TASK_TYPE_URGENT:
		return domaintask.TypeUrgent, nil
	case taskv1.TaskType_TASK_TYPE_NORMAL:
		return domaintask.TypeNormal, nil
	case taskv1.TaskType_TASK_TYPE_LOW:
		return domaintask.TypeLow, nil
	case taskv1.TaskType_TASK_TYPE_SCHEDULED:
		return domaintask.TypeScheduled, nil
	case taskv1.TaskType_TASK_TYPE_UNSPECIFIED:
		return "", errors.New("task type is required")
	default:
		return "", errors.New("unsupported task type")
	}
}

func stringToProtoTaskType(taskType string) taskv1.TaskType {
	switch taskType {
	case string(domaintask.TypeUrgent):
		return taskv1.TaskType_TASK_TYPE_URGENT
	case string(domaintask.TypeNormal):
		return taskv1.TaskType_TASK_TYPE_NORMAL
	case string(domaintask.TypeLow):
		return taskv1.TaskType_TASK_TYPE_LOW
	case string(domaintask.TypeScheduled):
		return taskv1.TaskType_TASK_TYPE_SCHEDULED
	default:
		return taskv1.TaskType_TASK_TYPE_UNSPECIFIED
	}
}

func stringToProtoTaskStatus(taskStatus string) taskv1.TaskStatus {
	switch taskStatus {
	case string(domaintask.StatusActive):
		return taskv1.TaskStatus_TASK_STATUS_ACTIVE
	case string(domaintask.StatusCompleted):
		return taskv1.TaskStatus_TASK_STATUS_COMPLETED
	default:
		return taskv1.TaskStatus_TASK_STATUS_UNSPECIFIED
	}
}
