package state

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

func newID(prefix string) (string, error) {
	var random [10]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("generate %s id: %w", prefix, err)
	}
	return fmt.Sprintf("%s_%x_%s", prefix, time.Now().UTC().UnixMilli(), hex.EncodeToString(random[:])), nil
}
