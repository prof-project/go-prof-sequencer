package main

import (
	pbBundleMerger "github.com/prof-project/go-prof-sequencer/api/v1"
	"log"
	"net/http"
	"time"
)

func main() {
	txPool := &TxBundlePool{
		bundles:    []*TxPoolBundle{},
		bundleMap:  make(map[string]*TxPoolBundle),
		customSort: sortByBlockNumber,
	}

	// Start the cleanup job for the pool
	txPool.startCleanupJob(5 * time.Second)

	//// Setup the interface to the bundle-merger
	//ToDo: define behaviour on disconnect
	// Attempt to connect to the gRPC server
	conn, err := connectToGRPCServer()
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pbBundleMerger.NewBundleServiceClient(conn)

	// Start the periodic bundle sender (every 10 seconds up to 10 bundles at a time)
	startPeriodicBundleSender(txPool, client, 10*time.Second, 5)

	// Register the handler and pass the txPool to it
	http.HandleFunc("/eth_sendBundle", handleBundleRequest(txPool))
	http.HandleFunc("/eth_cancelBundle", handleCancelBundleRequest(txPool))

	//ToDo: replace with a proper logger
	log.Println("Server is running on port 8080...")

	// Start the HTTP server
	log.Fatal(http.ListenAndServe(":8080", nil))
}
