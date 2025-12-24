package task

import (
	"testing"
	"time"

	"github.com/KasumiMercury/primind-central-backend/internal/task/domain/user"
)

func TestNewCompletedTask(t *testing.T) {
	userID, err := user.NewID()
	if err != nil {
		t.Fatalf("failed to create user ID: %v", err)
	}

	taskID, err := NewID()
	if err != nil {
		t.Fatalf("failed to create task ID: %v", err)
	}

	createdAt := time.Now().UTC().Truncate(time.Microsecond)
	targetAt := createdAt.Add(1 * time.Hour)
	scheduled := createdAt.Add(30 * time.Minute)
	color := MustColor("#FF6B6B")

	// Use TypeScheduled so that scheduledAt is allowed
	task, err := NewTask(
		taskID,
		userID,
		"Test Task",
		TypeScheduled,
		StatusActive,
		"Test Description",
		&scheduled,
		createdAt,
		targetAt,
		color,
	)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	tests := []struct {
		name        string
		completedAt time.Time
	}{
		{
			name:        "creates completed task with current time",
			completedAt: time.Now(),
		},
		{
			name:        "normalizes completedAt to UTC",
			completedAt: time.Now().In(time.FixedZone("JST", 9*60*60)),
		},
		{
			name:        "truncates completedAt to microseconds",
			completedAt: time.Now().Add(123 * time.Nanosecond),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completed := NewCompletedTask(task, tt.completedAt)

			if completed == nil {
				t.Fatal("NewCompletedTask returned nil")
			}

			// Verify all fields are copied correctly
			if completed.ID() != task.ID() {
				t.Errorf("ID mismatch: got %v, want %v", completed.ID(), task.ID())
			}

			if completed.UserID() != task.UserID() {
				t.Errorf("UserID mismatch: got %v, want %v", completed.UserID(), task.UserID())
			}

			if completed.Title() != task.Title() {
				t.Errorf("Title mismatch: got %q, want %q", completed.Title(), task.Title())
			}

			if completed.TaskType() != task.TaskType() {
				t.Errorf("TaskType mismatch: got %v, want %v", completed.TaskType(), task.TaskType())
			}

			if completed.Description() != task.Description() {
				t.Errorf("Description mismatch: got %q, want %q", completed.Description(), task.Description())
			}

			if completed.ScheduledAt() == nil || task.ScheduledAt() == nil {
				if completed.ScheduledAt() != task.ScheduledAt() {
					t.Errorf("ScheduledAt mismatch: got %v, want %v", completed.ScheduledAt(), task.ScheduledAt())
				}
			} else if !completed.ScheduledAt().Equal(*task.ScheduledAt()) {
				t.Errorf("ScheduledAt mismatch: got %v, want %v", *completed.ScheduledAt(), *task.ScheduledAt())
			}

			if !completed.CreatedAt().Equal(task.CreatedAt()) {
				t.Errorf("CreatedAt mismatch: got %v, want %v", completed.CreatedAt(), task.CreatedAt())
			}

			if !completed.TargetAt().Equal(task.TargetAt()) {
				t.Errorf("TargetAt mismatch: got %v, want %v", completed.TargetAt(), task.TargetAt())
			}

			if completed.Color() != task.Color() {
				t.Errorf("Color mismatch: got %v, want %v", completed.Color(), task.Color())
			}

			// Verify completedAt is normalized
			expectedCompletedAt := tt.completedAt.UTC().Truncate(time.Microsecond)
			if !completed.CompletedAt().Equal(expectedCompletedAt) {
				t.Errorf("CompletedAt not normalized: got %v, want %v", completed.CompletedAt(), expectedCompletedAt)
			}

			// Verify completedAt is in UTC
			if completed.CompletedAt().Location() != time.UTC {
				t.Errorf("CompletedAt not in UTC: got %v", completed.CompletedAt().Location())
			}
		})
	}
}

func TestNewCompletedTaskWithNilScheduledAt(t *testing.T) {
	userID, err := user.NewID()
	if err != nil {
		t.Fatalf("failed to create user ID: %v", err)
	}

	taskID, err := NewID()
	if err != nil {
		t.Fatalf("failed to create task ID: %v", err)
	}

	createdAt := time.Now().UTC().Truncate(time.Microsecond)
	targetAt := createdAt.Add(1 * time.Hour)
	color := MustColor("#FF6B6B")

	task, err := NewTask(
		taskID,
		userID,
		"Test Task",
		TypeNear,
		StatusActive,
		"",
		nil, // no scheduled time
		createdAt,
		targetAt,
		color,
	)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	completed := NewCompletedTask(task, time.Now())

	if completed.ScheduledAt() != nil {
		t.Errorf("ScheduledAt should be nil, got %v", completed.ScheduledAt())
	}
}
