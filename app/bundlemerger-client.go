package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	pbBundleMerger "github.com/prof-project/prof-grpc/go/profpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// connectToGRPCServer connects to a gRPC server with optional TLS.
func connectToGRPCServer(grpcURL string, useTLS bool) (*grpc.ClientConn, error) {
	log.Printf("Attempting to connect to gRPC server at %s...", grpcURL)

	// Define custom backoff settings for reconnection attempts
	backoffConfig := backoff.Config{
		BaseDelay:  1.0 * time.Second, // Start with a 1-second delay
		Multiplier: 1.6,               // Exponential backoff multiplier
		MaxDelay:   120 * time.Second, // Maximum backoff delay
	}

	var opts []grpc.DialOption

	if useTLS {
		// Create a tls.Config with InsecureSkipVerify set to true
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
		}
		creds := credentials.NewTLS(tlsConfig)
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	opts = append(opts, grpc.WithConnectParams(grpc.ConnectParams{
		Backoff:           backoffConfig,
		MinConnectTimeout: 5 * time.Second,
	}))

	// Connect to the server
	conn, err := grpc.NewClient(grpcURL, opts...)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %v", err)
	}

	log.Printf("Connected to gRPC server at %s", grpcURL)
	return conn, nil
}

func processBundleCollectionResponse(txPool *TxBundlePool, res *pbBundleMerger.BundlesResponse) error {
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

	return nil
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

func convertToGRPCBundles(bundles []*TxPoolBundle) []*pbBundleMerger.Bundle {
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
	return grpcBundles
}

func sendBundles(client pbBundleMerger.BundleServiceClient, txPool *TxBundlePool, bundleLimit int) {
	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Retrieve a batch of bundles ready for processing
	bundles := txPool.getBundlesForProcessing(bundleLimit, false)
	if len(bundles) == 0 {
		return // No bundles to process, skip this iteration
	}

	req := &pbBundleMerger.BundlesRequest{
		Bundles: convertToGRPCBundles(bundles),
	}

	// Send the request and receive the response
	resp, err := client.SendBundleCollections(ctx, req)
	if err != nil {
		log.Fatalf("Failed to send bundles: %v", err)
	}

	// Process the response
	for i, bundleResp := range resp.BundleResponses {
		log.Printf("Bundle %d: ReplacementUuid: %s, Status: %s, Success: %v",
			i+1, bundleResp.ReplacementUuid, bundleResp.Status, bundleResp.Success)
	}

	err = processBundleCollectionResponse(txPool, resp)
	if err != nil {
		log.Printf("Error processing responses: %v\n", err)
	}

	log.Printf("%d bundles sent via gRPC\n", len(bundles))
}

func startPeriodicBundleSender(txPool *TxBundlePool, interval time.Duration, bundleLimit int, grpcURL string, useTLS bool) {
	// Attempt to connect to the gRPC server
	conn, err := connectToGRPCServer(grpcURL, useTLS)
	if err != nil {
		log.Fatalf("Failed to connect to %s: %v", grpcURL, err)
	}
	defer conn.Close()

	client := pbBundleMerger.NewBundleServiceClient(conn)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		sendBundles(client, txPool, bundleLimit)
	}
}
