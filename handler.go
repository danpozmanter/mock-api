package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// handleRequest simulates latency, random failures, and returns the (possibly overridden)
// response. It also streams if the query parameter stream=true is present.
func handleRequest(w http.ResponseWriter, r *http.Request, path string, config *Config, simulator *ErrorSimulator) {
	// Simulate latency.
	chosenLatency := getLatency(config)
	log.Printf("Path %s: Sleeping for %f ms", path, chosenLatency)
	time.Sleep(time.Duration(chosenLatency) * time.Millisecond)

	// Possibly simulate an error.
	if simulator.ShouldError() {
		simulateError(w, r, config)
		return
	}

	responseData := getResponseData(path, config)
	if isStreaming(r) {
		streamResponse(w, responseData, config)
	} else {
		normalResponse(w, responseData)
	}
}

// getLatency selects low or high latency based on the configured frequency.
func getLatency(config *Config) float64 {
	return config.Latency.Low + rand.Float64()*(config.Latency.High-config.Latency.Low)
}

func sendJSONError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(ErrorResponse{Error: message}); err != nil {
		log.Printf("Error encoding error response: %v", err)
		// If JSON encoding fails, send a minimal JSON error
		if _, err := w.Write([]byte(`{"error":"Internal server error"}`)); err != nil {
			log.Printf("Error writing response: %v", err)
		}

	}
}

// simulateError writes an error response, streaming if requested.
func simulateError(w http.ResponseWriter, r *http.Request, config *Config) {
	log.Printf("Simulating error for request")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(config.ErrorResponse.Code)

	// Convert the error body to a JSON-compatible format
	errorBody := convertToJSONCompatible(config.ErrorResponse.Body)

	jsonBytes, err := json.Marshal(errorBody)
	if err != nil {
		log.Printf("Error encoding error response: %v", err)
		w.Write([]byte(`{"error":"Internal server error"}`))
		return
	}

	_, writeErr := w.Write(jsonBytes)
	if writeErr != nil {
		log.Printf("Error writing error response: %v", writeErr)
	}
}

// getResponseData returns an override response if present; otherwise, a default message.
func getResponseData(path string, config *Config) interface{} {
	// Normalize path by trimming trailing slashes
	normalizedPath := strings.TrimRight(path, "/")

	if override, ok := config.Responses[normalizedPath]; ok {
		switch v := override.(type) {
		case string:
			// If it's a string, try to decode it as JSON into a map
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(v), &result); err != nil {
				log.Printf("Failed to parse JSON string: %v", err)
				return map[string]string{"error": "Invalid JSON override"}
			}
			return result

		default:
			// For YAML structures, convert them properly
			converted := convertToJSONCompatible(override)
			return converted
		}
	}

	return map[string]string{"message": fmt.Sprintf("Response for %s", normalizedPath)}
}

// Simplified map conversion
func convertToJSONCompatible(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[fmt.Sprintf("%v", k)] = convertToJSONCompatible(v)
		}
		return m2
	case []interface{}:
		for i, v := range x {
			x[i] = convertToJSONCompatible(v)
		}
		return x
	default:
		return x
	}
}

func normalResponse(w http.ResponseWriter, responseData interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(responseData); err != nil {
		log.Printf("Error encoding response: %v", err)
		sendJSONError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
}

func isStreaming(r *http.Request) bool {
	return r.URL.Query().Get("stream") == "true"
}

func streamResponse(w http.ResponseWriter, responseData interface{}, config *Config) {
	w.Header().Set("Content-Type", "text/event-stream")
	jsonBytes, err := json.Marshal(responseData)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	// Divide the JSON into approximately 3 chunks.
	chunkCount := 3
	chunkSize := len(jsonBytes) / chunkCount
	if chunkSize == 0 {
		chunkSize = len(jsonBytes)
	}
	for i := 0; i < len(jsonBytes); i += chunkSize {
		end := i + chunkSize
		if end > len(jsonBytes) {
			end = len(jsonBytes)
		}
		chunk := jsonBytes[i:end]
		fmt.Fprintf(w, "data: %s\n\n", chunk)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		// Sleep between chunks.
		chosenLatency := getLatency(config)
		time.Sleep(time.Duration(chosenLatency) * time.Millisecond)
	}
	// Termination marker.
	fmt.Fprint(w, "data: [DONE]\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// marshalJSON converts v to a JSON string (or returns "{}" on error).
func marshalJSON(v interface{}) string {
	bytes, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(bytes)
}
