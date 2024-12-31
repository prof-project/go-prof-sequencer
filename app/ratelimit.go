// Package main implements the sequencer
package main

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	rateLimiter       = make(map[string][]time.Time)
	rateLimit         = 3000 // Max 3000 requests per minute
	windowSize        = time.Minute
	mu                sync.Mutex
	globalRateLimiter = make([]time.Time, 0)
	globalRateLimit   = 10000 // Max 10000 requests per minute globally
)

// rateLimitMiddleware is the middleware for rate limiting using a sliding window approach
func rateLimitMiddleware() gin.HandlerFunc {
	go periodicCleanup() // Start the periodic cleanup job

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()

		mu.Lock()
		defer mu.Unlock()

		// Check if the client has exceeded the rate limit
		if len(rateLimiter[clientIP]) >= rateLimit {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}

		// Check if the global rate limit has been exceeded
		if len(globalRateLimiter) >= globalRateLimit {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Global rate limit exceeded"})
			c.Abort()
			return
		}

		// Add the current timestamp
		rateLimiter[clientIP] = append(rateLimiter[clientIP], now)
		globalRateLimiter = append(globalRateLimiter, now)
		c.Next()
	}
}

// periodicCleanup removes old timestamps periodically
func periodicCleanup() {
	for {
		time.Sleep(windowSize/2 + 1*time.Second)

		mu.Lock()
		now := time.Now()

		// Clean up old timestamps for all clients
		for clientIP, timestamps := range rateLimiter {
			var newTimestamps []time.Time
			for _, t := range timestamps {
				if now.Sub(t) <= windowSize {
					newTimestamps = append(newTimestamps, t)
				}
			}
			rateLimiter[clientIP] = newTimestamps
		}

		// Clean up old timestamps for the global rate limiter
		var newGlobalTimestamps []time.Time
		for _, t := range globalRateLimiter {
			if now.Sub(t) <= windowSize {
				newGlobalTimestamps = append(newGlobalTimestamps, t)
			}
		}
		globalRateLimiter = newGlobalTimestamps

		mu.Unlock()
	}
}
