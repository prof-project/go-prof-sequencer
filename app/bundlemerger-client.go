package main

import (
	"context"
	"github.com/ethereum/go-ethereum/core/types"
	pb "github.com/prof-project/go-prof-sequencer/api/v1"
	"google.golang.org/grpc"
	"log"
	"time"
)

const (
	address = "localhost:50051"
)

func sendBundlesViaGRPC(bundles []*TxPoolBundle) error {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewBundleServiceClient(conn)

	// Build the gRPC bundles request
	var grpcBundles []*pb.Bundle
	for _, bundle := range bundles {
		grpcBundle := &pb.Bundle{
			Transactions:      serializeTransactions(bundle.Txs),
			BlockNumber:       bundle.BlockNumber,
			MinTimestamp:      bundle.MinTimestamp,      // Optional minimum timestamp
			MaxTimestamp:      bundle.MaxTimestamp,      // Optional maximum timestamp
			RevertingTxHashes: bundle.RevertingTxHashes, // Optional tx hashes allowed to revert
			ReplacementUuid:   bundle.ReplacementUuid,   // Optional replacement UUID
			Builders:          bundle.Builders,          // Optional list of builders
		}
		grpcBundles = append(grpcBundles, grpcBundle)
	}

	bundlesRequest := &pb.BundlesRequest{
		Bundles: grpcBundles, // Send the collection of bundles
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Adjust the timeout to 10 seconds
	defer cancel()

	// Send the bundle collection via gRPC
	res, err := client.SendBundles(ctx, bundlesRequest)
	if err != nil {
		log.Fatalf("Failed to send bundles: %v", err)
	}

	// Log the response statuses for each bundle
	for i, bundleRes := range res.BundleResponses {
		log.Printf("Bundle %d response: %v", i+1, bundleRes.Status)
	}

	//ToDo: introduce error handling

	return err
}

func serializeTransactions(transactions []*types.Transaction) []*pb.Transaction {
	var serialized []*pb.Transaction
	for _, tx := range transactions {
		serialized = append(serialized, &pb.Transaction{
			Data:  tx.Data(),
			Gas:   tx.Gas(),
			Nonce: tx.Nonce(),
			To:    tx.To().Hex(),
			Value: tx.Value().String(),
		})
	}
	return serialized
}

func startPeriodicBundleSender(txPool *TxBundlePool, interval time.Duration, bundleLimit int) {
	go func() {
		for {
			time.Sleep(interval)

			// Retrieve a batch of bundles ready for processing
			bundles := txPool.getBundlesForProcessing(bundleLimit)
			if len(bundles) == 0 {
				continue // No bundles to process, skip this iteration
			}

			// Send the bundles via gRPC
			err := sendBundlesViaGRPC(bundles)
			if err != nil {
				log.Printf("Error sending bundles via gRPC: %v\n", err)
				continue
			}

			// Mark the bundles as ready for deletion
			txPool.markBundlesForDeletion(bundles)

			log.Printf("%d bundles sent via gRPC and marked for deletion\n", len(bundles))
		}
	}()
}
