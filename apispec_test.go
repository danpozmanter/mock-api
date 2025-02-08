package main

import (
	"os"
	"strings"
	"testing"
)

const validAPISpec = `
paths:
  /test:
    get: {}
`

func TestLoadAPISpecFile(t *testing.T) {
	filename := "test_spec.yaml"
	if err := os.WriteFile(filename, []byte(validAPISpec), 0644); err != nil {
		t.Fatalf("Failed to write test API spec: %v", err)
	}
	defer os.Remove(filename)

	spec, err := loadAPISpec(filename)
	if err != nil {
		t.Fatalf("Expected API spec to load, got error: %v", err)
	}
	if len(spec.Paths) != 1 {
		t.Errorf("Expected 1 path, got: %d", len(spec.Paths))
	}
}

func TestLoadAPISpecInvalid(t *testing.T) {
	invalidSpec := `invalid: yaml: -`
	filename := "test_invalid_spec.yaml"
	if err := os.WriteFile(filename, []byte(invalidSpec), 0644); err != nil {
		t.Fatalf("Failed to write test API spec: %v", err)
	}
	defer os.Remove(filename)

	_, err := loadAPISpec(filename)
	if err == nil {
		t.Fatal("Expected error for invalid API spec, got nil")
	}
	if !strings.Contains(err.Error(), "error parsing API spec") {
		t.Errorf("Expected error message to mention parsing, got: %s", err.Error())
	}
}

func TestLoadAPISpecFileNotFound(t *testing.T) {
	_, err := loadAPISpec("non_existent_file.yaml")
	if err == nil || !strings.Contains(err.Error(), "error reading API spec file") {
		t.Fatalf("Expected file not found error, got: %v", err)
	}
}

func TestLoadAPISpecInvalidYAML(t *testing.T) {
	filename := "invalid_spec.yaml"
	invalidYAML := `invalid: yaml: -`

	if err := os.WriteFile(filename, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write invalid test API spec: %v", err)
	}
	defer os.Remove(filename)

	_, err := loadAPISpec(filename)
	if err == nil || !strings.Contains(err.Error(), "error parsing API spec") {
		t.Fatalf("Expected YAML parsing error, got: %v", err)
	}
}

func TestLoadAPISpecHTTPFailure(t *testing.T) {
	_, err := loadAPISpec("http://nonexistent-url.com/spec.yaml")
	if err == nil || !strings.Contains(err.Error(), "error fetching API spec from URL") {
		t.Fatalf("Expected HTTP fetch error, got: %v", err)
	}
}
