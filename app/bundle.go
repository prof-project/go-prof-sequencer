package main

import (
	"github.com/ethereum/go-ethereum/core/types"
)

// ToDo: use in TxPoolBundle
// Bundle represents the full bundle structure, including transactions and additional metadata
type Bundle struct {
	Transactions      []*types.Transaction // Processed and decoded Ethereum transactions
	BlockNumber       string               // Target block number (hex-encoded)
	MinTimestamp      int64                // Optional: Minimum timestamp for bundle validity
	MaxTimestamp      int64                // Optional: Maximum timestamp for bundle validity
	RevertingTxHashes []string             // Optional: List of transaction hashes allowed to revert
	ReplacementUuid   string               // Optional: UUID to replace or cancel the bundle
	Builders          []string             // Optional: List of builder names to share the bundle with
}

func createBundle(transactions []*types.Transaction, blockNumber string, minTimestamp, maxTimestamp int64, revertingTxHashes []string, replacementUuid string, builders []string) *Bundle {
	return &Bundle{
		Transactions:      transactions,
		BlockNumber:       blockNumber,
		MinTimestamp:      minTimestamp,
		MaxTimestamp:      maxTimestamp,
		RevertingTxHashes: revertingTxHashes,
		ReplacementUuid:   replacementUuid,
		Builders:          builders,
	}
}
