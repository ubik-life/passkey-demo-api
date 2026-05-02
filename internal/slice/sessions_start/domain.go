package sessions_start

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	s1 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_start"
	s2 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_finish"
)

// LoginSessionID — идентификатор сессии входа (UUID v4).
// Отдельный тип от RegistrationID: разные жизненные циклы, разные таблицы.
type LoginSessionID struct {
	value uuid.UUID
}

func generateLoginSessionID() LoginSessionID {
	return LoginSessionID{value: uuid.New()}
}

func (id LoginSessionID) String() string { return id.value.String() }
func (id LoginSessionID) Bytes() []byte  { b := id.value; return b[:] }

// SessionStartCommand — валидированная команда фазы 1 входа.
type SessionStartCommand struct {
	handle s1.Handle
}

// NewSessionStartCommand — делегирует валидацию NewHandle из S1.
func NewSessionStartCommand(req SessionStartRequest) (SessionStartCommand, error) {
	h, err := s1.NewHandle(req.Handle)
	if err != nil {
		return SessionStartCommand{}, fmt.Errorf("handle: %w", err)
	}
	return SessionStartCommand{handle: h}, nil
}

func (c SessionStartCommand) Handle() s1.Handle { return c.handle }

// UserWithCredentials — агрегат «пользователь + его непустой список credentials».
// Инвариант: len(credentials) >= 1 — закодирован в самом существовании структуры.
// Если в БД нет user или нет credentials — I/O возвращает ErrUserNotFound.
type UserWithCredentials struct {
	user        s2.User
	credentials []s2.Credential
}

func (u UserWithCredentials) User() s2.User            { return u.user }
func (u UserWithCredentials) Credentials() []s2.Credential { return u.credentials }

// LoginSession — доменная сущность сессии входа.
type LoginSession struct {
	id        LoginSessionID
	userID    s2.UserID
	challenge s1.Challenge
	expiresAt time.Time
}

// NewLoginSessionInput — value-агрегатор для конструктора («один data-аргумент»).
type NewLoginSessionInput struct {
	ID        LoginSessionID
	UserID    s2.UserID
	Challenge s1.Challenge
	TTL       time.Duration
	Now       time.Time
}

// NewLoginSession собирает доменную сущность. Падений нет — антецеденты
// проверены предыдущими конструкторами; expiresAt = Now + TTL.
func NewLoginSession(input NewLoginSessionInput) LoginSession {
	return LoginSession{
		id:        input.ID,
		userID:    input.UserID,
		challenge: input.Challenge,
		expiresAt: input.Now.Add(input.TTL),
	}
}

func (s LoginSession) ID() LoginSessionID      { return s.id }
func (s LoginSession) UserID() s2.UserID       { return s.userID }
func (s LoginSession) Challenge() s1.Challenge { return s.challenge }
func (s LoginSession) ExpiresAt() time.Time    { return s.expiresAt }

// LoginSessionIDFromString — рехидратор LoginSessionID для path-параметра S4.
func LoginSessionIDFromString(raw string) (LoginSessionID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return LoginSessionID{}, fmt.Errorf("loginSessionID: %w", err)
	}
	return LoginSessionID{value: id}, nil
}

// LoginSessionFromRow восстанавливает сущность из строки login_sessions (для S4).
func LoginSessionFromRow(rowID, rowUserID string, rowChallenge []byte, rowExpiresAtUnix int64) (LoginSession, error) {
	id, err := LoginSessionIDFromString(rowID)
	if err != nil {
		return LoginSession{}, fmt.Errorf("login session from row: %w", err)
	}
	userID, err := s2.UserIDFromString(rowUserID)
	if err != nil {
		return LoginSession{}, fmt.Errorf("login session from row: %w", err)
	}
	challenge, err := s1.ChallengeFromBytes(rowChallenge)
	if err != nil {
		return LoginSession{}, fmt.Errorf("login session from row: %w", err)
	}
	return LoginSession{
		id:        id,
		userID:    userID,
		challenge: challenge,
		expiresAt: time.Unix(rowExpiresAtUnix, 0).UTC(),
	}, nil
}
