package registrations_finish

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/protocol/webauthncose"
	"github.com/golang-jwt/jwt/v5"
	s1 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_start"
)

// ParsedAttestation — обёртка над protocol.ParsedCredentialCreationData.
type ParsedAttestation struct {
	parsed *protocol.ParsedCredentialCreationData
}

// VerifiedCredential — credential, прошедший верификацию против challenge свежей сессии.
type VerifiedCredential struct {
	credentialID []byte
	publicKey    []byte
	signCount    uint32
	transports   []string
}

func (v VerifiedCredential) CredentialID() []byte  { return v.credentialID }
func (v VerifiedCredential) PublicKey() []byte     { return v.publicKey }
func (v VerifiedCredential) SignCount() uint32     { return v.signCount }
func (v VerifiedCredential) Transports() []string { return v.transports }

// AttestationVerification — агрегатор для verifyAttestation.
type AttestationVerification struct {
	Fresh  FreshRegistrationSession
	Parsed ParsedAttestation
}

// GenerateTokenPairInput — агрегатор для generateTokenPair.
type GenerateTokenPairInput struct {
	User User
	Now  time.Time
}

// BuildTokenPairView — агрегатор для buildResponse.
type BuildTokenPairView struct {
	Access  AccessToken
	Refresh IssuedRefreshToken
}

// TokenPair — DTO ответа POST /v1/registrations/{id}/attestation 200.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// JWTConfig — конфигурация выдачи JWT.
type JWTConfig struct {
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	Issuer     string
}

func parseAttestation(raw []byte) (ParsedAttestation, error) {
	parsed, err := protocol.ParseCredentialCreationResponseBody(strings.NewReader(string(raw)))
	if err != nil {
		return ParsedAttestation{}, fmt.Errorf("%w: %v", ErrAttestationParse, err)
	}
	return ParsedAttestation{parsed: parsed}, nil
}

var supportedCredParams = []protocol.CredentialParameter{
	{Type: protocol.PublicKeyCredentialType, Algorithm: webauthncose.AlgES256},
	{Type: protocol.PublicKeyCredentialType, Algorithm: webauthncose.AlgRS256},
	{Type: protocol.PublicKeyCredentialType, Algorithm: webauthncose.AlgEdDSA},
}

func verifyAttestation(input AttestationVerification, rp s1.RPConfig) (VerifiedCredential, error) {
	challengeBytes := input.Fresh.Challenge().Bytes()
	challengeB64 := base64.RawURLEncoding.EncodeToString(challengeBytes[:])

	_, err := input.Parsed.parsed.Verify(
		challengeB64,
		rp.ID,
		[]string{rp.Origin},
		nil,
		protocol.TopOriginAutoVerificationMode,
		false,
		false,
		true,
		nil,
		supportedCredParams,
	)
	if err != nil {
		return VerifiedCredential{}, fmt.Errorf("%w: %v", ErrAttestationInvalid, err)
	}

	attData := input.Parsed.parsed.Response.AttestationObject.AuthData.AttData
	signCount := input.Parsed.parsed.Response.AttestationObject.AuthData.Counter

	var transports []string
	for _, t := range input.Parsed.parsed.Response.Transports {
		transports = append(transports, string(t))
	}

	return VerifiedCredential{
		credentialID: attData.CredentialID,
		publicKey:    attData.CredentialPublicKey,
		signCount:    signCount,
		transports:   transports,
	}, nil
}

func generateTokenPair(input GenerateTokenPairInput, signer ed25519.PrivateKey, jwtCfg JWTConfig) (IssuedTokenPair, error) {
	claims := jwt.RegisteredClaims{
		Issuer:    jwtCfg.Issuer,
		Subject:   input.User.ID().String(),
		IssuedAt:  jwt.NewNumericDate(input.Now),
		ExpiresAt: jwt.NewNumericDate(input.Now.Add(jwtCfg.AccessTTL)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	signed, err := token.SignedString(signer)
	if err != nil {
		return IssuedTokenPair{}, fmt.Errorf("sign access token: %w", err)
	}
	access := AccessToken{
		value:     signed,
		expiresAt: input.Now.Add(jwtCfg.AccessTTL),
	}

	refreshRaw := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, refreshRaw); err != nil {
		return IssuedTokenPair{}, fmt.Errorf("generate refresh token: %w", err)
	}
	plaintext := base64.RawURLEncoding.EncodeToString(refreshRaw)
	hashBytes := sha256.Sum256([]byte(plaintext))
	refresh := IssuedRefreshToken{
		plaintext: plaintext,
		hash:      hex.EncodeToString(hashBytes[:]),
		expiresAt: input.Now.Add(jwtCfg.RefreshTTL),
	}

	return IssuedTokenPair{Access: access, Refresh: refresh}, nil
}

func buildResponse(view BuildTokenPairView) TokenPair {
	return TokenPair{
		AccessToken:  view.Access.Value(),
		RefreshToken: view.Refresh.Plaintext(),
	}
}
