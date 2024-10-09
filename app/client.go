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

func sendBundleViaGRPC(bundle *Bundle) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewBundleServiceClient(conn)

	// Build the gRPC bundle request
	bundleRequest := &pb.BundleRequest{
		Transactions:      serializeTransactions(bundle.Transactions),
		BlockNumber:       bundle.BlockNumber,
		MinTimestamp:      bundle.MinTimestamp,      // Optional timestamp
		MaxTimestamp:      bundle.MaxTimestamp,      // Optional timestamp
		RevertingTxHashes: bundle.RevertingTxHashes, // Optional tx hashes allowed to revert
		ReplacementUuid:   bundle.ReplacementUuid,   // Optional replacement UUID
		Builders:          bundle.Builders,          // Optional list of builders
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := client.SendBundle(ctx, bundleRequest)
	if err != nil {
		log.Fatalf("Failed to send bundle: %v", err)
	}

	log.Printf("Bundler response: %v", res.Status)
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
