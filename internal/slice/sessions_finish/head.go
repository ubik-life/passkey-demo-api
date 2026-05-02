package sessions_finish

import (
	"fmt"

	s2 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_finish"
)

func ProcessSessionFinish(req SessionFinishRequest, deps Deps) (s2.TokenPair, error) {
	cmd, err := NewSessionFinishCommand(req)
	if err != nil {
		return s2.TokenPair{}, fmt.Errorf("command: %w", err)
	}

	session, err := deps.Store.LoadLoginSession(cmd.LoginSessionID())
	if err != nil {
		return s2.TokenPair{}, fmt.Errorf("load session: %w", err)
	}

	fresh, err := NewFreshLoginSession(NewFreshLoginSessionInput{Session: session, Now: deps.Clock.Now()})
	if err != nil {
		return s2.TokenPair{}, fmt.Errorf("fresh session: %w", err)
	}

	targetInput := LoadAssertionTargetInput{
		UserID:       fresh.UserID(),
		CredentialID: cmd.Parsed().CredentialID(),
	}
	target, err := deps.Store.LoadAssertionTarget(targetInput)
	if err != nil {
		return s2.TokenPair{}, fmt.Errorf("load target: %w", err)
	}

	verifyInput := AssertionVerification{Fresh: fresh, Parsed: cmd.Parsed(), Target: target}
	verified, err := verifyAssertion(verifyInput, deps.RP)
	if err != nil {
		return s2.TokenPair{}, fmt.Errorf("verify: %w", err)
	}

	tokenInput := s2.GenerateTokenPairInput{User: target.User(), Now: deps.Clock.Now()}
	issued, err := s2.GenerateTokenPair(tokenInput, deps.Signer, deps.JWT)
	if err != nil {
		return s2.TokenPair{}, fmt.Errorf("token pair: %w", err)
	}

	finishInput := FinishLoginInput{
		Credential:       target.Credential(),
		NewSignCount:     verified.NewSignCount(),
		RefreshTokenHash: issued.Refresh.Hash(),
		RefreshExpiresAt: issued.Refresh.ExpiresAt(),
		LoginSessionID:   fresh.ID(),
	}
	if err := deps.Store.FinishLogin(finishInput); err != nil {
		return s2.TokenPair{}, fmt.Errorf("finish login: %w", err)
	}

	view := s2.BuildTokenPairView{Access: issued.Access, Refresh: issued.Refresh}
	return s2.BuildResponse(view), nil
}
