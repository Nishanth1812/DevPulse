package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// Fingerprint returns a deterministic SHA-256 digest of structured inputs.
// Encoding the complete parts as one JSON array preserves type and part
// boundaries while encoding maps in a stable key order.
func Fingerprint(parts ...any) (string, error) {
	data, err := json.Marshal(parts)
	if err != nil {
		return "", fmt.Errorf("cache: fingerprint inputs: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
