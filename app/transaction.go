// Package main implements the sequencer
package main

import (
	"log"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func isValidTransaction(tx *types.Transaction) bool {
	// Check if the nonce is greater than 0 and the gas limit is reasonable (min 21000)
	if tx.Nonce() == 0 {
		log.Printf("Transaction has invalid nonce: %d", tx.Nonce())
		return false
	}

	if tx.Gas() < 21000 {
		log.Printf("Transaction has insufficient gas: %d", tx.Gas())
		return false
	}

	// ToDo: Add validation logic here

	return tx != nil && tx.Hash() != (common.Hash{})
}
