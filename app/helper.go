package main

import (
	"encoding/hex"
	"os"
)

// Decode hex string utility
func decodeHex(hexStr string) ([]byte, error) {
	if len(hexStr) > 1 && hexStr[:2] == "0x" {
		hexStr = hexStr[2:]
	}
	return hex.DecodeString(hexStr)
}

// getSecret reads a secret from a file
func getSecret(filePath string, defaultValue string) string {
	if data, err := os.ReadFile(filePath); err == nil {
		return string(data)
	}
	return defaultValue
}
