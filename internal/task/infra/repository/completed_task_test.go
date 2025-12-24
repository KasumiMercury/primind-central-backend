package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	domaintask "github.com/KasumiMercury/primind-central-backend/internal/task/domain/task"
	domainuser "github.com/KasumiMercury/primind-central-backend/internal/task/domain/user"
	"github.com/KasumiMercury/primind-central-backend/internal/testutil"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func setupArchiveDB(t *testing.T) *gorm.DB {
	t.Helper()

	ctx := context.Background()
	db, cleanup := testutil.SetupPostgresContainer(ctx, t)
	t.Cleanup(cleanup)

	if err := db.AutoMigrate(&TaskModel{}, &CompletedTaskModel{}); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	return db
}

func createTestTask(t *testing.T, db *gorm.DB, userID domainuser.ID) *domaintask.Task {
	t.Helper()

	taskID := domaintask.ID(uuid.Must(uuid.NewV7()))
	createdAt := time.Now().UTC().Truncate(time.Microsecond)
	targetAt := createdAt.Add(1 * time.Hour)

	task, err := domaintask.NewTask(
		taskID,
		userID,
		"Test Task",
		domaintask.TypeNear,
		domaintask.StatusActive,
		"Test Description",
		nil,
		createdAt,
		targetAt,
		domaintask.MustColor("#FF6B6B"),
	)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Save to database
	taskRepo := NewTaskRepository(db)
	if err := taskRepo.SaveTask(context.Background(), task); err != nil {
		t.Fatalf("failed to save task: %v", err)
	}

	return task
}

func TestArchiveTaskSuccess(t *testing.T) {
	db := setupArchiveDB(t)
	archiveRepo := NewTaskArchiveRepository(db)
	ctx := context.Background()

	userID, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to create user ID: %v", err)
	}

	// Create a task in the tasks table
	task := createTestTask(t, db, userID)

	// Create completed task
	completedAt := time.Now()

	completedTask, err := domaintask.NewCompletedTask(task, completedAt)
	if err != nil {
		t.Fatalf("failed to create completed task: %v", err)
	}

	// Archive the task
	err = archiveRepo.ArchiveTask(ctx, completedTask, task.ID(), userID)
	if err != nil {
		t.Fatalf("ArchiveTask failed: %v", err)
	}

	// Verify task is in completed_tasks table
	var completedRecord CompletedTaskModel
	if err := db.First(&completedRecord, "id = ?", task.ID().String()).Error; err != nil {
		t.Fatalf("completed task not found in completed_tasks: %v", err)
	}

	if completedRecord.Title != task.Title() {
		t.Errorf("Title mismatch: got %q, want %q", completedRecord.Title, task.Title())
	}

	if completedRecord.UserID != userID.String() {
		t.Errorf("UserID mismatch: got %q, want %q", completedRecord.UserID, userID.String())
	}

	expectedCompletedAt := completedAt.UTC().Truncate(time.Microsecond)
	if !completedRecord.CompletedAt.Equal(expectedCompletedAt) {
		t.Errorf("CompletedAt mismatch: got %v, want %v", completedRecord.CompletedAt, expectedCompletedAt)
	}

	// Verify task is removed from tasks table
	var taskRecord TaskModel

	err = db.First(&taskRecord, "id = ?", task.ID().String()).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("task should be deleted from tasks table, but got error: %v", err)
	}
}

func TestArchiveTaskNilTask(t *testing.T) {
	db := setupArchiveDB(t)
	archiveRepo := NewTaskArchiveRepository(db)
	ctx := context.Background()

	userID, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to create user ID: %v", err)
	}

	taskID := domaintask.ID(uuid.Must(uuid.NewV7()))

	err = archiveRepo.ArchiveTask(ctx, nil, taskID, userID)
	if !errors.Is(err, ErrTaskRequired) {
		t.Errorf("expected ErrTaskRequired, got %v", err)
	}
}

func TestArchiveTaskNotFound(t *testing.T) {
	db := setupArchiveDB(t)
	archiveRepo := NewTaskArchiveRepository(db)
	ctx := context.Background()

	userID, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to create user ID: %v", err)
	}

	// Create a task object but don't save it to the database
	taskID := domaintask.ID(uuid.Must(uuid.NewV7()))
	createdAt := time.Now().UTC().Truncate(time.Microsecond)
	targetAt := createdAt.Add(1 * time.Hour)

	task, err := domaintask.NewTask(
		taskID,
		userID,
		"Non-existent Task",
		domaintask.TypeNear,
		domaintask.StatusActive,
		"",
		nil,
		createdAt,
		targetAt,
		domaintask.MustColor("#FF6B6B"),
	)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	completedTask, err := domaintask.NewCompletedTask(task, time.Now())
	if err != nil {
		t.Fatalf("failed to create completed task: %v", err)
	}

	// Try to archive a task that doesn't exist in the database
	err = archiveRepo.ArchiveTask(ctx, completedTask, taskID, userID)
	if !errors.Is(err, domaintask.ErrTaskNotFound) {
		t.Errorf("expected ErrTaskNotFound, got %v", err)
	}

	// Verify nothing was inserted into completed_tasks (transaction rolled back)
	var count int64

	db.Model(&CompletedTaskModel{}).Where("id = ?", taskID.String()).Count(&count)

	if count != 0 {
		t.Errorf("expected no record in completed_tasks after rollback, got %d", count)
	}
}

func TestArchiveTaskTransactionRollback(t *testing.T) {
	db := setupArchiveDB(t)
	archiveRepo := NewTaskArchiveRepository(db)
	ctx := context.Background()

	userID, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to create user ID: %v", err)
	}

	// Create a task in the tasks table
	task := createTestTask(t, db, userID)

	completedTask, err := domaintask.NewCompletedTask(task, time.Now())
	if err != nil {
		t.Fatalf("failed to create completed task: %v", err)
	}

	// Try to archive with wrong userID (should fail on delete)
	wrongUserID, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to create wrong user ID: %v", err)
	}

	err = archiveRepo.ArchiveTask(ctx, completedTask, task.ID(), wrongUserID)
	if !errors.Is(err, domaintask.ErrTaskNotFound) {
		t.Errorf("expected ErrTaskNotFound, got %v", err)
	}

	// Verify original task still exists (transaction rolled back)
	var taskRecord TaskModel
	if err := db.First(&taskRecord, "id = ?", task.ID().String()).Error; err != nil {
		t.Errorf("original task should still exist after rollback: %v", err)
	}

	// Verify nothing was inserted into completed_tasks
	var count int64

	db.Model(&CompletedTaskModel{}).Where("id = ?", task.ID().String()).Count(&count)

	if count != 0 {
		t.Errorf("expected no record in completed_tasks after rollback, got %d", count)
	}
}
