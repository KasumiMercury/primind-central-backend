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
			name:          "TypeScheduled is not in map (uses dynamic mapping)",
			taskType:      TypeScheduled,
			expectedCount: 0,
			exists:        false,
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
		name                string
		taskType            Type
		expectedPercentages []float64
	}{
		{
			name:                "TypeShort percentages are 0.70 and 0.96",
			taskType:            TypeShort,
			expectedPercentages: []float64{0.70, 0.96},
		},
		{
			name:                "TypeNear percentages are 0.56 and 0.89",
			taskType:            TypeNear,
			expectedPercentages: []float64{0.56, 0.89},
		},
		{
			name:                "TypeRelaxed percentages are 0.35, 0.65, and 0.87",
			taskType:            TypeRelaxed,
			expectedPercentages: []float64{0.35, 0.65, 0.87},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intervals := DefaultReminderIntervals[tt.taskType]

			if len(intervals) != len(tt.expectedPercentages) {
				t.Fatalf("len(intervals) = %d, want %d", len(intervals), len(tt.expectedPercentages))
			}

			for i, expectedPct := range tt.expectedPercentages {
				expected := ReminderInterval(expectedPct)
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
		expectNil     bool
	}{
		{
			name:          "TypeShort returns 2 intervals",
			taskType:      TypeShort,
			expectedCount: 2,
			expectNil:     false,
		},
		{
			name:          "TypeNear returns 2 intervals",
			taskType:      TypeNear,
			expectedCount: 2,
			expectNil:     false,
		},
		{
			name:          "TypeRelaxed returns 3 intervals",
			taskType:      TypeRelaxed,
			expectedCount: 3,
			expectNil:     false,
		},
		{
			name:          "TypeScheduled returns nil (uses dynamic mapping)",
			taskType:      TypeScheduled,
			expectedCount: 0,
			expectNil:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetReminderIntervalsForType(tt.taskType)

			if tt.expectNil {
				if result != nil {
					t.Errorf("GetReminderIntervalsForType(%v) = %v, want nil", tt.taskType, result)
				}
			} else {
				if len(result) != tt.expectedCount {
					t.Errorf("GetReminderIntervalsForType(%v) returned %d intervals, want %d",
						tt.taskType, len(result), tt.expectedCount)
				}
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

func TestGetReminderIntervalsForDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		duration            time.Duration
		expectedPercentages []float64
		expectedCount       int
	}{
		{
			name:                "duration <= 30m returns short percentages",
			duration:            25 * time.Minute,
			expectedPercentages: []float64{0.70, 0.96},
			expectedCount:       2,
		},
		{
			name:                "duration exactly 30m returns short percentages",
			duration:            30 * time.Minute,
			expectedPercentages: []float64{0.70, 0.96},
			expectedCount:       2,
		},
		{
			name:                "duration > 30m and <= 3h returns near percentages",
			duration:            2 * time.Hour,
			expectedPercentages: []float64{0.56, 0.89},
			expectedCount:       2,
		},
		{
			name:                "duration exactly 3h returns near percentages",
			duration:            3 * time.Hour,
			expectedPercentages: []float64{0.56, 0.89},
			expectedCount:       2,
		},
		{
			name:                "duration > 3h returns relaxed percentages",
			duration:            12 * time.Hour,
			expectedPercentages: []float64{0.35, 0.65, 0.87},
			expectedCount:       3,
		},
		{
			name:                "duration 48h returns relaxed percentages",
			duration:            48 * time.Hour,
			expectedPercentages: []float64{0.35, 0.65, 0.87},
			expectedCount:       3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetReminderIntervalsForDuration(tt.duration)

			if len(result) != tt.expectedCount {
				t.Fatalf("GetReminderIntervalsForDuration(%v) returned %d intervals, want %d",
					tt.duration, len(result), tt.expectedCount)
			}

			for i, expectedPct := range tt.expectedPercentages {
				expected := ReminderInterval(expectedPct)
				if result[i] != expected {
					t.Errorf("intervals[%d] = %v, want %v", i, result[i], expected)
				}
			}
		})
	}
}

