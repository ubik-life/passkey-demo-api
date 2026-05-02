package sessions_start

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	s1 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_start"
	s2 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_finish"
)

// buildRequestOptions — 1 тест

func TestBuildRequestOptions_Happy(t *testing.T) {
	userID, err := s2.UserIDFromString("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)

	challenge, err := s1.GenerateChallenge()
	require.NoError(t, err)

	session := NewLoginSession(NewLoginSessionInput{
		ID:        generateLoginSessionID(),
		UserID:    userID,
		Challenge: challenge,
		TTL:       5 * time.Minute,
		Now:       time.Now(),
	})

	cred, err := s2.CredentialFromRow(
		[]byte{1, 2, 3},
		"550e8400-e29b-41d4-a716-446655440000",
		[]byte{4, 5, 6},
		0,
		"",
		time.Now().Unix(),
	)
	require.NoError(t, err)

	rp := s1.RPConfig{Name: "Test RP", ID: "localhost"}
	opts := buildRequestOptions(BuildRequestOptionsInput{
		Session:     session,
		Credentials: []s2.Credential{cred},
	}, rp)

	assert.Equal(t, "localhost", opts.RpID)
	assert.Equal(t, "preferred", opts.UserVerification)
	assert.NotEmpty(t, opts.Challenge)
	assert.Len(t, opts.AllowCredentials, 1)
	assert.Equal(t, "public-key", opts.AllowCredentials[0].Type)
	assert.NotEmpty(t, opts.AllowCredentials[0].ID)
}

// buildResponse — 1 тест

func TestBuildResponse_Happy(t *testing.T) {
	userID, err := s2.UserIDFromString("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)

	challenge, err := s1.GenerateChallenge()
	require.NoError(t, err)

	session := NewLoginSession(NewLoginSessionInput{
		ID:        generateLoginSessionID(),
		UserID:    userID,
		Challenge: challenge,
		TTL:       5 * time.Minute,
		Now:       time.Now(),
	})

	opts := RequestOptions{Challenge: "abc", RpID: "localhost"}
	resp := buildResponse(SessionStartView{Session: session, Options: opts})

	assert.Equal(t, session.ID().String(), resp.ID)
	assert.Equal(t, "abc", resp.Options.Challenge)
}
