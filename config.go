package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

// Config holds our configuration.
type Config struct {
	// Which API spec (YAML) to load.
	APISpec string `yaml:"api_spec"`
	// Latency configuration.
	Latency LatencyConfig `yaml:"latency"`
	// Override responses for specific endpoints.
	Responses map[string]interface{} `yaml:"responses"`
	// ErrorResponse now contains the error code, body, and frequency.
	ErrorResponse ErrorResponseConfig `yaml:"error_response"`
	// Prefix to insert before each endpoint URL. Defaults to "v1" if not provided.
	Prefix string `yaml:"prefix"`
}

// LatencyConfig specifies two latency values (in milliseconds)
// and the frequency of using the low latency.
type LatencyConfig struct {
	Low          int     `yaml:"low"`
	High         int     `yaml:"high"`
	LowFrequency float64 `yaml:"low_frequency"`
}

// ErrorResponseConfig now includes Frequency.
type ErrorResponseConfig struct {
	Code      int         `yaml:"code"`
	Body      interface{} `yaml:"body"`
	Frequency float64     `yaml:"frequency"`
}

// loadConfig reads and parses the YAML config file and returns an error if any required field is missing.
func loadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}

	// Check for missing required fields.
	missing := checkMissingConfig(&config)
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required configuration values: %s", strings.Join(missing, ", "))
	}

	// For optional fields, initialize defaults if needed.
	if config.Responses == nil {
		config.Responses = make(map[string]interface{})
	}

	return &config, nil
}

func checkMissingConfig(config *Config) []string {
	var missing []string
	if strings.TrimSpace(config.APISpec) == "" {
		missing = append(missing, "api_spec")
	}
	if config.Latency.Low == 0 {
		missing = append(missing, "latency.low")
	}
	if config.Latency.High == 0 {
		missing = append(missing, "latency.high")
	}
	if config.Latency.LowFrequency == 0 {
		missing = append(missing, "latency.low_frequency")
	}
	if config.ErrorResponse.Frequency == 0 {
		missing = append(missing, "error_response.frequency")
	}
	if config.ErrorResponse.Code == 0 {
		missing = append(missing, "error_response.code")
	}
	if config.ErrorResponse.Body == nil {
		missing = append(missing, "error_response.body")
	}
	if strings.TrimSpace(config.Prefix) == "" {
		missing = append(missing, "prefix")
	}
	return missing
}
