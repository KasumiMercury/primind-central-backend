package task

import (
	"errors"

	domaintask "github.com/KasumiMercury/primind-central-backend/internal/task/domain/task"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/authclient"
)

var (
	ErrUnauthorized              = authclient.ErrUnauthorized
	ErrCreateTaskRequestRequired = errors.New("create task request is required")
	ErrGetTaskRequestRequired    = errors.New("get tasks request is required")
	ErrTitleRequired             = errors.New("task title is required")
	ErrTaskNotFound              = domaintask.ErrTaskNotFound
	ErrTaskIDRequired            = errors.New("task ID is required")
	ErrTaskIDAlreadyExists       = domaintask.ErrTaskIDAlreadyExists
)
