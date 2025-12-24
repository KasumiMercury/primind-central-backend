package task

import (
	"context"

	"github.com/KasumiMercury/primind-central-backend/internal/task/domain/user"
)

//go:generate mockgen -source=archive_repository.go -destination=mock_archive_repository.go -package=task

type TaskArchiveRepository interface {
	ArchiveTask(ctx context.Context, completedTask *CompletedTask, taskID ID, userID user.ID) error
}
