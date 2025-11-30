package task

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
)

func TestNewIDSuccess(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "generates valid UUIDv7"},
		{name: "generates non-empty ID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := NewID()
			if err != nil {
				t.Fatalf("NewID() unexpected error: %v", err)
			}

			if id.String() == "" {
				t.Error("NewID() returned empty ID")
			}

			parsedUUID := uuid.UUID(id)
			if parsedUUID.Version() != 7 {
				t.Errorf("NewID() returned UUIDv%d, want v7", parsedUUID.Version())
			}
		})
	}
}

func TestNewIDFromStringSuccess(t *testing.T) {
	validID, err := NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	validIDStr := validID.String()

	tests := []struct {
		name  string
		input string
	}{
		{name: "valid UUIDv7 string", input: validIDStr},
		{name: "round-trip ID preservation", input: validIDStr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := NewIDFromString(tt.input)
			if err != nil {
				t.Fatalf("NewIDFromString(%q) unexpected error: %v", tt.input, err)
			}

			if id.String() != tt.input {
				t.Errorf("NewIDFromString(%q) = %q, want %q", tt.input, id.String(), tt.input)
			}

			parsedUUID := uuid.UUID(id)
			if parsedUUID.Version() != 7 {
				t.Errorf("NewIDFromString(%q) returned UUIDv%d, want v7", tt.input, parsedUUID.Version())
			}
		})
	}

	t.Run("round-trip consistency: NewID -> String -> NewIDFromString", func(t *testing.T) {
		originalID, err := NewID()
		if err != nil {
			t.Fatalf("NewID() error: %v", err)
		}

		idStr := originalID.String()

		parsedID, err := NewIDFromString(idStr)
		if err != nil {
			t.Fatalf("NewIDFromString(%q) error: %v", idStr, err)
		}

		if parsedID.String() != originalID.String() {
			t.Errorf("round-trip failed: got %q, want %q", parsedID.String(), originalID.String())
		}
	})
}

func TestNewIDFromStringErrors(t *testing.T) {
	uuidv4 := uuid.New()

	tests := []struct {
		name        string
		input       string
		expectedErr error
	}{
		{
			name:        "empty string",
			input:       "",
			expectedErr: ErrIDInvalidFormat,
		},
		{
			name:        "invalid UUID format",
			input:       "not-a-uuid",
			expectedErr: ErrIDInvalidFormat,
		},
		{
			name:        "non-UUID string",
			input:       "12345678-abcd",
			expectedErr: ErrIDInvalidFormat,
		},
		{
			name:        "UUIDv4 instead of v7",
			input:       uuidv4.String(),
			expectedErr: ErrIDInvalidV7,
		},
		{
			name:        "malformed UUID with correct length",
			input:       "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
			expectedErr: ErrIDInvalidFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := NewIDFromString(tt.input)
			if err == nil {
				t.Fatalf("NewIDFromString(%q) expected error, got ID: %s", tt.input, id.String())
			}

			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("NewIDFromString(%q) error = %v, want %v", tt.input, err, tt.expectedErr)
			}
		})
	}
}

func TestIDString(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "converts ID to string"},
		{name: "string is valid UUID format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := NewID()
			if err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			idStr := id.String()
			if idStr == "" {
				t.Error("ID.String() returned empty string")
			}

			_, err = uuid.Parse(idStr)
			if err != nil {
				t.Errorf("ID.String() = %q is not valid UUID format: %v", idStr, err)
			}

			if !strings.Contains(idStr, "-") {
				t.Errorf("ID.String() = %q does not appear to be UUID format (missing dashes)", idStr)
			}
		})
	}
}

func TestNewTypeSuccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		typeStr  string
		expected Type
	}{
		{
			name:     "urgent",
			typeStr:  "urgent",
			expected: TypeUrgent,
		},
		{
			name:     "normal",
			typeStr:  "normal",
			expected: TypeNormal,
		},
		{
			name:     "low",
			typeStr:  "low",
			expected: TypeLow,
		},
		{
			name:     "has_due_time",
			typeStr:  "has_due_time",
			expected: TypeHasDueTime,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskType, err := NewType(tt.typeStr)
			if err != nil {
				t.Fatalf("NewType(%q) unexpected error: %v", tt.typeStr, err)
			}

			if taskType != tt.expected {
				t.Errorf("NewType(%q) = %v, want %v", tt.typeStr, taskType, tt.expected)
			}
		})
	}
}

func TestNewTypeErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		expectedErr error
	}{
		{
			name:        "empty string",
			input:       "",
			expectedErr: ErrInvalidTaskType,
		},
		{
			name:        "invalid type string",
			input:       "invalid_type",
			expectedErr: ErrInvalidTaskType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskType, err := NewType(tt.input)
			if err == nil {
				t.Fatalf("NewType(%q) expected error, got type: %v", tt.input, taskType)
			}

			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("NewType(%q) error = %v, want %v", tt.input, err, tt.expectedErr)
			}
		})
	}
}

func NewStatusSuccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		statusStr string
		expected  Status
	}{
		{
			name:      "active",
			statusStr: "active",
			expected:  StatusActive,
		},
		{
			name:      "completed",
			statusStr: "completed",
			expected:  StatusCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskStatus, err := NewStatus(tt.statusStr)
			if err != nil {
				t.Fatalf("NewStatus(%q) unexpected error: %v", tt.statusStr, err)
			}

			if taskStatus != tt.expected {
				t.Errorf("NewStatus(%q) = %v, want %v", tt.statusStr, taskStatus, tt.expected)
			}
		})
	}
}

func TestNewStatusErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		expectedErr error
	}{
		{
			name:        "empty string",
			input:       "",
			expectedErr: ErrInvalidTaskStatus,
		},
		{
			name:        "invalid status string",
			input:       "invalid_status",
			expectedErr: ErrInvalidTaskStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskStatus, err := NewStatus(tt.input)
			if err == nil {
				t.Fatalf("NewStatus(%q) expected error, got status: %v", tt.input, taskStatus)
			}

			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("NewStatus(%q) error = %v, want %v", tt.input, err, tt.expectedErr)
			}
		})
	}
}

func TestNewTaskSuccess(t *testing.T) {
	t.Parallel()

	validID, err := NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	taskNormalType, err := NewType("normal")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	taskDueType, err := NewType("has_due_time")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	taskStatus, err := NewStatus("active")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	createTime := time.Now().UTC().Truncate(time.Microsecond)

	type args struct {
		id          ID
		userID      string
		title       string
		taskType    Type
		status      Status
		description *string
		dueTime     *time.Time
		createdAt   time.Time
	}

	tests := []struct {
		name     string
		args     args
		expected *Task
	}{
		{
			name: "creates non-due task without optional fields",
			args: args{
				id:          validID,
				userID:      "user123",
				title:       "Test Task",
				taskType:    taskNormalType,
				status:      taskStatus,
				description: nil,
				dueTime:     nil,
				createdAt:   createTime,
			},
			expected: &Task{
				id:          validID,
				userID:      "user123",
				title:       "Test Task",
				taskType:    taskNormalType,
				taskStatus:  taskStatus,
				description: nil,
				dueTime:     nil,
				createdAt:   createTime,
			},
		},
		{
			name: "creates due task with all fields",
			args: args{
				id:       validID,
				userID:   "user456",
				title:    "Another Task",
				taskType: taskDueType,
				status:   taskStatus,
				description: func() *string {
					desc := "This is a detailed description."

					return &desc
				}(),
				dueTime: func() *time.Time {
					due := createTime.Add(48 * time.Hour)

					return &due
				}(),
				createdAt: createTime,
			},
			expected: &Task{
				id:         validID,
				userID:     "user456",
				title:      "Another Task",
				taskType:   taskDueType,
				taskStatus: taskStatus,
				description: func() *string {
					desc := "This is a detailed description."

					return &desc
				}(),
				dueTime: func() *time.Time {
					due := createTime.Add(48 * time.Hour)

					return &due
				}(),
				createdAt: createTime,
			},
		},
		{
			name: "creates task with description",
			args: args{
				id:       validID,
				userID:   "user789",
				title:    "Task with Description",
				taskType: taskNormalType,
				status:   taskStatus,
				description: func() *string {
					desc := "Just a simple description."

					return &desc
				}(),
				dueTime:   nil,
				createdAt: createTime,
			},
			expected: &Task{
				id:         validID,
				userID:     "user789",
				title:      "Task with Description",
				taskType:   taskNormalType,
				taskStatus: taskStatus,
				description: func() *string {
					desc := "Just a simple description."

					return &desc
				}(),
				dueTime:   nil,
				createdAt: createTime,
			},
		},
		{
			name: "creates task with due time",
			args: args{
				id:          validID,
				userID:      "user101",
				title:       "Task with Due Time",
				taskType:    taskDueType,
				status:      taskStatus,
				description: nil,
				dueTime: func() *time.Time {
					due := createTime.Add(24 * time.Hour)

					return &due
				}(),
				createdAt: createTime,
			},
			expected: &Task{
				id:          validID,
				userID:      "user101",
				title:       "Task with Due Time",
				taskType:    taskDueType,
				taskStatus:  taskStatus,
				description: nil,
				dueTime: func() *time.Time {
					due := createTime.Add(24 * time.Hour)

					return &due
				}(),
				createdAt: createTime,
			},
		},
		{
			name: "1 rune title",
			args: args{
				id:          validID,
				userID:      "user102",
				title:       "A",
				taskType:    taskNormalType,
				status:      taskStatus,
				description: nil,
				dueTime:     nil,
				createdAt:   createTime,
			},
			expected: &Task{
				id:          validID,
				userID:      "user102",
				title:       "A",
				taskType:    taskNormalType,
				taskStatus:  taskStatus,
				description: nil,
				dueTime:     nil,
				createdAt:   createTime,
			},
		},
		{
			name: "500 rune title",
			args: args{
				id:       validID,
				userID:   "user103",
				title:    strings.Repeat("T", 500),
				taskType: taskNormalType,
				status:   taskStatus,
				description: func() *string {
					desc := "Description for 500 rune title."

					return &desc
				}(),
				dueTime:   nil,
				createdAt: createTime,
			},
			expected: &Task{
				id:         validID,
				userID:     "user103",
				title:      strings.Repeat("T", 500),
				taskType:   taskNormalType,
				taskStatus: taskStatus,
				description: func() *string {
					desc := "Description for 500 rune title."

					return &desc
				}(),
				dueTime:   nil,
				createdAt: createTime,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := NewTask(
				tt.args.id,
				tt.args.userID,
				tt.args.title,
				tt.args.taskType,
				tt.args.status,
				tt.args.description,
				tt.args.dueTime,
				tt.args.createdAt,
			)
			if err != nil {
				t.Fatalf("NewTask() unexpected error: %v", err)
			}

			opts := cmp.Options{
				cmp.AllowUnexported(Task{}),
			}
			if diff := cmp.Diff(tt.expected, task, opts); diff != "" {
				t.Errorf("NewTask() mismatch (-want +got):\n%s", diff)
			}

			if task.ID() != tt.expected.id {
				t.Errorf("Task.ID() = %v, want %v", task.ID(), tt.expected.ID())
			}

			if task.UserID() != tt.expected.userID {
				t.Errorf("Task.UserID() = %v, want %v", task.UserID(), tt.expected.userID)
			}

			if task.Title() != tt.expected.title {
				t.Errorf("Task.Title() = %v, want %v", task.Title(), tt.expected.title)
			}

			if task.TaskType() != tt.expected.taskType {
				t.Errorf("Task.Type() = %v, want %v", task.TaskType(), tt.expected.taskType)
			}

			if task.TaskStatus() != tt.expected.taskStatus {
				t.Errorf("Task.Status() = %v, want %v", task.TaskStatus(), tt.expected.taskStatus)
			}

			if (task.Description() == nil) != (tt.expected.description == nil) ||
				(task.Description() != nil && *task.Description() != *tt.expected.description) {
				t.Errorf("Task.Description() = %v, want %v", task.Description(), tt.expected.description)
			}

			if (task.DueTime() == nil) != (tt.expected.dueTime == nil) ||
				(task.DueTime() != nil && !task.DueTime().Equal(*tt.expected.dueTime)) {
				t.Errorf("Task.DueTime() = %v, want %v", task.DueTime(), tt.expected.dueTime)
			}

			if !task.CreatedAt().Equal(tt.expected.createdAt) {
				t.Errorf("Task.CreatedAt() = %v, want %v", task.CreatedAt(), tt.expected.createdAt)
			}
		})
	}
}

func TestNewTaskErrors(t *testing.T) {
	t.Parallel()

	validID, err := NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	taskNormalType, err := NewType("normal")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	taskDueType, err := NewType("has_due_time")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	taskStatus, err := NewStatus("active")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	createTime := time.Now().UTC().Truncate(time.Microsecond)

	type args struct {
		id          ID
		userID      string
		title       string
		taskType    Type
		status      Status
		description *string
		dueTime     *time.Time
		createdAt   time.Time
	}

	tests := []struct {
		name        string
		args        args
		expectedErr error
	}{
		{
			name: "empty userID",
			args: args{
				id:        validID,
				userID:    "",
				title:     "Test Task",
				taskType:  taskNormalType,
				status:    taskStatus,
				createdAt: createTime,
			},
			expectedErr: ErrUserIDEmpty,
		},
		{
			name: "empty title",
			args: args{
				id:        validID,
				userID:    "user123",
				title:     "",
				taskType:  taskNormalType,
				status:    taskStatus,
				createdAt: createTime,
			},
			expectedErr: ErrTitleEmpty,
		},
		{
			name: "501 rune title is too long",
			args: args{
				id:        validID,
				userID:    "user123",
				title:     strings.Repeat("T", 501),
				taskType:  taskNormalType,
				status:    taskStatus,
				createdAt: createTime,
			},
			expectedErr: ErrTitleTooLong,
		},
		{
			name: "nil due time for due type",
			args: args{
				id:        validID,
				userID:    "user123",
				title:     "Test Task",
				taskType:  taskDueType,
				status:    taskStatus,
				dueTime:   nil,
				createdAt: createTime,
			},
			expectedErr: ErrDueTimeRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := NewTask(
				tt.args.id,
				tt.args.userID,
				tt.args.title,
				tt.args.taskType,
				tt.args.status,
				tt.args.description,
				tt.args.dueTime,
				tt.args.createdAt,
			)
			if err == nil {
				t.Fatalf("NewTask() expected error, got task: %+v", task)
			}

			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("NewTask() error = %v, want %v", err, tt.expectedErr)
			}
		})
	}
}
