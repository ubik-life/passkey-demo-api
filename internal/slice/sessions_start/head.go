package sessions_start

import (
	"fmt"

	s1 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_start"
)

func ProcessSessionStart(req SessionStartRequest, deps Deps) (SessionStartResponse, error) {
	cmd, err := NewSessionStartCommand(req)
	if err != nil {
		return SessionStartResponse{}, fmt.Errorf("command: %w", err)
	}

	uwc, err := deps.Store.LoadUserCredentials(cmd.Handle())
	if err != nil {
		return SessionStartResponse{}, fmt.Errorf("load user: %w", err)
	}

	challenge, err := s1.GenerateChallenge()
	if err != nil {
		return SessionStartResponse{}, fmt.Errorf("challenge: %w", err)
	}

	sessionID := generateLoginSessionID()

	input := NewLoginSessionInput{
		ID:        sessionID,
		UserID:    uwc.User().ID(),
		Challenge: challenge,
		TTL:       deps.ChallengeTTL,
		Now:       deps.Clock.Now(),
	}
	session := NewLoginSession(input)

	if err := deps.Store.PersistLoginSession(session); err != nil {
		return SessionStartResponse{}, fmt.Errorf("persist: %w", err)
	}

	optInput := BuildRequestOptionsInput{
		Session:     session,
		Credentials: uwc.Credentials(),
	}
	options := buildRequestOptions(optInput, deps.RP)
	view := SessionStartView{Session: session, Options: options}
	return buildResponse(view), nil
}
