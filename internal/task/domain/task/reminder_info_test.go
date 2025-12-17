package task

import (
	"testing"
	"time"

	"github.com/KasumiMercury/primind-central-backend/internal/task/domain/user"
)

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

		info := CalculateReminderTimes(task, "test-user-id", nil)

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

		info := CalculateReminderTimes(task, "test-user-id", nil)

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

		info := CalculateReminderTimes(task, "test-user-id", nil)

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

		info := CalculateReminderTimes(task, "test-user-id", nil)

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
		info := CalculateReminderTimes(nil, "", nil)
		if info != nil {
			t.Error("expected nil for nil task")
		}
	})

	t.Run("filters out intervals exceeding targetAt for short scheduled task", func(t *testing.T) {
		taskID, err := NewID()
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		// Short scheduled task: targetAt = 30 minutes from createdAt
		// Low intervals (126, 232, 315 min) all exceed this
		scheduledAt := baseTime.Add(30 * time.Minute)

		task, err := NewTask(
			taskID,
			validUserID,
			"Short Scheduled Task",
			TypeScheduled,
			StatusActive,
			"",
			&scheduledAt,
			baseTime,
			scheduledAt,
			validColor,
		)
		if err != nil {
			t.Fatalf("NewTask() unexpected error: %v", err)
		}

		info := CalculateReminderTimes(task, "test-user-id", nil)

		if info == nil {
			t.Fatal("CalculateReminderTimes returned nil")
		}

		// All intervals exceed targetAt, so only targetAt should remain
		if len(info.ReminderTimes) != 1 {
			t.Errorf("got %d reminder times, want 1 (only targetAt)", len(info.ReminderTimes))
		}

		if !info.ReminderTimes[0].Equal(scheduledAt) {
			t.Errorf("ReminderTimes[0] = %v, want %v (targetAt)", info.ReminderTimes[0], scheduledAt)
		}
	})

	t.Run("deduplicates when interval exactly equals targetAt", func(t *testing.T) {
		taskID, err := NewID()
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		// targetAt = 126 minutes (matches first Low interval exactly)
		targetAt := baseTime.Add(126 * time.Minute)

		task, err := NewTask(
			taskID,
			validUserID,
			"Test Dedupe Task",
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

		info := CalculateReminderTimes(task, "test-user-id", nil)

		if info == nil {
			t.Fatal("CalculateReminderTimes returned nil")
		}

		// Only the 126min interval should remain (232, 315 filtered)
		// targetAt equals 126min so no duplicate should be added
		if len(info.ReminderTimes) != 1 {
			t.Errorf("got %d reminder times, want 1 (no duplicate)", len(info.ReminderTimes))
		}

		expected := baseTime.Add(126 * time.Minute)
		if !info.ReminderTimes[0].Equal(expected) {
			t.Errorf("ReminderTimes[0] = %v, want %v", info.ReminderTimes[0], expected)
		}
	})

	t.Run("partially filters intervals exceeding targetAt", func(t *testing.T) {
		taskID, err := NewID()
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		// targetAt = 200 minutes
		// Low intervals: 126min (kept), 232min (filtered), 315min (filtered)
		targetAt := baseTime.Add(200 * time.Minute)

		task, err := NewTask(
			taskID,
			validUserID,
			"Test Partial Filter Task",
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

		info := CalculateReminderTimes(task, "test-user-id", nil)

		if info == nil {
			t.Fatal("CalculateReminderTimes returned nil")
		}

		// Expected: 126min interval + targetAt (200min)
		if len(info.ReminderTimes) != 2 {
			t.Errorf("got %d reminder times, want 2", len(info.ReminderTimes))
		}

		expectedTimes := []time.Time{
			baseTime.Add(126 * time.Minute),
			baseTime.Add(200 * time.Minute),
		}

		for i, expected := range expectedTimes {
			if i >= len(info.ReminderTimes) {
				break
			}

			if !info.ReminderTimes[i].Equal(expected) {
				t.Errorf("ReminderTimes[%d] = %v, want %v", i, info.ReminderTimes[i], expected)
			}
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

		info := CalculateReminderTimes(task, "test-user-id", nil)

		if info.TaskID != task.ID() {
			t.Errorf("TaskID = %v, want %v", info.TaskID, task.ID())
		}

		if info.TaskType != task.TaskType() {
			t.Errorf("TaskType = %v, want %v", info.TaskType, task.TaskType())
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

		info := CalculateReminderTimes(task, "test-user-id", nil)

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

		info := CalculateReminderTimes(task, "test-user-id", nil)

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
