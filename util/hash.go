package util

import (
	// nolint:gosec
	"crypto/sha1"
	"encoding/hex"
)

// Sha1 computes SHA1 hash of the data.
// nolint:unparam,gosec
func Sha1(data []byte) (string, error) {
	hasher := sha1.New()
	hasher.Write(data)

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
