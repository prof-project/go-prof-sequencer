package main

import (
	"fmt"
	"google.golang.org/grpc"
	"io"
	"log"
	"net"
	"time"

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
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

// Implement the StreamBundleCollections rpc of the BundleService service
func (s *server) StreamBundleCollections(stream pbBundleMerger.BundleService_StreamBundleCollectionsServer) error {
	for {
		// Receive the next collection of bundles from the client
		req, err := stream.Recv()
		if err == io.EOF {
			// No more collections from the client
			return nil
		}
		if err != nil {
			return err
		}

		log.Printf("Received %d bundles", len(req.Bundles))

		// Prepare to collect responses for each bundle
		var bundleResponses []*pbBundleMerger.BundleResponse

		// Process each bundle in the collection
		for i, bundle := range req.Bundles {
			log.Printf("Processing bundle %d with %d transactions", i+1, len(bundle.Transactions))

			// Log details of each transaction in the bundle
			for j, tx := range bundle.Transactions {
				log.Printf("Transaction %d: To: %s, Nonce: %d, Gas: %d, Value: %s",
					j+1, tx.To, tx.Nonce, tx.Gas, tx.Value)
			}

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
		if err := stream.Send(response); err != nil {
			return err
		}
	}
}

// Simulate some bundle processing, like communication with miners or other external services
func simulateBundleProcessing(bundle *pbBundleMerger.Bundle) error {
	log.Printf("Simulating processing of bundle for BlockNumber %s", bundle.BlockNumber)
	time.Sleep(1 * time.Second)

	// Simulate a simple success case for now
	return nil
}
