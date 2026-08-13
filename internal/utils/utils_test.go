package utils

import (
	"testing"
	"time"
)

func TestDaysUntil(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		t    time.Time
		want int
	}{
		{"later today", now.Add(6 * time.Hour), 0},
		{"one day out", now.Add(24 * time.Hour), 1},
		{"three days out", now.Add(3 * 24 * time.Hour), 3},
		{"just passed", now.Add(-12 * time.Hour), -1},
		{"three days past", now.Add(-3 * 24 * time.Hour), -3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DaysUntil(tt.t); got != tt.want {
				t.Fatalf("DaysUntil(%v) = %d, want %d", tt.t, got, tt.want)
			}
		})
	}
}
