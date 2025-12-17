package task

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/KasumiMercury/primind-central-backend/internal/task/domain/user"
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
			name:     "scheduled",
			typeStr:  "scheduled",
			expected: TypeScheduled,
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

func TestNewStatusSuccess(t *testing.T) {
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

	validUserID, err := user.NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	taskNormalType, err := NewType("normal")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	taskscheduledType, err := NewType("scheduled")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	taskStatus, err := NewStatus("active")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	createTime := time.Now().UTC().Truncate(time.Microsecond)

	targetTime := createTime.Add(1 * time.Hour)
	scheduledTargetTime := createTime.Add(48 * time.Hour)
	scheduledTargetTime24h := createTime.Add(24 * time.Hour)

	validColor := MustColor("#FF6B6B")

	type args struct {
		id          ID
		userID      user.ID
		title       string
		taskType    Type
		status      Status
		description string
		scheduledAt *time.Time
		createdAt   time.Time
		targetAt    time.Time
		color       Color
	}

	tests := []struct {
		name     string
		args     args
		expected *Task
	}{
		{
			name: "creates non-scheduled task without optional fields",
			args: args{
				id:          validID,
				userID:      validUserID,
				title:       "",
				taskType:    taskNormalType,
				status:      taskStatus,
				description: "",
				scheduledAt: nil,
				createdAt:   createTime,
				targetAt:    targetTime,
				color:       validColor,
			},
			expected: &Task{
				id:          validID,
				userID:      validUserID,
				title:       "",
				taskType:    taskNormalType,
				taskStatus:  taskStatus,
				description: "",
				scheduledAt: nil,
				createdAt:   createTime,
				targetAt:    targetTime,
				color:       validColor,
			},
		},
		{
			name: "creates scheduled task with all fields",
			args: args{
				id:          validID,
				userID:      validUserID,
				title:       "Another Task",
				taskType:    taskscheduledType,
				status:      taskStatus,
				description: "This is a detailed description.",
				scheduledAt: func() *time.Time {
					scheduled := createTime.Add(48 * time.Hour)

					return &scheduled
				}(),
				createdAt: createTime,
				targetAt:  scheduledTargetTime,
				color:     validColor,
			},
			expected: &Task{
				id:          validID,
				userID:      validUserID,
				title:       "Another Task",
				taskType:    taskscheduledType,
				taskStatus:  taskStatus,
				description: "This is a detailed description.",
				scheduledAt: func() *time.Time {
					scheduled := createTime.Add(48 * time.Hour)

					return &scheduled
				}(),
				createdAt: createTime,
				targetAt:  scheduledTargetTime,
				color:     validColor,
			},
		},
		{
			name: "creates task with description",
			args: args{
				id:          validID,
				userID:      validUserID,
				title:       "Task with Description",
				taskType:    taskNormalType,
				status:      taskStatus,
				description: "Just a simple description.",
				scheduledAt: nil,
				createdAt:   createTime,
				targetAt:    targetTime,
				color:       validColor,
			},
			expected: &Task{
				id:          validID,
				userID:      validUserID,
				title:       "Task with Description",
				taskType:    taskNormalType,
				taskStatus:  taskStatus,
				description: "Just a simple description.",
				scheduledAt: nil,
				createdAt:   createTime,
				targetAt:    targetTime,
				color:       validColor,
			},
		},
		{
			name: "creates task with scheduled time",
			args: args{
				id:          validID,
				userID:      validUserID,
				title:       "Task with scheduled Time",
				taskType:    taskscheduledType,
				status:      taskStatus,
				description: "",
				scheduledAt: func() *time.Time {
					scheduled := createTime.Add(24 * time.Hour)

					return &scheduled
				}(),
				createdAt: createTime,
				targetAt:  scheduledTargetTime24h,
				color:     validColor,
			},
			expected: &Task{
				id:          validID,
				userID:      validUserID,
				title:       "Task with scheduled Time",
				taskType:    taskscheduledType,
				taskStatus:  taskStatus,
				description: "",
				scheduledAt: func() *time.Time {
					scheduled := createTime.Add(24 * time.Hour)

					return &scheduled
				}(),
				createdAt: createTime,
				targetAt:  scheduledTargetTime24h,
				color:     validColor,
			},
		},
		{
			name: "empty title",
			args: args{
				id:          validID,
				userID:      validUserID,
				title:       "",
				taskType:    taskNormalType,
				status:      taskStatus,
				description: "",
				scheduledAt: nil,
				createdAt:   createTime,
				targetAt:    targetTime,
				color:       validColor,
			},
			expected: &Task{
				id:          validID,
				userID:      validUserID,
				title:       "",
				taskType:    taskNormalType,
				taskStatus:  taskStatus,
				description: "",
				scheduledAt: nil,
				createdAt:   createTime,
				targetAt:    targetTime,
				color:       validColor,
			},
		},
		{
			name: "500 rune title",
			args: args{
				id:          validID,
				userID:      validUserID,
				title:       strings.Repeat("T", 500),
				taskType:    taskNormalType,
				status:      taskStatus,
				description: "Description for 500 rune title.",
				scheduledAt: nil,
				createdAt:   createTime,
				targetAt:    targetTime,
				color:       validColor,
			},
			expected: &Task{
				id:          validID,
				userID:      validUserID,
				title:       strings.Repeat("T", 500),
				taskType:    taskNormalType,
				taskStatus:  taskStatus,
				description: "Description for 500 rune title.",
				scheduledAt: nil,
				createdAt:   createTime,
				targetAt:    targetTime,
				color:       validColor,
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
				tt.args.scheduledAt,
				tt.args.createdAt,
				tt.args.targetAt,
				tt.args.color,
			)
			if err != nil {
				t.Fatalf("NewTask() unexpected error: %v", err)
			}

			opts := cmp.Options{
				cmp.AllowUnexported(Task{}, Color{}),
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

			if task.Description() != tt.expected.description {
				t.Errorf("Task.Description() = %v, want %v", task.Description(), tt.expected.description)
			}

			if (task.ScheduledAt() == nil) != (tt.expected.scheduledAt == nil) ||
				(task.ScheduledAt() != nil && !task.ScheduledAt().Equal(*tt.expected.scheduledAt)) {
				t.Errorf("Task.ScheduledAt() = %v, want %v", task.ScheduledAt(), tt.expected.scheduledAt)
			}

			if !task.CreatedAt().Equal(tt.expected.createdAt) {
				t.Errorf("Task.CreatedAt() = %v, want %v", task.CreatedAt(), tt.expected.createdAt)
			}

			if !task.TargetAt().Equal(tt.expected.targetAt) {
				t.Errorf("Task.TargetAt() = %v, want %v", task.TargetAt(), tt.expected.targetAt)
			}

			if task.Color().String() != tt.expected.color.String() {
				t.Errorf("Task.Color() = %v, want %v", task.Color(), tt.expected.color)
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

	validUserID, err := user.NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	taskUrgentType, err := NewType("urgent")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	taskNormalType, err := NewType("normal")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	taskLowType, err := NewType("low")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	taskscheduledType, err := NewType("scheduled")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	taskStatus, err := NewStatus("active")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	createTime := time.Now().UTC().Truncate(time.Microsecond)
	targetTime := createTime.Add(1 * time.Hour)

	validColor := MustColor("#FF6B6B")

	type args struct {
		id          ID
		userID      user.ID
		title       string
		taskType    Type
		status      Status
		description string
		scheduledAt *time.Time
		createdAt   time.Time
		targetAt    time.Time
		color       Color
	}

	tests := []struct {
		name        string
		args        args
		expectedErr error
	}{
		{
			name: "501 rune title is too long",
			args: args{
				id:        validID,
				userID:    validUserID,
				title:     strings.Repeat("T", 501),
				taskType:  taskNormalType,
				status:    taskStatus,
				createdAt: createTime,
				targetAt:  targetTime,
				color:     validColor,
			},
			expectedErr: ErrTitleTooLong,
		},
		{
			name: "nil scheduled time for scheduled type",
			args: args{
				id:          validID,
				userID:      validUserID,
				title:       "Test Task",
				taskType:    taskscheduledType,
				status:      taskStatus,
				scheduledAt: nil,
				createdAt:   createTime,
				targetAt:    targetTime,
				color:       validColor,
			},
			expectedErr: ErrScheduledAtRequired,
		},
		{
			name: "scheduled time provided for urgent type",
			args: args{
				id:       validID,
				userID:   validUserID,
				title:    "Test Task",
				taskType: taskUrgentType,
				status:   taskStatus,
				scheduledAt: func() *time.Time {
					scheduled := createTime.Add(24 * time.Hour)

					return &scheduled
				}(),
				createdAt: createTime,
				targetAt:  targetTime,
				color:     validColor,
			},
			expectedErr: ErrScheduledAtNotAllowed,
		},
		{
			name: "scheduled time provided for normal type",
			args: args{
				id:       validID,
				userID:   validUserID,
				title:    "Test Task",
				taskType: taskNormalType,
				status:   taskStatus,
				scheduledAt: func() *time.Time {
					scheduled := createTime.Add(24 * time.Hour)

					return &scheduled
				}(),
				createdAt: createTime,
				targetAt:  targetTime,
				color:     validColor,
			},
			expectedErr: ErrScheduledAtNotAllowed,
		},
		{
			name: "scheduled time provided for low type",
			args: args{
				id:       validID,
				userID:   validUserID,
				title:    "Test Task",
				taskType: taskLowType,
				status:   taskStatus,
				scheduledAt: func() *time.Time {
					scheduled := createTime.Add(24 * time.Hour)

					return &scheduled
				}(),
				createdAt: createTime,
				targetAt:  targetTime,
				color:     validColor,
			},
			expectedErr: ErrScheduledAtNotAllowed,
		},
		{
			name: "scheduled time before created at",
			args: args{
				id:       validID,
				userID:   validUserID,
				title:    "Test Task",
				taskType: taskscheduledType,
				status:   taskStatus,
				scheduledAt: func() *time.Time {
					scheduled := createTime.Add(-1 * time.Hour)

					return &scheduled
				}(),
				createdAt: createTime,
				targetAt:  targetTime,
				color:     validColor,
			},
			expectedErr: ErrScheduledAtBeforeCreatedAt,
		},
		{
			name: "empty color",
			args: args{
				id:        validID,
				userID:    validUserID,
				title:     "Test Task",
				taskType:  taskNormalType,
				status:    taskStatus,
				createdAt: createTime,
				targetAt:  targetTime,
				color:     Color{},
			},
			expectedErr: ErrColorEmpty,
		},
		{
			name: "invalid color format",
			args: args{
				id:        validID,
				userID:    validUserID,
				title:     "Test Task",
				taskType:  taskNormalType,
				status:    taskStatus,
				createdAt: createTime,
				targetAt:  targetTime,
				color:     Color{hex: "#FFF"},
			},
			expectedErr: ErrColorInvalidFormat,
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
				tt.args.scheduledAt,
				tt.args.createdAt,
				tt.args.targetAt,
				tt.args.color,
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

func TestCreateTaskWithPreGeneratedID(t *testing.T) {
	t.Parallel()

	validUserID, err := user.NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	taskNormalType, err := NewType("normal")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	taskscheduledType, err := NewType("scheduled")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	validColor := MustColor("#FF6B6B")

	tests := []struct {
		name        string
		taskID      *ID
		userID      user.ID
		title       string
		taskType    Type
		description string
		scheduledAt *time.Time
		color       Color
	}{
		{
			name: "create task with valid predefined UUIDv7",
			taskID: func() *ID {
				id, _ := NewID()

				return &id
			}(),
			userID:      validUserID,
			title:       "Task with predefined ID",
			taskType:    taskNormalType,
			description: "This task has a predefined ID",
			scheduledAt: nil,
			color:       validColor,
		},
		{
			name:        "create task without ID",
			taskID:      nil,
			userID:      validUserID,
			title:       "Task without ID",
			taskType:    taskNormalType,
			description: "This task will get an auto-generated ID",
			scheduledAt: nil,
			color:       validColor,
		},
		{
			name: "create scheduled task with predefined ID",
			taskID: func() *ID {
				id, _ := NewID()

				return &id
			}(),
			userID:      validUserID,
			title:       "scheduled Task with predefined ID",
			taskType:    taskscheduledType,
			description: "",
			scheduledAt: func() *time.Time {
				scheduled := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Microsecond)

				return &scheduled
			}(),
			color: validColor,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := CreateTask(
				tt.taskID,
				tt.userID,
				tt.title,
				tt.taskType,
				tt.description,
				tt.scheduledAt,
				tt.color,
			)
			if err != nil {
				t.Fatalf("CreateTask() unexpected error: %v", err)
			}

			if task == nil {
				t.Fatal("CreateTask() returned nil task")
			}

			if tt.taskID != nil {
				if task.ID().String() != tt.taskID.String() {
					t.Errorf("CreateTask() task ID = %v, want %v", task.ID().String(), tt.taskID.String())
				}
			} else {
				if task.ID().String() == "" {
					t.Error("CreateTask() auto-generated ID is empty")
				}

				parsedID, err := NewIDFromString(task.ID().String())
				if err != nil {
					t.Errorf("CreateTask() auto-generated ID is invalid: %v", err)
				}

				if parsedID.String() != task.ID().String() {
					t.Errorf("CreateTask() ID round-trip failed: got %v, want %v", parsedID.String(), task.ID().String())
				}
			}

			if task.UserID() != tt.userID {
				t.Errorf("CreateTask() userID = %v, want %v", task.UserID(), tt.userID)
			}

			if task.Title() != tt.title {
				t.Errorf("CreateTask() title = %v, want %v", task.Title(), tt.title)
			}

			if task.TaskType() != tt.taskType {
				t.Errorf("CreateTask() taskType = %v, want %v", task.TaskType(), tt.taskType)
			}

			if task.TaskStatus() != StatusPendingReminders {
				t.Errorf("CreateTask() taskStatus = %v, want %v", task.TaskStatus(), StatusPendingReminders)
			}

			if task.Description() != tt.description {
				t.Errorf("CreateTask() description = %v, want %v", task.Description(), tt.description)
			}

			if (task.ScheduledAt() == nil) != (tt.scheduledAt == nil) ||
				(task.ScheduledAt() != nil && !task.ScheduledAt().Equal(*tt.scheduledAt)) {
				t.Errorf("CreateTask() scheduledAt = %v, want %v", task.ScheduledAt(), tt.scheduledAt)
			}
		})
	}
}

func TestCreateTaskTargetAtCalculation(t *testing.T) {
	t.Parallel()

	validUserID, err := user.NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	validColor := MustColor("#FF6B6B")

	tests := []struct {
		name               string
		taskType           Type
		scheduledAt        *time.Time
		expectedOffsetFunc func(createdAt time.Time, scheduledAt *time.Time) time.Time
	}{
		{
			name:        "urgent task target_at is created_at + 15 minutes",
			taskType:    TypeUrgent,
			scheduledAt: nil,
			expectedOffsetFunc: func(createdAt time.Time, scheduledAt *time.Time) time.Time {
				return createdAt.Add(15 * time.Minute)
			},
		},
		{
			name:        "normal task target_at is created_at + 1 hour",
			taskType:    TypeNormal,
			scheduledAt: nil,
			expectedOffsetFunc: func(createdAt time.Time, scheduledAt *time.Time) time.Time {
				return createdAt.Add(1 * time.Hour)
			},
		},
		{
			name:        "low task target_at is created_at + 6 hours",
			taskType:    TypeLow,
			scheduledAt: nil,
			expectedOffsetFunc: func(createdAt time.Time, scheduledAt *time.Time) time.Time {
				return createdAt.Add(6 * time.Hour)
			},
		},
		{
			name:     "scheduled task target_at equals scheduled_at",
			taskType: TypeScheduled,
			scheduledAt: func() *time.Time {
				scheduled := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Microsecond)

				return &scheduled
			}(),
			expectedOffsetFunc: func(createdAt time.Time, scheduledAt *time.Time) time.Time {
				return *scheduledAt
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beforeCreate := time.Now().UTC().Truncate(time.Microsecond)

			task, err := CreateTask(
				nil,
				validUserID,
				"Test Task",
				tt.taskType,
				"Test description",
				tt.scheduledAt,
				validColor,
			)
			if err != nil {
				t.Fatalf("CreateTask() unexpected error: %v", err)
			}

			afterCreate := time.Now().UTC().Truncate(time.Microsecond)

			if task == nil {
				t.Fatal("CreateTask() returned nil task")
			}

			// Verify targetAt is set
			if task.TargetAt().IsZero() {
				t.Error("CreateTask() targetAt is zero")
			}

			// Verify targetAt is calculated correctly based on task type
			expectedMinTargetAt := tt.expectedOffsetFunc(beforeCreate, tt.scheduledAt).Truncate(time.Microsecond)
			expectedMaxTargetAt := tt.expectedOffsetFunc(afterCreate, tt.scheduledAt).Truncate(time.Microsecond)
			actualTargetAt := task.TargetAt().Truncate(time.Microsecond)

			if tt.taskType == TypeScheduled {
				// For scheduled tasks, target_at should exactly equal scheduled_at
				expectedTargetAt := tt.scheduledAt.UTC().Truncate(time.Microsecond)
				if !actualTargetAt.Equal(expectedTargetAt) {
					t.Errorf("CreateTask() targetAt = %v, want %v", actualTargetAt, expectedTargetAt)
				}
			} else {
				// For non-scheduled tasks, targetAt should be within the expected range
				if actualTargetAt.Before(expectedMinTargetAt) || actualTargetAt.After(expectedMaxTargetAt) {
					t.Errorf("CreateTask() targetAt = %v, expected between %v and %v",
						actualTargetAt, expectedMinTargetAt, expectedMaxTargetAt)
				}
			}

			// Verify targetAt is in UTC
			if task.TargetAt().Location() != time.UTC {
				t.Errorf("CreateTask() targetAt location = %v, want UTC", task.TargetAt().Location())
			}
		})
	}
}

func TestNewTaskTargetAtNormalization(t *testing.T) {
	t.Parallel()

	validID, err := NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	validUserID, err := user.NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	taskNormalType, err := NewType("normal")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	taskStatus, err := NewStatus("active")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	validColor := MustColor("#FF6B6B")

	t.Run("targetAt is normalized to UTC", func(t *testing.T) {
		createTime := time.Now().UTC().Truncate(time.Microsecond)
		// Create targetAt in a non-UTC timezone
		jst := time.FixedZone("JST", 9*60*60)
		targetAtJST := createTime.Add(1 * time.Hour).In(jst)

		task, err := NewTask(
			validID,
			validUserID,
			"Test Task",
			taskNormalType,
			taskStatus,
			"Test description",
			nil,
			createTime,
			targetAtJST,
			validColor,
		)
		if err != nil {
			t.Fatalf("NewTask() unexpected error: %v", err)
		}

		// Verify targetAt is in UTC
		if task.TargetAt().Location() != time.UTC {
			t.Errorf("Task.TargetAt().Location() = %v, want UTC", task.TargetAt().Location())
		}

		// Verify the time value is equivalent
		if !task.TargetAt().Equal(targetAtJST) {
			t.Errorf("Task.TargetAt() = %v, want equivalent to %v", task.TargetAt(), targetAtJST)
		}
	})

	t.Run("targetAt is truncated to microseconds", func(t *testing.T) {
		createTime := time.Now().UTC().Truncate(time.Microsecond)
		// Create targetAt with nanosecond precision
		targetAtNano := createTime.Add(1*time.Hour + 123*time.Nanosecond)

		task, err := NewTask(
			validID,
			validUserID,
			"Test Task",
			taskNormalType,
			taskStatus,
			"Test description",
			nil,
			createTime,
			targetAtNano,
			validColor,
		)
		if err != nil {
			t.Fatalf("NewTask() unexpected error: %v", err)
		}

		// Verify nanoseconds beyond microseconds are truncated
		expectedTargetAt := targetAtNano.UTC().Truncate(time.Microsecond)
		if !task.TargetAt().Equal(expectedTargetAt) {
			t.Errorf("Task.TargetAt() = %v, want %v (truncated)", task.TargetAt(), expectedTargetAt)
		}
	})
}
