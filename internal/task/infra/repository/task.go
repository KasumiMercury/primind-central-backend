package repository

import (
	"context"
	"errors"
	"time"

	domaintask "github.com/KasumiMercury/primind-central-backend/internal/task/domain/task"
	domainuser "github.com/KasumiMercury/primind-central-backend/internal/task/domain/user"
	"gorm.io/gorm"
)

type TaskModel struct {
	ID          string     `gorm:"type:uuid;primaryKey"`
	UserID      string     `gorm:"type:uuid;not null;index:idx_tasks_user_id"`
	Title       string     `gorm:"type:varchar(500);not null"`
	TaskType    string     `gorm:"type:varchar(50);not null;index:idx_tasks_task_type"`
	TaskStatus  string     `gorm:"type:varchar(50);not null;index:idx_tasks_task_status"`
	Description *string    `gorm:"type:text"`
	DueTime     *time.Time `gorm:"type:timestamptz"`
	CreatedAt   time.Time  `gorm:"not null;autoCreateTime"`
}

func (TaskModel) TableName() string {
	return "tasks"
}

type taskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) domaintask.TaskRepository {
	return &taskRepository{db: db}
}

func (r *taskRepository) SaveTask(ctx context.Context, task *domaintask.Task) error {
	if task == nil {
		return ErrTaskRequired
	}

	var description *string
	if task.Description() != nil {
		description = task.Description()
	}

	var dueTime *time.Time
	if task.DueTime() != nil {
		dueTime = task.DueTime()
	}

	record := TaskModel{
		ID:          task.ID().String(),
		UserID:      task.UserID().String(),
		Title:       task.Title(),
		TaskType:    string(task.TaskType()),
		TaskStatus:  string(task.TaskStatus()),
		Description: description,
		DueTime:     dueTime,
		CreatedAt:   task.CreatedAt(),
	}

	return r.db.WithContext(ctx).Create(&record).Error
}

func (r *taskRepository) GetTaskByID(ctx context.Context, id domaintask.ID, userID domainuser.ID) (*domaintask.Task, error) {
	var record TaskModel
	if err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id.String(), userID).
		First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domaintask.ErrTaskNotFound
		}

		return nil, err
	}

	recordTaskID, err := domaintask.NewIDFromString(record.ID)
	if err != nil {
		return nil, err
	}

	recordUserID, err := domainuser.NewIDFromString(record.UserID)
	if err != nil {
		return nil, err
	}

	taskType, err := domaintask.NewType(record.TaskType)
	if err != nil {
		return nil, err
	}

	taskStatus, err := domaintask.NewStatus(record.TaskStatus)
	if err != nil {
		return nil, err
	}

	return domaintask.NewTask(
		recordTaskID,
		recordUserID,
		record.Title,
		taskType,
		taskStatus,
		record.Description,
		record.DueTime,
		record.CreatedAt,
	)
}
