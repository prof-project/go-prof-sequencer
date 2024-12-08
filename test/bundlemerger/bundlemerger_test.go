package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	pbBundleMerger "github.com/prof-project/prof-grpc/go/profpb"
	"github.com/stretchr/testify/assert"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	pbBundleMerger.RegisterBundleServiceServer(s, &server{})
	go func() {
		if err := s.Serve(lis); err != nil {
			panic(err)
		}
	}()
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func TestHealthCheckEndpoint(t *testing.T) {
	req, err := http.NewRequest("GET", "/sequencer-testserver/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "ok", rr.Body.String())
}

func TestSendBundleCollections(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := pbBundleMerger.NewBundleServiceClient(conn)

	req := &pbBundleMerger.BundlesRequest{
		Bundles: []*pbBundleMerger.Bundle{
			{
				BlockNumber:  "12345",
				MinTimestamp: 1609459200,
				MaxTimestamp: 1609459260,
				Transactions: []*pbBundleMerger.BundleTransaction{
					{Data: []byte("tx1")},
					{Data: []byte("tx2")},
				},
				RevertingTxHashes: []string{"tx3"},
				ReplacementUuid:   "uuid-123",
				Builders:          []string{"builder1"},
			},
		},
	}

	resp, err := client.SendBundleCollections(ctx, req)
	if err != nil {
		t.Fatalf("SendBundleCollections failed: %v", err)
	}

	assert.Equal(t, 1, len(resp.BundleResponses))
	assert.True(t, resp.BundleResponses[0].Success)
}
