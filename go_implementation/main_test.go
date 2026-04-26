package main

import (
	"net/http"
	"testing"
	"time"
)

func TestMainStartup(t *testing.T) {
	go func() {
		main()
	}()

	// Wait for server to start
	time.Sleep(1 * time.Second)

	// Test server response
	res, err := http.Get("http://localhost:8080/v1/unknown")
	if err != nil {
		t.Fatalf("Failed to reach server: %v", err)
	}
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404 for unknown path, got %d", res.StatusCode)
	}
}
