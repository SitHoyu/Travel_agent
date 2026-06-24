package utils

import (
	"crypto/rand"
	"encoding/hex"
)

func NewID() string {
	buf := make([]byte, 8)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}
