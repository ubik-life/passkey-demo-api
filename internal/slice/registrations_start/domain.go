package registrations_start

import (
	"encoding/base64"
	"fmt"
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

func (s RegistrationSession) ID() RegistrationID  { return s.id }
func (s RegistrationSession) Handle() Handle       { return s.handle }
func (s RegistrationSession) Challenge() Challenge { return s.challenge }
func (s RegistrationSession) ExpiresAt() time.Time { return s.expiresAt }

// ChallengeFromBytes восстанавливает Challenge из сырых байт БД.
// Возвращает ошибку, если длина не равна 32.
func ChallengeFromBytes(b []byte) (Challenge, error) {
	if len(b) != 32 {
		return Challenge{}, fmt.Errorf("challenge: expected 32 bytes, got %d", len(b))
	}
	var c Challenge
	copy(c.bytes[:], b)
	return c, nil
}

// RegistrationIDFromString восстанавливает RegistrationID из UUID-строки.
func RegistrationIDFromString(s string) (RegistrationID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return RegistrationID{}, fmt.Errorf("registrationID: %w", err)
	}
	return RegistrationID{value: id}, nil
}

// RegistrationSessionFromRow восстанавливает сущность из строки БД.
// Не валидирует доменные инварианты повторно.
func RegistrationSessionFromRow(
	rowID string,
	rowHandle string,
	rowChallenge []byte,
	rowExpiresAtUnix int64,
) (RegistrationSession, error) {
	id, err := RegistrationIDFromString(rowID)
	if err != nil {
		return RegistrationSession{}, fmt.Errorf("session from row: %w", err)
	}
	handle, err := NewHandle(rowHandle)
	if err != nil {
		return RegistrationSession{}, fmt.Errorf("session from row: %w", err)
	}
	challenge, err := ChallengeFromBytes(rowChallenge)
	if err != nil {
		return RegistrationSession{}, fmt.Errorf("session from row: %w", err)
	}
	return RegistrationSession{
		id:        id,
		handle:    handle,
		challenge: challenge,
		expiresAt: time.Unix(rowExpiresAtUnix, 0).UTC(),
	}, nil
}
