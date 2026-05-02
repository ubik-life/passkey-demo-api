package registrations_finish

import "fmt"

func ProcessRegistrationFinish(req RegistrationFinishRequest, deps Deps) (TokenPair, error) {
	cmd, err := NewRegistrationFinishCommand(req)
	if err != nil {
		return TokenPair{}, fmt.Errorf("command: %w", err)
	}

	session, err := deps.Store.LoadRegistrationSession(cmd.RegID())
	if err != nil {
		return TokenPair{}, fmt.Errorf("load session: %w", err)
	}

	freshInput := NewFreshSessionInput{Session: session, Now: deps.Clock.Now()}
	fresh, err := NewFreshRegistrationSession(freshInput)
	if err != nil {
		return TokenPair{}, fmt.Errorf("fresh session: %w", err)
	}

	verifyInput := AttestationVerification{Fresh: fresh, Parsed: cmd.Parsed()}
	verified, err := verifyAttestation(verifyInput, deps.RP)
	if err != nil {
		return TokenPair{}, fmt.Errorf("verify: %w", err)
	}

	userInput := NewUserInput{
		ID:        generateUserID(),
		Handle:    fresh.Handle(),
		CreatedAt: deps.Clock.Now(),
	}
	user := NewUser(userInput)

	credInput := NewCredentialInput{User: user, Verified: verified, CreatedAt: deps.Clock.Now()}
	credential := NewCredential(credInput)

	tokenInput := GenerateTokenPairInput{User: user, Now: deps.Clock.Now()}
	issued, err := generateTokenPair(tokenInput, deps.Signer, deps.JWT)
	if err != nil {
		return TokenPair{}, fmt.Errorf("token pair: %w", err)
	}

	finishInput := FinishRegistrationInput{
		User:             user,
		Credential:       credential,
		RefreshTokenHash: issued.Refresh.Hash(),
		RefreshExpiresAt: issued.Refresh.ExpiresAt(),
		RegistrationID:   fresh.ID(),
	}
	if err := deps.Store.FinishRegistration(finishInput); err != nil {
		return TokenPair{}, fmt.Errorf("finish: %w", err)
	}

	view := BuildTokenPairView{Access: issued.Access, Refresh: issued.Refresh}
	return buildResponse(view), nil
}
