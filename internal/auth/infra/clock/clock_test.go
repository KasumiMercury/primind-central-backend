package clock

import (
	"testing"
	"time"
)

func TestRealClockSuccess(t *testing.T) {
	c := &RealClock{}
	now := c.Now()
	if time.Since(now) < 0 {
		t.Fatalf("expected now to be <= current time")
	}
}

func TestFixedClockSuccess(t *testing.T) {
	fixed := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewFixedClock(fixed)
	if got := c.Now(); !got.Equal(fixed) {
		t.Fatalf("expected fixed time %v, got %v", fixed, got)
	}
}
