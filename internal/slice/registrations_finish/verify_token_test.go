package registrations_finish

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	s1 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_start"
)

// ──────────────────────────────────────────────────────────────────────────────
// VerifyAccessToken — 6 тестов
// ──────────────────────────────────────────────────────────────────────────────

func buildValidToken(t *testing.T, priv ed25519.PrivateKey, userID UserID, issuer string, expiresAt time.Time) string {
	t.Helper()
	claims := jwt.RegisteredClaims{
		Issuer:    issuer,
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	signed, err := tok.SignedString(priv)
	require.NoError(t, err)
	return signed
}

func buildTestKeypair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	return pub, priv
}

func buildTestUserID(t *testing.T) UserID {
	t.Helper()
	userID, err := UserIDFromString("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)
	return userID
}

func TestVerifyAccessToken_Happy(t *testing.T) {
	pub, priv := buildTestKeypair(t)
	userID := buildTestUserID(t)
	jwtCfg := JWTConfig{AccessTTL: 15 * time.Minute, RefreshTTL: 24 * time.Hour, Issuer: "passkey-demo"}

	h, err := s1.NewHandle("testuser")
	require.NoError(t, err)
	user := NewUser(NewUserInput{ID: userID, Handle: h, CreatedAt: time.Now()})

	issued, err := generateTokenPair(GenerateTokenPairInput{User: user, Now: time.Now()}, priv, jwtCfg)
	require.NoError(t, err)

	input := VerifyAccessTokenInput{
		AccessTokenRaw: issued.Access.Value(),
		PublicKey:      pub,
		ExpectedIssuer: "passkey-demo",
		Now:            time.Now(),
	}
	authID, err := VerifyAccessToken(input)
	require.NoError(t, err)
	assert.Equal(t, userID.String(), authID.UserID().String())
}

func TestVerifyAccessToken_MalformedToken(t *testing.T) {
	pub, _ := buildTestKeypair(t)
	input := VerifyAccessTokenInput{
		AccessTokenRaw: "not.a.jwt",
		PublicKey:      pub,
		ExpectedIssuer: "passkey-demo",
		Now:            time.Now(),
	}
	_, err := VerifyAccessToken(input)
	assert.ErrorIs(t, err, ErrAccessTokenInvalid)
}

func TestVerifyAccessToken_SignatureMismatch(t *testing.T) {
	pub1, _ := buildTestKeypair(t)
	_, priv2 := buildTestKeypair(t)
	userID := buildTestUserID(t)

	// Подписываем ключом 2, верифицируем ключом 1.
	raw := buildValidToken(t, priv2, userID, "passkey-demo", time.Now().Add(15*time.Minute))
	input := VerifyAccessTokenInput{
		AccessTokenRaw: raw,
		PublicKey:      pub1,
		ExpectedIssuer: "passkey-demo",
		Now:            time.Now(),
	}
	_, err := VerifyAccessToken(input)
	assert.ErrorIs(t, err, ErrAccessTokenInvalid)
}

func TestVerifyAccessToken_Expired(t *testing.T) {
	pub, priv := buildTestKeypair(t)
	userID := buildTestUserID(t)
	past := time.Now().Add(-1 * time.Hour)
	raw := buildValidToken(t, priv, userID, "passkey-demo", past)

	input := VerifyAccessTokenInput{
		AccessTokenRaw: raw,
		PublicKey:      pub,
		ExpectedIssuer: "passkey-demo",
		Now:            time.Now(),
	}
	_, err := VerifyAccessToken(input)
	assert.ErrorIs(t, err, ErrAccessTokenInvalid)
}

func TestVerifyAccessToken_IssuerMismatch(t *testing.T) {
	pub, priv := buildTestKeypair(t)
	userID := buildTestUserID(t)
	raw := buildValidToken(t, priv, userID, "passkey-demo", time.Now().Add(15*time.Minute))

	input := VerifyAccessTokenInput{
		AccessTokenRaw: raw,
		PublicKey:      pub,
		ExpectedIssuer: "other-issuer",
		Now:            time.Now(),
	}
	_, err := VerifyAccessToken(input)
	assert.ErrorIs(t, err, ErrAccessTokenInvalid)
}

func TestVerifyAccessToken_SubjectNotUUID(t *testing.T) {
	pub, priv := buildTestKeypair(t)

	// Вручную строим JWT с Subject = "not-a-uuid".
	claims := jwt.RegisteredClaims{
		Issuer:    "passkey-demo",
		Subject:   "not-a-uuid",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	raw, err := tok.SignedString(priv)
	require.NoError(t, err)

	input := VerifyAccessTokenInput{
		AccessTokenRaw: raw,
		PublicKey:      pub,
		ExpectedIssuer: "passkey-demo",
		Now:            time.Now(),
	}
	_, err = VerifyAccessToken(input)
	assert.ErrorIs(t, err, ErrAccessTokenInvalid)
}
