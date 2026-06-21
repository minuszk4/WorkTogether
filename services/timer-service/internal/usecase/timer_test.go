package usecase

import (
	"testing"
)

func TestTimerStateInitialization(t *testing.T) {
	state := &PomodoroState{
		Status:           "focus",
		DurationSeconds:  1500,
		BreakSeconds:     300,
		RemainingSeconds: 1500,
		CurrentCycle:     1,
		TotalCycles:      4,
	}

	if state.Status != "focus" {
		t.Errorf("Expected status to be 'focus', got '%s'", state.Status)
	}
	if state.DurationSeconds != 1500 {
		t.Errorf("Expected duration to be 1500, got %d", state.DurationSeconds)
	}
	if state.RemainingSeconds != 1500 {
		t.Errorf("Expected remaining seconds to be 1500, got %d", state.RemainingSeconds)
	}
	if state.CurrentCycle != 1 {
		t.Errorf("Expected current cycle to be 1, got %d", state.CurrentCycle)
	}
	if state.TotalCycles != 4 {
		t.Errorf("Expected total cycles to be 4, got %d", state.TotalCycles)
	}
}
