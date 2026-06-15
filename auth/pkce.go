// Package auth provides OAuth PKCE authentication for OpenRouter.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// GenerateCodeVerifier generates a random code verifier for PKCE.
// The verifier is between 43 and 128 characters (we use 64).
func GenerateCodeVerifier() (string, error) {
	// Generate 48 random bytes (will be 64 chars when base64 encoded)
	b := make([]byte, 48)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64URLEncode(b), nil
}

// GenerateCodeChallenge generates a code challenge from the verifier using S256 method.
func GenerateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64URLEncode(h[:])
}

// base64URLEncode encodes bytes to URL-safe base64 without padding.
func base64URLEncode(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}
