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

Класс выбирается через `errors.Is(err, ErrXxx)` в маппинге ингресс-адаптера. Метод `Store.PersistRegistrationSession` оборачивает низкоуровневые ошибки SQLite в эти sentinel-значения через `fmt.Errorf("...: %w", ErrDBLocked)`.

### I/O-объект слайса 1 — `Store`

Применение **правила автономного IO-объекта** (Шаг 6 скилла + `feedback_io_autonomous_store`): зависимость `*sql.DB` инкапсулирована в объекте `Store`, головной модуль её не видит. В `Deps` слайса 1 — поле `Store *Store`, не сырой `*sql.DB`.

```go
// Store — автономный I/O-объект слайса 1, инкапсулирующий *sql.DB.
// Поле db неэкспортируемое — головной модуль обращается к БД исключительно через методы.
type Store struct {
    db *sql.DB
}

func NewStore(db *sql.DB) *Store

// PersistRegistrationSession — write-метод. Контракт: см. slices/01-registrations-start.md.
//
// Внутри: INSERT INTO registration_sessions (id, handle, challenge, expires_at) VALUES (?, ?, ?, ?).
// Маппинг: SQLITE_BUSY → ErrDBLocked, SQLITE_FULL → ErrDiskFull.
func (s *Store) PersistRegistrationSession(rs RegistrationSession) error
```

> **Техдолг кода S1.** В реализации (`internal/slice/registrations_start/`, PR #17) `*sql.DB` лежит сырым в `Deps`, `persistRegistrationSession` — пакетная функция. Карточка дизайна отражает целевое состояние; рефакторинг кода — отдельный тикет в `backlog.md` (root).

## Структуры слайса 2 — `registrations-finish`

Слайс 2 импортирует из слайса 1: `Handle`, `RegistrationID`, `Challenge`, `RegistrationSession`, `ErrDBLocked`, `ErrDiskFull`. Также **аддитивно требует** в слайсе 1 один экспортированный рехидратор для чтения сессии из БД (см. ниже).

### Аддитивные расширения слайса 1 (рехидратор)

`RegistrationSession` сейчас собирается только через `NewRegistrationSession(NewRegistrationSessionInput)` (с TTL и Now). Чтобы метод `Store.LoadRegistrationSession` слайса 2 мог восстановить сущность из строки БД (`expires_at` уже посчитан), слайс 1 экспортирует:

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

### I/O-объект слайса 2 — `Store`

Применение **правила автономного IO-объекта** (Шаг 6 скилла + `feedback_io_autonomous_store`): зависимость `*sql.DB` инкапсулирована в объекте `Store`, головной модуль её не видит. В `Deps` слайса 2 — поле `Store *Store`, не сырой `*sql.DB`.

```go
// Store — автономный I/O-объект слайса 2, инкапсулирующий *sql.DB.
// Поле db неэкспортируемое — головной модуль обращается к БД исключительно через методы.
type Store struct {
    db *sql.DB
}

func NewStore(db *sql.DB) *Store

// LoadRegistrationSession — read-метод. Контракт: см. slices/02-registrations-finish.md.
//
// Внутри: SELECT id, handle, challenge, expires_at FROM registration_sessions WHERE id = ?
// Если строки нет — ErrSessionNotFound.
// Маппинг: SQLITE_BUSY → ErrDBLocked. ErrDiskFull для read не различается.
// Рехидрирует через RegistrationSessionFromRow (S1).
func (s *Store) LoadRegistrationSession(id RegistrationID) (RegistrationSession, error)

// FinishRegistration — write-метод (атомарная транзакция, 4 операции).
// Контракт: см. slices/02-registrations-finish.md.
//
// Внутри одной транзакции:
//   INSERT INTO users (id, handle, created_at) VALUES (...)
//   INSERT INTO credentials (credential_id, user_id, public_key, sign_count, transports, created_at) VALUES (...)
//   INSERT INTO refresh_tokens (token_hash, user_id, expires_at) VALUES (...)
//   DELETE FROM registration_sessions WHERE id = ?
// При любой ошибке tx откатывается — ни одна из 4 операций не остаётся применённой.
//
// Маппинг: SQLITE_CONSTRAINT_UNIQUE на users.handle → ErrHandleTaken (race-condition);
//          SQLITE_BUSY → ErrDBLocked; SQLITE_FULL → ErrDiskFull.
type FinishRegistrationInput struct {
    User             User
    Credential       Credential
    RefreshTokenHash string
    RefreshExpiresAt time.Time
    RegistrationID   RegistrationID
}

func (s *Store) FinishRegistration(input FinishRegistrationInput) error
```

`Store.FinishRegistration` — единственная I/O write-операция слайса. Всё или ничего. Это критично для корректности: иначе при, например, успехе INSERT users + провале INSERT credentials получим «осиротевшего» пользователя без credential, который не сможет войти.

> **Техдолг кода S2.** В реализации (`internal/slice/registrations_finish/`, PR #21) `*sql.DB` лежит сырым в `Deps`, `loadRegistrationSession`/`finishRegistration` — пакетные функции. Карточка дизайна отражает целевое состояние; рефакторинг кода — отдельный тикет в `backlog.md` (root).

## Структуры слайса 3 — `sessions-start`

Слайс 3 импортирует из слайса 1: `Handle`, `Challenge`, `NewHandle`, `ErrDBLocked`, `ErrDiskFull` и (после аддитивного расширения) `GenerateChallenge`. Из слайса 2: `User`, `Credential`, `UserID` и (после аддитивных расширений) `UserFromRow`, `CredentialFromRow`, `UserIDFromString`.

### Аддитивные расширения слайса 1 для слайса 3

`generateChallenge` сейчас не экспортирована. Чтобы S3 (а далее и S4) могли создавать `Challenge` без дублирования генератора 32 случайных байт, S1 экспортирует:

```go
// GenerateChallenge — публичная обёртка над generateChallenge.
// Идентичная семантика: 32 случайных байта из crypto/rand.
// Реиспользуется S3 и S4.
func GenerateChallenge() (Challenge, error)
```

Юнит-теста на `GenerateChallenge` отдельно нет — генератор тот же, что покрыт `generateChallenge` в S1. Если внутренняя `generateChallenge` со временем будет переименована, публичная `GenerateChallenge` остаётся стабильной точкой входа для всех слайсов.

### Аддитивные расширения слайса 2 для слайса 3

`User` и `Credential` сейчас собираются только конструкторами `NewUser` / `NewCredential` (требуют рантайм-значения вроде `CreatedAt`, `Verified`). Чтобы метод `Store.LoadUserCredentials` слайса 3 мог восстановить сущности из строк БД, S2 экспортирует рехидраторы:

```go
// UserFromRow восстанавливает сущность из строки users(id, handle, created_at).
// Не валидирует доменные инварианты повторно — данные валидировались при INSERT.
// Возвращает ошибку только при синтаксической непригодности строк
// (UUID не парсится, handle не проходит NewHandle).
func UserFromRow(rowID string, rowHandle string, rowCreatedAtUnix int64) (User, error)

// CredentialFromRow восстанавливает credential из строки
// credentials(credential_id, user_id, public_key, sign_count, transports, created_at).
// transports парсится из CSV-строки (см. миграцию 0003).
func CredentialFromRow(
    rowCredentialID []byte,
    rowUserID string,
    rowPublicKey []byte,
    rowSignCount uint32,
    rowTransports string,
    rowCreatedAtUnix int64,
) (Credential, error)

// UserIDFromString — рехидратор UserID. Используется CredentialFromRow и в S4
// (load login session → user_id → load credentials).
func UserIDFromString(s string) (UserID, error)  // ошибка только если UUID невалиден
```

`Handle` восстанавливается через существующий `NewHandle(rowHandle)` — длина уже проверена при INSERT, повторная валидация не нагрузка.

### Request DTO (ингресс-адаптер слайса 3)

```go
// SessionStartRequest — невалидированный вход из HTTP-адаптера.
// Маппится из тела POST /v1/sessions.
type SessionStartRequest struct {
    Handle string `json:"handle"`
}
```

### Доменные структуры

```go
// LoginSessionID — идентификатор сессии входа (UUID v4).
// Отдельный тип от RegistrationID: разные жизненные циклы, разные таблицы;
// система типов запрещает передать один туда, где ожидается другой.
type LoginSessionID struct {
    value uuid.UUID
}

func generateLoginSessionID() LoginSessionID
func (id LoginSessionID) String() string
func (id LoginSessionID) Bytes() []byte

// SessionStartCommand — валидированная команда фазы 1 входа.
// Поля неэкспортируемые. Создаётся только через NewSessionStartCommand.
type SessionStartCommand struct {
    handle Handle
}

func NewSessionStartCommand(req SessionStartRequest) (SessionStartCommand, error)
// Делегирует: NewHandle(req.Handle).
// Failure: ErrHandleEmpty / ErrHandleTooShort / ErrHandleTooLong
// (импорт из S1, обёрнутые через fmt.Errorf("handle: %w", err)).

func (c SessionStartCommand) Handle() Handle

// UserWithCredentials — агрегат «пользователь + его непустой список credentials».
// Создаётся только методом Store.LoadUserCredentials. Поля неэкспортируемые.
// Инвариант: len(credentials) >= 1 — закодирован в самом существовании структуры.
// Если в БД нет user или нет credentials — I/O возвращает ErrUserNotFound,
// и эта структура с пустым списком не существует по построению.
type UserWithCredentials struct {
    user        User
    credentials []Credential
}

func (u UserWithCredentials) User() User
func (u UserWithCredentials) Credentials() []Credential  // длина >= 1, гарантированно

// LoginSession — доменная сущность сессии входа.
// Объединяет всё, что должно быть сохранено и прочитано фазой 2.
// Поля неэкспортируемые. Создаётся только через NewLoginSession.
type LoginSession struct {
    id        LoginSessionID
    userID    UserID
    challenge Challenge
    expiresAt time.Time
}

// NewLoginSessionInput — value-объект-агрегатор для конструктора.
// «Один data-аргумент» (Шаг 3 program-design.skill).
type NewLoginSessionInput struct {
    ID        LoginSessionID
    UserID    UserID
    Challenge Challenge
    TTL       time.Duration
    Now       time.Time
}

// NewLoginSession собирает доменную сущность.
// Антецедент: ID и UserID — валидные UUID v4 (гарантировано предыдущими шагами);
// Challenge — 32 байта; TTL > 0 (из конфига); Now — момент создания.
// Падений нет — все инварианты на нижних уровнях.
func NewLoginSession(input NewLoginSessionInput) LoginSession

func (s LoginSession) ID() LoginSessionID
func (s LoginSession) UserID() UserID
func (s LoginSession) Challenge() Challenge
func (s LoginSession) ExpiresAt() time.Time
```

### Response DTO (формируется логикой, сериализуется адаптером)

```go
// RequestOptions — подмножество PublicKeyCredentialRequestOptions
// (WebAuthn Level 2). Точно соответствует схеме OpenAPI.
type RequestOptions struct {
    Challenge        string                  `json:"challenge"`        // base64url
    RpID             string                  `json:"rpId,omitempty"`
    AllowCredentials []AllowCredentialDescriptor `json:"allowCredentials,omitempty"`
    UserVerification string                  `json:"userVerification,omitempty"` // "preferred"
    Timeout          int                     `json:"timeout,omitempty"`
}

type AllowCredentialDescriptor struct {
    Type string `json:"type"`           // "public-key"
    ID   string `json:"id"`             // base64url(credential_id)
}

// SessionStartResponse — ответ POST /v1/sessions 201.
type SessionStartResponse struct {
    ID      string         `json:"id"`      // LoginSessionID.String()
    Options RequestOptions `json:"options"`
}

// BuildRequestOptionsInput — value-агрегатор для buildRequestOptions.
// «Один data-аргумент»: вместо buildRequestOptions(s, creds) — buildRequestOptions(input).
type BuildRequestOptionsInput struct {
    Session     LoginSession
    Credentials []Credential  // непустой по инварианту UserWithCredentials
}

// SessionStartView — value-агрегатор для buildResponse.
type SessionStartView struct {
    Session LoginSession
    Options RequestOptions
}
```

### Ошибки

```go
// Класс ошибок предусловий для слайса 3.
// Маппятся ингресс-адаптером в 404 NOT_FOUND.
var (
    ErrUserNotFound = errors.New("user: not found")  // → 404 NOT_FOUND
)

// ErrHandle*, ErrDBLocked, ErrDiskFull импортируются из S1.
```

`ErrUserNotFound` — единственная новая sentinel-ошибка слайса. Покрывает оба случая (нет user / у user нет credentials), потому что для клиента поведение идентично — пройти регистрацию заново. Различение в логах I/O-объекта.

### I/O-объект слайса 3 — `Store`

Применение **правила автономного IO-объекта** (Шаг 6 скилла + `feedback_io_autonomous_store`): зависимость `*sql.DB` инкапсулирована в объекте `Store`, головной модуль её не видит. В `Deps` слайса 3 — поле `Store *Store`, не сырой `*sql.DB`. Имя `Store` отражает тип интеграции (БД).

```go
// Store — автономный I/O-объект слайса 3, инкапсулирующий *sql.DB.
// Поле db неэкспортируемое — только методы Store работают с БД,
// головной модуль обращается к БД исключительно через них.
type Store struct {
    db *sql.DB
}

// NewStore — единственный конструктор. Принимает открытый пул из инфраструктурного
// модуля (cmd/api/main.go → wire.go), возвращает готовый объект.
func NewStore(db *sql.DB) *Store

// LoadUserCredentials — read-метод. Контракт: см. slices/03-sessions-start.md.
//
// Внутри: SELECT id, handle, created_at FROM users WHERE handle = ?
//          (если нет строки → ErrUserNotFound),
//          SELECT credential_id, user_id, public_key, sign_count, transports, created_at
//            FROM credentials WHERE user_id = ?
//          (если нет строк → ErrUserNotFound),
//          сборка UserWithCredentials через UserFromRow / CredentialFromRow (S2).
//
// Маппинг: SQLITE_BUSY → ErrDBLocked. ErrDiskFull для read не различается.
func (s *Store) LoadUserCredentials(h Handle) (UserWithCredentials, error)

// PersistLoginSession — write-метод. Контракт: см. slices/03-sessions-start.md.
//
// Внутри: INSERT INTO login_sessions (id, user_id, challenge, expires_at) VALUES (?, ?, ?, ?).
//
// Маппинг: SQLITE_BUSY → ErrDBLocked, SQLITE_FULL → ErrDiskFull.
func (s *Store) PersistLoginSession(ls LoginSession) error
```

**Один объект, два метода (read + write).** По правилу Шага 6 «один режим работы» каждый метод — отдельный I/O-модуль (юнитами не покрывается, проверяется компонентным сценарием). Объединение в один объект — не нарушение правила: правило про методы, не про типы. Альтернатива — два объекта (`UserCredentialReader`, `LoginSessionWriter`) — даёт больше типов без выигрыша в инкапсуляции (всё равно одна `*sql.DB` под капотом). Берём один `Store`.

> **Технический долг S1/S2.** В реализованных слайсах 1 и 2 в `Deps` лежит сырой `*sql.DB`, а I/O — пакетные функции (`persistRegistrationSession`, `loadRegistrationSession`, `finishRegistration`), не методы автономного объекта. Это нарушение Шага 6 и `feedback_io_autonomous_store`, которое не правится в этой ветке (правило «связанные правки — одна ветка»: ветка дизайна S3 не должна расширяться на рефакторинг реализаций S1/S2). Долг фиксируется отдельным тикетом и закрывается одним PR — либо вместе с дизайном S4 (когда S4 потребует свой `Store`), либо отдельной refactor-сессией оператора.

## Структуры слайса 4 — `sessions-finish`

Слайс 4 импортирует:
- из слайса 1: `Challenge`, `ChallengeFromBytes`, `ErrDBLocked`, `ErrDiskFull`;
- из слайса 2: `User`, `UserID`, `UserIDFromString`, `Credential`, `JWTConfig`, `AccessToken`, `IssuedRefreshToken`, `IssuedTokenPair`, `GenerateTokenPairInput`, `BuildTokenPairView`, `TokenPair` и (после аддитивных расширений) `GenerateTokenPair`, `BuildResponse`;
- из слайса 3: `LoginSessionID`, `LoginSession` и (после аддитивных расширений) `LoginSessionIDFromString`, `LoginSessionFromRow`.

### Аддитивные расширения слайса 2 для слайса 4

`generateTokenPair` и `buildResponse` сейчас — пакетные функции, не экспортированы. Чтобы S4 переиспользовал их без дублирования логики выпуска токенов и сериализации `TokenPair`, S2 экспортирует:

```go
// GenerateTokenPair — публичная обёртка над generateTokenPair.
// Идентичная семантика: Ed25519-подписанный access JWT + 32-байтный refresh с hex(sha256) хешем.
// Реиспользуется S4.
func GenerateTokenPair(input GenerateTokenPairInput) (IssuedTokenPair, error)
// deps: signer ed25519.PrivateKey, jwtCfg JWTConfig (приходят через Deps слайса)

// BuildResponse — публичная обёртка над buildResponse.
// Сериализует пару выданных токенов в DTO TokenPair.
// Реиспользуется S4.
func BuildResponse(view BuildTokenPairView) TokenPair
```

Юнит-тесты S2 на `generateTokenPair` и `buildResponse` остаются прежними (тесты вызывают публичные обёртки). Если внутренняя `generateTokenPair`/`buildResponse` со временем будет переименована, публичные `GenerateTokenPair`/`BuildResponse` остаются стабильной точкой входа для S4.

`AccessToken`, `IssuedRefreshToken`, `IssuedTokenPair`, `GenerateTokenPairInput`, `BuildTokenPairView`, `JWTConfig`, `TokenPair` уже описаны в секции «Структуры слайса 2» — у каждого экспортированы либо публичные методы (`Value()`, `Plaintext()`, `Hash()`, `ExpiresAt()`), либо публичные поля (для value-агрегаторов). Дополнительных рехидраторов для этих типов S4 не требует — он не читает access/refresh из БД, а только генерирует свежую пару.

### Аддитивные расширения слайса 3 для слайса 4

`LoginSession` и `LoginSessionID` сейчас собираются только конструкторами (`NewLoginSession`, `generateLoginSessionID`). Чтобы S4 мог распарсить path-параметр `{id}` и восстановить сессию из строки БД, S3 экспортирует:

```go
// LoginSessionIDFromString — рехидратор LoginSessionID для path-параметра.
// Не валидирует доменные инварианты повторно — UUID-парсинг.
// Возвращает ошибку только если строка не парсится как UUID.
func LoginSessionIDFromString(s string) (LoginSessionID, error)

// LoginSessionFromRow восстанавливает сущность из строки login_sessions(id, user_id, challenge, expires_at).
// Не валидирует доменные инварианты повторно — данные валидировались при INSERT.
// Возвращает ошибку только при синтаксической непригодности строк
// (UUID не парсится, challenge не 32 байта).
func LoginSessionFromRow(
    rowID string,
    rowUserID string,
    rowChallenge []byte,
    rowExpiresAtUnix int64,
) (LoginSession, error)
```

`UserID` восстанавливается через существующий рехидратор `UserIDFromString` (S2 экспорт); `Challenge` — через `ChallengeFromBytes` (S1 экспорт).

### Request DTO (ингресс-адаптер слайса 4)

```go
// SessionFinishRequest — невалидированный вход из HTTP-адаптера.
// Маппится из path-параметра {id} и тела POST /v1/sessions/{id}/assertion.
type SessionFinishRequest struct {
    LoginSessionIDRaw string  // path-параметр (raw UUID-строка)
    AssertionBody     []byte  // тело запроса (JSON AssertionRequest по OpenAPI)
}
```

### Доменные структуры

```go
// ParsedAssertion — типизированная обёртка над *protocol.ParsedCredentialAssertionData
// из github.com/go-webauthn/webauthn/protocol. Создаётся только через parseAssertion.
type ParsedAssertion struct {
    parsed *protocol.ParsedCredentialAssertionData
}

func (p ParsedAssertion) CredentialID() []byte  // raw credential ID из распарсенного assertion

// parseAssertion парсит JSON-тело AssertionRequest через go-webauthn protocol.
// Не верифицирует — только синтаксис и базовая структура (clientDataJSON,
// authenticatorData парсятся как CBOR/binary, поля заполняются).
func parseAssertion(raw []byte) (ParsedAssertion, error)
// Failure: ErrAssertionParse (некорректный JSON, отсутствуют обязательные поля,
// authenticatorData не парсится).

// SessionFinishCommand — валидированная команда фазы 2 входа.
// Поля неэкспортируемые. Создаётся только через NewSessionFinishCommand.
type SessionFinishCommand struct {
    loginSessionID LoginSessionID
    parsed         ParsedAssertion
}

func NewSessionFinishCommand(req SessionFinishRequest) (SessionFinishCommand, error)
// Делегирует: LoginSessionIDFromString(req.LoginSessionIDRaw), parseAssertion(req.AssertionBody).
// Failure: ErrInvalidLoginSessionID, ErrAssertionParse.

func (c SessionFinishCommand) LoginSessionID() LoginSessionID
func (c SessionFinishCommand) Parsed() ParsedAssertion

// FreshLoginSession — LoginSession с инвариантом «не истекла на момент конструкции».
// Создаётся только через NewFreshLoginSession. Применение подправила «подтип, не guard»
// (program-design.skill Шаг 3) — то же решение, что в S2 для FreshRegistrationSession.
type FreshLoginSession struct {
    session LoginSession
}

type NewFreshLoginSessionInput struct {
    Session LoginSession
    Now     time.Time
}

// NewFreshLoginSession проверяет инвариант: input.Now < input.Session.ExpiresAt().
// Если истекла — ErrLoginSessionExpired, структура не создаётся.
func NewFreshLoginSession(input NewFreshLoginSessionInput) (FreshLoginSession, error)

func (f FreshLoginSession) ID() LoginSessionID
func (f FreshLoginSession) UserID() UserID
func (f FreshLoginSession) Challenge() Challenge

// AssertionTarget — агрегат «пользователь + его credential по credentialID».
// Создаётся только методом Store.LoadAssertionTarget. Поля неэкспортируемые.
// Инвариант: credential.UserID() == user.ID() — закодирован в самом существовании структуры.
// Если в БД нет credential по credentialID или credential принадлежит другому user — I/O
// возвращает ErrCredentialNotFound, и эта структура с «чужим» credential не существует
// по построению. Это — то же решение, что в S3 для UserWithCredentials: инвариант
// инкапсулирован в I/O-возврате, не как guard выше по пайпу.
type AssertionTarget struct {
    user       User
    credential Credential
}

func (t AssertionTarget) User() User
func (t AssertionTarget) Credential() Credential

// LoadAssertionTargetInput — value-объект-агрегатор для Store.LoadAssertionTarget.
// «Один data-аргумент» (Шаг 3 program-design.skill).
type LoadAssertionTargetInput struct {
    UserID       UserID  // из FreshLoginSession.UserID() — «свой» user
    CredentialID []byte  // из ParsedAssertion.CredentialID() — какой ключ предъявлен
}

// VerifiedAssertion — assertion, прошедший верификацию против challenge свежей сессии
// и публичного ключа credential'а. Создаётся только через verifyAssertion. Поля неэкспортируемые.
type VerifiedAssertion struct {
    newSignCount uint32  // счётчик из authenticatorData (после успеха Verify)
}

func (v VerifiedAssertion) NewSignCount() uint32

// AssertionVerification — value-объект-агрегатор для verifyAssertion.
// «Один data-аргумент» (Шаг 3 program-design.skill).
type AssertionVerification struct {
    Fresh  FreshLoginSession  // non-expired сессия (инвариант в типе)
    Parsed ParsedAssertion    // распарсенный assertion (синтаксис)
    Target AssertionTarget    // «свой» user + credential для верификации (инвариант в типе)
}

// verifyAssertion проверяет assertion против challenge свежей сессии и публичного ключа.
// Использует protocol.ParsedCredentialAssertionData.Verify под капотом.
func verifyAssertion(input AssertionVerification) (VerifiedAssertion, error)
// deps: rpConfig (нужны ID и Origin)
// Failure: ErrAssertionInvalid (challenge mismatch, RP ID hash mismatch,
// origin mismatch, signature invalid, sign-count clone-warning и любые другие провалы Verify).
```

### Ошибки

```go
// Класс ошибок валидации входа фазы 2 (антецедент конструктора команды).
// Маппятся ингресс-адаптером в 422 VALIDATION_ERROR.
var (
    ErrInvalidLoginSessionID = errors.New("loginSessionID: not a valid UUID")
    ErrAssertionParse        = errors.New("assertion: cannot parse")
)

// Класс ошибок предусловий и верификации.
var (
    ErrLoginSessionNotFound = errors.New("login session: not found")    // → 404 NOT_FOUND
    ErrLoginSessionExpired  = errors.New("login session: expired")       // → 404 NOT_FOUND
    ErrCredentialNotFound   = errors.New("credential: not found")        // → 404 NOT_FOUND (нет credential или принадлежит другому user)
    ErrAssertionInvalid     = errors.New("assertion: verification failed") // → 422 ASSERTION_INVALID
)

// ErrDBLocked / ErrDiskFull импортируются из слайса 1 (общий контракт SQLite).
```

`ErrLoginSessionExpired` маппится в **404 NOT_FOUND** (а не в отдельный код): для клиента истёкшая сессия неотличима от несуществующей — оба требуют начать фазу 1 заново. Различение нужно только в логах.

`ErrCredentialNotFound` объединяет два случая: credential не существует в БД и credential существует, но принадлежит другому пользователю (попытка использовать чужой ключ в рамках своей login_session). Для клиента поведение идентично — 404; различение в логах метода `Store.LoadAssertionTarget` (`logger.Warn("credential mismatch", "user_id", uid, "cred_user_id", cuid)`).

`db_disk_full` (`ErrDiskFull`) на этом эндпоинте **возможен** (write-tx — UPDATE credentials + INSERT refresh_tokens + DELETE login_sessions), декларирован OpenAPI и должен маппиться адаптером в 507. Но компонентного сценария на `db_disk_full` именно здесь **нет** (по сознательной раскладке `slices.md`: `db_disk_full` → S2). Это — часть декларированного OpenAPI-контракта, не мёртвая логика.

### I/O-объект слайса 4 — `Store`

Применение **правила автономного IO-объекта** (Шаг 6 скилла + `feedback_io_autonomous_store`): зависимость `*sql.DB` инкапсулирована в объекте `Store`, головной модуль её не видит. В `Deps` слайса 4 — поле `Store *Store`, не сырой `*sql.DB`.

```go
// Store — автономный I/O-объект слайса 4, инкапсулирующий *sql.DB.
// Поле db неэкспортируемое — головной модуль обращается к БД исключительно через методы.
type Store struct {
    db *sql.DB
}

// NewStore — единственный конструктор. Принимает открытый пул из инфраструктурного
// модуля (cmd/api/main.go → wire.go), возвращает готовый объект.
func NewStore(db *sql.DB) *Store

// LoadLoginSession — read-метод. Контракт: см. slices/04-sessions-finish.md.
//
// Внутри: SELECT id, user_id, challenge, expires_at FROM login_sessions WHERE id = ?
// Если строки нет — ErrLoginSessionNotFound.
// Маппинг: SQLITE_BUSY → ErrDBLocked. ErrDiskFull для read не различается.
// Рехидрирует через LoginSessionFromRow (S3 экспорт).
func (s *Store) LoadLoginSession(id LoginSessionID) (LoginSession, error)

// LoadAssertionTarget — read-метод. Контракт: см. slices/04-sessions-finish.md.
//
// Внутри: SELECT credential_id, user_id, public_key, sign_count, transports, created_at
//          FROM credentials WHERE credential_id = ?
//          (если нет строки → ErrCredentialNotFound),
//          сравнить credential.user_id с input.UserID
//          (если не совпадает → ErrCredentialNotFound — единый класс ошибки),
//          SELECT id, handle, created_at FROM users WHERE id = ?
//          (если нет строки → ErrCredentialNotFound — пользователь удалён,
//           но credential остался; для клиента то же 404).
//          Сборка AssertionTarget через CredentialFromRow / UserFromRow (S2 рехидраторы).
//
// Маппинг: SQLITE_BUSY → ErrDBLocked. ErrDiskFull для read не различается.
func (s *Store) LoadAssertionTarget(input LoadAssertionTargetInput) (AssertionTarget, error)

// FinishLogin — write-метод (атомарная транзакция, 3 операции).
// Контракт: см. slices/04-sessions-finish.md.
//
// Внутри одной транзакции:
//   UPDATE credentials SET sign_count = ? WHERE credential_id = ?
//   INSERT INTO refresh_tokens (token_hash, user_id, expires_at) VALUES (...)
//   DELETE FROM login_sessions WHERE id = ?
// При любой ошибке tx откатывается — ни одна из 3 операций не остаётся применённой.
//
// Маппинг: SQLITE_BUSY → ErrDBLocked; SQLITE_FULL → ErrDiskFull.
type FinishLoginInput struct {
    Credential       Credential       // нужен только credential_id и user_id (для INSERT refresh_tokens.user_id)
    NewSignCount     uint32           // из VerifiedAssertion.NewSignCount()
    RefreshTokenHash string           // hex(sha256(plaintext)) из IssuedRefreshToken.Hash()
    RefreshExpiresAt time.Time        // из IssuedRefreshToken.ExpiresAt()
    LoginSessionID   LoginSessionID   // из FreshLoginSession.ID() — DELETE FROM login_sessions
}

func (s *Store) FinishLogin(input FinishLoginInput) error
```

`Store.FinishLogin` — единственная I/O write-операция слайса. Всё или ничего: иначе при, например, успехе UPDATE credentials + провале INSERT refresh_tokens получим обновлённый sign_count без выданного refresh — клиент уйдёт с 5xx, но сервер «считает», что предыдущий assertion использован, и при ретрае signCount уже больше и Verify провалится с ErrAssertionInvalid.

**Один объект, три метода (два read + один write).** По правилу Шага 6 «один режим работы» каждый метод — отдельный I/O-модуль (юнитами не покрывается, проверяется компонентным сценарием). Объединение в один объект — не нарушение правила: правило про методы, не про типы.

**Решение — `LoadLoginSession` и `LoadAssertionTarget` как два отдельных метода, не один.** Альтернатива — слить в один read-метод `LoadLoginContext({id, credentialID}) -> {Fresh, Target}` — нарушает Шаг 3 «один data-аргумент» (методу пришлось бы принимать кортеж id+credentialID без естественной доменной структуры) и склеивает два логически независимых SELECT'а: «найти сессию» и «найти credential свою» — каждый со своим классом ошибок (`ErrLoginSessionNotFound` vs `ErrCredentialNotFound`). Слияние ухудшит читаемость failure-маппинга в адаптере. Берём два метода.

