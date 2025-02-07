package main

import (
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

	// Register each endpoint from the spec with the provided prefix.
	for path, methods := range spec.Paths {
		for method := range methods {
			trimmedPrefix := strings.Trim(config.Prefix, "/")
			trimmedPath := strings.TrimLeft(path, "/")
			fullPath := "/" + trimmedPrefix + "/" + trimmedPath
			httpMethod := strings.ToUpper(method)
			// Capture fullPath in closure.
			ep := fullPath
			router.HandleFunc(ep, func(w http.ResponseWriter, r *http.Request) {
				handleRequest(w, r, ep, config)
			}).Methods(httpMethod)
			log.Printf("Registered endpoint: %s %s", httpMethod, fullPath)
		}
	}

	// Catch-all handler.
	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Found", http.StatusNotFound)
	})

	addr := ":" + *port
	log.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
