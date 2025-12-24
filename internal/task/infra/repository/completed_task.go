package repository

import (
	"context"
	"time"

	domaintask "github.com/KasumiMercury/primind-central-backend/internal/task/domain/task"
	domainuser "github.com/KasumiMercury/primind-central-backend/internal/task/domain/user"
	"gorm.io/gorm"
)

type CompletedTaskModel struct {
	ID          string     `gorm:"type:uuid;primaryKey"`
	UserID      string     `gorm:"type:uuid;not null;index:idx_completed_tasks_user_id"`
	Title       string     `gorm:"type:varchar(500);not null"`
	TaskType    string     `gorm:"type:varchar(50);not null"`
	Description string     `gorm:"type:text"`
	ScheduledAt *time.Time `gorm:"type:timestamptz"`
	CreatedAt   time.Time  `gorm:"type:timestamptz;not null"`
	TargetAt    time.Time  `gorm:"type:timestamptz;not null"`
	Color       string     `gorm:"type:varchar(7);not null"`
	CompletedAt time.Time  `gorm:"type:timestamptz;not null;index:idx_completed_tasks_completed_at"`
}

func (CompletedTaskModel) TableName() string {
	return "completed_tasks"
}

type taskArchiveRepository struct {
	db *gorm.DB
}

func NewTaskArchiveRepository(db *gorm.DB) domaintask.TaskArchiveRepository {
	return &taskArchiveRepository{db: db}
}

func (r *taskArchiveRepository) ArchiveTask(
	ctx context.Context,
	completedTask *domaintask.CompletedTask,
	taskID domaintask.ID,
	userID domainuser.ID,
) error {
	if completedTask == nil {
		return ErrTaskRequired
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Insert into completed_tasks
		var scheduledAt *time.Time
		if completedTask.ScheduledAt() != nil {
			scheduledAt = completedTask.ScheduledAt()
		}

		record := CompletedTaskModel{
			ID:          completedTask.ID().String(),
			UserID:      completedTask.UserID().String(),
			Title:       completedTask.Title(),
			TaskType:    string(completedTask.TaskType()),
			Description: completedTask.Description(),
			ScheduledAt: scheduledAt,
			CreatedAt:   completedTask.CreatedAt(),
			TargetAt:    completedTask.TargetAt(),
			Color:       completedTask.Color().String(),
			CompletedAt: completedTask.CompletedAt(),
		}

		if err := tx.Create(&record).Error; err != nil {
			return err
		}

		// Delete from tasks table
		result := tx.
			Where("id = ? AND user_id = ?", taskID.String(), userID.String()).
			Delete(&TaskModel{})

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return domaintask.ErrTaskNotFound
		}

		return nil
	})
}
