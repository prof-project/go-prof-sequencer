package main

import (
	"flag"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	// Add command-line flag for gRPC URL
	grpcURL := flag.String("grpc-url", "127.0.0.1:50051", "URL for gRPC connection to bundle merger")
	flag.Parse()

	// Log the gRPC URL being used
	log.Printf("Using gRPC URL: %s", *grpcURL)

	txPool := &TxBundlePool{
		bundles:    []*TxPoolBundle{},
		bundleMap:  make(map[string]*TxPoolBundle),
		customSort: sortByBlockNumber,
	}

	// Set the Gin to debug mode
	// ToDo: change to release mode in production
	gin.SetMode(gin.DebugMode)

	// Start the cleanup job for the pool
	txPool.startCleanupJob(5 * time.Second)

	// Start the periodic bundle sender
	go startPeriodicBundleSender(txPool, 5*time.Second, 1, *grpcURL)

	// Create a new Gin router
	r := gin.New()

	// Use the custom logger middleware to log all HTTP requests
	r.Use(CustomLogger())

	// ToDo: define the trusted proxies in production
	r.SetTrustedProxies(nil)

	// Apply JWT authentication and rate limiting to protected routes
	protected := r.Group("/sequencer", jwtAuthMiddleware([]string{"user"}), rateLimitMiddleware())
	{
		protected.POST("/eth_sendBundle", handleBundleRequest(txPool))
		protected.POST("/eth_cancelBundle", handleCancelBundleRequest(txPool))
	}

	// Apply rate limiting to unprotected routes
	unprotected := r.Group("/sequencer", rateLimitMiddleware())
	{
		// Health check endpoint
		unprotected.GET("/health", healthHandler)

		// JWT login endpoint
		unprotected.POST("/login", jwtLoginHandler)
	}

	// ToDo: replace with a proper logger
	log.Println("Server is running on port 80...")

	// Start the HTTP server
	log.Fatal(r.Run(":80"))
}

// CustomLogger is a middleware function that logs detailed information about each request
func CustomLogger() gin.HandlerFunc {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	return func(c *gin.Context) {
		startTime := time.Now()

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(startTime)

		// Get status code
		statusCode := c.Writer.Status()

		// Log details
		logger.WithFields(logrus.Fields{
			"status_code":  statusCode,
			"latency_time": latency,
			"client_ip":    c.ClientIP(),
			"method":       c.Request.Method,
			"path":         c.Request.URL.Path,
			"user_agent":   c.Request.UserAgent(),
			"error":        c.Errors.ByType(gin.ErrorTypePrivate).String(),
		}).Info("Request details")
	}
}
