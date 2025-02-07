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
