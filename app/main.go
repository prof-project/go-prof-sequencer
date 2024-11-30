package main

import (
	"flag"
	"log"
	"time"

	"github.com/gin-gonic/gin"
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

	r := gin.Default()
	// ToDo: define the trusted proxies in production
	r.SetTrustedProxies(nil)

	// Apply JWT authentication and rate limiting to protected routes
	protected := r.Group("/sequencer", jwtAuthMiddleware([]string{"user"}), rateLimitMiddleware())
	{
		protected.POST("/eth_sendBundle", func(c *gin.Context) {
			handleBundleRequest(txPool)(c.Writer, c.Request)
		})
		protected.POST("/eth_cancelBundle", func(c *gin.Context) {
			handleCancelBundleRequest(txPool)(c.Writer, c.Request)
		})
	}

	// Apply rate limiting to unprotected routes
	unprotected := r.Group("/sequencer", rateLimitMiddleware())
	{
		// Health check endpoint
		unprotected.GET("/health", func(c *gin.Context) {
			healthHandler(c.Writer, c.Request)
		})

		// JWT login endpoint
		unprotected.POST("/login", jwtLoginHandler)
	}

	// ToDo: replace with a proper logger
	log.Println("Server is running on port 80...")

	// Start the HTTP server
	log.Fatal(r.Run(":80"))
}
