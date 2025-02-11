// Package main implements an HTTP server that handles API requests based on a configuration file
package main

import (
	"flag"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// setupFlags initializes and parses command-line flags for server configuration.
// It returns the paths to the config file and the port number to listen on.
func setupFlags() (configFile string, port string) {
	configFilePtr := flag.String("config", "config.yaml", "Path to config file")
	portPtr := flag.String("port", "8080", "Port to listen on")
	flag.Parse()
	return *configFilePtr, *portPtr
}

// initializeServer loads and validates the server configuration and API specification.
// It returns the parsed config and API spec along with any error encountered.
func initializeServer(configFile string) (*Config, *APISpec, error) {
	config, err := loadConfig(configFile)
	if err != nil {
		return nil, nil, err
	}
	log.Printf("Loaded config: %+v", config)

	spec, err := loadAPISpec(config.APISpec)
	if err != nil {
		return nil, nil, err
	}
	log.Printf("Loaded API spec with %d paths", len(spec.Paths))
	return config, spec, nil
}

// buildFullPath constructs the complete URL path by combining the prefix and path.
// It ensures proper formatting by trimming extra slashes.
func buildFullPath(prefix, path string) string {
	trimmedPrefix := strings.Trim(prefix, "/")
	trimmedPath := strings.TrimLeft(path, "/")
	return "/" + trimmedPrefix + "/" + trimmedPath
}

// registerMethodHandlers sets up route handlers for all HTTP methods defined in the API spec.
// It returns a map of valid HTTP methods for the given path.
func registerMethodHandlers(router *mux.Router, fullPath string, methods map[string]interface{}, config *Config) map[string]bool {
	validMethods := make(map[string]bool)
	simulator := NewErrorSimulator(config.ErrorResponse.Frequency)
	for method := range methods {
		httpMethod := strings.ToUpper(method)
		validMethods[httpMethod] = true
		router.HandleFunc(fullPath, func(w http.ResponseWriter, r *http.Request) {
			handleRequest(w, r, fullPath, config, simulator)
		}).Methods(httpMethod)
		log.Printf("Registered endpoint: %s %s", httpMethod, fullPath)
	}
	return validMethods
}

// registerMethodNotAllowedHandler sets up a handler for requests using unsupported HTTP methods.
func registerMethodNotAllowedHandler(router *mux.Router, fullPath string) {
	router.HandleFunc(fullPath, func(w http.ResponseWriter, r *http.Request) {
		sendJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
	})
}

// registerNotFoundHandler sets up a handler for requests to undefined paths.
func registerNotFoundHandler(router *mux.Router) {
	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sendJSONError(w, http.StatusNotFound, "Not found")
	})
}

// setupRouter configures the HTTP router with all endpoints from the API spec.
// It returns the configured router ready for use.
func setupRouter(config *Config, spec *APISpec) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	pathMethods := make(map[string]map[string]bool)

	for path, methods := range spec.Paths {
		fullPath := buildFullPath(config.Prefix, path)
		pathMethods[fullPath] = registerMethodHandlers(router, fullPath, methods, config)
		registerMethodNotAllowedHandler(router, fullPath)
	}

	registerNotFoundHandler(router)
	return router
}

// main initializes and starts the HTTP server with the configured router.
// It handles command-line flags, loads configuration, and sets up all routes.
func main() {
	configFile, port := setupFlags()

	config, spec, err := initializeServer(configFile)
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}

	router := setupRouter(config, spec)
	log.Printf("Loaded responses: %+v", config.Responses)

	addr := ":" + port
	log.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
