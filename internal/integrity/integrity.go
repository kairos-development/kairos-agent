package integrity

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

var (
	beginMarker = []byte("KAIROS_ATTEST_BEGIN")
	endMarker   = []byte("KAIROS_ATTEST_END")
)

// HashExcludingAttestationBlock computes the binary hash excluding the attestation block.
func HashExcludingAttestationBlock(body []byte) string {
	start := indexOf(body, beginMarker)
	end := indexOf(body, endMarker)
	if start >= 0 && end > start {
		end += len(endMarker)
		trimmed := append([]byte(nil), body[:start]...)
		trimmed = append(trimmed, body[end:]...)
		body = trimmed
	}
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}

// VerifyBinary verifies the runtime self-integrity hash.
func VerifyBinary(path string, expected string) (Status, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return StatusDegraded, fmt.Errorf("read binary: %w", err)
	}
	if HashExcludingAttestationBlock(body) != expected {
		return StatusDegraded, nil
	}
	return StatusTrusted, nil
}

func indexOf(body []byte, needle []byte) int {
	for i := 0; i+len(needle) <= len(body); i++ {
		match := true
		for j := range needle {
			if body[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
