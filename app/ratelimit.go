package main

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	rateLimiter = make(map[string][]time.Time)
	rateLimit   = 300 // Max 300 requests per minute
	windowSize  = time.Minute
	mu          sync.Mutex
)

// rateLimitMiddleware is the middleware for rate limiting using a sliding window approach
func rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()

		mu.Lock()
		defer mu.Unlock()

		// Clean up old timestamps
		timestamps := rateLimiter[clientIP]
		var newTimestamps []time.Time
		for _, t := range timestamps {
			if now.Sub(t) <= windowSize {
				newTimestamps = append(newTimestamps, t)
			}
		}
		rateLimiter[clientIP] = newTimestamps

		// Check if the client has exceeded the rate limit
		if len(rateLimiter[clientIP]) >= rateLimit {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}

		// Add the current timestamp
		rateLimiter[clientIP] = append(rateLimiter[clientIP], now)
		c.Next()
	}
}
