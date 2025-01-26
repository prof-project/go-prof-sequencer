// Package main implements the sequencer
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	pbBundleMerger "github.com/prof-project/prof-grpc/go/profpb"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// connectToGRPCServer connects to a gRPC server with optional TLS.
func connectToGRPCServer(grpcURL string, useTLS bool) (*grpc.ClientConn, error) {
	log.Info().Str("grpc_url", grpcURL).Msg("Attempting to connect to gRPC server")

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
	conn, err := grpc.Dial(grpcURL, opts...)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %v", err)
	}

	log.Info().Str("grpc_url", grpcURL).Msg("Connected to gRPC server")
	return conn, nil
}

func processBundleCollectionResponse(txPool *TxBundlePool, res *pbBundleMerger.BundlesResponse) error {
	// Process each bundle's response
	for _, bundleRes := range res.BundleResponses {
		if bundleRes.Success {
			err := txPool.cancelBundleByUUID(bundleRes.ReplacementUuid)
			if err != nil {
				return err
			}
			log.Info().Str("uuid", bundleRes.ReplacementUuid).Str("status", bundleRes.Status).Msg("Bundle processed successfully")
		} else {
			log.Error().Str("uuid", bundleRes.ReplacementUuid).Str("status", bundleRes.Status).Msg("Bundle processing failed")
		}
	}

	return nil
}

func serializeTransactions(transactions []*types.Transaction) []*pbBundleMerger.BundleTransaction {
	var serialized []*pbBundleMerger.BundleTransaction
	for _, tx := range transactions {
		data, err := rlp.EncodeToBytes(tx)
		if err != nil {
			log.Error().Err(err).Msg("Failed to serialize transaction")
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
			ReplacementUuid:   bundle.ReplacementUUID,
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
		log.Fatal().Err(err).Msg("Failed to send bundles")
	}

	// Process the response
	for i, bundleResp := range resp.BundleResponses {
		log.Info().
			Int("index", i+1).
			Str("uuid", bundleResp.ReplacementUuid).
			Str("status", bundleResp.Status).
			Bool("success", bundleResp.Success).
			Msg("Bundle response received")
	}

	err = processBundleCollectionResponse(txPool, resp)
	if err != nil {
		log.Error().Err(err).Msg("Error processing responses")
	}

	log.Info().Int("bundles_sent", len(bundles)).Msg("Bundles sent via gRPC")
}

func startPeriodicBundleSender(txPool *TxBundlePool, interval time.Duration, bundleLimit int, grpcURL string, useTLS bool) {
	// Attempt to connect to the gRPC server
	conn, err := connectToGRPCServer(grpcURL, useTLS)
	if err != nil {
		log.Fatal().Err(err).Str("grpc_url", grpcURL).Msg("Failed to connect to gRPC server")
	}
	defer conn.Close()

	client := pbBundleMerger.NewBundleServiceClient(conn)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		sendBundles(client, txPool, bundleLimit)
	}
}
