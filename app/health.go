package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// healthHandler is the handler for the health check endpoint
func healthHandler(c *gin.Context) {
	// Check the health and return a status code accordingly
	if isHealthy() {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	} else {
		c.JSON(http.StatusOK, gin.H{"status": "not healthy"})
	}
}

// helper functions
// isHealthy checks the health of the service
// ToDo: implement a proper health check
func isHealthy() bool {
	return true
}
