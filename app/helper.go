package main

import "encoding/hex"

// Decode hex string utility
func decodeHex(s string) ([]byte, error) {
	return hex.DecodeString(s[2:]) // Trim the "0x" prefix before decoding
}
