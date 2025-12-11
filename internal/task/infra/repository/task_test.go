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

	createdAt := time.Now().UTC().Truncate(time.Microsecond)
	targetAt := createdAt.Add(1 * time.Hour)

	task, err := domaintask.NewTask(
		domaintask.ID(taskId),
		userId,
		"Test Task",
		"normal",
		"active",
		"",
		nil,
		createdAt,
		targetAt,
		domaintask.MustColor("#FF6B6B"),
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

	// Verify targetAt is persisted and retrieved correctly
	if !retrievedTask.TargetAt().Equal(task.TargetAt()) {
		t.Errorf("expected targetAt %v, got %v", task.TargetAt(), retrievedTask.TargetAt())
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
	createdAt := time.Now().UTC().Truncate(time.Microsecond)
	targetAt := createdAt.Add(1 * time.Hour)

	task, err := domaintask.NewTask(
		taskID,
		user1ID,
		"User1's Task",
		"normal",
		"active",
		"",
		nil,
		createdAt,
		targetAt,
		domaintask.MustColor("#FF6B6B"),
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
	if err := db.Exec("INSERT INTO tasks (id, user_id, title, task_type, task_status, created_at, target_at, color) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		corruptedTaskID1, userID, "Corrupted Type Task", "invalid_type", "active", time.Now(), time.Now().Add(1*time.Hour), "#FF6B6B").Error; err != nil {
		t.Fatalf("failed to insert corrupted task type: %v", err)
	}

	corruptedID1, _ := domaintask.NewIDFromString(corruptedTaskID1)
	if _, err := repo.GetTaskByID(ctx, corruptedID1, userID); !errors.Is(err, domaintask.ErrInvalidTaskType) {
		t.Fatalf("expected ErrInvalidTaskType, got %v", err)
	}

	// Scenario 5: GetTaskByID with invalid TaskStatus in database
	corruptedTaskID2 := uuid.Must(uuid.NewV7()).String()
	if err := db.Exec("INSERT INTO tasks (id, user_id, title, task_type, task_status, created_at, target_at, color) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		corruptedTaskID2, userID, "Corrupted Status Task", "normal", "invalid_status", time.Now(), time.Now().Add(1*time.Hour), "#FF6B6B").Error; err != nil {
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
	fixedTargetAt := fixedTime.Add(1 * time.Hour)
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
		fixedTargetAt,
		domaintask.MustColor("#FF6B6B"),
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

	if !record.TargetAt.Equal(fixedTargetAt) {
		t.Fatalf("expected TargetAt %v, got %v", fixedTargetAt, record.TargetAt)
	}

	// Scenario 2: Multiple tasks with same timestamp
	taskID2 := domaintask.ID(uuid.Must(uuid.NewV7()))
	fixedTargetAt2 := fixedTime.Add(15 * time.Minute)

	task2, err := domaintask.NewTask(
		taskID2,
		userID,
		"Fixed Time Task 2",
		"urgent",
		"active",
		"",
		nil,
		fixedTime,
		fixedTargetAt2,
		domaintask.MustColor("#4ECDC4"),
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

	// Verify targetAt is different for different task types
	if record.TargetAt.Equal(record2.TargetAt) {
		t.Fatalf("expected different TargetAt for different task types, got %v and %v", record.TargetAt, record2.TargetAt)
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
	createdAt := time.Now().UTC().Truncate(time.Microsecond)
	targetAt := createdAt.Add(1 * time.Hour)

	task, err := domaintask.NewTask(
		taskID1,
		userID,
		"Existing Task",
		"normal",
		"active",
		"",
		nil,
		createdAt,
		targetAt,
		domaintask.MustColor("#FF6B6B"),
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
	createdAt := time.Now().UTC().Truncate(time.Microsecond)
	targetAt := createdAt.Add(1 * time.Hour)

	task, err := domaintask.NewTask(
		predefinedID,
		userID,
		"Task with predefined ID",
		"normal",
		"active",
		"This task has a predefined ID",
		nil,
		createdAt,
		targetAt,
		domaintask.MustColor("#FF6B6B"),
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
	createdAt := time.Now().UTC().Truncate(time.Microsecond)
	targetAt := createdAt.Add(1 * time.Hour)

	task1, err := domaintask.NewTask(
		sharedTaskID,
		user1ID,
		"Task 1",
		"normal",
		"active",
		"",
		nil,
		createdAt,
		targetAt,
		domaintask.MustColor("#FF6B6B"),
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
		createdAt,
		targetAt,
		domaintask.MustColor("#FF6B6B"),
	)
	if err != nil {
		t.Fatalf("failed to create task2: %v", err)
	}

	err = repo.SaveTask(context.Background(), task2)
	if err == nil {
		t.Fatal("SaveTask for duplicate ID should have failed, but succeeded")
	}
}

func TestListActiveTasksByUserID(t *testing.T) {
	db := setupTaskDB(t)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	userID, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to create user ID: %v", err)
	}

	otherUserID, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to create other user ID: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)

	// Task 1: target_at = now + 1h, created_at = now
	task1ID := domaintask.ID(uuid.Must(uuid.NewV7()))

	task1, err := domaintask.NewTask(
		task1ID,
		userID,
		"Task 1",
		"normal",
		"active",
		"",
		nil,
		now,
		now.Add(1*time.Hour),
		domaintask.MustColor("#FF6B6B"),
	)
	if err != nil {
		t.Fatalf("failed to create task1: %v", err)
	}

	if err := repo.SaveTask(ctx, task1); err != nil {
		t.Fatalf("failed to save task1: %v", err)
	}

	// Task 2: target_at = now + 30min, created_at = now
	task2ID := domaintask.ID(uuid.Must(uuid.NewV7()))

	task2, err := domaintask.NewTask(
		task2ID,
		userID,
		"Task 2",
		"urgent",
		"active",
		"",
		nil,
		now,
		now.Add(30*time.Minute),
		domaintask.MustColor("#4ECDC4"),
	)
	if err != nil {
		t.Fatalf("failed to create task2: %v", err)
	}

	if err := repo.SaveTask(ctx, task2); err != nil {
		t.Fatalf("failed to save task2: %v", err)
	}

	// Task 3: target_at = now + 30min, created_at = now + 1s (same target, newer created - should come first for ties)
	task3ID := domaintask.ID(uuid.Must(uuid.NewV7()))

	task3, err := domaintask.NewTask(
		task3ID,
		userID,
		"Task 3",
		"normal",
		"active",
		"",
		nil,
		now.Add(1*time.Second),
		now.Add(30*time.Minute),
		domaintask.MustColor("#45B7D1"),
	)
	if err != nil {
		t.Fatalf("failed to create task3: %v", err)
	}

	if err := repo.SaveTask(ctx, task3); err != nil {
		t.Fatalf("failed to save task3: %v", err)
	}

	// Task 4: COMPLETED status (should NOT be returned)
	task4ID := domaintask.ID(uuid.Must(uuid.NewV7()))

	task4, err := domaintask.NewTask(
		task4ID,
		userID,
		"Task 4",
		"normal",
		"completed",
		"",
		nil,
		now,
		now.Add(20*time.Minute),
		domaintask.MustColor("#96CEB4"),
	)
	if err != nil {
		t.Fatalf("failed to create task4: %v", err)
	}

	if err := repo.SaveTask(ctx, task4); err != nil {
		t.Fatalf("failed to save task4: %v", err)
	}

	// Task 5: Different user (should NOT be returned)
	task5ID := domaintask.ID(uuid.Must(uuid.NewV7()))

	task5, err := domaintask.NewTask(
		task5ID,
		otherUserID,
		"Task 5",
		"normal",
		"active",
		"",
		nil,
		now,
		now.Add(15*time.Minute),
		domaintask.MustColor("#FFEAA7"),
	)
	if err != nil {
		t.Fatalf("failed to create task5: %v", err)
	}

	if err := repo.SaveTask(ctx, task5); err != nil {
		t.Fatalf("failed to save task5: %v", err)
	}

	t.Run("sort by target_at ascending with created_at descending for ties", func(t *testing.T) {
		tasks, err := repo.ListActiveTasksByUserID(ctx, userID, domaintask.SortTypeTargetAt)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(tasks) != 3 {
			t.Fatalf("expected 3 tasks, got %d", len(tasks))
		}

		// Expected order: task3 (30min, newer created), task2 (30min, older created), task1 (1h)
		expectedOrder := []string{task3ID.String(), task2ID.String(), task1ID.String()}
		for i, task := range tasks {
			if task.ID().String() != expectedOrder[i] {
				t.Errorf("position %d: expected %s, got %s", i, expectedOrder[i], task.ID().String())
			}
		}
	})

	t.Run("returns empty slice for user with no active tasks", func(t *testing.T) {
		emptyUserID, err := domainuser.NewID()
		if err != nil {
			t.Fatalf("failed to create empty user ID: %v", err)
		}

		tasks, err := repo.ListActiveTasksByUserID(ctx, emptyUserID, domaintask.SortTypeTargetAt)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(tasks) != 0 {
			t.Fatalf("expected 0 tasks, got %d", len(tasks))
		}
	})

	t.Run("returns error for invalid sort type", func(t *testing.T) {
		_, err := repo.ListActiveTasksByUserID(ctx, userID, domaintask.SortType("invalid"))
		if !errors.Is(err, domaintask.ErrInvalidSortType) {
			t.Fatalf("expected ErrInvalidSortType, got %v", err)
		}
	})

	t.Run("user isolation - does not return other users tasks", func(t *testing.T) {
		tasks, err := repo.ListActiveTasksByUserID(ctx, otherUserID, domaintask.SortTypeTargetAt)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(tasks) != 1 {
			t.Fatalf("expected 1 task for other user, got %d", len(tasks))
		}

		if tasks[0].ID().String() != task5ID.String() {
			t.Errorf("expected task5, got %s", tasks[0].ID().String())
		}
	})
}

func TestUpdateTask(t *testing.T) {
	db := setupTaskDB(t)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	userID, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to create user ID: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	targetAt := now.Add(1 * time.Hour)

	t.Run("update task successfully", func(t *testing.T) {
		taskID := domaintask.ID(uuid.Must(uuid.NewV7()))

		task, err := domaintask.NewTask(
			taskID,
			userID,
			"Original Title",
			"normal",
			"active",
			"Original Description",
			nil,
			now,
			targetAt,
			domaintask.MustColor("#FF6B6B"),
		)
		if err != nil {
			t.Fatalf("failed to create task: %v", err)
		}

		if err := repo.SaveTask(ctx, task); err != nil {
			t.Fatalf("failed to save task: %v", err)
		}

		// Create updated task
		updatedTask, err := domaintask.NewTask(
			taskID,
			userID,
			"Updated Title",
			"normal",
			"completed",
			"Updated Description",
			nil,
			now,
			targetAt,
			domaintask.MustColor("#4ECDC4"),
		)
		if err != nil {
			t.Fatalf("failed to create updated task: %v", err)
		}

		if err := repo.UpdateTask(ctx, updatedTask); err != nil {
			t.Fatalf("failed to update task: %v", err)
		}

		// Verify the update
		retrieved, err := repo.GetTaskByID(ctx, taskID, userID)
		if err != nil {
			t.Fatalf("failed to retrieve updated task: %v", err)
		}

		if retrieved.Title() != "Updated Title" {
			t.Errorf("expected title 'Updated Title', got '%s'", retrieved.Title())
		}

		if retrieved.TaskStatus() != domaintask.StatusCompleted {
			t.Errorf("expected status 'completed', got '%s'", retrieved.TaskStatus())
		}

		if retrieved.Description() != "Updated Description" {
			t.Errorf("expected description 'Updated Description', got '%s'", retrieved.Description())
		}

		if retrieved.Color().String() != "#4ECDC4" {
			t.Errorf("expected color '#4ECDC4', got '%s'", retrieved.Color().String())
		}
	})

	t.Run("update non-existent task returns ErrTaskNotFound", func(t *testing.T) {
		nonExistentID := domaintask.ID(uuid.Must(uuid.NewV7()))

		task, err := domaintask.NewTask(
			nonExistentID,
			userID,
			"Non-existent Task",
			"normal",
			"active",
			"",
			nil,
			now,
			targetAt,
			domaintask.MustColor("#FF6B6B"),
		)
		if err != nil {
			t.Fatalf("failed to create task: %v", err)
		}

		err = repo.UpdateTask(ctx, task)
		if !errors.Is(err, domaintask.ErrTaskNotFound) {
			t.Fatalf("expected ErrTaskNotFound, got %v", err)
		}
	})

	t.Run("update with nil task returns ErrTaskRequired", func(t *testing.T) {
		err := repo.UpdateTask(ctx, nil)
		if !errors.Is(err, ErrTaskRequired) {
			t.Fatalf("expected ErrTaskRequired, got %v", err)
		}
	})

	t.Run("user isolation - cannot update other user's task", func(t *testing.T) {
		otherUserID, err := domainuser.NewID()
		if err != nil {
			t.Fatalf("failed to create other user ID: %v", err)
		}

		taskID := domaintask.ID(uuid.Must(uuid.NewV7()))

		// Create task owned by userID
		task, err := domaintask.NewTask(
			taskID,
			userID,
			"User1's Task",
			"normal",
			"active",
			"",
			nil,
			now,
			targetAt,
			domaintask.MustColor("#FF6B6B"),
		)
		if err != nil {
			t.Fatalf("failed to create task: %v", err)
		}

		if err := repo.SaveTask(ctx, task); err != nil {
			t.Fatalf("failed to save task: %v", err)
		}

		// Try to update with otherUserID
		maliciousUpdate, err := domaintask.NewTask(
			taskID,
			otherUserID, // Different user
			"Malicious Update",
			"normal",
			"active",
			"",
			nil,
			now,
			targetAt,
			domaintask.MustColor("#FF6B6B"),
		)
		if err != nil {
			t.Fatalf("failed to create malicious update task: %v", err)
		}

		err = repo.UpdateTask(ctx, maliciousUpdate)
		if !errors.Is(err, domaintask.ErrTaskNotFound) {
			t.Fatalf("expected ErrTaskNotFound for user isolation, got %v", err)
		}

		// Verify original task is unchanged
		original, err := repo.GetTaskByID(ctx, taskID, userID)
		if err != nil {
			t.Fatalf("failed to retrieve original task: %v", err)
		}

		if original.Title() != "User1's Task" {
			t.Errorf("expected original title unchanged, got '%s'", original.Title())
		}
	})

	t.Run("update scheduled task with new scheduled_at", func(t *testing.T) {
		taskID := domaintask.ID(uuid.Must(uuid.NewV7()))
		scheduledAt := now.Add(2 * time.Hour)
		scheduledTargetAt := scheduledAt

		task, err := domaintask.NewTask(
			taskID,
			userID,
			"Scheduled Task",
			"scheduled",
			"active",
			"",
			&scheduledAt,
			now,
			scheduledTargetAt,
			domaintask.MustColor("#FF6B6B"),
		)
		if err != nil {
			t.Fatalf("failed to create scheduled task: %v", err)
		}

		if err := repo.SaveTask(ctx, task); err != nil {
			t.Fatalf("failed to save scheduled task: %v", err)
		}

		// Update with new scheduled_at
		newScheduledAt := now.Add(4 * time.Hour)
		newTargetAt := newScheduledAt

		updatedTask, err := domaintask.NewTask(
			taskID,
			userID,
			"Scheduled Task",
			"scheduled",
			"active",
			"",
			&newScheduledAt,
			now,
			newTargetAt,
			domaintask.MustColor("#FF6B6B"),
		)
		if err != nil {
			t.Fatalf("failed to create updated scheduled task: %v", err)
		}

		if err := repo.UpdateTask(ctx, updatedTask); err != nil {
			t.Fatalf("failed to update scheduled task: %v", err)
		}

		// Verify the update
		retrieved, err := repo.GetTaskByID(ctx, taskID, userID)
		if err != nil {
			t.Fatalf("failed to retrieve updated scheduled task: %v", err)
		}

		if retrieved.ScheduledAt() == nil {
			t.Fatalf("expected scheduled_at to be set, got nil")
		}

		if !retrieved.ScheduledAt().Equal(newScheduledAt) {
			t.Errorf("expected scheduled_at %v, got %v", newScheduledAt, retrieved.ScheduledAt())
		}

		if !retrieved.TargetAt().Equal(newTargetAt) {
			t.Errorf("expected target_at %v, got %v", newTargetAt, retrieved.TargetAt())
		}
	})
}
