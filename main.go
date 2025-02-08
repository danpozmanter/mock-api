package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

func main() {
	// Command-line flags.
	configFile := flag.String("config", "config.yaml", "Path to config file")
	port := flag.String("port", "8080", "Port to listen on")
	flag.Parse()

	// Load configuration.
	config, err := loadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Printf("Loaded config: %+v", config)

	// Load the API spec.
	spec, err := loadAPISpec(config.APISpec)
	if err != nil {
		log.Fatalf("Failed to load API spec: %v", err)
	}
	log.Printf("Loaded API spec with %d paths", len(spec.Paths))

	// Create router.
	router := mux.NewRouter().StrictSlash(true)

	// Store valid methods for each path
	pathMethods := make(map[string]map[string]bool)

	// Register each endpoint from the spec with the provided prefix.
	for path, methods := range spec.Paths {
		trimmedPrefix := strings.Trim(config.Prefix, "/")
		trimmedPath := strings.TrimLeft(path, "/")
		fullPath := "/" + trimmedPrefix + "/" + trimmedPath

		// Store valid methods for this path
		pathMethods[fullPath] = make(map[string]bool)

		for method := range methods {
			httpMethod := strings.ToUpper(method)
			pathMethods[fullPath][httpMethod] = true

			// Capture fullPath in closure.
			ep := fullPath
			router.HandleFunc(ep, func(w http.ResponseWriter, r *http.Request) {
				handleRequest(w, r, ep, config)
			}).Methods(httpMethod)
			log.Printf("Registered endpoint: %s %s", httpMethod, fullPath)
		}

		// Add a handler for unsupported methods on valid paths
		ep := fullPath
		router.HandleFunc(ep, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			if err := json.NewEncoder(w).Encode(ErrorResponse{
				Error: "Method not allowed",
			}); err != nil {
				log.Printf("Error encoding method not allowed response: %v", err)
			}

		})
	}

	// Catch-all handler for unknown paths
	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(ErrorResponse{
			Error: "Not found",
		}); err != nil {
			log.Printf("Error encoding not found response: %v", err)
		}

	})

	log.Printf("Loaded responses: %+v", config.Responses)

	addr := ":" + *port
	log.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
