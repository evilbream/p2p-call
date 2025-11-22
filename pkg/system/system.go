package system

import (
	"crypto/rand"
	"encoding/hex"
)

func GenerateSessionID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
