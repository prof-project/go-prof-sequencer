package main

import (
	"context"
	"log"
	"net"

	pbBundleMerger "github.com/prof-project/go-prof-sequencer/api/v1"
	"google.golang.org/grpc"
)

type server struct {
	pbBundleMerger.UnimplementedBundleServiceServer
}

func (s *server) SendBundle(ctx context.Context, req *pbBundleMerger.BundleRequest) (*pbBundleMerger.BundleResponse, error) {
	log.Printf("Received bundle with %d transactions", len(req.Transactions))

	// Perform merge with public Ethereum block proposal
	// (e.g., communicate with a miner or MEV searcher)

	return &pbBundleMerger.BundleResponse{Status: "Bundle merged successfully"}, nil
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
