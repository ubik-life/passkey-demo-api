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
    ListenAddr   string         // PASSKEY_LISTEN_ADDR, например ":8080"
    DBPath       string         // PASSKEY_DB_PATH, например "/var/lib/passkey/db.sqlite"
    RP           RPConfig       // PASSKEY_RP_NAME, PASSKEY_RP_ID
    ChallengeTTL time.Duration  // PASSKEY_CHALLENGE_TTL, например "5m"
    JWT          JWTConfig      // зарезервировано для слайсов 2/4 (генерация Ed25519 при старте)
}

type JWTConfig struct {
    AccessTTL  time.Duration  // PASSKEY_JWT_ACCESS_TTL,  например "15m"
    RefreshTTL time.Duration  // PASSKEY_JWT_REFRESH_TTL, например "720h"
}
```

`RPConfig` — описан в `messages.md`.

`JWTConfig` — заглушка на этой итерации, не используется в слайсе 1. Документируется здесь, чтобы инфраструктура не пересобиралась при переходе к слайсам 2/4. Ed25519 ключ генерируется при старте процесса (см. `CLAUDE.md`: «приватный ключ не персистится»).

## Старт процесса

Псевдокод `cmd/api/main.go`:

```
main():
    cfg, err := app.LoadConfig()         // env → AppConfig; падение при невалидном
    log := slog.New(slog.NewJSONHandler(os.Stdout, ...))
    clk := clock.System{}

    db, err := db.Open(cfg.DBPath)       // открыть пул, применить миграции
    defer db.Close()

    deps := wire.Build(cfg, db, log, clk)

    mux := chi.NewRouter()
    registrationsStart.Register(mux, deps.RegistrationsStart)
    // в следующих итерациях:
    // registrationsFinish.Register(mux, deps.RegistrationsFinish)
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

## Миграции

```
internal/db/migrations/
└── 0001_registration_sessions.sql
```

Содержимое (для слайса 1):

```sql
CREATE TABLE registration_sessions (
    id          TEXT PRIMARY KEY,    -- UUID v4 string form
    handle      TEXT NOT NULL,
    challenge   BLOB NOT NULL,        -- 32 bytes
    expires_at  INTEGER NOT NULL      -- unix seconds
);

CREATE INDEX idx_registration_sessions_expires_at ON registration_sessions(expires_at);
```

Миграции применяются на старте `db.Open` через `goose` или эквивалент (см. `AGENTS.md §15`). `0001_*` создаёт таблицу для challenge'ей фазы 1 регистрации.

Миграции для будущих слайсов (`users`, `credentials`, `login_sessions`, `refresh_tokens`) добавятся в их соответствующих итерациях. Каждый слайс добавляет ровно те таблицы, с которыми работает его I/O-модуль.

## Health-эндпоинт

`GET /health` → 200 со статическим телом — уже реализован в placeholder-сервисе (`devlog/06`). Подключается до регистрации слайсов; используется compose-healthcheck в `component-tests`.

## Что **не** делает инфраструктурный модуль

- Не валидирует входящие запросы (это — конструкторы домена в слайсах).
- Не оркестрирует слайсы между собой (vertical slice — независимы по построению).
- Не содержит бизнес-логики (поэтому без юнит-тестов; проверяется компонентными).

## Тестирование

- **Юнитов нет** (бизнес-логики нет).
- **Компонентные тесты** прогоняют каждый слайс через его реальный вход — HTTP-запрос для HTTP-слайсов (все 6 слайсов в этом сервисе). См. `component-tests/features/`.
