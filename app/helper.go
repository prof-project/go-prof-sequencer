// Package main implements the sequencer
package main

import (
	"encoding/hex"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Decode hex string utility
func decodeHex(hexStr string) ([]byte, error) {
	if len(hexStr) > 1 && hexStr[:2] == "0x" {
		hexStr = hexStr[2:]
	}
	return hex.DecodeString(hexStr)
}

// getSecret reads a secret from a file
func getSecret(filePath string, defaultValue string) string {
	if data, err := os.ReadFile(filePath); err == nil {
		return string(data)
	}
	return defaultValue
}

// PrometheusMiddleware is a Gin middleware for Prometheus metrics
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()

		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		httpRequestsTotal.WithLabelValues(path, c.Request.Method, strconv.Itoa(c.Writer.Status())).Inc()
		httpRequestDuration.WithLabelValues(path, c.Request.Method).Observe(duration)
	}
}
