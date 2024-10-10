package main

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
	"log"
	"net/http"
)

type SendBundleRequest struct {
	JSONRPC string             `json:"jsonrpc"` // JSON-RPC version,
	ID      int                `json:"id"`      // ID of the request
	Method  string             `json:"method"`  // Method name, should be "eth_sendBundle"
	Params  []SendBundleParams `json:"params"`  // Array containing bundle params objects
}

type SendBundleParams struct {
	Txs               []string `json:"txs"`                         // Array of signed transactions (hex strings)
	BlockNumber       string   `json:"blockNumber"`                 // Hex-encoded block number
	MinTimestamp      int64    `json:"minTimestamp,omitempty"`      // Optional minimum timestamp
	MaxTimestamp      int64    `json:"maxTimestamp,omitempty"`      // Optional maximum timestamp
	RevertingTxHashes []string `json:"revertingTxHashes,omitempty"` // Optional list of tx hashes allowed to revert
	ReplacementUuid   string   `json:"replacementUuid,omitempty"`   // Optional replacement UUID
	Builders          []string `json:"builders,omitempty"`          // Optional list of builder names
}

// Handle eth_sendBundle requests
func handleBundleRequest(txPool *TxBundlePool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		// Ensure there are bundles in the params
		if len(req.Params) == 0 {
			http.Error(w, "Missing params", http.StatusBadRequest)
			return
		}

		// Process each bundle in the Params array
		for _, params := range req.Params {
			// If no ReplacementUuid is provided, generate one
			if params.ReplacementUuid == "" {
				newUuid := uuid.New().String()
				log.Printf("Generated new UUID for bundle: %s\n", newUuid)
				params.ReplacementUuid = newUuid
			}

			// Decode the hex-encoded transactions
			var validTxs []*types.Transaction
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
					validTxs = append(validTxs, tx)
				} else {
					log.Printf("Skipping invalid transaction: %+v\n", tx)
				}
			}

			// Ensure at least one valid transaction exists in the bundle
			if len(validTxs) == 0 {
				log.Printf("No valid transactions in the bundle for UUID: %s\n", params.ReplacementUuid)
				continue
			}

			// Create the TxPoolBundle using the validated transactions
			bundle := TxPoolBundle{
				Txs:               validTxs,
				BlockNumber:       params.BlockNumber,
				MinTimestamp:      params.MinTimestamp,
				MaxTimestamp:      params.MaxTimestamp,
				RevertingTxHashes: params.RevertingTxHashes,
				ReplacementUuid:   params.ReplacementUuid, // Use either the provided or newly generated UUID
				Builders:          params.Builders,
			}

			// Add the bundle to the transaction pool
			err := txPool.addBundle(&bundle, false)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to add bundle with UUID %s to pool: %v", bundle.ReplacementUuid, err), http.StatusBadRequest)
				log.Printf("Failed to add bundle with UUID %s to pool: %v\n", bundle.ReplacementUuid, err)
				return
			}

			log.Printf("Bundle with UUID %s received and added to the pool", bundle.ReplacementUuid)
		}

		//ToDo: implement proper reply
		// Respond with success after all bundles are processed
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("Bundles received and added to the pool"))
	}
}

type CancelBundleRequest struct {
	JSONRPC string               `json:"jsonrpc"` // JSON-RPC version
	ID      int                  `json:"id"`      // ID of the request
	Method  string               `json:"method"`  // Method name, should be "eth_cancelBundle"
	Params  []CancelBundleParams `json:"params"`  // Array containing CancelBundleParams objects
}

type CancelBundleParams struct {
	ReplacementUuid string `json:"replacementUuid"` // UUIDv4 to uniquely identify the bundle to cancel
}

// Handle eth_cancelBundle requests
func handleCancelBundleRequest(txPool *TxBundlePool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CancelBundleRequest

		// Parse the JSON body into the CancelBundleRequest struct
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request payload: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Ensure the method is eth_cancelBundle
		if req.Method != "eth_cancelBundle" {
			http.Error(w, "Invalid method: "+req.Method, http.StatusBadRequest)
			return
		}

		// Ensure there are bundles in the params
		if len(req.Params) == 0 {
			http.Error(w, "Missing params", http.StatusBadRequest)
			return
		}

		var canceledBundles []string
		var failedBundles []string

		// Process each bundle in the Params array
		for _, param := range req.Params {
			// Check if ReplacementUuid is provided
			if param.ReplacementUuid == "" {
				failedBundles = append(failedBundles, "missing UUID")
				continue
			}

			// Attempt to cancel the bundle by replacementUuid
			err := txPool.cancelBundleByUuid(param.ReplacementUuid)
			if err != nil {
				log.Printf("Failed to cancel bundle with UUID: %s, error: %v\n", param.ReplacementUuid, err)
				failedBundles = append(failedBundles, param.ReplacementUuid)
				continue
			}

			canceledBundles = append(canceledBundles, param.ReplacementUuid)
		}

		// Create the response
		response := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      req.ID,
		}

		// Handle partial success or failure
		if len(failedBundles) > 0 {
			response["error"] = fmt.Sprintf("Failed to cancel bundles with UUIDs: %v", failedBundles)
			w.WriteHeader(http.StatusMultiStatus) // 207 Multi-Status (to indicate partial success)
		} else {
			response["result"] = "All bundles canceled successfully"
			w.WriteHeader(http.StatusOK)
		}

		// Respond with the JSON result
		responseBody, _ := json.Marshal(response)
		w.Write(responseBody)
	}
}
