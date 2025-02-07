package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

// handleRequest simulates latency, random failures, and returns the (possibly overridden)
// response. It also streams if the query parameter stream=true is present.
func handleRequest(w http.ResponseWriter, r *http.Request, path string, config *Config) {
	// Simulate latency.
	chosenLatency := chooseLatency(config)
	log.Printf("Path %s: Sleeping for %d ms", path, chosenLatency)
	time.Sleep(time.Duration(chosenLatency) * time.Millisecond)

	// Possibly simulate an error.
	if rand.Float64() < config.ErrorFrequency {
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

// chooseLatency selects low or high latency based on the configured frequency.
func chooseLatency(config *Config) int {
	if rand.Float64() < config.Latency.LowFrequency {
		return config.Latency.Low
	}
	return config.Latency.High
}

// simulateError writes an error response, streaming if requested.
func simulateError(w http.ResponseWriter, r *http.Request, config *Config) {
	log.Printf("Simulating error for request")
	if isStreaming(r) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: %s\n\n", marshalJSON(config.ErrorResponse.Body))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(config.ErrorResponse.Code)
		_ = json.NewEncoder(w).Encode(config.ErrorResponse.Body)
	}
}

// getResponseData returns an override response if present; otherwise, a default message.
func getResponseData(path string, config *Config) interface{} {
	if override, ok := config.Responses[path]; ok {
		return override
	}
	return map[string]string{"message": fmt.Sprintf("Default response for %s", path)}
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
		chunkCount = 1
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
		chosenLatency := chooseLatency(config)
		time.Sleep(time.Duration(chosenLatency) * time.Millisecond)
	}
	// Termination marker.
	fmt.Fprint(w, "data: [DONE]\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func normalResponse(w http.ResponseWriter, responseData interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(responseData)
}

// marshalJSON converts v to a JSON string (or returns "{}" on error).
func marshalJSON(v interface{}) string {
	bytes, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(bytes)
}
