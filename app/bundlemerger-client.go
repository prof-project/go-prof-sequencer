package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	pbBundleMerger "github.com/prof-project/prof-grpc/go/profpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
)

func connectToGRPCServer(grpcURL string) (*grpc.ClientConn, error) {
	// Define custom backoff settings for reconnection attempts
	backoffConfig := backoff.Config{
		BaseDelay:  1.0 * time.Second, // Start with a 1-second delay
		Multiplier: 1.6,               // Exponential backoff multiplier
		MaxDelay:   120 * time.Second, // Maximum backoff delay
	}

	// Connect to gRPC server with a retry mechanism
	conn, err := grpc.Dial(grpcURL,
		grpc.WithInsecure(),
		grpc.WithBlock(), // Block until connection is established
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff:           backoffConfig,
			MinConnectTimeout: 5 * time.Second, // Minimum time to wait for connection
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server at %s: %v", grpcURL, err)
	}
	return conn, nil
}

func streamBundleCollections(bundles []*TxPoolBundle, stream pbBundleMerger.BundleService_StreamBundleCollectionsClient) error {
	// Convert bundles from TxPoolBundle to the gRPC Bundles format
	var grpcBundles []*pbBundleMerger.Bundle
	for _, bundle := range bundles {
		grpcBundles = append(grpcBundles, &pbBundleMerger.Bundle{
			Transactions:      serializeTransactions(bundle.Txs),
			ReplacementUuid:   bundle.ReplacementUuid,
			BlockNumber:       bundle.BlockNumber,
			MinTimestamp:      bundle.MinTimestamp,
			MaxTimestamp:      bundle.MaxTimestamp,
			RevertingTxHashes: bundle.RevertingTxHashes,
			Builders:          bundle.Builders,
		})
	}

	// Send the collection of bundles to the server
	err := stream.Send(&pbBundleMerger.BundlesRequest{
		Bundles: grpcBundles,
	})
	if err != nil {
		return err
	}

	return err
}

func processBundleCollectionResponse(txPool *TxBundlePool, stream pbBundleMerger.BundleService_StreamBundleCollectionsClient) error {
	for {
		// Receive the response from the server
		res, err := stream.Recv()
		if err == io.EOF {
			// No more responses from the server
			return nil
		}
		if err != nil {
			return err
		}

		// Process each bundle's response
		for _, bundleRes := range res.BundleResponses {
			if bundleRes.Success {
				err := txPool.cancelBundleByUuid(bundleRes.ReplacementUuid)
				if err != nil {
					return err
				}
				log.Printf("Bundle %s processed successfully: %s\n", bundleRes.ReplacementUuid, bundleRes.Status)
			} else {
				log.Printf("Bundle %s failed: %s\n", bundleRes.ReplacementUuid, bundleRes.Status)
			}
		}
	}
}

func serializeTransactions(transactions []*types.Transaction) []*pbBundleMerger.BundleTransaction {
	var serialized []*pbBundleMerger.BundleTransaction
	for _, tx := range transactions {
		data, err := rlp.EncodeToBytes(tx)
		if err != nil {
			log.Printf("Failed to serialize transaction: %v\n", err)
			continue
		}
		serialized = append(serialized, &pbBundleMerger.BundleTransaction{Data: data})
	}
	return serialized
}

func startPeriodicBundleSender(txPool *TxBundlePool, interval time.Duration, bundleLimit int, grpcURL string) {
	go func() {
		// Attempt to connect to the gRPC server
		conn, err := connectToGRPCServer(grpcURL)
		if err != nil {
			log.Fatalf("Failed to connect to %s: %v", grpcURL, err)
		}
		defer conn.Close()

		client := pbBundleMerger.NewBundleServiceClient(conn)

		for {
			// Open the gRPC stream, with retry logic if the stream fails
			stream, err := client.StreamBundleCollections(context.Background())
			if err != nil {
				log.Printf("Failed to open stream: %v\n", err)
				time.Sleep(5 * time.Second) // Wait before retrying
				continue
			}

			// Start a goroutine to process server responses
			go func() {
				err := processBundleCollectionResponse(txPool, stream)
				if err != nil {
					log.Printf("Error processing responses: %v\n", err)
				}
			}()

			for {
				time.Sleep(interval)

				// Retrieve a batch of bundles ready for processing
				bundles := txPool.getBundlesForProcessing(bundleLimit, false)
				if len(bundles) == 0 {
					continue // No bundles to process, skip this iteration
				}

				// Send the bundles via gRPC stream
				err := streamBundleCollections(bundles, stream)
				if err != nil {
					log.Printf("Error sending bundles via gRPC: %v\n", err)
					break // Exit the loop and attempt to reconnect
				}

				log.Printf("%d bundles sent via gRPC\n", len(bundles))
			}

			// Close the stream if it fails
			stream.CloseSend()
		}
	}()
}
