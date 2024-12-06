package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"google.golang.org/grpc"

	pbBundleMerger "github.com/prof-project/prof-grpc/go/profpb"
)

type server struct {
	pbBundleMerger.UnimplementedBundleServiceServer
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pbBundleMerger.RegisterBundleServiceServer(s, &server{})

	log.Println("gRPC server running on port 50051...")

	// Start health check endpoint
	go startHealthCheck()

	// Log incoming connections
	go func() {
		for {
			conn, err := lis.Accept()
			if err != nil {
				log.Printf("Failed to accept connection: %v", err)
				continue
			}
			log.Printf("Accepted new connection from %v", conn.RemoteAddr())
			conn.Close()
		}
	}()

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

// Start a simple HTTP server for health checks
func startHealthCheck() {
	http.HandleFunc("/sequencer/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	log.Println("Health check endpoint running on port 80...")
	if err := http.ListenAndServe(":80", nil); err != nil {
		log.Fatalf("Failed to start health check endpoint: %v", err)
	}
}

// Implement the SendBundleCollections rpc of the BundleService service
func (s *server) SendBundleCollections(ctx context.Context, req *pbBundleMerger.BundlesRequest) (*pbBundleMerger.BundlesResponse, error) {
	log.Printf("Received %d bundles", len(req.Bundles))

	// Prepare to collect responses for each bundle
	var bundleResponses []*pbBundleMerger.BundleResponse

	// Process each bundle in the collection
	for i, bundle := range req.Bundles {
		log.Printf("Processing bundle %d with %d transactions", i+1, len(bundle.Transactions))

		// Log other bundle information
		log.Printf("Bundle BlockNumber: %s, MinTimestamp: %d, MaxTimestamp: %d",
			bundle.BlockNumber, bundle.MinTimestamp, bundle.MaxTimestamp)

		// Optional fields
		if len(bundle.RevertingTxHashes) > 0 {
			log.Printf("RevertingTxHashes: %v", bundle.RevertingTxHashes)
		}
		if bundle.ReplacementUuid != "" {
			log.Printf("ReplacementUuid: %s", bundle.ReplacementUuid)
		}
		if len(bundle.Builders) > 0 {
			log.Printf("Builders: %v", bundle.Builders)
		}

		// Simulate some processing (e.g., interacting with miners or MEV searchers)
		err := simulateBundleProcessing(bundle)
		var statusMessage string
		var success bool

		if err != nil {
			log.Printf("Error processing bundle %d: %v", i+1, err)
			statusMessage = fmt.Sprintf("Failed to merge bundle: %v", err)
			success = false
		} else {
			log.Printf("Bundle %d processed successfully", i+1)
			statusMessage = "Bundle merged successfully"
			success = true
		}

		// Add response for the current bundle, including its UUID and processing result
		bundleResponses = append(bundleResponses, &pbBundleMerger.BundleResponse{
			ReplacementUuid: bundle.ReplacementUuid,
			Status:          statusMessage,
			Success:         success,
		})
	}

	// Send the response for the entire collection of bundles
	response := &pbBundleMerger.BundlesResponse{
		BundleResponses: bundleResponses,
	}
	return response, nil
}

// Simulate some bundle processing, like communication with miners or other external services
func simulateBundleProcessing(bundle *pbBundleMerger.Bundle) error {
	log.Printf("Simulating processing of bundle for BlockNumber %s", bundle.BlockNumber)
	time.Sleep(1 * time.Second)

	// Simulate a simple success case for now
	return nil
}
