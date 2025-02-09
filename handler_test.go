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
			Low:  10,
			High: 20,
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

// Test HandleRequest - Normal Response
func TestHandleRequestNormal(t *testing.T) {
	config := createTestConfig()
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

// Test HandleRequest - Struct Override
func TestHandleRequestWithStructOverride(t *testing.T) {
	config := createTestConfig()
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

// Test HandleRequest - Streaming
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

// Test HandleRequest - Error Response
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

// Test HandleRequest - Unknown Path
func TestHandleRequestUnknownPath(t *testing.T) {
	config := createTestConfig()

	req := httptest.NewRequest("GET", "http://example.com/v1/unknown", nil)
	w := httptest.NewRecorder()

	handleRequest(w, req, "/v1/unknown", config)
	res := w.Result()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", res.StatusCode)
	}

	var responseData map[string]string
	if err := json.NewDecoder(res.Body).Decode(&responseData); err != nil {
		t.Fatalf("Error decoding JSON: %v", err)
	}

	if responseData["message"] == "" {
		t.Errorf("Expected default response message, got empty")
	}
}

// Test Send JSON Error
func TestSendJSONError(t *testing.T) {
	w := httptest.NewRecorder()
	sendJSONError(w, http.StatusInternalServerError, "test error")

	res := w.Result()
	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", res.StatusCode)
	}

	var responseData map[string]string
	if err := json.NewDecoder(res.Body).Decode(&responseData); err != nil {
		t.Fatalf("Error decoding JSON: %v", err)
	}

	if responseData["error"] != "test error" {
		t.Errorf("Expected 'test error', got: %v", responseData["error"])
	}
}

// Test Simulate Error - Streaming
func TestSimulateErrorStreaming(t *testing.T) {
	config := createTestConfig()
	config.ErrorResponse.Frequency = 1.0 // Always trigger error

	req := httptest.NewRequest("GET", "http://example.com/v1/test?stream=true", nil)
	w := httptest.NewRecorder()
	simulateError(w, req, config)

	res := w.Result()
	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500 for error while streaming, got %d", res.StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, `"error":"simulated error"`) {
		t.Errorf("Expected simulated error in stream, got: %s", body)
	}
}

// Test Stream Response Encoding Failure
func TestStreamResponseEncodingFailure(t *testing.T) {
	w := httptest.NewRecorder()
	config := createTestConfig()

	// Force JSON marshalling failure by passing an invalid type
	data := make(chan int)

	streamResponse(w, data, config)

	res := w.Result()
	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected 500 Internal Server Error, got %d", res.StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Internal server error") {
		t.Errorf("Expected internal server error message, got: %s", body)
	}
}

// Test Get Response Data - Invalid JSON
func TestGetResponseDataWithInvalidJSON(t *testing.T) {
	config := createTestConfig()
	config.Responses["/v1/test"] = `{"message":}` // Invalid JSON

	responseData := getResponseData("/v1/test", config)
	if _, ok := responseData.(map[string]string); !ok {
		t.Errorf("Expected a map response, got: %T", responseData)
	}
}

// Test Convert To JSON Compatible
func TestConvertToJSONCompatible(t *testing.T) {
	input := map[interface{}]interface{}{
		"key1": "value1",
		"key2": map[interface{}]interface{}{
			"nestedKey": "nestedValue",
		},
		"key3": []interface{}{1, 2, 3},
	}

	result := convertToJSONCompatible(input)
	output, ok := result.(map[string]interface{})
	if !ok || output["key1"] != "value1" {
		t.Errorf("Unexpected conversion result: %v", result)
	}
}

// Test Normal Response Encoding Failure
func TestNormalResponseEncodingFailure(t *testing.T) {
	w := httptest.NewRecorder()

	// Invalid data type (channel) to force encoding failure
	data := make(chan int)

	normalResponse(w, data)

	res := w.Result()
	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected 500 Internal Server Error, got %d", res.StatusCode)
	}
}

// Test Marshal JSON Invalid Input
func TestMarshalJSONInvalidInput(t *testing.T) {
	// Force error by passing an unmarshalable type
	data := make(chan int)
	result := marshalJSON(data)

	if result != "{}" {
		t.Errorf("Expected '{}', got: %s", result)
	}
}
