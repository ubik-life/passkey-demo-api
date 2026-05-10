package sessions_logout

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSessionLogoutCommand_Happy(t *testing.T) {
	req := SessionLogoutRequest{AccessTokenRaw: "some.jwt.token"}
	cmd, err := NewSessionLogoutCommand(req)
	require.NoError(t, err)
	assert.Equal(t, "some.jwt.token", cmd.AccessTokenRaw())
}

func TestNewSessionLogoutCommand_EmptyToken(t *testing.T) {
	req := SessionLogoutRequest{AccessTokenRaw: ""}
	_, err := NewSessionLogoutCommand(req)
	assert.ErrorIs(t, err, ErrMissingBearer)
}
