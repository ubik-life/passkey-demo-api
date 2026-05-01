package registrations_start

import (
	"encoding/base64"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Handle — валидированный логин пользователя.
type Handle struct {
	value string
}

func NewHandle(raw string) (Handle, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return Handle{}, ErrHandleEmpty
	}
	if len(trimmed) < 3 {
		return Handle{}, ErrHandleTooShort
	}
	if len(trimmed) > 64 {
		return Handle{}, ErrHandleTooLong
	}
	return Handle{value: trimmed}, nil
}

func (h Handle) Value() string { return h.value }

// RegistrationStartCommand — валидированная команда фазы 1 регистрации.
type RegistrationStartCommand struct {
	handle Handle
}

func NewRegistrationStartCommand(req RegistrationStartRequest) (RegistrationStartCommand, error) {
	h, err := NewHandle(req.Handle)
	if err != nil {
		return RegistrationStartCommand{}, err
	}
	return RegistrationStartCommand{handle: h}, nil
}

func (c RegistrationStartCommand) Handle() Handle { return c.handle }

// Challenge — 32 случайных байта (WebAuthn Level 2 §13.4.3).
type Challenge struct {
	bytes [32]byte
}

func (c Challenge) Bytes() [32]byte { return c.bytes }

func (c Challenge) Base64URL() string {
	return base64.RawURLEncoding.EncodeToString(c.bytes[:])
}

// RegistrationID — идентификатор регистрационной сессии (UUID v4).
type RegistrationID struct {
	value uuid.UUID
}

func (id RegistrationID) String() string { return id.value.String() }

func (id RegistrationID) Bytes() []byte {
	b := id.value
	return b[:]
}

// NewRegistrationSessionInput — value-агрегатор для NewRegistrationSession.
type NewRegistrationSessionInput struct {
	ID        RegistrationID
	Handle    Handle
	Challenge Challenge
	TTL       time.Duration
	Now       time.Time
}

// RegistrationSession — доменная сущность фазы 1 регистрации.
type RegistrationSession struct {
	id        RegistrationID
	handle    Handle
	challenge Challenge
	expiresAt time.Time
}

func NewRegistrationSession(input NewRegistrationSessionInput) RegistrationSession {
	return RegistrationSession{
		id:        input.ID,
		handle:    input.Handle,
		challenge: input.Challenge,
		expiresAt: input.Now.Add(input.TTL),
	}
}

func (s RegistrationSession) ID() RegistrationID { return s.id }
func (s RegistrationSession) Handle() Handle      { return s.handle }
func (s RegistrationSession) Challenge() Challenge { return s.challenge }
func (s RegistrationSession) ExpiresAt() time.Time { return s.expiresAt }
