// Package jobs provides job run tracking for generation operations.
package jobs

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// GenerateRunID generates a unique run ID.
// Format: YYYYMMDD-HHMM-{8 char random hex}
// Example: 20260415-1430-a7f9c2d1
func GenerateRunID() string {
	now := time.Now()
	timestamp := now.Format("20060102-1504")

	randBytes := make([]byte, 4)
	rand.Read(randBytes) //nolint:errcheck — crypto/rand.Read never fails on supported platforms
	randHex := hex.EncodeToString(randBytes)

	return fmt.Sprintf("%s-%s", timestamp, randHex)
}
