// Package main implements the sequencer
package main

import (
	"context"
	"flag"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

func main() {
	// Set up logging to a file so Promtail can read it.
	if err := os.MkdirAll("/app/logs", 0755); err != nil {
		log.Fatal().Err(err).Msg("Failed to create log directory")
	}
	logFile, err := os.OpenFile("/app/logs/app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open log file")
	}
	defer func() {
		if err := logFile.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close log file")
		}
	}()

	// Initialize Zerolog to write to both console and log file
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = zerolog.New(io.MultiWriter(logFile, os.Stdout)).With().Timestamp().Logger()

	// Make Gin write logs to file and console
	gin.DefaultWriter = io.MultiWriter(logFile, os.Stdout)
	gin.DefaultErrorWriter = io.MultiWriter(logFile, os.Stderr)

	log.Info().Msg("Starting prof-sequencer application...")

	// Set up OpenTelemetry trace exporter (OTLP -> Tempo).
	ctx := context.Background()
	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint("tempo:14268"), // Adjust if your Tempo port differs
		otlptracehttp.WithInsecure(),              // Usually no TLS in local Docker
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create OTLP exporter")
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("prof-sequencer"),
		)),
	)
	otel.SetTracerProvider(tp)
	defer func() {
		_ = tp.Shutdown(ctx)
	}()

	// Add command-line flags for gRPC URL and useTLS
	grpcURL := flag.String("grpc-url", "127.0.0.1:50051", "URL for gRPC connection to bundle merger")
	useTLS := flag.Bool("use-tls", false, "Use TLS for gRPC connection")
	flag.Parse()

	// Log the gRPC URL and useTLS flag being used
	log.Info().Str("grpc_url", *grpcURL).Bool("use_tls", *useTLS).Msg("gRPC configuration")

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
	go startPeriodicBundleSender(txPool, 1*time.Second, 100, *grpcURL, *useTLS)

	// Create a new Gin router
	rMain := gin.New()

	// Use the OTel Gin middleware
	rMain.Use(otelgin.Middleware("prof-sequencer"))

	// Use the custom logger middleware to log all HTTP requests
	rMain.Use(CustomLogger())

	// ToDo: define the trusted proxies in production
	rMain.SetTrustedProxies(nil)

	// Apply JWT authentication and rate limiting to protected routes
	protected := rMain.Group("/sequencer", jwtAuthMiddleware([]string{"user"}), rateLimitMiddleware())
	{
		protected.POST("/eth_sendBundle", handleBundleRequest(txPool))
		protected.POST("/eth_cancelBundle", handleCancelBundleRequest(txPool))
	}

	// Apply rate limiting to unprotected routes
	unprotected := rMain.Group("/sequencer", rateLimitMiddleware())
	{
		// Health check endpoint
		unprotected.GET("/health", healthHandler)

		// JWT login endpoint
		unprotected.POST("/login", jwtLoginHandler)
	}

	// Start the HTTP server with graceful shutdown
	srv := &http.Server{
		Addr:    ":80",
		Handler: rMain,
	}

	// Create a new Gin router
	rPrometheus := gin.New()

	// Expose Prometheus metrics on `/metrics`. We wrap the standard promhttp.Handler.
	rPrometheus.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Start the Prometheus metrics server
	metricsSrv := &http.Server{
		Addr:    ":8080",
		Handler: rPrometheus,
	}

	// Listen for signals to gracefully shut down
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info().Msg("Starting Gin server on :80")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("ListenAndServe error")
		}
	}()

	go func() {
		log.Info().Msg("Starting Prometheus metrics server on :8080")
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Metrics ListenAndServe error")
		}
	}()

	<-quit
	log.Info().Msg("Shutting down servers...")

	ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctxShutDown); err != nil {
		log.Fatal().Err(err).Msg("HTTP server forced to shutdown")
	}
	if err := metricsSrv.Shutdown(ctxShutDown); err != nil {
		log.Fatal().Err(err).Msg("Metrics server forced to shutdown")
	}

	log.Info().Msg("Servers exited properly")
}

// CustomLogger is a middleware function that logs detailed information about each request
func CustomLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		log.Info().
			Str("client_ip", c.ClientIP()).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Dur("latency", duration).
			Msg("request details")
	}
}
