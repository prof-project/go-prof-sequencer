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
	"github.com/natefinch/lumberjack"
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
	// Add command-line flags for Prometheus metrics, log level and logging mode
	enableMetrics := flag.Bool("enable-metrics", false, "Enable Prometheus metrics endpoint")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error, fatal, panic)")
	logToFile := flag.Bool("log-to-file", false, "Log to file and stdout (true) or only stdout (false)")
	// Add command-line flags for gRPC URL and useTLS
	grpcURL := flag.String("grpc-url", "127.0.0.1:50051", "URL for gRPC connection to bundle merger")
	useTLS := flag.Bool("use-tls", false, "Use TLS for gRPC connection")
	flag.Parse()

	// Set log level
	level, err := zerolog.ParseLevel(*logLevel)
	if err != nil {
		log.Fatal().Err(err).Msg("Invalid log level")
	}
	zerolog.SetGlobalLevel(level)

	// Set up logging to a file with rotation if enabled
	var logWriters []io.Writer
	if *logToFile {
		logFile := &lumberjack.Logger{
			Filename:   "/app/logs/app.log",
			MaxSize:    5, // megabytes
			MaxBackups: 3,
			MaxAge:     28,    // days
			Compress:   false, // disabled by default
		}
		defer func() {
			if err := logFile.Close(); err != nil {
				log.Error().Err(err).Msg("Failed to close log file")
			}
		}()
		logWriters = append(logWriters, logFile)
	}
	logWriters = append(logWriters, os.Stdout)

	// Initialize Zerolog to write to the configured outputs
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = zerolog.New(io.MultiWriter(logWriters...)).With().Timestamp().Logger()

	// Make Gin write logs to the configured outputs
	gin.DefaultWriter = io.MultiWriter(logWriters...)
	gin.DefaultErrorWriter = io.MultiWriter(logWriters...)

	log.Info().Msg("Starting prof-sequencer application...")

	// Set up OpenTelemetry trace exporter (OTLP -> Tempo).
	ctx := context.Background()
	traceExporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpoint("tempo:4318"), otlptracehttp.WithInsecure())
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

	// Log the gRPC URL and useTLS flag being used
	log.Info().Str("grpc_url", *grpcURL).Bool("use_tls", *useTLS).Msg("gRPC configuration")

	txPool := &TxBundlePool{
		bundles:    []*TxPoolBundle{},
		bundleMap:  make(map[string]*TxPoolBundle),
		customSort: sortByBlockNumber,
	}

	// Set Gin to Release mode
	gin.SetMode(gin.ReleaseMode)

	// Start the cleanup job for the pool
	txPool.startCleanupJob(5 * time.Second)

	// Start the periodic bundle sender
	go startPeriodicBundleSender(txPool, 1*time.Second, 100, *grpcURL, *useTLS)

	// Create a new Gin router
	rMain := gin.New()

	// Use the OTel Gin middleware
	rMain.Use(otelgin.Middleware("prof-sequencer"))

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

	// Create a new Gin router for Prometheus metrics if enabled
	var metricsSrv *http.Server
	if *enableMetrics {
		rPrometheus := gin.New()
		rPrometheus.GET("/metrics", gin.WrapH(promhttp.Handler()))
		metricsSrv = &http.Server{
			Addr:    ":8080",
			Handler: rPrometheus,
		}
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

	if *enableMetrics {
		go func() {
			log.Info().Msg("Starting Prometheus metrics server on :8080")
			if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatal().Err(err).Msg("Metrics ListenAndServe error")
			}
		}()
	}

	<-quit
	log.Info().Msg("Shutting down servers...")

	ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctxShutDown); err != nil {
		log.Fatal().Err(err).Msg("HTTP server forced to shutdown")
	}
	if *enableMetrics {
		if err := metricsSrv.Shutdown(ctxShutDown); err != nil {
			log.Fatal().Err(err).Msg("Metrics server forced to shutdown")
		}
	}

	log.Info().Msg("Servers exited properly")
}
