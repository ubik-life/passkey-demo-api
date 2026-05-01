package registrations_finish

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
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
	issued, err := generateTokenPair(GenerateTokenPairInput{User: user, Now: now}, priv, jwtCfg, rand.Reader)
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

// ──────────────────────────────────────────────────────────────────────────────
// ProcessRegistrationFinish — честный тест с in-memory SQLite (7 тестов)
// ──────────────────────────────────────────────────────────────────────────────

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	for _, stmt := range []string{
		`CREATE TABLE registration_sessions (id TEXT PRIMARY KEY, handle TEXT NOT NULL, challenge BLOB NOT NULL, expires_at INTEGER NOT NULL)`,
		`CREATE TABLE users (id TEXT PRIMARY KEY, handle TEXT NOT NULL UNIQUE, created_at INTEGER NOT NULL)`,
		`CREATE TABLE credentials (credential_id BLOB PRIMARY KEY, user_id TEXT NOT NULL, public_key BLOB NOT NULL, sign_count INTEGER NOT NULL, transports TEXT NOT NULL DEFAULT '', created_at INTEGER NOT NULL)`,
		`CREATE TABLE refresh_tokens (token_hash TEXT PRIMARY KEY, user_id TEXT NOT NULL, expires_at INTEGER NOT NULL, revoked_at INTEGER NULL)`,
	} {
		_, err := db.Exec(stmt)
		require.NoError(t, err)
	}

	t.Cleanup(func() { db.Close() })
	return db
}

func insertSession(t *testing.T, db *sql.DB, session s1.RegistrationSession) {
	t.Helper()
	ch := session.Challenge().Bytes()
	_, err := db.Exec(
		`INSERT INTO registration_sessions (id, handle, challenge, expires_at) VALUES (?, ?, ?, ?)`,
		session.ID().String(),
		session.Handle().Value(),
		ch[:],
		session.ExpiresAt().Unix(),
	)
	require.NoError(t, err)
}

type testClock struct{ now time.Time }

func (c testClock) Now() time.Time { return c.now }

func buildDeps(t *testing.T, db *sql.DB) Deps {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	return Deps{
		DB:     db,
		Clock:  testClock{time.Now()},
		Logger: nil,
		RP:     s1.RPConfig{Name: "Passkey Demo", ID: "localhost", Origin: "http://localhost"},
		JWT:    JWTConfig{AccessTTL: 15 * time.Minute, RefreshTTL: 720 * time.Hour, Issuer: "passkey-demo"},
		Signer: priv,
		Rand:   rand.Reader,
	}
}

func TestProcessRegistrationFinish_Happy(t *testing.T) {
	db := openTestDB(t)
	auth, rp, cred, session := buildVirtualSession(t, "dian")
	insertSession(t, db, session)

	body := buildAttestationBody(t, auth, rp, cred, session)

	deps := buildDeps(t, db)
	resp, err := ProcessRegistrationFinish(RegistrationFinishRequest{
		RegistrationIDRaw: session.ID().String(),
		AttestationBody:   body,
	}, deps)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
}

func TestProcessRegistrationFinish_InvalidRegID(t *testing.T) {
	db := openTestDB(t)
	deps := buildDeps(t, db)
	_, err := ProcessRegistrationFinish(RegistrationFinishRequest{
		RegistrationIDRaw: "bad-uuid",
		AttestationBody:   []byte(`{}`),
	}, deps)
	assert.ErrorIs(t, err, ErrInvalidRegID)
}

func TestProcessRegistrationFinish_InvalidAttestationBody(t *testing.T) {
	db := openTestDB(t)
	deps := buildDeps(t, db)
	_, err := ProcessRegistrationFinish(RegistrationFinishRequest{
		RegistrationIDRaw: "550e8400-e29b-41d4-a716-446655440000",
		AttestationBody:   []byte(`{invalid`),
	}, deps)
	assert.ErrorIs(t, err, ErrAttestationParse)
}

func TestProcessRegistrationFinish_SessionNotFound(t *testing.T) {
	db := openTestDB(t)
	auth, rp, cred, session := buildVirtualSession(t, "evee")
	body := buildAttestationBody(t, auth, rp, cred, session)

	deps := buildDeps(t, db)
	_, err := ProcessRegistrationFinish(RegistrationFinishRequest{
		RegistrationIDRaw: session.ID().String(),
		AttestationBody:   body,
	}, deps)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestProcessRegistrationFinish_SessionExpired(t *testing.T) {
	db := openTestDB(t)
	auth, rp, cred, session := buildVirtualSession(t, "fran")

	ch := session.Challenge().Bytes()
	_, err := db.Exec(
		`INSERT INTO registration_sessions (id, handle, challenge, expires_at) VALUES (?, ?, ?, ?)`,
		session.ID().String(), session.Handle().Value(), ch[:],
		time.Now().Add(-1*time.Minute).Unix(),
	)
	require.NoError(t, err)

	body := buildAttestationBody(t, auth, rp, cred, session)
	deps := buildDeps(t, db)
	_, err = ProcessRegistrationFinish(RegistrationFinishRequest{
		RegistrationIDRaw: session.ID().String(),
		AttestationBody:   body,
	}, deps)
	assert.ErrorIs(t, err, ErrSessionExpired)
}

func TestProcessRegistrationFinish_AttestationInvalid(t *testing.T) {
	db := openTestDB(t)
	auth, rp, _, session := buildVirtualSession(t, "grac")
	insertSession(t, db, session)

	// Build attestation for a different credential (wrong challenge binding)
	wrongCred := vwa.NewCredential(vwa.KeyTypeEC2)
	auth.AddCredential(wrongCred)
	wrongChallengeBytes := make([]byte, 32)
	wrongChallengeBytes[0] = 0xAB
	wrongOptions := vwa.AttestationOptions{
		Challenge:      wrongChallengeBytes,
		RelyingPartyID: rp.ID,
	}
	wrongBody := []byte(vwa.CreateAttestationResponse(rp, auth, wrongCred, wrongOptions))

	deps := buildDeps(t, db)
	_, err := ProcessRegistrationFinish(RegistrationFinishRequest{
		RegistrationIDRaw: session.ID().String(),
		AttestationBody:   wrongBody,
	}, deps)
	assert.True(t, isVerifyOrAttestationError(err), "expected attestation error, got: %v", err)
}

func isVerifyOrAttestationError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return bytes.Contains([]byte(msg), []byte("attestation")) ||
		bytes.Contains([]byte(msg), []byte("verify")) ||
		bytes.Contains([]byte(msg), []byte("session"))
}

func TestProcessRegistrationFinish_CatastrophicRand(t *testing.T) {
	db := openTestDB(t)
	auth, rp, cred, session := buildVirtualSession(t, "hele")
	insertSession(t, db, session)

	body := buildAttestationBody(t, auth, rp, cred, session)
	deps := buildDeps(t, db)
	deps.Rand = &alwaysFailReader{}

	_, err := ProcessRegistrationFinish(RegistrationFinishRequest{
		RegistrationIDRaw: session.ID().String(),
		AttestationBody:   body,
	}, deps)
	require.Error(t, err)
}

type alwaysFailReader struct{}

func (r *alwaysFailReader) Read(p []byte) (int, error) {
	return 0, assert.AnError
}
