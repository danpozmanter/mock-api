package main

import (
	"os"
	"strings"
	"testing"
)

const validConfig = `
api_spec: "spec.yaml"
latency:
  low: 100
  high: 1000
responses:
  /v1/test:
    response: "{\"message\":\"override\"}"
error_response:
  code: 500
  body:
    error: "simulated error"
  frequency: 0.05
prefix: "v1"
`

func TestLoadConfigValid(t *testing.T) {
	filename := "test_config.yaml"
	if err := os.WriteFile(filename, []byte(validConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	defer os.Remove(filename)

	config, err := loadConfig(filename)
	if err != nil {
		t.Fatalf("Expected config to load, got error: %v", err)
	}
	if config.APISpec != "spec.yaml" {
		t.Errorf("Expected api_spec to be spec.yaml, got: %s", config.APISpec)
	}
	if config.Latency.Low != 100 {
		t.Errorf("Expected latency.low to be 100, got: %f", config.Latency.Low)
	}
	if !strings.Contains(config.Prefix, "v1") {
		t.Errorf("Expected prefix to contain v1, got: %s", config.Prefix)
	}
}

func TestLoadConfigMissingValues(t *testing.T) {
	invalidConfig := `
api_spec: ""
latency:
  low: 0
  high: 0
  low_frequency: 0
error_response:
  code: 0
  body: null
error_frequency: 0
prefix: ""
`
	filename := "test_invalid_config.yaml"
	if err := os.WriteFile(filename, []byte(invalidConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	defer os.Remove(filename)

	_, err := loadConfig(filename)
	if err == nil {
		t.Fatal("Expected error for missing config values, got nil")
	}
	expectedFields := []string{"api_spec", "latency.low", "latency.high", "latency.low_frequency", "error_response.frequency", "error_response.code", "error_response.body", "prefix"}
	for _, field := range expectedFields {
		if !strings.Contains(err.Error(), field) {
			t.Errorf("Expected error message to contain %s", field)
		}
	}
}

func TestLoadConfigMissingSections(t *testing.T) {
	invalidConfig := `
api_spec: ""
error_response:
  code: 0
  body: null
prefix: ""
`
	filename := "test_invalid_config.yaml"
	if err := os.WriteFile(filename, []byte(invalidConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	defer os.Remove(filename)

	_, err := loadConfig(filename)
	if err == nil {
		t.Fatal("Expected error for missing config values, got nil")
	}
	expectedFields := []string{"api_spec", "latency.low", "latency.high", "latency.low_frequency", "error_response.frequency", "error_response.code", "error_response.body", "prefix"}
	for _, field := range expectedFields {
		if !strings.Contains(err.Error(), field) {
			t.Errorf("Expected error message to contain %s", field)
		}
	}
}
