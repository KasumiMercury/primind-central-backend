package task

import (
	"testing"
	"time"
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
			name:          "TypeShort has 2 intervals",
			taskType:      TypeShort,
			expectedCount: 2,
			exists:        true,
		},
		{
			name:          "TypeNear has 2 intervals",
			taskType:      TypeNear,
			expectedCount: 2,
			exists:        true,
		},
		{
			name:          "TypeRelaxed has 3 intervals",
			taskType:      TypeRelaxed,
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
			name:            "TypeShort intervals are 3 and 5 minutes",
			taskType:        TypeShort,
			expectedMinutes: []int{3, 5},
		},
		{
			name:            "TypeNear intervals are 33 and 53 minutes",
			taskType:        TypeNear,
			expectedMinutes: []int{33, 53},
		},
		{
			name:            "TypeRelaxed intervals are 126, 232, and 315 minutes",
			taskType:        TypeRelaxed,
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
			name:          "TypeShort returns 2 intervals",
			taskType:      TypeShort,
			expectedCount: 2,
		},
		{
			name:          "TypeNear returns 2 intervals",
			taskType:      TypeNear,
			expectedCount: 2,
		},
		{
			name:          "TypeRelaxed returns 3 intervals",
			taskType:      TypeRelaxed,
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

func TestReminderIntervalTypeConversion(t *testing.T) {
	t.Parallel()

	t.Run("ReminderInterval can be converted to time.Duration", func(t *testing.T) {
		interval := DefaultReminderIntervalsShort[0] // 3 minutes
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

