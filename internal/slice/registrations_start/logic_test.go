package registrations_start

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateChallenge_Happy(t *testing.T) {
	c, err := generateChallenge()
	require.NoError(t, err)
	assert.NotEmpty(t, c.Base64URL())
	assert.Len(t, c.Bytes(), 32)
}

func TestGenerateRegistrationID_Happy(t *testing.T) {
	id := generateRegistrationID()
	assert.NotEmpty(t, id.String())
	assert.Len(t, id.Bytes(), 16)
}

func TestBuildCreationOptions_Happy(t *testing.T) {
	s := makeTestSession(t)
	rp := RPConfig{Name: "Test RP", ID: "localhost"}

	opts := buildCreationOptions(s, rp)

	assert.Equal(t, "Test RP", opts.RP.Name)
	assert.Equal(t, "localhost", opts.RP.ID)
	assert.NotEmpty(t, opts.User.ID)
	assert.Equal(t, "alice", opts.User.Name)
	assert.Equal(t, "alice", opts.User.DisplayName)
	assert.NotEmpty(t, opts.Challenge)
	assert.Equal(t, "none", opts.Attestation)
	assert.Len(t, opts.PubKeyCredParams, 2)
}

func TestBuildResponse_Happy(t *testing.T) {
	s := makeTestSession(t)
	opts := buildCreationOptions(s, RPConfig{Name: "Test RP", ID: "localhost"})

	resp := buildResponse(RegistrationStartView{Session: s, Options: opts})

	assert.Equal(t, s.ID().String(), resp.ID)
	assert.Equal(t, opts.Challenge, resp.Options.Challenge)
}

func makeTestSession(t *testing.T) RegistrationSession {
	t.Helper()
	id := generateRegistrationID()
	h, err := NewHandle("alice")
	require.NoError(t, err)
	ch, err := generateChallenge()
	require.NoError(t, err)
	return NewRegistrationSession(NewRegistrationSessionInput{
		ID:        id,
		Handle:    h,
		Challenge: ch,
		TTL:       5 * time.Minute,
		Now:       time.Now(),
	})
}
