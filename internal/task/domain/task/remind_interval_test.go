package task

import (
	"testing"
	"time"

	"github.com/KasumiMercury/primind-central-backend/internal/task/domain/user"
)

func TestDefaultReminderIntervals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		taskType      Type
		expectedCount int
		exists        bool
	}{
		{
			name:          "TypeUrgent has 2 intervals",
			taskType:      TypeUrgent,
			expectedCount: 2,
			exists:        true,
		},
		{
			name:          "TypeNormal has 2 intervals",
			taskType:      TypeNormal,
			expectedCount: 2,
			exists:        true,
		},
		{
			name:          "TypeLow has 3 intervals",
			taskType:      TypeLow,
			expectedCount: 3,
			exists:        true,
		},
		{
			name:          "TypeScheduled uses Low intervals (3 intervals)",
			taskType:      TypeScheduled,
			expectedCount: 3,
			exists:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intervals, ok := DefaultReminderIntervals[tt.taskType]
			if ok != tt.exists {
				t.Errorf("DefaultReminderIntervals[%v] exists = %v, want %v", tt.taskType, ok, tt.exists)
			}

			if ok && len(intervals) != tt.expectedCount {
				t.Errorf("len(DefaultReminderIntervals[%v]) = %d, want %d", tt.taskType, len(intervals), tt.expectedCount)
			}
		})
	}
}

func TestDefaultReminderIntervalsValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		taskType        Type
		expectedMinutes []int
	}{
		{
			name:            "TypeUrgent intervals are 3 and 5 minutes",
			taskType:        TypeUrgent,
			expectedMinutes: []int{3, 5},
		},
		{
			name:            "TypeNormal intervals are 33 and 53 minutes",
			taskType:        TypeNormal,
			expectedMinutes: []int{33, 53},
		},
		{
			name:            "TypeLow intervals are 126, 232, and 315 minutes",
			taskType:        TypeLow,
			expectedMinutes: []int{126, 232, 315},
		},
		{
			name:            "TypeScheduled uses Low intervals (126, 232, 315 minutes)",
			taskType:        TypeScheduled,
			expectedMinutes: []int{126, 232, 315},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intervals := DefaultReminderIntervals[tt.taskType]

			if len(intervals) != len(tt.expectedMinutes) {
				t.Fatalf("len(intervals) = %d, want %d", len(intervals), len(tt.expectedMinutes))
			}

			for i, expectedMin := range tt.expectedMinutes {
				expected := ReminderInterval(time.Duration(expectedMin) * time.Minute)
				if intervals[i] != expected {
					t.Errorf("intervals[%d] = %v, want %v", i, intervals[i], expected)
				}
			}
		})
	}
}

func TestGetReminderIntervalsForType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		taskType      Type
		expectedCount int
	}{
		{
			name:          "TypeUrgent returns 2 intervals",
			taskType:      TypeUrgent,
			expectedCount: 2,
		},
		{
			name:          "TypeNormal returns 2 intervals",
			taskType:      TypeNormal,
			expectedCount: 2,
		},
		{
			name:          "TypeLow returns 3 intervals",
			taskType:      TypeLow,
			expectedCount: 3,
		},
		{
			name:          "TypeScheduled returns 3 intervals (using Low)",
			taskType:      TypeScheduled,
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetReminderIntervalsForType(tt.taskType)
			if len(result) != tt.expectedCount {
				t.Errorf("GetReminderIntervalsForType(%v) returned %d intervals, want %d",
					tt.taskType, len(result), tt.expectedCount)
			}
		})
	}
}

func TestGetReminderIntervalsForTypeUnknown(t *testing.T) {
	t.Parallel()

	t.Run("unknown type returns nil", func(t *testing.T) {
		result := GetReminderIntervalsForType(Type("unknown"))
		if result != nil {
			t.Errorf("GetReminderIntervalsForType(unknown) = %v, want nil", result)
		}
	})
}

func TestCalculateReminderTimes(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	validUserID, err := user.NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	validColor := MustColor("#FF6B6B")

	t.Run("TypeUrgent returns 3 reminder times (2 intervals + targetAt)", func(t *testing.T) {
		taskID, err := NewID()
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		// targetAt = createdAt + 15 minutes (ActivePeriodUrgent)
		targetAt := baseTime.Add(15 * time.Minute)

		task, err := NewTask(
			taskID,
			validUserID,
			"Test Urgent Task",
			TypeUrgent,
			StatusActive,
			"",
			nil,
			baseTime,
			targetAt,
			validColor,
		)
		if err != nil {
			t.Fatalf("NewTask() unexpected error: %v", err)
		}

		info := CalculateReminderTimes(task)

		if info == nil {
			t.Fatal("CalculateReminderTimes returned nil")
		}

		if len(info.ReminderTimes) != 3 {
			t.Errorf("got %d reminder times, want 3", len(info.ReminderTimes))
		}

		// Expected: 3min, 5min, 15min (targetAt)
		expectedTimes := []time.Time{
			baseTime.Add(3 * time.Minute),
			baseTime.Add(5 * time.Minute),
			baseTime.Add(15 * time.Minute),
		}

		for i, expected := range expectedTimes {
			if !info.ReminderTimes[i].Equal(expected) {
				t.Errorf("ReminderTimes[%d] = %v, want %v",
					i, info.ReminderTimes[i], expected)
			}
		}
	})

	t.Run("TypeNormal returns 3 reminder times (2 intervals + targetAt)", func(t *testing.T) {
		taskID, err := NewID()
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		// targetAt = createdAt + 60 minutes (ActivePeriodNormal)
		targetAt := baseTime.Add(60 * time.Minute)

		task, err := NewTask(
			taskID,
			validUserID,
			"Test Normal Task",
			TypeNormal,
			StatusActive,
			"",
			nil,
			baseTime,
			targetAt,
			validColor,
		)
		if err != nil {
			t.Fatalf("NewTask() unexpected error: %v", err)
		}

		info := CalculateReminderTimes(task)

		if info == nil {
			t.Fatal("CalculateReminderTimes returned nil")
		}

		if len(info.ReminderTimes) != 3 {
			t.Errorf("got %d reminder times, want 3", len(info.ReminderTimes))
		}

		// Expected: 33min, 53min, 60min (targetAt)
		expectedTimes := []time.Time{
			baseTime.Add(33 * time.Minute),
			baseTime.Add(53 * time.Minute),
			baseTime.Add(60 * time.Minute),
		}

		for i, expected := range expectedTimes {
			if !info.ReminderTimes[i].Equal(expected) {
				t.Errorf("ReminderTimes[%d] = %v, want %v",
					i, info.ReminderTimes[i], expected)
			}
		}
	})

	t.Run("TypeLow returns 4 reminder times (3 intervals + targetAt)", func(t *testing.T) {
		taskID, err := NewID()
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		// targetAt = createdAt + 360 minutes (ActivePeriodLow = 6 hours)
		targetAt := baseTime.Add(360 * time.Minute)

		task, err := NewTask(
			taskID,
			validUserID,
			"Test Low Task",
			TypeLow,
			StatusActive,
			"",
			nil,
			baseTime,
			targetAt,
			validColor,
		)
		if err != nil {
			t.Fatalf("NewTask() unexpected error: %v", err)
		}

		info := CalculateReminderTimes(task)

		if info == nil {
			t.Fatal("CalculateReminderTimes returned nil")
		}

		if len(info.ReminderTimes) != 4 {
			t.Errorf("got %d reminder times, want 4", len(info.ReminderTimes))
		}

		// Expected: 126min, 232min, 315min, 360min (targetAt)
		expectedTimes := []time.Time{
			baseTime.Add(126 * time.Minute),
			baseTime.Add(232 * time.Minute),
			baseTime.Add(315 * time.Minute),
			baseTime.Add(360 * time.Minute),
		}

		for i, expected := range expectedTimes {
			if !info.ReminderTimes[i].Equal(expected) {
				t.Errorf("ReminderTimes[%d] = %v, want %v",
					i, info.ReminderTimes[i], expected)
			}
		}
	})

	t.Run("TypeScheduled returns 4 reminder times (using Low intervals + targetAt)", func(t *testing.T) {
		taskID, err := NewID()
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		// For scheduled tasks, targetAt = scheduledAt
		scheduledAt := baseTime.Add(24 * time.Hour)

		task, err := NewTask(
			taskID,
			validUserID,
			"Test Scheduled Task",
			TypeScheduled,
			StatusActive,
			"",
			&scheduledAt,
			baseTime,
			scheduledAt, // targetAt equals scheduledAt for scheduled tasks
			validColor,
		)
		if err != nil {
			t.Fatalf("NewTask() unexpected error: %v", err)
		}

		info := CalculateReminderTimes(task)

		if info == nil {
			t.Fatal("CalculateReminderTimes returned nil")
		}

		// TypeScheduled uses Low intervals (3) + targetAt = 4 reminder times
		if len(info.ReminderTimes) != 4 {
			t.Errorf("got %d reminder times, want 4", len(info.ReminderTimes))
		}

		// Expected: 126min, 232min, 315min from createdAt, then scheduledAt
		expectedTimes := []time.Time{
			baseTime.Add(126 * time.Minute),
			baseTime.Add(232 * time.Minute),
			baseTime.Add(315 * time.Minute),
			scheduledAt,
		}

		for i, expected := range expectedTimes {
			if !info.ReminderTimes[i].Equal(expected) {
				t.Errorf("ReminderTimes[%d] = %v, want %v",
					i, info.ReminderTimes[i], expected)
			}
		}
	})

	t.Run("nil task returns nil", func(t *testing.T) {
		info := CalculateReminderTimes(nil)
		if info != nil {
			t.Error("expected nil for nil task")
		}
	})

	t.Run("ReminderInfo contains correct task ID and type", func(t *testing.T) {
		taskID, err := NewID()
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		targetAt := baseTime.Add(15 * time.Minute)

		task, err := NewTask(
			taskID,
			validUserID,
			"Test Task",
			TypeUrgent,
			StatusActive,
			"",
			nil,
			baseTime,
			targetAt,
			validColor,
		)
		if err != nil {
			t.Fatalf("NewTask() unexpected error: %v", err)
		}

		info := CalculateReminderTimes(task)

		if info.TaskID != task.ID() {
			t.Errorf("TaskID = %v, want %v", info.TaskID, task.ID())
		}

		if info.TaskType != task.TaskType() {
			t.Errorf("TaskType = %v, want %v", info.TaskType, task.TaskType())
		}
	})
}

func TestReminderIntervalTypeConversion(t *testing.T) {
	t.Parallel()

	t.Run("ReminderInterval can be converted to time.Duration", func(t *testing.T) {
		interval := DefaultReminderIntervalsUrgent[0] // 3 minutes
		duration := time.Duration(interval)

		if duration != 3*time.Minute {
			t.Errorf("time.Duration(interval) = %v, want %v", duration, 3*time.Minute)
		}
	})

	t.Run("ReminderInterval can be used with time.Add", func(t *testing.T) {
		baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		interval := ReminderInterval(33 * time.Minute)

		result := baseTime.Add(time.Duration(interval))
		expected := baseTime.Add(33 * time.Minute)

		if !result.Equal(expected) {
			t.Errorf("baseTime.Add(interval) = %v, want %v", result, expected)
		}
	})
}

func TestCalculateReminderTimesWithCreateTask(t *testing.T) {
	t.Parallel()

	validUserID, err := user.NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	validColor := MustColor("#FF6B6B")

	t.Run("works correctly with CreateTask for urgent type", func(t *testing.T) {
		task, err := CreateTask(
			nil,
			validUserID,
			"Urgent Task",
			TypeUrgent,
			"",
			nil,
			validColor,
		)
		if err != nil {
			t.Fatalf("CreateTask() unexpected error: %v", err)
		}

		info := CalculateReminderTimes(task)

		if info == nil {
			t.Fatal("CalculateReminderTimes returned nil")
		}

		// urgent: 2 intervals + targetAt = 3 reminder times
		if len(info.ReminderTimes) != 3 {
			t.Errorf("got %d reminder times, want 3", len(info.ReminderTimes))
		}

		// Verify the last reminder time equals targetAt
		lastReminderTime := info.ReminderTimes[len(info.ReminderTimes)-1]
		if !lastReminderTime.Equal(task.TargetAt()) {
			t.Errorf("last reminder time = %v, want %v (targetAt)", lastReminderTime, task.TargetAt())
		}

		// Verify all reminder times are before or equal to targetAt
		for i, reminderTime := range info.ReminderTimes {
			if reminderTime.After(task.TargetAt()) {
				t.Errorf("ReminderTimes[%d] = %v is after targetAt %v", i, reminderTime, task.TargetAt())
			}
		}

		// Verify reminder times are in ascending order
		for i := 1; i < len(info.ReminderTimes); i++ {
			if !info.ReminderTimes[i].After(info.ReminderTimes[i-1]) {
				t.Errorf("ReminderTimes[%d] = %v is not after ReminderTimes[%d] = %v",
					i, info.ReminderTimes[i], i-1, info.ReminderTimes[i-1])
			}
		}
	})

	t.Run("works correctly with CreateTask for scheduled type", func(t *testing.T) {
		scheduledAt := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Microsecond)

		task, err := CreateTask(
			nil,
			validUserID,
			"Scheduled Task",
			TypeScheduled,
			"",
			&scheduledAt,
			validColor,
		)
		if err != nil {
			t.Fatalf("CreateTask() unexpected error: %v", err)
		}

		info := CalculateReminderTimes(task)

		if info == nil {
			t.Fatal("CalculateReminderTimes returned nil")
		}

		// scheduled uses low intervals (3) + targetAt = 4 reminder times
		if len(info.ReminderTimes) != 4 {
			t.Errorf("got %d reminder times, want 4", len(info.ReminderTimes))
		}

		// Verify the last reminder time equals targetAt (which equals scheduledAt)
		lastReminderTime := info.ReminderTimes[len(info.ReminderTimes)-1]
		if !lastReminderTime.Equal(task.TargetAt()) {
			t.Errorf("last reminder time = %v, want %v (targetAt)", lastReminderTime, task.TargetAt())
		}
	})
}
