package main

import (
	"math/rand"
	"sync/atomic"
)

// ErrorSimulator maintains state for probabilistic error simulation.
// It dynamically adjusts error probability to maintain a target error frequency
// over time, making it more suitable for testing than simple random checking.
type ErrorSimulator struct {
	// targetFrequency is the desired proportion of errors (0.0 to 1.0)
	targetFrequency float64
	// totalRequests tracks the number of times ShouldError has been called
	totalRequests uint64
	// totalErrors tracks how many times we've returned true for an error
	totalErrors uint64
}

// NewErrorSimulator creates and initializes a new ErrorSimulator with the specified
// target error frequency. The frequency should be between 0.0 (never error)
// and 1.0 (always error).
//
// Example:
//
//	simulator := NewErrorSimulator(0.5) // 50% error rate
func NewErrorSimulator(frequency float64) *ErrorSimulator {
	return &ErrorSimulator{
		targetFrequency: frequency,
	}
}

// ShouldError determines if the current request should return an error.
// It maintains the target error frequency by dynamically adjusting the
// probability based on the actual error rate so far.
//
// The function is safe for concurrent use across multiple goroutines.
//
// Returns true if the current request should simulate an error.
func (e *ErrorSimulator) ShouldError() bool {
	requests := atomic.AddUint64(&e.totalRequests, 1)
	currentErrors := atomic.LoadUint64(&e.totalErrors)

	// Calculate current error rate
	currentRate := float64(currentErrors) / float64(requests)

	// Adjust probability to converge toward target frequency:
	// - If below target: increase error probability by 50%
	// - If above target: decrease error probability by 50%
	adjustedProb := e.targetFrequency
	if currentRate < e.targetFrequency {
		adjustedProb = e.targetFrequency * 1.5
	} else if currentRate > e.targetFrequency {
		adjustedProb = e.targetFrequency * 0.5
	}

	// Make the decision and update error count if needed
	shouldError := rand.Float64() < adjustedProb
	if shouldError {
		atomic.AddUint64(&e.totalErrors, 1)
	}

	return shouldError
}

// GetCurrentErrorRate returns the actual error rate observed so far.
// This can be used to verify that the error simulation is maintaining
// the desired frequency over time.
//
// Returns a float64 between 0.0 and 1.0. Returns 0.0 if no requests
// have been made yet.
func (e *ErrorSimulator) GetCurrentErrorRate() float64 {
	requests := atomic.LoadUint64(&e.totalRequests)
	if requests == 0 {
		return 0
	}
	return float64(atomic.LoadUint64(&e.totalErrors)) / float64(requests)
}
