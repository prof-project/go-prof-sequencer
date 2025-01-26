// Package main implements the sequencer
package main

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
)

func isValidTransaction(tx *types.Transaction) bool {
	// Check if the gas limit is reasonable (min 21000)
	if tx.Gas() < 21000 {
		log.Error().Uint64("gas", tx.Gas()).Msg("Transaction has insufficient gas")
		return false
	}

	// ToDo: Add validation logic here

	return tx != nil && tx.Hash() != (common.Hash{})
}
