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
// parseAssertion — 4 теста
// ──────────────────────────────────────────────────────────────────────────────

func TestParseAssertion_Happy(t *testing.T) {
	vcred, _, _, auth, rp, session := buildVirtualLoginSession(t)
	body := buildAssertionBody(t, auth, rp, vcred, session)
	pa, err := parseAssertion(body)
	require.NoError(t, err)
	assert.NotNil(t, pa.parsed)
}

func TestParseAssertion_InvalidJSON(t *testing.T) {
	_, err := parseAssertion([]byte(`{not valid json`))
	assert.ErrorIs(t, err, ErrAssertionParse)
}

func TestParseAssertion_MissingResponseField(t *testing.T) {
	_, err := parseAssertion([]byte(`{"id":"abc","rawId":"abc","type":"public-key"}`))
	assert.ErrorIs(t, err, ErrAssertionParse)
}

func TestParseAssertion_InvalidAuthenticatorData(t *testing.T) {
	body := `{"id":"dGVzdA","rawId":"dGVzdA","type":"public-key","response":{"clientDataJSON":"dGVzdA","authenticatorData":"////invalid","signature":"MEQ"}}`
	_, err := parseAssertion([]byte(body))
	assert.ErrorIs(t, err, ErrAssertionParse)
}

// ──────────────────────────────────────────────────────────────────────────────
// verifyAssertion — 2 теста
// ──────────────────────────────────────────────────────────────────────────────

func TestVerifyAssertion_Happy(t *testing.T) {
	vcred, _, target, auth, rp, session := buildVirtualLoginSession(t)
	body := buildAssertionBody(t, auth, rp, vcred, session)

	pa, err := parseAssertion(body)
	require.NoError(t, err)

	fresh, err := NewFreshLoginSession(NewFreshLoginSessionInput{Session: session, Now: time.Now()})
	require.NoError(t, err)

	rpCfg := s1.RPConfig{Name: "Passkey Demo", ID: "localhost", Origin: "http://localhost"}
	verified, err := verifyAssertion(AssertionVerification{
		Fresh:  fresh,
		Parsed: pa,
		Target: target,
	}, rpCfg)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, verified.NewSignCount(), uint32(0))
}

func TestVerifyAssertion_WrongChallenge(t *testing.T) {
	vcred, _, target, auth, rp, session := buildVirtualLoginSession(t)
	body := buildAssertionBody(t, auth, rp, vcred, session)

	pa, err := parseAssertion(body)
	require.NoError(t, err)

	diffChallengeBytes := make([]byte, 32)
	diffChallengeBytes[0] = 0xFF
	diffCh, err := s1.ChallengeFromBytes(diffChallengeBytes)
	require.NoError(t, err)
	userID, err := s2.UserIDFromString("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)
	diffID, err := s3.LoginSessionIDFromString("660e8400-e29b-41d4-a716-446655440002")
	require.NoError(t, err)
	wrongSession := s3.NewLoginSession(s3.NewLoginSessionInput{
		ID: diffID, UserID: userID, Challenge: diffCh,
		TTL: 5 * time.Minute, Now: time.Now(),
	})

	fresh, err := NewFreshLoginSession(NewFreshLoginSessionInput{Session: wrongSession, Now: time.Now()})
	require.NoError(t, err)

	rpCfg := s1.RPConfig{Name: "Passkey Demo", ID: "localhost", Origin: "http://localhost"}
	_, err = verifyAssertion(AssertionVerification{
		Fresh:  fresh,
		Parsed: pa,
		Target: target,
	}, rpCfg)
	assert.ErrorIs(t, err, ErrAssertionInvalid)
}

// ──────────────────────────────────────────────────────────────────────────────
// helpers
// ──────────────────────────────────────────────────────────────────────────────

// buildVirtualLoginSession создаёт virtualwebauthn аутентификатор, строит s2.Credential
// из attestation-ответа (чтобы получить COSE public key), и собирает login-сессию.
func buildVirtualLoginSession(t *testing.T) (
	vwa.Credential,
	s2.Credential,
	AssertionTarget,
	vwa.Authenticator,
	vwa.RelyingParty,
	s3.LoginSession,
) {
	t.Helper()

	rp := vwa.RelyingParty{Name: "Passkey Demo", ID: "localhost", Origin: "http://localhost"}
	auth := vwa.NewAuthenticator()
	vcred := vwa.NewCredential(vwa.KeyTypeEC2)
	auth.AddCredential(vcred)

	// Делаем attestation, чтобы извлечь COSE public key из authenticatorData.
	attChallengeBytes := make([]byte, 32)
	_, err := rand.Read(attChallengeBytes)
	require.NoError(t, err)

	attOptions := vwa.AttestationOptions{
		Challenge:        attChallengeBytes,
		RelyingPartyID:   rp.ID,
		RelyingPartyName: rp.Name,
	}
	attRespJSON := vwa.CreateAttestationResponse(rp, auth, vcred, attOptions)

	// Парсим через protocol напрямую, чтобы достать credentialID и publicKey.
	attParsed, err := protocol.ParseCredentialCreationResponseBody(strings.NewReader(attRespJSON))
	require.NoError(t, err)

	attData := attParsed.Response.AttestationObject.AuthData.AttData
	credentialID := attData.CredentialID
	publicKey := attData.CredentialPublicKey

	userID, err := s2.UserIDFromString("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)
	credential, err := s2.CredentialFromRow(credentialID, userID.String(), publicKey, 0, "", time.Now().Unix())
	require.NoError(t, err)

	h, err := s1.NewHandle("testuser")
	require.NoError(t, err)
	user := s2.NewUser(s2.NewUserInput{ID: userID, Handle: h, CreatedAt: time.Now()})
	target := AssertionTarget{user: user, credential: credential}

	loginChallengeBytes := make([]byte, 32)
	_, err = rand.Read(loginChallengeBytes)
	require.NoError(t, err)
	loginCh, err := s1.ChallengeFromBytes(loginChallengeBytes)
	require.NoError(t, err)
	loginID, err := s3.LoginSessionIDFromString("660e8400-e29b-41d4-a716-446655440001")
	require.NoError(t, err)
	loginSession := s3.NewLoginSession(s3.NewLoginSessionInput{
		ID: loginID, UserID: userID, Challenge: loginCh,
		TTL: 5 * time.Minute, Now: time.Now(),
	})

	return vcred, credential, target, auth, rp, loginSession
}

func buildAssertionBody(
	t *testing.T,
	auth vwa.Authenticator,
	rp vwa.RelyingParty,
	cred vwa.Credential,
	session s3.LoginSession,
) []byte {
	t.Helper()
	challengeBytes := session.Challenge().Bytes()
	options := vwa.AssertionOptions{
		Challenge:      challengeBytes[:],
		RelyingPartyID: rp.ID,
	}
	respJSON := vwa.CreateAssertionResponse(rp, auth, cred, options)
	return []byte(respJSON)
}
