package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

// APISpec is a minimal structure to parse the “paths” from an API YAML.
type APISpec struct {
	Paths map[string]map[string]interface{} `yaml:"paths"`
}

// loadAPISpec loads (via HTTP GET or file read) and parses the API YAML.
func loadAPISpec(specURL string) (*APISpec, error) {
	var data []byte
	var err error
	if strings.HasPrefix(specURL, "http://") || strings.HasPrefix(specURL, "https://") {
		resp, err := http.Get(specURL)
		if err != nil {
			return nil, fmt.Errorf("error fetching API spec from URL: %v", err)
		}
		defer resp.Body.Close()
		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading API spec HTTP response: %v", err)
		}
	} else {
		data, err = os.ReadFile(specURL)
		if err != nil {
			return nil, fmt.Errorf("error reading API spec file: %v", err)
		}
	}
	var spec APISpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("error parsing API spec: %v", err)
	}
	return &spec, nil
}
