package registrations_start

import "fmt"

func ProcessRegistrationStart(req RegistrationStartRequest, deps Deps) (RegistrationStartResponse, error) {
	cmd, err := NewRegistrationStartCommand(req)
	if err != nil {
		return RegistrationStartResponse{}, fmt.Errorf("command: %w", err)
	}

	challenge, err := generateChallenge()
	if err != nil {
		return RegistrationStartResponse{}, fmt.Errorf("challenge: %w", err)
	}

	id := generateRegistrationID()

	input := NewRegistrationSessionInput{
		ID:        id,
		Handle:    cmd.Handle(),
		Challenge: challenge,
		TTL:       deps.ChallengeTTL,
		Now:       deps.Clock.Now(),
	}
	session := NewRegistrationSession(input)

	if err := persistRegistrationSession(deps.DB, session); err != nil {
		return RegistrationStartResponse{}, fmt.Errorf("persist: %w", err)
	}

	options := buildCreationOptions(session, deps.RP)
	view := RegistrationStartView{Session: session, Options: options}
	return buildResponse(view), nil
}
