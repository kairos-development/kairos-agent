package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedPublicKey_Success(t *testing.T) {
	key, err := EmbeddedPublicKey()
	require.NoError(t, err)
	assert.NotNil(t, key)
	assert.Len(t, key, ed25519.PublicKeySize)
}

func TestVerifyJWT_Success(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"EdDSA","typ":"JWT"}`))
	claimsBody, _ := json.Marshal(Claims{Subject: "demo", HWID: "test-hwid", Expiry: time.Now().Add(time.Hour).Unix()})
	payload := base64.RawURLEncoding.EncodeToString(claimsBody)
	signed := header + "." + payload
	signature := base64.RawURLEncoding.EncodeToString(ed25519.Sign(privateKey, []byte(signed)))

	claims, err := VerifyJWT(signed+"."+signature, publicKey)
	require.NoError(t, err)
	assert.Equal(t, "demo", claims.Subject)
	assert.Equal(t, "test-hwid", claims.HWID)
}

func TestVerifyJWT_MalformedToken_TwoParts(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	_, err = VerifyJWT("header.payload", publicKey)
	assert.ErrorIs(t, err, ErrMalformedToken)
}

func TestVerifyJWT_MalformedToken_OnePart(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	_, err = VerifyJWT("header", publicKey)
	assert.ErrorIs(t, err, ErrMalformedToken)
}

func TestVerifyJWT_MalformedToken_FourParts(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	_, err = VerifyJWT("header.payload.signature.extra", publicKey)
	assert.ErrorIs(t, err, ErrMalformedToken)
}

func TestVerifyJWT_InvalidPayloadEncoding(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	_, err = VerifyJWT("header.invalid!!!.signature", publicKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode payload")
}

func TestVerifyJWT_InvalidSignatureEncoding(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"EdDSA"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"test"}`))

	_, err = VerifyJWT(header+"."+payload+".invalid!!!", publicKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode signature")
}

func TestVerifyJWT_InvalidSignature(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"EdDSA","typ":"JWT"}`))
	claimsBody, _ := json.Marshal(Claims{Subject: "demo", Expiry: time.Now().Add(time.Hour).Unix()})
	payload := base64.RawURLEncoding.EncodeToString(claimsBody)

	// Use wrong signature
	wrongSignature := base64.RawURLEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))

	_, err = VerifyJWT(header+"."+payload+"."+wrongSignature, publicKey)
	assert.ErrorIs(t, err, ErrInvalidSignature)
}

func TestVerifyJWT_InvalidClaimsJSON(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"EdDSA","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`invalid json`))
	signed := header + "." + payload
	signature := base64.RawURLEncoding.EncodeToString(ed25519.Sign(privateKey, []byte(signed)))

	_, err = VerifyJWT(signed+"."+signature, publicKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode claims")
}

func TestEvaluateState_Licensed_NotExpired(t *testing.T) {
	now := time.Now().UTC()
	claims := Claims{Expiry: now.Add(time.Hour).Unix()}

	state := EvaluateState(now, claims, nil)
	assert.Equal(t, StateLicensed, state)
}

func TestEvaluateState_Grace_NoExpiry_NoGraceStart(t *testing.T) {
	now := time.Now().UTC()
	claims := Claims{Expiry: 0}

	state := EvaluateState(now, claims, nil)
	assert.Equal(t, StateGrace, state)
}

func TestEvaluateState_Grace_NoExpiry_WithinGracePeriod(t *testing.T) {
	now := time.Now().UTC()
	claims := Claims{Expiry: 0}
	graceStart := now.Add(-24 * time.Hour)

	state := EvaluateState(now, claims, &graceStart)
	assert.Equal(t, StateGrace, state)
}

func TestEvaluateState_RiskOnly_NoExpiry_GraceExpired(t *testing.T) {
	now := time.Now().UTC()
	claims := Claims{Expiry: 0}
	graceStart := now.Add(-49 * time.Hour)

	state := EvaluateState(now, claims, &graceStart)
	assert.Equal(t, StateRiskOnly, state)
}

func TestEvaluateState_Grace_Expired_NoGraceStart(t *testing.T) {
	now := time.Now().UTC()
	claims := Claims{Expiry: now.Add(-time.Hour).Unix()}

	state := EvaluateState(now, claims, nil)
	assert.Equal(t, StateGrace, state)
}

func TestEvaluateState_Grace_Expired_WithinGracePeriod(t *testing.T) {
	now := time.Now().UTC()
	claims := Claims{Expiry: now.Add(-time.Hour).Unix()}
	graceStart := now.Add(-24 * time.Hour)

	state := EvaluateState(now, claims, &graceStart)
	assert.Equal(t, StateGrace, state)
}

func TestEvaluateState_RiskOnly_Expired_GraceExpired(t *testing.T) {
	now := time.Now().UTC()
	claims := Claims{Expiry: now.Add(-time.Hour).Unix()}
	graceStart := now.Add(-49 * time.Hour)

	state := EvaluateState(now, claims, &graceStart)
	assert.Equal(t, StateRiskOnly, state)
}

func TestEvaluateState_Grace_ExactlyAtGraceBoundary(t *testing.T) {
	now := time.Now().UTC()
	claims := Claims{Expiry: now.Add(-time.Hour).Unix()}
	graceStart := now.Add(-48 * time.Hour)

	state := EvaluateState(now, claims, &graceStart)
	assert.Equal(t, StateRiskOnly, state)
}

func TestEvaluateState_Grace_JustBeforeGraceBoundary(t *testing.T) {
	now := time.Now().UTC()
	claims := Claims{Expiry: now.Add(-time.Hour).Unix()}
	graceStart := now.Add(-48*time.Hour + time.Second)

	state := EvaluateState(now, claims, &graceStart)
	assert.Equal(t, StateGrace, state)
}

func TestChecksum_Success(t *testing.T) {
	token := "test.token.signature"
	checksum := Checksum(token)

	assert.NotEmpty(t, checksum)
	assert.Len(t, checksum, 64) // SHA256 hex = 64 chars
}

func TestChecksum_Deterministic(t *testing.T) {
	token := "test.token.signature"
	checksum1 := Checksum(token)
	checksum2 := Checksum(token)

	assert.Equal(t, checksum1, checksum2)
}

func TestChecksum_DifferentTokens(t *testing.T) {
	token1 := "test.token.signature1"
	token2 := "test.token.signature2"

	checksum1 := Checksum(token1)
	checksum2 := Checksum(token2)

	assert.NotEqual(t, checksum1, checksum2)
}

func TestChecksum_EmptyToken(t *testing.T) {
	checksum := Checksum("")
	assert.NotEmpty(t, checksum)
	assert.Len(t, checksum, 64)
}

func TestStateConstants(t *testing.T) {
	assert.Equal(t, State("demo"), StateDemo)
	assert.Equal(t, State("licensed"), StateLicensed)
	assert.Equal(t, State("grace"), StateGrace)
	assert.Equal(t, State("risk_only"), StateRiskOnly)
	assert.Equal(t, State("unlicensed"), StateUnlicensed)
}

func TestVerifyJWT_WithAllClaimsFields(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"EdDSA","typ":"JWT"}`))
	claimsBody, _ := json.Marshal(Claims{
		Subject: "test-user",
		HWID:    "test-hwid-123",
		Expiry:  time.Now().Add(24 * time.Hour).Unix(),
	})
	payload := base64.RawURLEncoding.EncodeToString(claimsBody)
	signed := header + "." + payload
	signature := base64.RawURLEncoding.EncodeToString(ed25519.Sign(privateKey, []byte(signed)))

	claims, err := VerifyJWT(signed+"."+signature, publicKey)
	require.NoError(t, err)
	assert.Equal(t, "test-user", claims.Subject)
	assert.Equal(t, "test-hwid-123", claims.HWID)
	assert.Greater(t, claims.Expiry, time.Now().Unix())
}

func TestEvaluateState_Licensed_FarFuture(t *testing.T) {
	now := time.Now().UTC()
	claims := Claims{Expiry: now.Add(365 * 24 * time.Hour).Unix()}

	state := EvaluateState(now, claims, nil)
	assert.Equal(t, StateLicensed, state)
}

func TestEvaluateState_RiskOnly_ExactlyAt48Hours(t *testing.T) {
	now := time.Now().UTC()
	claims := Claims{Expiry: 0}
	graceStart := now.Add(-48 * time.Hour)

	state := EvaluateState(now, claims, &graceStart)
	assert.Equal(t, StateRiskOnly, state)
}
