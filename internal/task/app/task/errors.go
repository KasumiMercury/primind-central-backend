package task

import (
	"errors"

	domaintask "github.com/KasumiMercury/primind-central-backend/internal/task/domain/task"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/authclient"
)

var (
	ErrUnauthorized                   = authclient.ErrUnauthorized
	ErrAuthServiceUnavailable         = authclient.ErrAuthServiceUnavailable
	ErrCreateTaskRequestRequired      = errors.New("create task request is required")
	ErrGetTaskRequestRequired         = errors.New("get tasks request is required")
	ErrListActiveTasksRequestRequired = errors.New("list active tasks request is required")
	ErrUpdateTaskRequestRequired      = errors.New("update task request is required")
	ErrDeleteTaskRequestRequired      = errors.New("delete task request is required")
	ErrTitleRequired                  = errors.New("task title is required")
	ErrTaskNotFound                   = domaintask.ErrTaskNotFound
	ErrTaskIDRequired                 = errors.New("task ID is required")
	ErrTaskIDAlreadyExists            = domaintask.ErrTaskIDAlreadyExists
	ErrInvalidSortType                = domaintask.ErrInvalidSortType
)
