package main

import (
	"math"
	"testing"
)

// TestNewErrorSimulator verifies that the ErrorSimulator is correctly
// initialized with various target frequencies.
func TestNewErrorSimulator(t *testing.T) {
	tests := []struct {
		name      string
		frequency float64
		want      float64
	}{
		{"zero frequency", 0.0, 0.0},
		{"half frequency", 0.5, 0.5},
		{"full frequency", 1.0, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim := NewErrorSimulator(tt.frequency)
			if sim.targetFrequency != tt.want {
				t.Errorf("NewErrorSimulator(%v) got targetFrequency = %v, want %v",
					tt.frequency, sim.targetFrequency, tt.want)
			}
		})
	}
}

// TestGetCurrentErrorRate verifies that the error rate calculation
// is correct for various combinations of requests and errors.
func TestGetCurrentErrorRate(t *testing.T) {
	tests := []struct {
		name          string
		totalRequests uint64
		totalErrors   uint64
		expectedRate  float64
	}{
		{"no requests", 0, 0, 0.0},
		{"no errors", 100, 0, 0.0},
		{"all errors", 100, 100, 1.0},
		{"half errors", 100, 50, 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim := NewErrorSimulator(0.5)
			sim.totalRequests = tt.totalRequests
			sim.totalErrors = tt.totalErrors

			got := sim.GetCurrentErrorRate()
			if got != tt.expectedRate {
				t.Errorf("GetCurrentErrorRate() = %v, want %v", got, tt.expectedRate)
			}
		})
	}
}

// TestShouldError verifies that the error simulation converges to the
// target frequency over a large number of iterations. It tests various
// target frequencies and ensures the actual error rate stays within
// an acceptable tolerance range.
func TestShouldError(t *testing.T) {
	testCases := []struct {
		name           string
		targetFreq     float64
		iterations     int
		toleranceRange float64
	}{
		{"zero frequency", 0.0, 1000, 0.05},
		{"quarter frequency", 0.25, 1000, 0.05},
		{"half frequency", 0.5, 1000, 0.05},
		{"high frequency", 0.75, 1000, 0.05},
		{"full frequency", 1.0, 1000, 0.05},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sim := NewErrorSimulator(tc.targetFreq)
			errorCount := 0

			for i := 0; i < tc.iterations; i++ {
				if sim.ShouldError() {
					errorCount++
				}
			}

			actualFreq := float64(errorCount) / float64(tc.iterations)
			difference := math.Abs(actualFreq - tc.targetFreq)

			if difference > tc.toleranceRange {
				t.Errorf("ShouldError() frequency = %v, want %v (±%v), diff: %v",
					actualFreq, tc.targetFreq, tc.toleranceRange, difference)
			}
		})
	}
}

// TestShouldError_Adjustment verifies that the probability adjustment
// mechanism works correctly. It checks that the error rate increases
// when we're below target and decreases when we're above target.
func TestShouldError_Adjustment(t *testing.T) {
	sim := NewErrorSimulator(0.5)

	// Test adjustment when below target frequency
	sim.totalRequests = 100
	sim.totalErrors = 25 // 25% error rate, below 50% target

	errorCount := 0
	iterations := 100

	for i := 0; i < iterations; i++ {
		if sim.ShouldError() {
			errorCount++
		}
	}

	adjustedRate := float64(errorCount) / float64(iterations)
	if adjustedRate <= 0.5 {
		t.Errorf("Expected increased error rate when below target, got %v", adjustedRate)
	}

	// Test adjustment when above target frequency
	sim = NewErrorSimulator(0.5)
	sim.totalRequests = 100
	sim.totalErrors = 75 // 75% error rate, above 50% target

	errorCount = 0
	for i := 0; i < iterations; i++ {
		if sim.ShouldError() {
			errorCount++
		}
	}

	adjustedRate = float64(errorCount) / float64(iterations)
	if adjustedRate >= 0.5 {
		t.Errorf("Expected decreased error rate when above target, got %v", adjustedRate)
	}
}

// TestConcurrency verifies that the ErrorSimulator handles concurrent
// access correctly. It launches multiple goroutines that simultaneously
// call ShouldError and GetCurrentErrorRate, then verifies that the
// total request count is accurate and no race conditions occurred.
func TestConcurrency(t *testing.T) {
	sim := NewErrorSimulator(0.5)
	iterations := 1000
	goroutines := 10

	done := make(chan bool)

	// Launch concurrent goroutines
	for i := 0; i < goroutines; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				sim.ShouldError()
				sim.GetCurrentErrorRate()
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < goroutines; i++ {
		<-done
	}

	expectedRequests := uint64(goroutines * iterations)
	if sim.totalRequests != expectedRequests {
		t.Errorf("Expected %d total requests, got %d", expectedRequests, sim.totalRequests)
	}
}
