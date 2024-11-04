//go:build noauth

package main

import "github.com/gin-gonic/gin"

// jwtAuthMiddleware is a no-op middleware for disabling authentication
func jwtAuthMiddleware(requiredRoles []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("username", "testuser")
		c.Next()
	}
}
