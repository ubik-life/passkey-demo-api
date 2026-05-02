package sessions_finish

import (
	"fmt"
	"time"

	s1 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_start"
	s2 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_finish"
	s3 "github.com/ubik-life/passkey-demo-api/internal/slice/sessions_start"
)

// SessionFinishRequest — невалидированный вход из HTTP-адаптера.
type SessionFinishRequest struct {
	LoginSessionIDRaw string
	AssertionBody     []byte
}

// SessionFinishCommand — валидированная команда фазы 2 входа.
type SessionFinishCommand struct {
	loginSessionID s3.LoginSessionID
	parsed         ParsedAssertion
}

// NewSessionFinishCommand собирает команду из DTO.
// Делегирует LoginSessionIDFromString (S3) и parseAssertion.
func NewSessionFinishCommand(req SessionFinishRequest) (SessionFinishCommand, error) {
	id, err := s3.LoginSessionIDFromString(req.LoginSessionIDRaw)
	if err != nil {
		return SessionFinishCommand{}, fmt.Errorf("%w: %v", ErrInvalidLoginSessionID, err)
	}
	parsed, err := parseAssertion(req.AssertionBody)
	if err != nil {
		return SessionFinishCommand{}, err
	}
	return SessionFinishCommand{loginSessionID: id, parsed: parsed}, nil
}

func (c SessionFinishCommand) LoginSessionID() s3.LoginSessionID { return c.loginSessionID }
func (c SessionFinishCommand) Parsed() ParsedAssertion           { return c.parsed }

// FreshLoginSession — LoginSession с инвариантом «не истекла».
// Применение подправила «подтип, не guard» (program-design.skill Шаг 3).
type FreshLoginSession struct {
	session s3.LoginSession
}

// NewFreshLoginSessionInput — агрегатор для конструктора подтипа.
type NewFreshLoginSessionInput struct {
	Session s3.LoginSession
	Now     time.Time
}

// NewFreshLoginSession проверяет инвариант: input.Now < input.Session.ExpiresAt().
func NewFreshLoginSession(input NewFreshLoginSessionInput) (FreshLoginSession, error) {
	if !input.Now.Before(input.Session.ExpiresAt()) {
		return FreshLoginSession{}, ErrLoginSessionExpired
	}
	return FreshLoginSession{session: input.Session}, nil
}

func (f FreshLoginSession) ID() s3.LoginSessionID { return f.session.ID() }
func (f FreshLoginSession) UserID() s2.UserID      { return f.session.UserID() }
func (f FreshLoginSession) Challenge() s1.Challenge { return f.session.Challenge() }

// AssertionTarget — агрегат «пользователь + его credential».
// Инвариант: credential.UserID() == user.ID() — закодирован в существовании структуры.
// Создаётся только методом Store.LoadAssertionTarget.
type AssertionTarget struct {
	user       s2.User
	credential s2.Credential
}

func (t AssertionTarget) User() s2.User           { return t.user }
func (t AssertionTarget) Credential() s2.Credential { return t.credential }

// LoadAssertionTargetInput — агрегатор для Store.LoadAssertionTarget.
type LoadAssertionTargetInput struct {
	UserID       s2.UserID
	CredentialID []byte
}

// VerifiedAssertion — assertion, прошедший верификацию.
// Создаётся только через verifyAssertion.
type VerifiedAssertion struct {
	newSignCount uint32
}

func (v VerifiedAssertion) NewSignCount() uint32 { return v.newSignCount }

// AssertionVerification — агрегатор для verifyAssertion.
type AssertionVerification struct {
	Fresh  FreshLoginSession
	Parsed ParsedAssertion
	Target AssertionTarget
}

// FinishLoginInput — агрегатор для Store.FinishLogin.
type FinishLoginInput struct {
	Credential       s2.Credential
	NewSignCount     uint32
	RefreshTokenHash string
	RefreshExpiresAt time.Time
	LoginSessionID   s3.LoginSessionID
}
