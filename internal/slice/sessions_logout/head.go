package sessions_logout

import (
	"fmt"

	s2 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_finish"
)

func ProcessSessionLogout(req SessionLogoutRequest, deps Deps) error {
	cmd, err := NewSessionLogoutCommand(req)
	if err != nil {
		return fmt.Errorf("command: %w", err)
	}

	verifyInput := s2.VerifyAccessTokenInput{
		AccessTokenRaw: cmd.AccessTokenRaw(),
		PublicKey:      deps.Verifier,
		ExpectedIssuer: deps.JWT.Issuer,
		Now:            deps.Clock.Now(),
	}
	authID, err := s2.VerifyAccessToken(verifyInput)
	if err != nil {
		return fmt.Errorf("verify token: %w", err)
	}

	revInput := RevokeUserSessionsInput{
		UserID: authID.UserID(),
		Now:    deps.Clock.Now(),
	}
	if err := deps.Store.RevokeUserSessions(revInput); err != nil {
		return fmt.Errorf("revoke sessions: %w", err)
	}

	return nil
}
