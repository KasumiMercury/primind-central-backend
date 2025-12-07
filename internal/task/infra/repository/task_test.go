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

func setupTaskDB(t *testing.T) *gorm.DB {
	t.Helper()

	ctx := context.Background()
	db, cleanup := testutil.SetupPostgresContainer(ctx, t)
	t.Cleanup(cleanup)

	if err := db.AutoMigrate(&TaskModel{}); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	return db
}

func TestTaskRepositoryIntegrationSuccess(t *testing.T) {
	db := setupTaskDB(t)
	repo := NewTaskRepository(db)

	taskId := uuid.Must(uuid.NewV7())

	userId, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to create user ID: %v", err)
	}

	task, err := domaintask.NewTask(
		domaintask.ID(taskId),
		userId,
		"Test Task",
		"normal",
		"active",
		"",
		nil,
		time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Test Create
	if err := repo.SaveTask(context.Background(), task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Test GetByID
	retrievedTask, err := repo.GetTaskByID(context.Background(), task.ID(), userId)
	if err != nil {
		t.Fatalf("failed to get task by ID: %v", err)
	}

	if retrievedTask.Title() != task.Title() {
		t.Errorf("expected title %q, got %q", task.Title(), retrievedTask.Title())
	}
}

func TestTaskRepositoryIntegrationError(t *testing.T) {
	db := setupTaskDB(t)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	// Scenario 1: SaveTask with nil task
	if err := repo.SaveTask(ctx, nil); !errors.Is(err, ErrTaskRequired) {
		t.Fatalf("expected ErrTaskRequired, got %v", err)
	}

	// Scenario 2: GetTaskByID for non-existent task
	nonExistentID := domaintask.ID(uuid.Must(uuid.NewV7()))

	userID, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to create user ID: %v", err)
	}

	if _, err := repo.GetTaskByID(ctx, nonExistentID, userID); !errors.Is(err, domaintask.ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}

	// Scenario 3: GetTaskByID with user isolation
	user1ID, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to create user1 ID: %v", err)
	}

	user2ID, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to create user2 ID: %v", err)
	}

	taskID := domaintask.ID(uuid.Must(uuid.NewV7()))

	task, err := domaintask.NewTask(
		taskID,
		user1ID,
		"User1's Task",
		"normal",
		"active",
		"",
		nil,
		time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	if err := repo.SaveTask(ctx, task); err != nil {
		t.Fatalf("failed to save task: %v", err)
	}

	// Try to retrieve with user2's ID
	if _, err := repo.GetTaskByID(ctx, taskID, user2ID); !errors.Is(err, domaintask.ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound for user isolation, got %v", err)
	}

	// Scenario 4: GetTaskByID with invalid TaskType in database
	corruptedTaskID1 := uuid.Must(uuid.NewV7()).String()
	if err := db.Exec("INSERT INTO tasks (id, user_id, title, task_type, task_status, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		corruptedTaskID1, userID, "Corrupted Type Task", "invalid_type", "active", time.Now()).Error; err != nil {
		t.Fatalf("failed to insert corrupted task type: %v", err)
	}

	corruptedID1, _ := domaintask.NewIDFromString(corruptedTaskID1)
	if _, err := repo.GetTaskByID(ctx, corruptedID1, userID); !errors.Is(err, domaintask.ErrInvalidTaskType) {
		t.Fatalf("expected ErrInvalidTaskType, got %v", err)
	}

	// Scenario 5: GetTaskByID with invalid TaskStatus in database
	corruptedTaskID2 := uuid.Must(uuid.NewV7()).String()
	if err := db.Exec("INSERT INTO tasks (id, user_id, title, task_type, task_status, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		corruptedTaskID2, userID, "Corrupted Status Task", "normal", "invalid_status", time.Now()).Error; err != nil {
		t.Fatalf("failed to insert corrupted task status: %v", err)
	}

	corruptedID2, _ := domaintask.NewIDFromString(corruptedTaskID2)
	if _, err := repo.GetTaskByID(ctx, corruptedID2, userID); !errors.Is(err, domaintask.ErrInvalidTaskStatus) {
		t.Fatalf("expected ErrInvalidTaskStatus, got %v", err)
	}
}

func TestTaskRepositoryWithFixedClock(t *testing.T) {
	db := setupTaskDB(t)
	repo := NewTaskRepository(db)

	fixedTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	taskID := domaintask.ID(uuid.Must(uuid.NewV7()))

	userID, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to create user ID: %v", err)
	}

	// Create task with specific CreatedAt
	task, err := domaintask.NewTask(
		taskID,
		userID,
		"Fixed Time Task",
		"normal",
		"active",
		"",
		nil,
		fixedTime,
	)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	if err := repo.SaveTask(context.Background(), task); err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	// Verify timestamp in database
	var record TaskModel
	if err := db.First(&record, "id = ?", taskID.String()).Error; err != nil {
		t.Fatalf("failed to query task: %v", err)
	}

	if !record.CreatedAt.Equal(fixedTime) {
		t.Fatalf("expected CreatedAt %v, got %v", fixedTime, record.CreatedAt)
	}

	// Scenario 2: Multiple tasks with same timestamp
	taskID2 := domaintask.ID(uuid.Must(uuid.NewV7()))

	task2, err := domaintask.NewTask(
		taskID2,
		userID,
		"Fixed Time Task 2",
		"urgent",
		"active",
		"",
		nil,
		fixedTime,
	)
	if err != nil {
		t.Fatalf("failed to create task2: %v", err)
	}

	if err := repo.SaveTask(context.Background(), task2); err != nil {
		t.Fatalf("SaveTask for task2 failed: %v", err)
	}

	var record2 TaskModel
	if err := db.First(&record2, "id = ?", taskID2.String()).Error; err != nil {
		t.Fatalf("failed to query task2: %v", err)
	}

	if !record2.CreatedAt.Equal(fixedTime) {
		t.Fatalf("expected task2 CreatedAt %v, got %v", fixedTime, record2.CreatedAt)
	}

	// Verify both tasks have identical timestamps
	if !record.CreatedAt.Equal(record2.CreatedAt) {
		t.Fatalf("expected both tasks to have identical CreatedAt, got %v and %v", record.CreatedAt, record2.CreatedAt)
	}
}

func TestExistsTaskByID(t *testing.T) {
	db := setupTaskDB(t)
	repo := NewTaskRepository(db)

	userID, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to create user ID: %v", err)
	}

	taskID1 := domaintask.ID(uuid.Must(uuid.NewV7()))

	task, err := domaintask.NewTask(
		taskID1,
		userID,
		"Existing Task",
		"normal",
		"active",
		"",
		nil,
		time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	if err := repo.SaveTask(context.Background(), task); err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	tests := []struct {
		name     string
		taskID   domaintask.ID
		expected bool
	}{
		{
			name:     "task exists returns true",
			taskID:   taskID1,
			expected: true,
		},
		{
			name:     "task does not exist returns false",
			taskID:   domaintask.ID(uuid.Must(uuid.NewV7())),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := repo.ExistsTaskByID(context.Background(), tt.taskID)
			if err != nil {
				t.Fatalf("ExistsTaskByID failed: %v", err)
			}

			if exists != tt.expected {
				t.Errorf("ExistsTaskByID(%s) = %v, want %v", tt.taskID.String(), exists, tt.expected)
			}
		})
	}
}

func TestSaveTaskWithPredefinedID(t *testing.T) {
	db := setupTaskDB(t)
	repo := NewTaskRepository(db)

	userID, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to create user ID: %v", err)
	}

	predefinedID := domaintask.ID(uuid.Must(uuid.NewV7()))

	task, err := domaintask.NewTask(
		predefinedID,
		userID,
		"Task with predefined ID",
		"normal",
		"active",
		"This task has a predefined ID",
		nil,
		time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	if err := repo.SaveTask(context.Background(), task); err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	retrievedTask, err := repo.GetTaskByID(context.Background(), predefinedID, userID)
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}

	if retrievedTask.ID().String() != predefinedID.String() {
		t.Errorf("Retrieved task ID = %s, want %s", retrievedTask.ID().String(), predefinedID.String())
	}

	if retrievedTask.Title() != task.Title() {
		t.Errorf("Retrieved task title = %s, want %s", retrievedTask.Title(), task.Title())
	}
}

func TestSaveTaskDuplicateID(t *testing.T) {
	db := setupTaskDB(t)
	repo := NewTaskRepository(db)

	user1ID, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to create user1 ID: %v", err)
	}

	user2ID, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to create user2 ID: %v", err)
	}

	sharedTaskID := domaintask.ID(uuid.Must(uuid.NewV7()))

	task1, err := domaintask.NewTask(
		sharedTaskID,
		user1ID,
		"Task 1",
		"normal",
		"active",
		"",
		nil,
		time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to create task1: %v", err)
	}

	if err := repo.SaveTask(context.Background(), task1); err != nil {
		t.Fatalf("SaveTask for task1 failed: %v", err)
	}

	task2, err := domaintask.NewTask(
		sharedTaskID,
		user2ID,
		"Task 2",
		"normal",
		"active",
		"",
		nil,
		time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to create task2: %v", err)
	}

	err = repo.SaveTask(context.Background(), task2)
	if err == nil {
		t.Fatal("SaveTask for duplicate ID should have failed, but succeeded")
	}
}
