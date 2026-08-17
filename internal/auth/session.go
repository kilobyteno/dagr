package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

const DefaultTokenBytes = 32

// NewSessionToken returns a raw opaque token and its SHA-256 hex digest for storage.
func NewSessionToken(size int) (rawToken string, tokenHash string, err error) {
	if size < 16 {
		size = DefaultTokenBytes
	}
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("generate token: %w", err)
	}
	rawToken = base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(rawToken))
	tokenHash = hex.EncodeToString(sum[:])
	return rawToken, tokenHash, nil
}

// HashToken returns the SHA-256 hex digest of a raw session token.
func HashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}
