package sessions_finish

import (
	"crypto/rand"
	"strings"
	"testing"
	"time"

	vwa "github.com/descope/virtualwebauthn"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	s1 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_start"
	s2 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_finish"
	s3 "github.com/ubik-life/passkey-demo-api/internal/slice/sessions_start"
)

// ──────────────────────────────────────────────────────────────────────────────
// NewSessionFinishCommand — 3 теста
// ──────────────────────────────────────────────────────────────────────────────

func TestNewSessionFinishCommand_Happy(t *testing.T) {
	body := makeAssertionBody(t)
	cmd, err := NewSessionFinishCommand(SessionFinishRequest{
		LoginSessionIDRaw: "550e8400-e29b-41d4-a716-446655440000",
		AssertionBody:     body,
	})
	require.NoError(t, err)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", cmd.LoginSessionID().String())
	assert.NotNil(t, cmd.Parsed().parsed)
}

func TestNewSessionFinishCommand_InvalidLoginSessionID(t *testing.T) {
	body := makeAssertionBody(t)
	_, err := NewSessionFinishCommand(SessionFinishRequest{
		LoginSessionIDRaw: "not-a-uuid",
		AssertionBody:     body,
	})
	assert.ErrorIs(t, err, ErrInvalidLoginSessionID)
}

func TestNewSessionFinishCommand_InvalidAssertionBody(t *testing.T) {
	_, err := NewSessionFinishCommand(SessionFinishRequest{
		LoginSessionIDRaw: "550e8400-e29b-41d4-a716-446655440000",
		AssertionBody:     []byte(`{not valid json`),
	})
	assert.ErrorIs(t, err, ErrAssertionParse)
}

// ──────────────────────────────────────────────────────────────────────────────
// NewFreshLoginSession — 2 теста
// ──────────────────────────────────────────────────────────────────────────────

func TestNewFreshLoginSession_Happy(t *testing.T) {
	session := buildSimpleLoginSession(t)
	now := time.Now()
	fresh, err := NewFreshLoginSession(NewFreshLoginSessionInput{Session: session, Now: now})
	require.NoError(t, err)
	assert.Equal(t, session.ID().String(), fresh.ID().String())
}

func TestNewFreshLoginSession_Expired(t *testing.T) {
	session := buildSimpleLoginSession(t)
	future := session.ExpiresAt().Add(time.Second)
	_, err := NewFreshLoginSession(NewFreshLoginSessionInput{Session: session, Now: future})
	assert.ErrorIs(t, err, ErrLoginSessionExpired)
}

// ──────────────────────────────────────────────────────────────────────────────
// helpers
// ──────────────────────────────────────────────────────────────────────────────

func buildSimpleLoginSession(t *testing.T) s3.LoginSession {
	t.Helper()
	userID, err := s2.UserIDFromString("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)
	challenge, err := s1.GenerateChallenge()
	require.NoError(t, err)
	id, err := s3.LoginSessionIDFromString("660e8400-e29b-41d4-a716-446655440001")
	require.NoError(t, err)
	return s3.NewLoginSession(s3.NewLoginSessionInput{
		ID:        id,
		UserID:    userID,
		Challenge: challenge,
		TTL:       5 * time.Minute,
		Now:       time.Now(),
	})
}

// makeAssertionBody строит синтаксически корректный assertion-body через virtualwebauthn.
// Не используется для верификации — только для теста парсера.
func makeAssertionBody(t *testing.T) []byte {
	t.Helper()

	rp := vwa.RelyingParty{Name: "Passkey Demo", ID: "localhost", Origin: "http://localhost"}
	auth := vwa.NewAuthenticator()
	vcred := vwa.NewCredential(vwa.KeyTypeEC2)
	auth.AddCredential(vcred)

	// Достаём credentialID из attestation-ответа.
	attChallengeBytes := make([]byte, 32)
	_, err := rand.Read(attChallengeBytes)
	require.NoError(t, err)
	attRespJSON := vwa.CreateAttestationResponse(rp, auth, vcred, vwa.AttestationOptions{
		Challenge: attChallengeBytes, RelyingPartyID: rp.ID, RelyingPartyName: rp.Name,
	})
	attParsed, err := protocol.ParseCredentialCreationResponseBody(strings.NewReader(attRespJSON))
	require.NoError(t, err)
	_ = attParsed // только чтобы убедиться, что парсинг прошёл

	// Генерируем assertion с произвольным challenge.
	loginChallenge := make([]byte, 32)
	_, err = rand.Read(loginChallenge)
	require.NoError(t, err)
	respJSON := vwa.CreateAssertionResponse(rp, auth, vcred, vwa.AssertionOptions{
		Challenge: loginChallenge, RelyingPartyID: rp.ID,
	})
	return []byte(respJSON)
}
