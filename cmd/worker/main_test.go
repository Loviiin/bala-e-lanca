package main

import (
	"testing"

	"github.com/Loviiin/okok-scale-logger/internal/ble"
)

func TestStabilityTrackerDoesNotRepeatWithinMinimumInterval(t *testing.T) {
	tracker := newStabilityTracker(3)
	reading := ble.Reading{WeightKg: 99.4}

	for range 2 {
		stable, isNew := tracker.observe(reading)
		if stable || isNew {
			t.Fatalf("leitura prematura marcada como nova: stable=%v new=%v", stable, isNew)
		}
	}
	stable, isNew := tracker.observe(reading)
	if !stable || !isNew {
		t.Fatalf("leitura estável não foi gravada: stable=%v new=%v", stable, isNew)
	}

	for range 3 {
		stable, isNew = tracker.observe(ble.Reading{WeightKg: 10.7})
		if isNew {
			t.Fatalf("segunda leitura passou durante o cooldown: stable=%v new=%v", stable, isNew)
		}
	}
}
