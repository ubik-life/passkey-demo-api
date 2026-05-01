package registrations_finish

import "errors"

var (
	ErrInvalidRegID     = errors.New("regID: not a valid UUID")
	ErrAttestationParse = errors.New("attestation: cannot parse")

	ErrSessionNotFound    = errors.New("session: not found")
	ErrSessionExpired     = errors.New("session: expired")
	ErrAttestationInvalid = errors.New("attestation: verification failed")

	ErrHandleTaken = errors.New("user: handle already taken")
)
