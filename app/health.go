package main

import (
	"fmt"
	"net/http"
)

// healthHandler is the handler for the health check endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	// Check the health and return a status code accordingly
	if isHealthy() {
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprint(w, "Service is healthy")
		if err != nil {
			return
		}
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := fmt.Fprint(w, "Service is not healthy")
		if err != nil {
			return
		}
	}
}

// helper functions
// isHealthy checks the health of the service
// ToDo: implement a proper health check
func isHealthy() bool {
	return true
}
