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

### Разделяемые структуры между слайсами

Появятся при добавлении слайсов 2-6. На текущей итерации каталог замкнут на одном слайсе.
