package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func createTestConfig() *Config {
	return &Config{
		APISpec: "spec.yaml",
		Latency: LatencyConfig{
			Low:          10,
			High:         20,
			LowFrequency: 1.0, // always low latency for testing
		},
		Responses: map[string]interface{}{
			"/v1/test": map[string]string{"message": "override"},
		},
		ErrorResponse: ErrorResponseConfig{
			Code:      500,
			Body:      map[string]string{"error": "simulated error"},
			Frequency: 0.0, // no error by default
		},
		Prefix: "v1",
	}
}

func TestHandleRequestNormal(t *testing.T) {
	config := createTestConfig()
	// Use raw JSON string as override
	config.Responses["/v1/test"] = `{"custom":"data","value":123}`

	req := httptest.NewRequest("GET", "http://example.com/?stream=false", nil)
	w := httptest.NewRecorder()
	handleRequest(w, req, "/v1/test", config)
	res := w.Result()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", res.StatusCode)
	}

	var responseData map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&responseData); err != nil {
		t.Fatalf("Error decoding JSON: %v", err)
	}

	if responseData["custom"] != "data" || responseData["value"].(float64) != 123 {
		t.Errorf("Expected custom override data, got %v", responseData)
	}
}

func TestHandleRequestWithStructOverride(t *testing.T) {
	config := createTestConfig()
	// Test with a map/struct override
	config.Responses["/v1/struct"] = map[string]interface{}{
		"status": "success",
		"code":   200,
	}

	req := httptest.NewRequest("GET", "http://example.com/?stream=false", nil)
	w := httptest.NewRecorder()
	handleRequest(w, req, "/v1/struct", config)
	res := w.Result()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", res.StatusCode)
	}

	var responseData map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&responseData); err != nil {
		t.Fatalf("Error decoding JSON: %v", err)
	}

	if responseData["status"] != "success" || responseData["code"].(float64) != 200 {
		t.Errorf("Expected struct override data, got %v", responseData)
	}
}

func TestHandleRequestStreaming(t *testing.T) {
	config := createTestConfig()
	req := httptest.NewRequest("GET", "http://example.com/?stream=true", nil)
	w := httptest.NewRecorder()
	start := time.Now()
	handleRequest(w, req, "/v1/test", config)
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
		t.Errorf("Expected streaming termination marker [DONE], got %s", body)
	}
}

func TestHandleRequestError(t *testing.T) {
	config := createTestConfig()
	config.ErrorResponse.Frequency = 1.0 // force error
	req := httptest.NewRequest("GET", "http://example.com/?stream=false", nil)
	w := httptest.NewRecorder()
	handleRequest(w, req, "/v1/test", config)
	res := w.Result()
	if res.StatusCode != 500 {
		t.Errorf("Expected status 500, got %d", res.StatusCode)
	}
	var errResp map[string]string
	if err := json.NewDecoder(res.Body).Decode(&errResp); err != nil {
		t.Errorf("Error decoding JSON: %v", err)
	}
	if errResp["error"] != "simulated error" {
		t.Errorf("Expected simulated error, got %s", errResp["error"])
	}
}
