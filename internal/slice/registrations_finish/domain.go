package registrations_finish

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	s1 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_start"
)

// RegistrationFinishRequest — невалидированный вход из HTTP-адаптера.
type RegistrationFinishRequest struct {
	RegistrationIDRaw string
	AttestationBody   []byte
}

// RegistrationFinishCommand — валидированная команда фазы 2 регистрации.
type RegistrationFinishCommand struct {
	regID  s1.RegistrationID
	parsed ParsedAttestation
}

func NewRegistrationFinishCommand(req RegistrationFinishRequest) (RegistrationFinishCommand, error) {
	regID, err := s1.RegistrationIDFromString(req.RegistrationIDRaw)
	if err != nil {
		return RegistrationFinishCommand{}, fmt.Errorf("%w: %v", ErrInvalidRegID, err)
	}
	parsed, err := parseAttestation(req.AttestationBody)
	if err != nil {
		return RegistrationFinishCommand{}, err
	}
	return RegistrationFinishCommand{regID: regID, parsed: parsed}, nil
}

func (c RegistrationFinishCommand) RegID() s1.RegistrationID { return c.regID }
func (c RegistrationFinishCommand) Parsed() ParsedAttestation { return c.parsed }

// FreshRegistrationSession — RegistrationSession с инвариантом «не истекла».
// Применение подправила «подтип, не guard» (program-design.skill Шаг 3).
type FreshRegistrationSession struct {
	session s1.RegistrationSession
}

// NewFreshSessionInput — агрегатор для конструктора подтипа.
type NewFreshSessionInput struct {
	Session s1.RegistrationSession
	Now     time.Time
}

func NewFreshRegistrationSession(input NewFreshSessionInput) (FreshRegistrationSession, error) {
	if !input.Now.Before(input.Session.ExpiresAt()) {
		return FreshRegistrationSession{}, ErrSessionExpired
	}
	return FreshRegistrationSession{session: input.Session}, nil
}

func (f FreshRegistrationSession) ID() s1.RegistrationID  { return f.session.ID() }
func (f FreshRegistrationSession) Handle() s1.Handle       { return f.session.Handle() }
func (f FreshRegistrationSession) Challenge() s1.Challenge { return f.session.Challenge() }

// UserID — идентификатор пользователя (UUID v4).
type UserID struct {
	value uuid.UUID
}

func generateUserID() UserID {
	return UserID{value: uuid.New()}
}

func (id UserID) String() string { return id.value.String() }

// User — доменная сущность пользователя.
type User struct {
	id        UserID
	handle    s1.Handle
	createdAt time.Time
}

// NewUserInput — агрегатор для NewUser.
type NewUserInput struct {
	ID        UserID
	Handle    s1.Handle
	CreatedAt time.Time
}

func NewUser(input NewUserInput) User {
	return User{
		id:        input.ID,
		handle:    input.Handle,
		createdAt: input.CreatedAt,
	}
}

func (u User) ID() UserID         { return u.id }
func (u User) Handle() s1.Handle  { return u.handle }
func (u User) CreatedAt() time.Time { return u.createdAt }

// Credential — доменная сущность WebAuthn credential.
type Credential struct {
	credentialID []byte
	userID       UserID
	publicKey    []byte
	signCount    uint32
	transports   []string
	createdAt    time.Time
}

// NewCredentialInput — агрегатор для NewCredential.
type NewCredentialInput struct {
	User      User
	Verified  VerifiedCredential
	CreatedAt time.Time
}

func NewCredential(input NewCredentialInput) Credential {
	return Credential{
		credentialID: input.Verified.CredentialID(),
		userID:       input.User.ID(),
		publicKey:    input.Verified.PublicKey(),
		signCount:    input.Verified.SignCount(),
		transports:   input.Verified.Transports(),
		createdAt:    input.CreatedAt,
	}
}

func (c Credential) CredentialID() []byte  { return c.credentialID }
func (c Credential) UserID() UserID        { return c.userID }
func (c Credential) PublicKey() []byte     { return c.publicKey }
func (c Credential) SignCount() uint32     { return c.signCount }
func (c Credential) Transports() []string { return c.transports }

// AccessToken — подписанный JWT с метаданными.
type AccessToken struct {
	value     string
	expiresAt time.Time
}

func (t AccessToken) Value() string      { return t.value }
func (t AccessToken) ExpiresAt() time.Time { return t.expiresAt }

// IssuedRefreshToken — refresh token: plaintext клиенту, hash в БД.
type IssuedRefreshToken struct {
	plaintext string
	hash      string
	expiresAt time.Time
}

func (t IssuedRefreshToken) Plaintext() string    { return t.plaintext }
func (t IssuedRefreshToken) Hash() string         { return t.hash }
func (t IssuedRefreshToken) ExpiresAt() time.Time { return t.expiresAt }

// IssuedTokenPair — внутренняя пара токенов до сериализации.
type IssuedTokenPair struct {
	Access  AccessToken
	Refresh IssuedRefreshToken
}

// FinishRegistrationInput — агрегатор для finishRegistration.
type FinishRegistrationInput struct {
	User             User
	Credential       Credential
	RefreshTokenHash string
	RefreshExpiresAt time.Time
	RegistrationID   s1.RegistrationID
}
