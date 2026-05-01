# Messages — passkey-demo

Каталог структур данных, которыми обмениваются модули.

## Соглашения

- **Идиома Go.** В Go идиоматический эквивалент `Result<T, Error>` — пара возвратов `(T, error)`. Везде, где в скилле для языко-независимого описания стоит `Result<T, Error>`, в Go-коде это означает функцию, возвращающую `(T, error)`.
- **Доменные структуры с инвариантами.** Поля **неэкспортируемые**. Создаются только через конструктор `NewT(...) -> (T, error)`. Невалидные данные → структура не создаётся.
- **Request/Response DTO.** Поля экспортируемые, без правил домена. Используются только на границе ингресс-адаптера (парсинг входа, сериализация выхода).

## Структуры слайса 1 — `registrations-start`

### Request DTO (ингресс-адаптер)

```go
// RegistrationStartRequest — невалидированный вход из HTTP-адаптера.
// Маппится из тела POST /v1/registrations.
type RegistrationStartRequest struct {
    Handle string `json:"handle"`
}
```

### Доменные структуры

```go
// Handle — валидированный логин пользователя.
// Создаётся только через NewHandle. Поле неэкспортируемое.
type Handle struct {
    value string
}

func NewHandle(raw string) (Handle, error)  // см. контракт в slices/01-registrations-start.md
func (h Handle) Value() string

// RegistrationStartCommand — валидированная команда фазы 1 регистрации.
// Создаётся только через NewRegistrationStartCommand.
type RegistrationStartCommand struct {
    handle Handle
}

func NewRegistrationStartCommand(req RegistrationStartRequest) (RegistrationStartCommand, error)
func (c RegistrationStartCommand) Handle() Handle

// Challenge — 32 случайных байта (требование WebAuthn Level 2 §13.4.3).
// Создаётся через generateChallenge() в логике слайса.
type Challenge struct {
    bytes [32]byte
}

func (c Challenge) Bytes() [32]byte
func (c Challenge) Base64URL() string  // base64url без padding для передачи в WebAuthn options

// RegistrationID — идентификатор регистрационной сессии (UUID v4).
type RegistrationID struct {
    value uuid.UUID
}

func (id RegistrationID) String() string
func (id RegistrationID) Bytes() []byte  // сырое представление (для base64url(user.id))

// RegistrationSession — доменная сущность фазы 1 регистрации.
// Объединяет всё, что должно быть сохранено и прочитано фазой 2.
// Поля неэкспортируемые. Создаётся только через NewRegistrationSession.
type RegistrationSession struct {
    id        RegistrationID
    handle    Handle
    challenge Challenge
    expiresAt time.Time
}

// NewRegistrationSessionInput — value-объект-агрегатор для конструктора.
// Применение правила «один data-аргумент» (Шаг 3 program-design.skill):
// у NewRegistrationSession не плоский список из пяти полей, а один
// input-объект, который собирается шагом пайпа из локальных значений.
type NewRegistrationSessionInput struct {
    ID        RegistrationID
    Handle    Handle
    Challenge Challenge
    TTL       time.Duration
    Now       time.Time
}

// NewRegistrationSession собирает доменную сущность.
// Антецедент: ID и Handle — валидные доменные значения (гарантировано
// конструкторами); Challenge — 32 байта; TTL > 0; Now — момент создания.
// Консеквент: expiresAt = Now + TTL. Падений нет — все инварианты уже
// проверены на нижних уровнях.
func NewRegistrationSession(input NewRegistrationSessionInput) RegistrationSession

func (s RegistrationSession) ID() RegistrationID
func (s RegistrationSession) Handle() Handle
func (s RegistrationSession) Challenge() Challenge
func (s RegistrationSession) ExpiresAt() time.Time
```

### Конфигурация (передаётся в логические модули как value object)

```go
// RPConfig — конфигурация Relying Party для WebAuthn-options.
// Заполняется инфраструктурным модулем из env, дальше неизменна.
type RPConfig struct {
    Name string  // RP.name (отображаемое имя)
    ID   string  // RP.id (домен, например "localhost" для dev)
}
```

### Response DTO (формируется логикой, сериализуется адаптером)

```go
// CreationOptions — подмножество PublicKeyCredentialCreationOptions
// (WebAuthn Level 2). Точно соответствует схеме OpenAPI.
type CreationOptions struct {
    RP               RPInfo            `json:"rp"`
    User             UserInfo          `json:"user"`
    Challenge        string            `json:"challenge"`        // base64url
    PubKeyCredParams []PubKeyCredParam `json:"pubKeyCredParams"`
    Timeout          int               `json:"timeout,omitempty"`
    Attestation      string            `json:"attestation"`       // "none" по умолчанию
}

type RPInfo struct {
    Name string `json:"name"`
    ID   string `json:"id,omitempty"`
}

type UserInfo struct {
    ID          string `json:"id"`           // base64url(RegistrationID.bytes)
    Name        string `json:"name"`         // handle
    DisplayName string `json:"displayName"`  // handle (для demo)
}

type PubKeyCredParam struct {
    Type string `json:"type"` // "public-key"
    Alg  int    `json:"alg"`  // -7 (ES256), -8 (EdDSA)
}

// RegistrationStartResponse — ответ POST /v1/registrations 201.
type RegistrationStartResponse struct {
    ID      string          `json:"id"`      // RegistrationID.String()
    Options CreationOptions `json:"options"`
}

// RegistrationStartView — value-агрегатор для buildResponse.
// Применение «один data-аргумент»: вместо buildResponse(s, options)
// — buildResponse(view).
type RegistrationStartView struct {
    Session RegistrationSession
    Options CreationOptions
}
```

### Ошибки

```go
// Класс ошибок валидации (антецедент конструктора).
// Маппятся ингресс-адаптером в 422 VALIDATION_ERROR.
var (
    ErrHandleEmpty       = errors.New("handle: empty")
    ErrHandleTooShort    = errors.New("handle: too short (min 3)")
    ErrHandleTooLong     = errors.New("handle: too long (max 64)")
)

// Класс ошибок инфраструктуры (режимы отказа SQLite).
// Маппятся ингресс-адаптером в 503 / 507 по карте режимов отказа.
var (
    ErrDBLocked   = errors.New("db: locked")    // SQLITE_BUSY → 503 db_locked + Retry-After
    ErrDiskFull   = errors.New("db: disk full") // SQLITE_FULL → 507 db_disk_full
)
```

Класс выбирается через `errors.Is(err, ErrXxx)` в маппинге ингресс-адаптера. I/O-модуль `persistRegistrationSession` оборачивает низкоуровневые ошибки SQLite в эти sentinel-значения через `fmt.Errorf("...: %w", ErrDBLocked)`.

## Структуры слайса 2 — `registrations-finish`

Слайс 2 импортирует из слайса 1: `Handle`, `RegistrationID`, `Challenge`, `RegistrationSession`, `ErrDBLocked`, `ErrDiskFull`. Также **аддитивно требует** в слайсе 1 один экспортированный рехидратор для чтения сессии из БД (см. ниже).

### Аддитивные расширения слайса 1 (рехидратор)

`RegistrationSession` сейчас собирается только через `NewRegistrationSession(NewRegistrationSessionInput)` (с TTL и Now). Чтобы I/O `loadRegistrationSession` слайса 2 мог восстановить сущность из строки БД (`expires_at` уже посчитан), слайс 1 экспортирует:

```go
// RegistrationSessionFromRow восстанавливает сущность из строки БД.
// Не валидирует доменные инварианты повторно — данные уже валидировались
// при записи. Возвращает ошибку только при синтаксической непригодности
// строк (UUID не парсится, challenge не 32 байта).
func RegistrationSessionFromRow(
    rowID string,
    rowHandle string,
    rowChallenge []byte,
    rowExpiresAtUnix int64,
) (RegistrationSession, error)

// ChallengeFromBytes — рехидратор Challenge.
func ChallengeFromBytes(b []byte) (Challenge, error)  // ошибка только если len != 32

// RegistrationIDFromString — рехидратор RegistrationID.
func RegistrationIDFromString(s string) (RegistrationID, error)  // ошибка только если UUID невалиден
```

`Handle` восстанавливается через существующий `NewHandle(rowHandle)` — длина уже проверена при INSERT, повторная валидация не нагрузка.

### Расширение `RPConfig` (слайс 1)

Добавляется поле `Origin` — требуется верификатором WebAuthn в слайсе 2 для проверки `clientDataJSON.origin`.

```go
type RPConfig struct {
    Name   string  // RP.name
    ID     string  // RP.id (домен)
    Origin string  // ожидаемый origin для clientDataJSON, например "http://localhost"
}
```

Env: `PASSKEY_RP_ORIGIN` (новый), дефолт `http://localhost` для dev.

### Request DTO (ингресс-адаптер слайса 2)

```go
// RegistrationFinishRequest — невалидированный вход из HTTP-адаптера.
// Маппится из path-параметра {id} и тела POST /v1/registrations/{id}/attestation.
type RegistrationFinishRequest struct {
    RegistrationIDRaw string          // path-параметр (raw UUID-строка)
    AttestationBody   []byte          // тело запроса (JSON AttestationRequest по OpenAPI)
}
```

### Доменные структуры

```go
// ParsedAttestation — типизированная обёртка над *protocol.ParsedCredentialCreationData
// из github.com/go-webauthn/webauthn/protocol. Создаётся только через parseAttestation.
type ParsedAttestation struct {
    parsed *protocol.ParsedCredentialCreationData
}

// parseAttestation парсит JSON-тело AttestationRequest через go-webauthn protocol.
// Не верифицирует — только синтаксис и базовая структура (clientDataJSON,
// attestationObject парсятся как CBOR, поля заполняются).
func parseAttestation(raw []byte) (ParsedAttestation, error)
// Failure: ErrAttestationParse (некорректный JSON, отсутствуют обязательные поля,
// CBOR не парсится).

// RegistrationFinishCommand — валидированная команда фазы 2 регистрации.
// Поля неэкспортируемые. Создаётся только через NewRegistrationFinishCommand.
type RegistrationFinishCommand struct {
    regID  RegistrationID
    parsed ParsedAttestation
}

func NewRegistrationFinishCommand(req RegistrationFinishRequest) (RegistrationFinishCommand, error)
// Делегирует: RegistrationIDFromString(req.RegistrationIDRaw), parseAttestation(req.AttestationBody).
// Failure: ErrInvalidRegID, ErrAttestationParse.

func (c RegistrationFinishCommand) RegID() RegistrationID
func (c RegistrationFinishCommand) Parsed() ParsedAttestation

// FreshRegistrationSession — RegistrationSession с инвариантом «не истекла на момент
// конструкции». Создаётся только через NewFreshRegistrationSession. Применение
// подправила «подтип, не guard» (program-design.skill Шаг 3).
type FreshRegistrationSession struct {
    session RegistrationSession
}

type NewFreshSessionInput struct {
    Session RegistrationSession
    Now     time.Time
}

// NewFreshRegistrationSession проверяет инвариант: input.Now < input.Session.ExpiresAt().
// Если истекла — ErrSessionExpired, структура не создаётся.
func NewFreshRegistrationSession(input NewFreshSessionInput) (FreshRegistrationSession, error)

func (f FreshRegistrationSession) ID() RegistrationID
func (f FreshRegistrationSession) Handle() Handle
func (f FreshRegistrationSession) Challenge() Challenge

// VerifiedCredential — credential, прошедший верификацию против challenge свежей сессии.
// Создаётся только через verifyAttestation. Поля неэкспортируемые.
type VerifiedCredential struct {
    credentialID []byte    // raw credential ID из authenticatorData
    publicKey    []byte    // публичный ключ в формате CBOR/COSE
    signCount    uint32
    transports   []string  // hints от аутентификатора, могут быть пустыми
}

func (v VerifiedCredential) CredentialID() []byte
func (v VerifiedCredential) PublicKey() []byte
func (v VerifiedCredential) SignCount() uint32
func (v VerifiedCredential) Transports() []string

// AttestationVerification — value-объект-агрегатор для verifyAttestation.
// Применение «один data-аргумент» (Шаг 3 program-design.skill).
type AttestationVerification struct {
    Fresh  FreshRegistrationSession
    Parsed ParsedAttestation
}

// verifyAttestation проверяет attestation против challenge свежей сессии.
// Использует protocol.ParsedCredentialCreationData.Verify под капотом.
func verifyAttestation(input AttestationVerification) (VerifiedCredential, error)
// deps: rpConfig (нужен ID и Origin)
// Failure: ErrAttestationInvalid (challenge mismatch, RP ID hash mismatch,
// origin mismatch, signature invalid и любые другие провалы Verify).

// UserID — идентификатор пользователя (UUID v4).
type UserID struct {
    value uuid.UUID
}

func generateUserID() UserID
func (id UserID) String() string

// User — доменная сущность пользователя. Поля неэкспортируемые.
type User struct {
    id        UserID
    handle    Handle
    createdAt time.Time
}

type NewUserInput struct {
    ID        UserID
    Handle    Handle
    CreatedAt time.Time
}

// NewUser собирает доменную сущность.
// Антецедент: ID — UUID v4 (гарантировано generateUserID); Handle — валидное доменное
// значение (приходит из FreshRegistrationSession.Handle()); CreatedAt — момент.
// Падений нет — все инварианты на нижних уровнях.
func NewUser(input NewUserInput) User

func (u User) ID() UserID
func (u User) Handle() Handle
func (u User) CreatedAt() time.Time

// Credential — доменная сущность WebAuthn credential. Поля неэкспортируемые.
type Credential struct {
    credentialID []byte
    userID       UserID
    publicKey    []byte
    signCount    uint32
    transports   []string
    createdAt    time.Time
}

type NewCredentialInput struct {
    User      User
    Verified  VerifiedCredential
    CreatedAt time.Time
}

// NewCredential собирает credential из верифицированного результата + пользователя.
// Антецедент: User — валидная сущность; Verified — успех verifyAttestation;
// CreatedAt — момент. Падений нет.
func NewCredential(input NewCredentialInput) Credential

func (c Credential) CredentialID() []byte
func (c Credential) UserID() UserID
func (c Credential) PublicKey() []byte
func (c Credential) SignCount() uint32
func (c Credential) Transports() []string

// AccessToken — подписанный JWT (Ed25519, компактная сериализация) с метаданными.
type AccessToken struct {
    value     string    // JWT compact serialization
    expiresAt time.Time
}

func (t AccessToken) Value() string
func (t AccessToken) ExpiresAt() time.Time

// IssuedRefreshToken — refresh token, выданный пользователю + хеш для БД.
// Plaintext отдаётся клиенту, hash хранится. Сравнение в S5 (logout) — по hash.
type IssuedRefreshToken struct {
    plaintext string    // base64url(32 байта crypto/rand)
    hash      string    // hex(sha256(plaintext))
    expiresAt time.Time
}

func (t IssuedRefreshToken) Plaintext() string
func (t IssuedRefreshToken) Hash() string
func (t IssuedRefreshToken) ExpiresAt() time.Time

// IssuedTokenPair — внутренняя пара (access + refresh) до сериализации в DTO.
type IssuedTokenPair struct {
    Access  AccessToken
    Refresh IssuedRefreshToken
}

type GenerateTokenPairInput struct {
    User User
    Now  time.Time
}

// generateTokenPair выдаёт пару токенов.
// deps: signer (ed25519.PrivateKey, генерится при старте процесса), jwtCfg (TTL),
//       rand io.Reader (источник случайности для refresh token; в проде — crypto/rand).
// Failure: catastrophic — ошибка чтения rand или подписи Ed25519 (теоретическая).
func generateTokenPair(input GenerateTokenPairInput) (IssuedTokenPair, error)
```

### Конфигурация JWT

```go
// JWTConfig — конфигурация выдачи JWT для слайса 2 (и далее 4/5/6).
// Заполняется инфраструктурным модулем из env, дальше неизменна.
type JWTConfig struct {
    AccessTTL  time.Duration  // PASSKEY_JWT_ACCESS_TTL,  дефолт "15m"
    RefreshTTL time.Duration  // PASSKEY_JWT_REFRESH_TTL, дефолт "720h"
    Issuer     string         // PASSKEY_JWT_ISSUER, дефолт "passkey-demo"
}
```

Ed25519 keypair (`signer ed25519.PrivateKey`) **не** в JWTConfig — генерируется при старте процесса (`crypto/ed25519.GenerateKey`), не персистится. Передаётся в слайс через `Deps.Signer`. См. `infrastructure.md`.

### Response DTO (формируется логикой, сериализуется адаптером)

```go
// TokenPair — ответ POST /v1/registrations/{id}/attestation 200.
// Точно соответствует схеме TokenPair в OpenAPI.
type TokenPair struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
}

// BuildTokenPairView — value-агрегатор для buildResponse.
// Применение «один data-аргумент»: вместо buildResponse(access, refresh)
// — buildResponse(view).
type BuildTokenPairView struct {
    Access  AccessToken
    Refresh IssuedRefreshToken
}
```

### Ошибки

```go
// Класс ошибок валидации входа фазы 2 (антецедент конструктора команды).
// Маппятся ингресс-адаптером в 422 VALIDATION_ERROR.
var (
    ErrInvalidRegID     = errors.New("regID: not a valid UUID")
    ErrAttestationParse = errors.New("attestation: cannot parse")
)

// Класс ошибок предусловий и верификации.
var (
    ErrSessionNotFound    = errors.New("session: not found")     // → 404 NOT_FOUND
    ErrSessionExpired     = errors.New("session: expired")        // → 404 NOT_FOUND
    ErrAttestationInvalid = errors.New("attestation: verification failed")  // → 422 VALIDATION_ERROR
)

// Класс ошибок инфраструктуры финального write-tx.
// ErrDBLocked / ErrDiskFull импортируются из слайса 1 (общий контракт SQLite).
var (
    ErrHandleTaken = errors.New("user: handle already taken")     // → 422 HANDLE_TAKEN (UNIQUE на users.handle)
)
```

`ErrSessionExpired` мапится в **404 NOT_FOUND** (а не в отдельный код): для клиента истёкшая сессия неотличима от несуществующей — оба требуют начать фазу 1 заново. Различение нужно только в логах (адаптер логирует разные ошибки одним и тем же 404).

`ErrHandleTaken` — race-condition: между `POST /v1/registrations` user A и `POST /v1/registrations/{id}/attestation` user A кто-то зарегистрировался под тем же handle. Маппится в `422 HANDLE_TAKEN`. Pre-check `findUserByHandle` не делаем — лишний раунд-трип не закрывает race; UNIQUE-констрейнт в любом случае единственная честная защита.

### I/O контракты

```go
// loadRegistrationSession читает запись из registration_sessions по id.
// Внутри: SELECT id, handle, challenge, expires_at FROM registration_sessions WHERE id = ?
// Если строки нет — ErrSessionNotFound. SQLite-ошибки оборачиваются в стандартный класс.
func loadRegistrationSession(id RegistrationID) (RegistrationSession, error)
// deps: *sql.DB
// Антецедент: id — валидный UUID; миграции применены.
// Консеквент:
//   Success: RegistrationSession, восстановленная через RegistrationSessionFromRow.
//   Failure: ErrSessionNotFound, ErrDBLocked, низкоуровневые SQLite (→ 500).

// finishRegistration выполняет одну транзакцию:
//   INSERT INTO users (id, handle, created_at) VALUES (...)
//   INSERT INTO credentials (credential_id, user_id, public_key, sign_count, transports, created_at) VALUES (...)
//   INSERT INTO refresh_tokens (token_hash, user_id, expires_at) VALUES (...)
//   DELETE FROM registration_sessions WHERE id = ?
type FinishRegistrationInput struct {
    User             User
    Credential       Credential
    RefreshTokenHash string
    RefreshExpiresAt time.Time
    RegistrationID   RegistrationID
}

func finishRegistration(input FinishRegistrationInput) error
// deps: *sql.DB
// Антецедент: все доменные значения валидны; миграции 0001-0004 применены.
// Консеквент:
//   Success: все 4 операции прошли; tx закоммичена.
//   Failure:
//     ErrHandleTaken — UNIQUE violation на users.handle (race);
//     ErrDBLocked — SQLITE_BUSY;
//     ErrDiskFull — SQLITE_FULL;
//     любой другой провал tx — низкоуровневая ошибка (→ 500).
// Атомарность: при любой ошибке tx откатывается, ни одна из 4 операций не остаётся применённой.
```

`finishRegistration` — единственная I/O write-операция слайса. Всё или ничего. Это критично для корректности: иначе при, например, успехе INSERT users + провале INSERT credentials получим «осиротевшего» пользователя без credential, который не сможет войти.


