package sessions_finish

import "errors"

var (
	ErrInvalidLoginSessionID = errors.New("loginSessionID: not a valid UUID")
	ErrAssertionParse        = errors.New("assertion: cannot parse")
	ErrLoginSessionNotFound  = errors.New("login session: not found")
	ErrLoginSessionExpired   = errors.New("login session: expired")
	ErrCredentialNotFound    = errors.New("credential: not found")
	ErrAssertionInvalid      = errors.New("assertion: verification failed")
)
