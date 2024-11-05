package main

import "encoding/hex"

// Decode hex string utility
func decodeHex(hexStr string) ([]byte, error) {
	if len(hexStr) > 1 && hexStr[:2] == "0x" {
		hexStr = hexStr[2:]
	}
	return hex.DecodeString(hexStr)
}
