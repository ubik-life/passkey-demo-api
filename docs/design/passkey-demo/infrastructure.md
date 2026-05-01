# Infrastructure — passkey-demo

Инфраструктурный модуль приложения. Один на всю программу. **Бизнес-логики нет.**

Задачи:
1. Прочитать конфиг из env.
2. Открыть пул соединений к SQLite, применить миграции.
3. Инициализировать общие зависимости (логгер, clock).
4. Поднять HTTP-сервер.
5. Зарегистрировать роуты слайсов на их ингресс-адаптеры, инжектируя зависимости.

В этой итерации (S1) подключается только слайс 1.

## Размещение

```
cmd/api/main.go              -- entry point, собирает приложение и поднимает HTTP-сервер
internal/app/config.go       -- env → AppConfig
internal/app/wire.go         -- DI: собирает Deps для каждого слайса
internal/db/db.go            -- открыть SQLite, применить миграции
internal/db/migrations/      -- *.sql
internal/clock/clock.go      -- интерфейс Clock + дефолтная реализация
internal/slice/registrations_start/    -- слайс 1 (по карточке слайса)
```

## Конфигурация

```go
// AppConfig — всё, что нужно сервису из env.
type AppConfig struct {
    ListenAddr   string         // SERVICE_ADDR, дефолт ":8080"
    DBPath       string         // SQLITE_PATH, обязателен
    RP           RPConfig       // PASSKEY_RP_NAME, PASSKEY_RP_ID, PASSKEY_RP_ORIGIN
    ChallengeTTL time.Duration  // PASSKEY_CHALLENGE_TTL, дефолт "5m"
    JWT          JWTConfig      // PASSKEY_JWT_*  (используется слайсами 2/4/5/6)
}

type JWTConfig struct {
    AccessTTL  time.Duration  // PASSKEY_JWT_ACCESS_TTL,  дефолт "15m"
    RefreshTTL time.Duration  // PASSKEY_JWT_REFRESH_TTL, дефолт "720h"
    Issuer     string         // PASSKEY_JWT_ISSUER,      дефолт "passkey-demo"
}
```

`RPConfig` — описан в `messages.md` (расширяется в S2 полем `Origin`).

Полный набор env-переменных:

| Имя | Дефолт | Откуда | Назначение |
|---|---|---|---|
| `SERVICE_ADDR` | `:8080` | S1 | listen address HTTP-сервера |
| `SQLITE_PATH` | (обязательна) | S1 | путь к файлу SQLite |
| `PASSKEY_RP_NAME` | `Passkey Demo` | S1 | RP.name в WebAuthn options |
| `PASSKEY_RP_ID` | `localhost` | S1 | RP.id (домен) |
| `PASSKEY_RP_ORIGIN` | `http://localhost` | **S2** | ожидаемый origin в `clientDataJSON` |
| `PASSKEY_CHALLENGE_TTL` | `5m` | S1 | TTL регистрационной сессии |
| `PASSKEY_JWT_ACCESS_TTL` | `15m` | **S2** | TTL access JWT |
| `PASSKEY_JWT_REFRESH_TTL` | `720h` | **S2** | TTL refresh token (хранится в БД, hashed) |
| `PASSKEY_JWT_ISSUER` | `passkey-demo` | **S2** | claim `iss` в JWT |

## Ed25519 keypair (S2)

Генерируется **при старте процесса** через `crypto/ed25519.GenerateKey(crypto/rand.Reader)`. **Не персистится** (по `CLAUDE.md`). Перезапуск процесса инвалидирует все ранее выданные access-токены — это сознательный выбор demo-сервиса. Refresh-токены не инвалидируются (они opaque, не подписаны — валидируются по hash в БД).

```go
// internal/app/wire.go
type Signer struct {
    Private ed25519.PrivateKey
    Public  ed25519.PublicKey
}

func generateSigner() (Signer, error) {
    pub, priv, err := ed25519.GenerateKey(rand.Reader)
    if err != nil {
        return Signer{}, fmt.Errorf("generate ed25519 keypair: %w", err)
    }
    return Signer{ Private: priv, Public: pub }, nil
}
```

Для S2 в `Deps` передаётся только `Private`. Public key понадобится в S5/S6 (валидация access JWT) — там подцепляется тот же `Signer`.

## Старт процесса

Псевдокод `cmd/api/main.go`:

```
main():
    cfg, err := app.LoadConfig()         // env → AppConfig; падение при невалидном
    log := slog.New(slog.NewJSONHandler(os.Stdout, ...))
    clk := clock.System{}

    signer, err := generateSigner()      // ed25519 keypair, не персистится  [S2]
    if err != nil { fatal }

    db, err := db.Open(cfg.DBPath)       // открыть пул, применить миграции 0001-0004
    defer db.Close()

    deps := wire.Build(cfg, db, log, clk, signer, rand.Reader)

    mux := chi.NewRouter()
    registrationsStart.Register(mux, deps.RegistrationsStart)
    registrationsFinish.Register(mux, deps.RegistrationsFinish)   // [S2]
    // в следующих итерациях:
    // sessionsStart.Register(mux, deps.SessionsStart)
    // ...

    srv := &http.Server{ Addr: cfg.ListenAddr, Handler: mux }
    log.Info("listening", "addr", cfg.ListenAddr)
    srv.ListenAndServe()
```

## Подключение слайса 1

Слайс предоставляет публичную функцию `Register(mux chi.Router, deps Deps)`, которая регистрирует свой ингресс-адаптер на роуте `POST /v1/registrations`.

```go
// internal/slice/registrations_start/register.go
package registrations_start

func Register(mux chi.Router, deps Deps) {
    h := newHTTPHandler(deps)
    mux.Post("/v1/registrations", h.ServeHTTP)
}

// Deps — зависимости слайса. Инжектируются wire.go.
type Deps struct {
    DB           *sql.DB
    Clock        clock.Clock
    Logger       *slog.Logger
    RP           RPConfig
    ChallengeTTL time.Duration
}
```

Wire берёт эти поля из `AppConfig` и общих зависимостей и собирает `Deps` для каждого слайса. Слайсы между собой `Deps` не делят (vertical slice — изоляция).

## Подключение слайса 2 (S2)

```go
// internal/slice/registrations_finish/register.go
package registrations_finish

func Register(mux chi.Router, deps Deps) {
    h := newHTTPHandler(deps)
    mux.Post("/v1/registrations/{id}/attestation", h.ServeHTTP)
}

// Deps — зависимости слайса 2.
type Deps struct {
    DB       *sql.DB
    Clock    clock.Clock
    Logger   *slog.Logger
    RP       registrations_start.RPConfig  // импорт из S1; в S2 используется поле Origin
    JWT      JWTConfig
    Signer   ed25519.PrivateKey            // приватный ключ Ed25519, генерится при старте
    Rand     io.Reader                     // источник случайности (crypto/rand.Reader в проде)
}
```

`Rand` — отдельный dep вместо неявного `crypto/rand`, чтобы юнит-тесты `generateTokenPair` могли подсунуть детерминированный генератор и проверить совпадение hash refresh-токена.

## Миграции

```
internal/db/migrations/
├── 0001_registration_sessions.sql   [S1]
├── 0002_users.sql                    [S2]
├── 0003_credentials.sql              [S2]
└── 0004_refresh_tokens.sql           [S2]
```

Миграции применяются на старте `db.Open` через `goose` (см. `AGENTS.md §15`). Один файл — одна таблица + связанные индексы; легче читать историю и откатывать частично.

### `0001_registration_sessions.sql` (S1, существующая)

```sql
CREATE TABLE registration_sessions (
    id          TEXT PRIMARY KEY,    -- UUID v4 string
    handle      TEXT NOT NULL,
    challenge   BLOB NOT NULL,        -- 32 bytes
    expires_at  INTEGER NOT NULL      -- unix seconds
);
CREATE INDEX idx_registration_sessions_expires_at ON registration_sessions(expires_at);
```

### `0002_users.sql` (S2)

```sql
-- +goose Up
CREATE TABLE users (
    id          TEXT PRIMARY KEY,                -- UUID v4 string
    handle      TEXT NOT NULL UNIQUE,            -- UNIQUE: SQLITE_CONSTRAINT_UNIQUE → ErrHandleTaken
    created_at  INTEGER NOT NULL                 -- unix seconds
);
CREATE UNIQUE INDEX idx_users_handle ON users(handle);

-- +goose Down
DROP INDEX IF EXISTS idx_users_handle;
DROP TABLE IF EXISTS users;
```

### `0003_credentials.sql` (S2)

```sql
-- +goose Up
CREATE TABLE credentials (
    credential_id  BLOB    PRIMARY KEY,           -- raw credential ID из authenticatorData
    user_id        TEXT    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    public_key     BLOB    NOT NULL,              -- CBOR/COSE public key
    sign_count     INTEGER NOT NULL,              -- счётчик из authenticatorData (uint32)
    transports     TEXT    NOT NULL DEFAULT '',   -- CSV: "usb,nfc,ble" — может быть пустой
    created_at     INTEGER NOT NULL
);
CREATE INDEX idx_credentials_user_id ON credentials(user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_credentials_user_id;
DROP TABLE IF EXISTS credentials;
```

`ON DELETE CASCADE` оставляем для будущего: если в каком-то слайсе понадобится удалять пользователя, credentials уйдут вместе. На S2 удалений `users` нет.

### `0004_refresh_tokens.sql` (S2)

```sql
-- +goose Up
CREATE TABLE refresh_tokens (
    token_hash  TEXT    PRIMARY KEY,              -- hex(sha256(plaintext))
    user_id     TEXT    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at  INTEGER NOT NULL,
    revoked_at  INTEGER NULL                      -- заполняется в S5 при logout
);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);

-- +goose Down
DROP INDEX IF EXISTS idx_refresh_tokens_expires_at;
DROP INDEX IF EXISTS idx_refresh_tokens_user_id;
DROP TABLE IF EXISTS refresh_tokens;
```

`revoked_at` — для S5 (logout). На S2 строки создаются с `revoked_at = NULL`. Валидация в S5/S6: `expires_at > now AND revoked_at IS NULL`.

### Будущие миграции

`login_sessions` (challenge'и фазы 1 входа) — слайс 3. Добавится отдельным файлом `0005_login_sessions.sql` в его итерации.

## Health-эндпоинт

`GET /health` → 200 со статическим телом — уже реализован в placeholder-сервисе (`devlog/06`). Подключается до регистрации слайсов; используется compose-healthcheck в `component-tests`.

## Что **не** делает инфраструктурный модуль

- Не валидирует входящие запросы (это — конструкторы домена в слайсах).
- Не оркестрирует слайсы между собой (vertical slice — независимы по построению).
- Не содержит бизнес-логики (поэтому без юнит-тестов; проверяется компонентными).

## Тестирование

- **Юнитов нет** (бизнес-логики нет).
- **Компонентные тесты** прогоняют каждый слайс через его реальный вход — HTTP-запрос для HTTP-слайсов (все 6 слайсов в этом сервисе). См. `component-tests/features/`.
