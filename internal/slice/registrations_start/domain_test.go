package registrations_start

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NewHandle — 4 теста

func TestNewHandle_Happy(t *testing.T) {
	h, err := NewHandle("alice")
	require.NoError(t, err)
	assert.Equal(t, "alice", h.Value())
}

func TestNewHandle_Empty(t *testing.T) {
	_, err := NewHandle("   ")
	assert.ErrorIs(t, err, ErrHandleEmpty)
}

func TestNewHandle_TooShort(t *testing.T) {
	_, err := NewHandle("ab")
	assert.ErrorIs(t, err, ErrHandleTooShort)
}

func TestNewHandle_TooLong(t *testing.T) {
	_, err := NewHandle(string(make([]byte, 65)))
	assert.ErrorIs(t, err, ErrHandleTooLong)
}

// NewRegistrationStartCommand — 2 теста

func TestNewRegistrationStartCommand_Happy(t *testing.T) {
	cmd, err := NewRegistrationStartCommand(RegistrationStartRequest{Handle: "alice"})
	require.NoError(t, err)
	assert.Equal(t, "alice", cmd.Handle().Value())
}

func TestNewRegistrationStartCommand_InvalidHandle(t *testing.T) {
	_, err := NewRegistrationStartCommand(RegistrationStartRequest{Handle: "ab"})
	assert.ErrorIs(t, err, ErrHandleTooShort)
}

// NewRegistrationSession — 1 тест

func TestNewRegistrationSession_Happy(t *testing.T) {
	id := generateRegistrationID()
	h, _ := NewHandle("alice")
	ch, _ := generateChallenge()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	s := NewRegistrationSession(NewRegistrationSessionInput{
		ID:        id,
		Handle:    h,
		Challenge: ch,
		TTL:       5 * time.Minute,
		Now:       now,
	})

	assert.Equal(t, id.String(), s.ID().String())
	assert.Equal(t, "alice", s.Handle().Value())
	assert.Equal(t, now.Add(5*time.Minute), s.ExpiresAt())
}
