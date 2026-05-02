package sessions_start

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	s1 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_start"
	s2 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_finish"
)

// NewSessionStartCommand — 2 теста

func TestNewSessionStartCommand_Happy(t *testing.T) {
	cmd, err := NewSessionStartCommand(SessionStartRequest{Handle: "alice"})
	require.NoError(t, err)
	assert.Equal(t, "alice", cmd.Handle().Value())
}

func TestNewSessionStartCommand_InvalidHandle(t *testing.T) {
	_, err := NewSessionStartCommand(SessionStartRequest{Handle: "ab"})
	assert.ErrorIs(t, err, s1.ErrHandleTooShort)
}

// generateLoginSessionID — 1 тест

func TestGenerateLoginSessionID_Happy(t *testing.T) {
	id := generateLoginSessionID()
	assert.NotEmpty(t, id.String())
	assert.Len(t, id.Bytes(), 16)
}

// NewLoginSession — 1 тест

func TestNewLoginSession_Happy(t *testing.T) {
	userID, err := s2.UserIDFromString("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)

	challenge, err := s1.GenerateChallenge()
	require.NoError(t, err)

	now := time.Now()
	ttl := 5 * time.Minute
	session := NewLoginSession(NewLoginSessionInput{
		ID:        generateLoginSessionID(),
		UserID:    userID,
		Challenge: challenge,
		TTL:       ttl,
		Now:       now,
	})

	assert.NotEmpty(t, session.ID().String())
	assert.Equal(t, userID.String(), session.UserID().String())
	assert.Equal(t, now.Add(ttl).Unix(), session.ExpiresAt().Unix())
}
