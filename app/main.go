package main

import (
	"fmt"
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

	// Start the periodic bundle sender (every 10 seconds up to 10 bundles at a time)
	startPeriodicBundleSender(txPool, 5*time.Second, 100)

	// Register the handler and pass the txPool to it
	http.HandleFunc("/eth_sendBundle", handleBundleRequest(txPool))
	http.HandleFunc("/eth_cancelBundle", handleCancelBundleRequest(txPool))

	// Health check endpoint
	http.HandleFunc("/health", healthHandler)

	// ToDo: replace with a proper logger
	log.Println("Server is running on port 8084...")

	// Start the HTTP server
	log.Fatal(http.ListenAndServe(":8084", nil))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	// Check the health and return a status code accordingly
	if isHealthy() {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Service is healthy")
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Service is not healthy")
	}
}

// ToDo: implement a proper health check
func isHealthy() bool {
	return true
}
