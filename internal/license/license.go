package license

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrMalformedToken reports a malformed JWT token.
	ErrMalformedToken = errors.New("malformed token")
	// ErrInvalidSignature reports a JWT signature mismatch.
	ErrInvalidSignature = errors.New("invalid token signature")
)

// DeveloperPublicKeyHex is the embedded public verification key.
const DeveloperPublicKeyHex = "8e6ad6bbf836d42c802564bb1313e8091deab74d69d3ee821345d54b66b7e82d"

// Claims is the local license claims envelope.
type Claims struct {
	Subject string `json:"sub"`
	HWID    string `json:"hwid"`
	Expiry  int64  `json:"exp"`
}

// EmbeddedPublicKey returns the built-in developer verification key.
func EmbeddedPublicKey() (ed25519.PublicKey, error) {
	body, err := hex.DecodeString(DeveloperPublicKeyHex)
	if err != nil {
		return nil, fmt.Errorf("decode public key: %w", err)
	}
	return ed25519.PublicKey(body), nil
}

// VerifyJWT verifies an Ed25519 JWT token without network calls.
func VerifyJWT(token string, key ed25519.PublicKey) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrMalformedToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, fmt.Errorf("decode payload: %w", err)
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Claims{}, fmt.Errorf("decode signature: %w", err)
	}
	signed := []byte(parts[0] + "." + parts[1])
	if !ed25519.Verify(key, signed, signature) {
		return Claims{}, ErrInvalidSignature
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, fmt.Errorf("decode claims: %w", err)
	}
	return claims, nil
}

// EvaluateState evaluates the local license state with grace handling.
func EvaluateState(now time.Time, claims Claims, graceStartedAt *time.Time) State {
	expiresAt := time.Unix(claims.Expiry, 0).UTC()
	if claims.Expiry == 0 {
		if graceStartedAt == nil {
			return StateGrace
		}
		if now.Sub(graceStartedAt.UTC()) >= 48*time.Hour {
			return StateRiskOnly
		}
		return StateGrace
	}
	if now.Before(expiresAt) {
		return StateLicensed
	}
	if graceStartedAt == nil {
		return StateGrace
	}
	if now.Sub(graceStartedAt.UTC()) >= 48*time.Hour {
		return StateRiskOnly
	}
	return StateGrace
}

// Checksum returns a token checksum suitable for diagnostics.
func Checksum(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}
