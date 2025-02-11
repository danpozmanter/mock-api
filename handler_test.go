package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// createTestConfig initializes a test configuration with default values.
func createTestConfig() *Config {
	return &Config{
		APISpec: "spec.yaml",
		Latency: LatencyConfig{
			Low:  10,
			High: 20,
		},
		Responses: map[string]interface{}{
			"/v1/test": map[string]string{"message": "override"},
		},
		ErrorResponse: ErrorResponseConfig{
			Code:      500,
			Body:      map[string]string{"error": "simulated error"},
			Frequency: 0.0, // No error by default.
		},
		Prefix: "v1",
	}
}

// TestHandleRequest_Success verifies that a normal response is returned correctly.
func TestHandleRequest_Success(t *testing.T) {
	config := createTestConfig()
	config.Responses["/v1/test"] = `{"custom":"data","value":123}`
	errorSim := NewErrorSimulator(0.0)

	req := httptest.NewRequest("GET", "http://example.com/?stream=false", nil)
	w := httptest.NewRecorder()
	handleRequest(w, req, "/v1/test", config, errorSim)
	res := w.Result()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", res.StatusCode)
	}

	var responseData map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&responseData); err != nil {
		t.Fatalf("Error decoding JSON: %v", err)
	}

	if responseData["custom"] != "data" || responseData["value"].(float64) != 123 {
		t.Errorf("Unexpected response data: %v", responseData)
	}
}

// TestHandleRequest_StructOverride ensures structured overrides work.
func TestHandleRequest_StructOverride(t *testing.T) {
	config := createTestConfig()
	config.Responses["/v1/struct"] = map[string]interface{}{
		"status": "success",
		"code":   200,
	}
	errorSim := NewErrorSimulator(0.0)

	req := httptest.NewRequest("GET", "http://example.com/?stream=false", nil)
	w := httptest.NewRecorder()
	handleRequest(w, req, "/v1/struct", config, errorSim)
	res := w.Result()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", res.StatusCode)
	}

	var responseData map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&responseData); err != nil {
		t.Fatalf("Error decoding JSON: %v", err)
	}

	if responseData["status"] != "success" || responseData["code"].(float64) != 200 {
		t.Errorf("Unexpected struct response data: %v", responseData)
	}
}

// TestHandleRequest_Streaming validates correct streaming behavior.
func TestHandleRequest_Streaming(t *testing.T) {
	config := createTestConfig()
	errorSim := NewErrorSimulator(0.0)

	req := httptest.NewRequest("GET", "http://example.com/?stream=true", nil)
	w := httptest.NewRecorder()
	start := time.Now()
	handleRequest(w, req, "/v1/test", config, errorSim)
	elapsed := time.Since(start)

	if elapsed > 2*time.Second {
		t.Errorf("Streaming took too long: %v", elapsed)
	}

	res := w.Result()
	if ct := res.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Errorf("Expected Content-Type text/event-stream, got %s", ct)
	}

	body := w.Body.String()
	if !strings.Contains(body, "[DONE]") {
		t.Errorf("Missing streaming termination marker [DONE], got: %s", body)
	}
}

// TestHandleRequest_Error verifies simulated error responses.
func TestHandleRequest_Error(t *testing.T) {
	config := createTestConfig()
	errorSim := NewErrorSimulator(1.0) // 100% error rate.

	req := httptest.NewRequest("GET", "http://example.com/?stream=false", nil)
	w := httptest.NewRecorder()
	handleRequest(w, req, "/v1/test", config, errorSim)
	res := w.Result()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("Expected status 500, got %d", res.StatusCode)
	}

	var errResp map[string]string
	if err := json.NewDecoder(res.Body).Decode(&errResp); err != nil {
		t.Fatalf("Error decoding JSON: %v", err)
	}

	if errResp["error"] != "simulated error" {
		t.Errorf("Unexpected error response: %v", errResp)
	}
}

// TestHandleRequest_UnknownPath ensures a default response is returned for unknown paths.
func TestHandleRequest_UnknownPath(t *testing.T) {
	config := createTestConfig()
	errorSim := NewErrorSimulator(0.0)

	req := httptest.NewRequest("GET", "http://example.com/v1/unknown", nil)
	w := httptest.NewRecorder()
	handleRequest(w, req, "/v1/unknown", config, errorSim)
	res := w.Result()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", res.StatusCode)
	}

	var responseData map[string]string
	if err := json.NewDecoder(res.Body).Decode(&responseData); err != nil {
		t.Fatalf("Error decoding JSON: %v", err)
	}

	if responseData["message"] == "" {
		t.Errorf("Expected default response message, got empty")
	}
}

// TestSendJSONError ensures error responses are properly formatted.
func TestSendJSONError(t *testing.T) {
	w := httptest.NewRecorder()
	sendJSONError(w, http.StatusInternalServerError, "test error")

	res := w.Result()
	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("Expected status 500, got %d", res.StatusCode)
	}

	var responseData map[string]string
	if err := json.NewDecoder(res.Body).Decode(&responseData); err != nil {
		t.Fatalf("Error decoding JSON: %v", err)
	}

	if responseData["error"] != "test error" {
		t.Errorf("Unexpected error message: %v", responseData["error"])
	}
}

// TestGetLatency ensures latency values fall within the expected range.
func TestGetLatency(t *testing.T) {
	config := createTestConfig()
	for i := 0; i < 100; i++ {
		latency := getLatency(config)
		if latency < config.Latency.Low || latency > config.Latency.High {
			t.Errorf("Latency %f is out of range [%f, %f]", latency, config.Latency.Low, config.Latency.High)
		}
	}
}

// TestConvertToJSONCompatible checks conversion of complex structures.
func TestConvertToJSONCompatible(t *testing.T) {
	input := map[interface{}]interface{}{
		"nested": map[interface{}]interface{}{
			"key": "value",
		},
		"empty": map[interface{}]interface{}{},
	}

	expected := map[string]interface{}{
		"nested": map[string]interface{}{
			"key": "value",
		},
		"empty": map[string]interface{}{},
	}

	result := convertToJSONCompatible(input)
	if !deepEqual(result, expected) {
		t.Errorf("Expected %+v, got %+v", expected, result)
	}
}

// deepEqual checks deep equality between two objects.
func deepEqual(a, b interface{}) bool {
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}
