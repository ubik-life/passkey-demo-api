package sessions_logout

import "fmt"

// SessionLogoutRequest — невалидированный вход из HTTP-адаптера.
type SessionLogoutRequest struct {
	AccessTokenRaw string
}

// SessionLogoutCommand — валидированная команда выхода.
type SessionLogoutCommand struct {
	accessTokenRaw string
}

func NewSessionLogoutCommand(req SessionLogoutRequest) (SessionLogoutCommand, error) {
	if req.AccessTokenRaw == "" {
		return SessionLogoutCommand{}, fmt.Errorf("%w", ErrMissingBearer)
	}
	return SessionLogoutCommand{accessTokenRaw: req.AccessTokenRaw}, nil
}

func (c SessionLogoutCommand) AccessTokenRaw() string { return c.accessTokenRaw }
