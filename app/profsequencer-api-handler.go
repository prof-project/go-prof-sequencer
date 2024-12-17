// Package main implements the sequencer
package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SendBundleRequest represents a request to send a bundle.
type SendBundleRequest struct {
	JSONRPC string             `json:"jsonrpc"` // JSON-RPC version,
	ID      int                `json:"id"`      // ID of the request
	Method  string             `json:"method"`  // Method name, should be "eth_sendBundle"
	Params  []SendBundleParams `json:"params"`  // Array containing bundle params objects
}

// SendBundleParams represents parameters for sending a bundle.
type SendBundleParams struct {
	Txs               []string `json:"txs"`                         // Array of signed transactions (hex strings)
	BlockNumber       string   `json:"blockNumber"`                 // Hex-encoded block number
	MinTimestamp      int64    `json:"minTimestamp,omitempty"`      // Optional minimum timestamp
	MaxTimestamp      int64    `json:"maxTimestamp,omitempty"`      // Optional maximum timestamp
	RevertingTxHashes []string `json:"revertingTxHashes,omitempty"` // Optional list of tx hashes allowed to revert
	ReplacementUUID   string   `json:"replacementUuid,omitempty"`   // Optional replacement UUID
	Builders          []string `json:"builders,omitempty"`          // Optional list of builder names
}

func handleBundleRequest(txPool *TxBundlePool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req SendBundleRequest

		// Read the raw request body
		rawBody, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read request body: " + err.Error()})
			return
		}

		// Decode the incoming request body into the SendBundleRequest struct
		if err := json.Unmarshal(rawBody, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
			return
		}

		// Ensure the method is eth_sendBundle
		if req.Method != "eth_sendBundle" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid method: " + req.Method})
			return
		}

		// Ensure there are bundles in the params
		if len(req.Params) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing params"})
			return
		}

		var processedBundles []string
		var failedBundles []string

		// Process each bundle in the Params array
		for _, params := range req.Params {
			// If no ReplacementUUID is provided, generate one
			if params.ReplacementUUID == "" {
				newUUID := uuid.New().String()
				log.Printf("Generated new UUID for bundle: %s\n", newUUID)
				params.ReplacementUUID = newUUID
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
				// Optional logging
				if false {
					// Derive the sender address based on the transaction type
					var signer types.Signer
					switch tx.Type() {
					case types.LegacyTxType:
						signer = types.HomesteadSigner{}
					case types.AccessListTxType:
						signer = types.NewEIP2930Signer(tx.ChainId())
					case types.DynamicFeeTxType:
						signer = types.NewLondonSigner(tx.ChainId())
					default:
						log.Printf("Unsupported transaction type: %d\n", tx.Type())
						continue
					}

					from, err := types.Sender(signer, tx)
					if err != nil {
						log.Printf("Failed to derive sender address: %v\n", err)
						continue
					}

					// Get the signature values
					v, r, s := tx.RawSignatureValues()

					log.Printf("Transaction details: %+v\n", tx)
					log.Printf("Sender address: %s\n", from.Hex())
					log.Printf("Signature values: v=%d, r=%x, s=%x\n", v, r, s)
				}

				if isValidTransaction(tx) {
					validTxs = append(validTxs, tx)
					log.Printf("Valid transaction: %+v\n", tx)
					log.Printf("Transaction nonce: %d\n", tx.Nonce())
				} else {
					log.Printf("Skipping invalid transaction: %+v\n", tx)
				}
			}

			// Ensure at least one valid transaction exists in the bundle
			if len(validTxs) == 0 {
				log.Printf("No valid transactions in the bundle for UUID: %s\n", params.ReplacementUUID)
				failedBundles = append(failedBundles, params.ReplacementUUID)
				continue
			}

			// Create the TxPoolBundle using the validated transactions
			bundle := TxPoolBundle{
				Txs:               validTxs,
				BlockNumber:       params.BlockNumber,
				MinTimestamp:      params.MinTimestamp,
				MaxTimestamp:      params.MaxTimestamp,
				RevertingTxHashes: params.RevertingTxHashes,
				ReplacementUUID:   params.ReplacementUUID,
				Builders:          params.Builders,
			}

			// Log details of each transaction in the bundle
			for j, tx := range bundle.Txs {
				log.Printf("Transaction %d: To: %v, Nonce: %d, Gas: %d, Value: %v",
					j+1, tx.To(), tx.Nonce(), tx.Gas(), tx.Value())
			}

			// Add the bundle to the transaction pool
			err := txPool.addBundle(&bundle, false)
			if err != nil {
				log.Printf("Failed to add bundle with UUID %s to pool: %v\n", bundle.ReplacementUUID, err)
				failedBundles = append(failedBundles, bundle.ReplacementUUID)
				continue
			}

			processedBundles = append(processedBundles, bundle.ReplacementUUID)
			log.Printf("Bundle with UUID %s received and added to the pool", bundle.ReplacementUUID)
		}

		// Respond with the result
		response := map[string]interface{}{
			"processedBundles": processedBundles,
			"failedBundles":    failedBundles,
		}

		c.JSON(http.StatusAccepted, response)
	}
}

// CancelBundleRequest represents a request to cancel a bundle.
type CancelBundleRequest struct {
	JSONRPC string               `json:"jsonrpc"` // JSON-RPC version
	ID      int                  `json:"id"`      // ID of the request
	Method  string               `json:"method"`  // Method name, should be "eth_cancelBundle"
	Params  []CancelBundleParams `json:"params"`  // Array containing CancelBundleParams objects
}

// CancelBundleParams represents parameters for canceling a bundle.
type CancelBundleParams struct {
	ReplacementUUID string `json:"replacementUuid"` // UUIDv4 to uniquely identify the bundle to cancel
}

// Handle eth_cancelBundle requests
func handleCancelBundleRequest(txPool *TxBundlePool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CancelBundleRequest

		// Parse the JSON body into the CancelBundleRequest struct
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
			return
		}

		// Ensure the method is eth_cancelBundle
		if req.Method != "eth_cancelBundle" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid method: " + req.Method})
			return
		}

		// Ensure there are bundles in the params
		if len(req.Params) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing params"})
			return
		}

		var canceledBundles []string
		var failedBundles []string

		// Process each bundle in the Params array
		for _, param := range req.Params {
			// Check if ReplacementUUID is provided
			if param.ReplacementUUID == "" {
				failedBundles = append(failedBundles, "missing UUID")
				continue
			}

			// Attempt to cancel the bundle by replacementUUID
			err := txPool.cancelBundleByUUID(param.ReplacementUUID)
			if err != nil {
				log.Printf("Failed to cancel bundle with UUID: %s, error: %v\n", param.ReplacementUUID, err)
				failedBundles = append(failedBundles, param.ReplacementUUID)
				continue
			}

			canceledBundles = append(canceledBundles, param.ReplacementUUID)
		}

		// Create the response
		response := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      req.ID,
		}

		// Handle partial success or failure
		if len(failedBundles) > 0 {
			response["error"] = map[string]interface{}{
				"message":       "Failed to cancel some bundles",
				"failedBundles": failedBundles,
			}
			c.JSON(http.StatusMultiStatus, response) // 207 Multi-Status (to indicate partial success)
		} else {
			response["result"] = "All bundles canceled successfully"
			c.JSON(http.StatusOK, response)
		}
	}
}
