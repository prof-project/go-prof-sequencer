//go:build noauth

// Package main implements the sequencer
package main

import "github.com/gin-gonic/gin"

// jwtLoginHandler is a no-op handler for disabling authentication
func jwtLoginHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"token": "test-token",
	})
}

// jwtAuthMiddleware is a no-op middleware for disabling authentication
func jwtAuthMiddleware(_ []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("username", "testuser")
		c.Next()
	}
}
