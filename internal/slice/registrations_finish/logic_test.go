package registrations_finish

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	vwa "github.com/descope/virtualwebauthn"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	s1 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_start"
)

// ──────────────────────────────────────────────────────────────────────────────
// Вспомогательные функции
// ──────────────────────────────────────────────────────────────────────────────

func buildVirtualSession(t *testing.T, handle string) (vwa.Authenticator, vwa.RelyingParty, vwa.Credential, s1.RegistrationSession) {
	t.Helper()

	rp := vwa.RelyingParty{Name: "Passkey Demo", ID: "localhost", Origin: "http://localhost"}
	auth := vwa.NewAuthenticator()
	cred := vwa.NewCredential(vwa.KeyTypeEC2)
	auth.AddCredential(cred)

	h, err := s1.NewHandle(handle)
	require.NoError(t, err)

	challengeBytes := make([]byte, 32)
	_, err = rand.Read(challengeBytes)
	require.NoError(t, err)

	ch, err := s1.ChallengeFromBytes(challengeBytes)
	require.NoError(t, err)

	id, err := s1.RegistrationIDFromString("550e8400-e29b-41d4-" + handle[:4] + "-446655440000"[:12])
	if err != nil {
		// fallback to a fixed UUID per handle by using a seeded id
		id, err = s1.RegistrationIDFromString("550e8400-e29b-41d4-a716-446655440000")
		require.NoError(t, err)
	}

	session := s1.NewRegistrationSession(s1.NewRegistrationSessionInput{
		ID:        id,
		Handle:    h,
		Challenge: ch,
		TTL:       5 * time.Minute,
		Now:       time.Now(),
	})

	return auth, rp, cred, session
}

func buildAttestationBody(t *testing.T, auth vwa.Authenticator, rp vwa.RelyingParty, cred vwa.Credential, session s1.RegistrationSession) []byte {
	t.Helper()

	challengeBytes := session.Challenge().Bytes()
	options := vwa.AttestationOptions{
		Challenge:        challengeBytes[:],
		RelyingPartyID:   rp.ID,
		RelyingPartyName: rp.Name,
	}
	respJSON := vwa.CreateAttestationResponse(rp, auth, cred, options)
	return []byte(respJSON)
}

// ──────────────────────────────────────────────────────────────────────────────
// parseAttestation — 4 теста
// ──────────────────────────────────────────────────────────────────────────────

func TestParseAttestation_Happy(t *testing.T) {
	auth, rp, cred, session := buildVirtualSession(t, "alice")
	body := buildAttestationBody(t, auth, rp, cred, session)
	pa, err := parseAttestation(body)
	require.NoError(t, err)
	assert.NotNil(t, pa.parsed)
}

func TestParseAttestation_InvalidJSON(t *testing.T) {
	_, err := parseAttestation([]byte(`{not valid json`))
	assert.ErrorIs(t, err, ErrAttestationParse)
}

func TestParseAttestation_MissingResponseField(t *testing.T) {
	_, err := parseAttestation([]byte(`{"id":"abc","rawId":"abc","type":"public-key"}`))
	assert.ErrorIs(t, err, ErrAttestationParse)
}

func TestParseAttestation_InvalidCBOR(t *testing.T) {
	body := `{"id":"dGVzdA","rawId":"dGVzdA","type":"public-key","response":{"clientDataJSON":"dGVzdA","attestationObject":"////invalid"}}`
	_, err := parseAttestation([]byte(body))
	assert.ErrorIs(t, err, ErrAttestationParse)
}

// ──────────────────────────────────────────────────────────────────────────────
// verifyAttestation — 2 теста
// ──────────────────────────────────────────────────────────────────────────────

func TestVerifyAttestation_Happy(t *testing.T) {
	auth, rp, cred, session := buildVirtualSession(t, "bob1")
	body := buildAttestationBody(t, auth, rp, cred, session)

	pa, err := parseAttestation(body)
	require.NoError(t, err)

	fresh, err := NewFreshRegistrationSession(NewFreshSessionInput{
		Session: session, Now: time.Now(),
	})
	require.NoError(t, err)

	rpCfg := s1.RPConfig{Name: "Passkey Demo", ID: "localhost", Origin: "http://localhost"}
	vc, err := verifyAttestation(AttestationVerification{Fresh: fresh, Parsed: pa}, rpCfg)
	require.NoError(t, err)
	assert.NotEmpty(t, vc.CredentialID())
	assert.NotEmpty(t, vc.PublicKey())
}

func TestVerifyAttestation_WrongChallenge(t *testing.T) {
	auth, rp, cred, session := buildVirtualSession(t, "char")
	body := buildAttestationBody(t, auth, rp, cred, session)

	pa, err := parseAttestation(body)
	require.NoError(t, err)

	differentChallenge := make([]byte, 32)
	differentChallenge[0] = 0xFF
	diffCh, err := s1.ChallengeFromBytes(differentChallenge)
	require.NoError(t, err)
	h, _ := s1.NewHandle("char")
	id, _ := s1.RegistrationIDFromString("550e8400-e29b-41d4-a716-446655440002")
	wrongSession := s1.NewRegistrationSession(s1.NewRegistrationSessionInput{
		ID: id, Handle: h, Challenge: diffCh,
		TTL: 5 * time.Minute, Now: time.Now(),
	})

	fresh, err := NewFreshRegistrationSession(NewFreshSessionInput{
		Session: wrongSession, Now: time.Now(),
	})
	require.NoError(t, err)

	rpCfg := s1.RPConfig{Name: "Passkey Demo", ID: "localhost", Origin: "http://localhost"}
	_, err = verifyAttestation(AttestationVerification{Fresh: fresh, Parsed: pa}, rpCfg)
	assert.ErrorIs(t, err, ErrAttestationInvalid)
}

// ──────────────────────────────────────────────────────────────────────────────
// generateTokenPair — 1 тест
// ──────────────────────────────────────────────────────────────────────────────

func TestGenerateTokenPair_Happy(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	h, _ := s1.NewHandle("alice")
	user := NewUser(NewUserInput{ID: generateUserID(), Handle: h, CreatedAt: time.Now()})

	jwtCfg := JWTConfig{
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 720 * time.Hour,
		Issuer:     "passkey-demo",
	}
	now := time.Now()
	issued, err := generateTokenPair(GenerateTokenPairInput{User: user, Now: now}, priv, jwtCfg)
	require.NoError(t, err)
	assert.NotEmpty(t, issued.Access.Value())
	assert.NotEmpty(t, issued.Refresh.Plaintext())
	assert.NotEmpty(t, issued.Refresh.Hash())
	assert.Equal(t, now.Add(jwtCfg.AccessTTL).Unix(), issued.Access.ExpiresAt().Unix())
}

// ──────────────────────────────────────────────────────────────────────────────
// buildResponse — 1 тест
// ──────────────────────────────────────────────────────────────────────────────

func TestBuildResponse_Happy(t *testing.T) {
	access := AccessToken{value: "access.tok.en", expiresAt: time.Now()}
	refresh := IssuedRefreshToken{plaintext: "myplaintext", hash: "myhash", expiresAt: time.Now()}
	tp := buildResponse(BuildTokenPairView{Access: access, Refresh: refresh})
	assert.Equal(t, "access.tok.en", tp.AccessToken)
	assert.Equal(t, "myplaintext", tp.RefreshToken)
}

type testClock struct{ now time.Time }

func (c testClock) Now() time.Time { return c.now }
