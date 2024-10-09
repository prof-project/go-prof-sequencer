package main

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/core/types"
	"log"
	"net/http"
	"sync"
)

type SendBundleParams struct {
	Txs               []string `json:"txs"`                         // Array of signed transactions (hex strings)
	BlockNumber       string   `json:"blockNumber"`                 // Hex-encoded block number
	MinTimestamp      int64    `json:"minTimestamp,omitempty"`      // Optional minimum timestamp
	MaxTimestamp      int64    `json:"maxTimestamp,omitempty"`      // Optional maximum timestamp
	RevertingTxHashes []string `json:"revertingTxHashes,omitempty"` // Optional list of tx hashes allowed to revert
	ReplacementUuid   string   `json:"replacementUuid,omitempty"`   // Optional replacement UUID
	Builders          []string `json:"builders,omitempty"`          // Optional list of builder names
}

type SendBundleRequest struct {
	JSONRPC string             `json:"jsonrpc"` // JSON-RPC version, e.g., "2.0"
	ID      int                `json:"id"`      // ID of the request
	Method  string             `json:"method"`  // Method name, should be "eth_sendBundle"
	Params  []SendBundleParams `json:"params"`  // Array containing a single bundle params object
}

var transactionPool = []*types.Transaction{}
var poolMutex sync.Mutex

func main() {
	http.HandleFunc("/eth_sendBundle", handleBundleRequest)
	log.Println("Server is running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// Handle eth_sendBundle requests
func handleBundleRequest(w http.ResponseWriter, r *http.Request) {
	var req SendBundleRequest // Declare the correct struct for the request

	// Decode the incoming request body into the SendBundleRequest struct
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Ensure the method is eth_sendBundle
	if req.Method != "eth_sendBundle" {
		http.Error(w, "Invalid method: "+req.Method, http.StatusBadRequest)
		return
	}

	// Extract the parameters (we're only expecting one bundle in params)
	if len(req.Params) == 0 {
		http.Error(w, "Missing params", http.StatusBadRequest)
		return
	}
	params := req.Params[0]

	// Decode the hex-encoded transactions
	var transactions []*types.Transaction
	for _, txHex := range params.Txs {
		txData, err := decodeHex(txHex)
		if err != nil {
			log.Printf("Failed to decode transaction hex: %v\n", err)
			continue
		}
		tx := new(types.Transaction)
		err = tx.UnmarshalBinary(txData)
		if err != nil {
			log.Printf("Failed to unmarshal transaction: %v\n", err)
			continue
		}

		if isValidTransaction(tx) {
			transactions = append(transactions, tx)
		} else {
			log.Printf("Skipping invalid transaction: %+v\n", tx)
		}
	}

	// Validate and create the bundle (only valid transactions will be included)
	bundle := createBundle(
		transactions,
		params.BlockNumber,
		params.MinTimestamp,
		params.MaxTimestamp,
		params.RevertingTxHashes,
		params.ReplacementUuid,
		params.Builders,
	)

	// Process the bundle in a separate goroutine (e.g., send it via gRPC)
	go sendBundleViaGRPC(bundle)

	// Respond with success
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Bundle received and processing started"))
}
