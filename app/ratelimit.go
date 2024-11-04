package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"sync"
	"time"
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
		username, exists := c.Get("username")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		user := username.(string)
		now := time.Now()

		mu.Lock()
		defer mu.Unlock()

		// Clean up old timestamps
		timestamps := rateLimiter[user]
		var newTimestamps []time.Time
		for _, t := range timestamps {
			if now.Sub(t) <= windowSize {
				newTimestamps = append(newTimestamps, t)
			}
		}
		rateLimiter[user] = newTimestamps

		// Check if the user has exceeded the rate limit
		if len(rateLimiter[user]) >= rateLimit {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}

		// Add the current timestamp
		rateLimiter[user] = append(rateLimiter[user], now)
		c.Next()
	}
}
