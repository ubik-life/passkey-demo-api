# Slice 01 — `registrations-start`

## Идентификатор входа

`HTTP POST /v1/registrations`

## Что делает (в одну фразу)

Принимает handle пользователя, создаёт регистрационную сессию с challenge, возвращает `id` сессии и `PublicKeyCredentialCreationOptions` для передачи в `navigator.credentials.create()` на клиенте.

## OpenAPI

`api-specification/openapi.yaml`, `paths./registrations.post`. Контракт:

- Запрос: `application/json` `{handle: string, 3..64}`
- Ответ 201: `{id: uuid, options: PublicKeyCredentialCreationOptions}`
- Возможные ошибки: 422 `VALIDATION_ERROR`, 503 `db_locked` (+ `Retry-After`), 507 `db_disk_full`

## Gherkin-сценарии слайса

`component-tests/features/registrations.feature`:

- `Сценарий: Создание challenge регистрации` — единственный сценарий, который **проверяет результат** этого слайса напрямую (`Тогда ответ 201`).
- `Сценарий: Завершение регистрации` и `Сценарий: Диск переполнен при завершении регистрации` — используют `POST /v1/registrations` в **When**-шагах, но их **Then**-шаги относятся к слайсу 2 (`registrations-finish`). Корректность работы слайса 1 в них проверяется неявно: чтобы сценарии прошли, фаза 1 должна вернуть валидные `id` и `options`.

В таблице `## Gherkin-mapping` ниже ведётся учёт **только Then-шагов первого сценария**.

## Дерево модулей

```
ингресс-адаптер: HTTP handler POST /v1/registrations
    ├── parse JSON body → RegistrationStartRequest
    └── (после головного модуля) format Response → 201 + JSON,
        либо error → 422 / 503 + Retry-After / 507
        │
головной модуль слайса: ProcessRegistrationStart
    ├── (1) NewRegistrationStartCommand(req)         → RegistrationStartCommand
    │       └── NewHandle(raw)                       → Handle                [конструктор-лист]
    ├── (2) generateChallenge()                      → Challenge             [логика-лист]
    ├── (3) generateRegistrationID()                 → RegistrationID        [логика-лист]
    ├── (4) NewRegistrationSession(input)            → RegistrationSession   [конструктор-сборщик]
    ├── (5) persistRegistrationSession(s)            → error                 [I/O — SQLite write; dep: db]
    ├── (6) buildCreationOptions(s)                  → CreationOptions       [логика; dep: rpConfig]
    └── (7) buildResponse(view)                      → RegistrationStartResponse  [логика]
```

Каждый узел — **один data-аргумент** (Шаг 3 скилла opus'а). Зависимости (`db`, `rpConfig`, `clock`) вынесены и не считаются стрелками графа.

## Псевдокод пайпа головного модуля

```
ProcessRegistrationStart(req: RegistrationStartRequest, deps: Deps)
    -> (RegistrationStartResponse, error):

    | NewRegistrationStartCommand(req)               -> RegistrationStartCommand
    | generateChallenge()                             -> Challenge
    | generateRegistrationID()                        -> RegistrationID
    | input := NewRegistrationSessionInput{ ID, Handle, Challenge,
                                             TTL: deps.cfg.ChallengeTTL,
                                             Now: deps.clock.Now() }
    | NewRegistrationSession(input)                  -> RegistrationSession
    | persistRegistrationSession(session)             -> error    [dep: deps.db]
    | buildCreationOptions(session)                   -> CreationOptions  [dep: deps.cfg.RP]
    | view := RegistrationStartView{ Session, Options }
    | buildResponse(view)                             -> RegistrationStartResponse
```

Ошибки протекают через ранний `return zero, err`. Идиома Go: на каждом шаге `err != nil` → `return RegistrationStartResponse{}, fmt.Errorf("step: %w", err)`.

`NewRegistrationSessionInput{...}` и `RegistrationStartView{...}` — это **сборка локальных value-объектов** (Go-литерал структуры), формальные строки пайпа без отдельного контракта. На графе они не отдельные узлы, а часть стрелки во входящий конструктор/функцию. Их роль — соблюсти «один data-аргумент» на следующем шаге без введения лишних доменных сущностей.

## Контракты модулей

### `NewHandle`

- **Сигнатура:** `NewHandle(raw string) -> (Handle, error)`
- **Input (data):** `raw string` — сырой handle из DTO.
- **Dependencies (deps):** —
- **Что делает:** валидирует строку handle и возвращает доменное значение или ошибку.
- **Антецедент:** `raw` после `strings.TrimSpace` имеет длину 3..64.
- **Консеквент:**
  - Success: `Handle.value` — строка длиной 3..64, без leading/trailing whitespace.
  - Failure: `ErrHandleEmpty` (пустая строка после trim), `ErrHandleTooShort` (len < 3), `ErrHandleTooLong` (len > 64).

### `NewRegistrationStartCommand`

- **Сигнатура:** `NewRegistrationStartCommand(req RegistrationStartRequest) -> (RegistrationStartCommand, error)`
- **Input (data):** `req RegistrationStartRequest`
- **Dependencies (deps):** —
- **Что делает:** собирает доменную команду из DTO. Делегирует валидацию `NewHandle`.
- **Антецедент:** `req.Handle` соответствует антецеденту `NewHandle`.
- **Консеквент:**
  - Success: команда с валидным `Handle`.
  - Failure: ошибка из `NewHandle`, обёрнутая в `fmt.Errorf("handle: %w", err)`.

### `generateChallenge`

- **Сигнатура:** `generateChallenge() -> (Challenge, error)`
- **Input (data):** void
- **Dependencies (deps):** — (`crypto/rand` неявно; для тестируемости при необходимости вводится `Randomness io.Reader` отдельным рефактором)
- **Что делает:** создаёт challenge из 32 случайных байт.
- **Антецедент:** —
- **Консеквент:**
  - Success: `Challenge.bytes` — 32 байта из `crypto/rand.Read`.
  - Failure: ошибка чтения системной энтропии (теоретическая; обрабатываем как catastrophic).

### `generateRegistrationID`

- **Сигнатура:** `generateRegistrationID() -> RegistrationID`
- **Input (data):** void
- **Dependencies (deps):** —
- **Что делает:** возвращает свежий UUID v4.
- **Антецедент:** —
- **Консеквент:** валидный UUID v4. Возврат без `error` — `uuid.New()` не падает.

### `NewRegistrationSession`

- **Сигнатура:** `NewRegistrationSession(input NewRegistrationSessionInput) -> RegistrationSession`
- **Input (data):** `input NewRegistrationSessionInput` (см. `messages.md`).
- **Dependencies (deps):** —
- **Что делает:** собирает доменную сущность регистрационной сессии.
- **Антецедент:**
  - `input.ID` — валидный UUID v4 (гарантировано конструктором);
  - `input.Handle` — валидное доменное значение (гарантировано конструктором);
  - `input.Challenge` — 32 байта (гарантировано конструктором);
  - `input.TTL > 0`;
  - `input.Now` — момент создания.
- **Консеквент:**
  - Success: `RegistrationSession` с `expiresAt = input.Now + input.TTL`.
  - Failure не возвращается — все доменные инварианты уже проверены конструкторами вложенных сущностей; `TTL > 0` гарантируется конфигом, `Now` не «проваливается».

### `persistRegistrationSession`

- **Сигнатура:** `persistRegistrationSession(s RegistrationSession) -> error`
- **Input (data):** `s RegistrationSession`
- **Dependencies (deps):** `*sql.DB`
- **Что делает:** одной транзакцией вставляет запись в таблицу `registration_sessions(id, handle, challenge, expires_at)`.
- **Антецедент:**
  - `s` — валидная доменная сущность (гарантировано `NewRegistrationSession`);
  - `db` — открытый пул соединений, миграция применена.
- **Консеквент:**
  - Success: запись в БД, `expires_at = s.ExpiresAt()`.
  - Failure:
    - `ErrDBLocked` — `SQLITE_BUSY` (lock contention), запись **не** создана. Маппится ингресс-адаптером в 503 + `Retry-After`.
    - `ErrDiskFull` — `SQLITE_FULL` (диск переполнен), запись **не** создана. Маппится в 507.
    - другие низкоуровневые ошибки SQLite оборачиваются в общую внутреннюю ошибку (маппинг → 500). Не различаются по контракту.

### `buildCreationOptions`

- **Сигнатура:** `buildCreationOptions(s RegistrationSession) -> CreationOptions`
- **Input (data):** `s RegistrationSession`
- **Dependencies (deps):** `RPConfig`
- **Что делает:** собирает `PublicKeyCredentialCreationOptions` по схеме OpenAPI из доменной сессии и конфига RP.
- **Антецедент:** `s` валидна; `RPConfig.Name` и `RPConfig.ID` непустые.
- **Консеквент:**
  - `RP.Name = rpConfig.Name`, `RP.ID = rpConfig.ID`.
  - `User.ID = base64url(s.ID().Bytes())`, `User.Name = s.Handle().Value()`, `User.DisplayName = s.Handle().Value()`.
  - `Challenge = s.Challenge().Base64URL()`.
  - `PubKeyCredParams = [{type:"public-key", alg:-7}, {type:"public-key", alg:-8}]`.
  - `Attestation = "none"`.
  - `Timeout` — не задаётся (`omitempty`).
- Падений нет — чистая функция без `error`.

### `buildResponse`

- **Сигнатура:** `buildResponse(view RegistrationStartView) -> RegistrationStartResponse`
- **Input (data):** `view RegistrationStartView { Session RegistrationSession; Options CreationOptions }` — value-агрегатор для соблюдения «один data-аргумент». См. `messages.md`.
- **Dependencies (deps):** —
- **Что делает:** упаковывает доменное представление ответа в DTO.
- **Антецедент:** аргументы валидны (Session — доменная сущность, Options — собран `buildCreationOptions`).
- **Консеквент:** `Response.ID = view.Session.ID().String()`, `Response.Options = view.Options`.

### Ингресс-адаптер: HTTP handler `POST /v1/registrations`

- **Что делает:**
  1. Парсит тело JSON в `RegistrationStartRequest`. Если JSON синтаксически невалиден — 400 с `error.code: VALIDATION_ERROR` (без обращения к головному модулю).
  2. Вызывает `ProcessRegistrationStart(req, deps)`.
  3. На Success: пишет `201 Created`, `Content-Type: application/json`, тело — `RegistrationStartResponse`.
  4. На Failure: маппит ошибки в HTTP-ответ (см. таблицу маппинга ниже).
- **Никакой бизнес-логики, никакой валидации полей** — только парсинг и маппинг.
- **Юнит-тестами не покрывается** — проверяется компонентным тестом слайса через реальный HTTP-вход (Шаг 8.1 скилла opus'а).

### Маппинг ошибок в ингресс-адаптере

| Класс ошибки                                                  | HTTP-статус | Заголовки           | Тело (`error.code`)         |
|---------------------------------------------------------------|-------------|---------------------|-----------------------------|
| Невалидный JSON (`json.Decoder.Decode` failed)                | 400          | —                   | `VALIDATION_ERROR`           |
| `ErrHandleEmpty` / `ErrHandleTooShort` / `ErrHandleTooLong`   | 422          | —                   | `VALIDATION_ERROR`           |
| `ErrDBLocked`                                                  | 503          | `Retry-After: 1`    | `db_locked`                  |
| `ErrDiskFull`                                                  | 507          | —                   | `db_disk_full`               |
| Любая другая (catastrophic из `generateChallenge`, неожиданные SQLite-ошибки) | 500 | — | `INTERNAL_ERROR`         |

## Gherkin-mapping

Только Then-шаги, которые проверяют **результат** этого слайса. Сценарии 2 и 3 проверяют слайс 2 (`registrations-finish`) и не входят в эту таблицу.

| Сценарий                          | Then-шаг       | Кто обеспечивает (узел графа / маппинг адаптера)                                                                              |
|-----------------------------------|----------------|-------------------------------------------------------------------------------------------------------------------------------|
| Создание challenge регистрации    | `Тогда ответ 201` | Узлы (1)–(7) головного модуля (Success-путь) → ингресс-адаптер: `201 Created` + JSON-сериализация `RegistrationStartResponse` |

### Чек-лист сверки 8.5

1. [x] **Узел существует.** Узлы (1)–(7) описаны в дереве и в контрактах выше; ингресс-адаптер описан с маппингом.
2. [x] **Ветка соответствует.** Then `ответ 201` — Success-ветка пайпа; ингресс-адаптер пишет 201.
3. [x] **Формат ответа адаптера согласован.** OpenAPI декларирует 201 + `RegistrationInitResponse`; адаптер сериализует `RegistrationStartResponse` с теми же полями (`id`, `options`). Структура `CreationOptions` в `messages.md` соответствует схеме `PublicKeyCredentialCreationOptions` в OpenAPI.
4. [x] **Все Then покрыты.** В сценарии «Создание challenge регистрации» один Then-шаг, он покрыт.

`[x] Gherkin-mapping сверен.`

### Замечание о сценариях 2 и 3

Сценарии «Завершение регистрации» и «Диск переполнен при завершении регистрации» используют этот слайс через When-шаги. Чтобы они прошли:

- ингресс-адаптер должен возвращать **сериализуемый** `RegistrationStartResponse` (godog WebAuthn-степ парсит тело и собирает attestation из `options.challenge`);
- I/O `persistRegistrationSession` должен **реально записать** challenge в БД, чтобы фаза 2 (слайс 2) могла его прочитать.

Эти неявные требования покрыты контрактом (`buildResponse` строит правильный JSON; I/O имеет Success-ветку, гарантирующую запись), но **не** добавляют строк в таблицу Gherkin-mapping этого слайса — они формализуются как стрелки графа в `contracts-graph.md` и проверяются интеграционно через сценарии слайса 2.

## Юнит-тесты по формуле

`N_юнит_тестов = 1 (happy path) + Σ (ветки антецедента)` — **только модули логики** (Шаг 8.1 скилла opus'а: «I/O — трубы, юнитами не покрываются»; ингресс-адаптер — тоже).

| Модуль                            | Happy | Ветки антецедента                                              | Итого |
|-----------------------------------|-------|----------------------------------------------------------------|-------|
| `NewHandle`                       | 1     | пусто/whitespace, len<3, len>64                                | 4     |
| `NewRegistrationStartCommand`     | 1     | (склейка; ветки покрыты в `NewHandle`, тут одна ошибка-обёртка) | 2     |
| `generateChallenge`               | 1     | (теоретическая ошибка `crypto/rand` — пропускаем в MVP)         | 1     |
| `generateRegistrationID`          | 1     | —                                                              | 1     |
| `NewRegistrationSession`          | 1     | —                                                              | 1     |
| `buildCreationOptions`            | 1     | —                                                              | 1     |
| `buildResponse`                   | 1     | —                                                              | 1     |
| `ProcessRegistrationStart` (head) | 1     | ошибка из `NewRegistrationStartCommand`, ошибка из `persistRegistrationSession` | 3 |
| **Итого**                         |       |                                                                | **14** |

Что **не** в таблице (и почему):

- `persistRegistrationSession` — I/O-модуль, по сути труба. Юнитов нет. Success-путь проверяется компонентным сценарием **«Создание challenge регистрации»** (этого слайса; если запись не дойдёт в БД — фаза 2 в сценариях 2-3 не сможет прочитать challenge, упадут). Failure-ветки `ErrDBLocked` / `ErrDiskFull` проверяются компонентными сценариями **других слайсов** по правилу различимости: `db_locked` → слайс 4 (`POST /v1/sessions/{id}/assertion`), `db_disk_full` → слайс 2 (`POST /v1/registrations/{id}/attestation`). На уровне маппинга ошибок в HTTP-статус **слайс 1 обязан** их корректно отдавать (503/507), но Gherkin-сценария на эту тему именно для `POST /v1/registrations` нет — это сознательный выбор в раскладке режимов отказа.
- **Ингресс-адаптер** — парсинг и маппинг ошибок, юнитов нет. Проверяется реальным HTTP-вызовом в компонентном сценарии.
- `головной модуль` (`ProcessRegistrationStart`) **есть** в таблице юнитов — это модуль логики (оркестратор пайпа), его антецедент: разбор ошибок от конструктора и от I/O. Тестируется с моками I/O.

Замечания по покрытию:

- Покрытие 100% строк/веток модулей логики (включая головной) достигается этими 14 юнит-тестами.
- I/O-модуль и ингресс-адаптер не входят в метрику покрытия юнитов; их корректность доказана компонентными сценариями (черный ящик через реальный HTTP).

## Definition of Done слайса

Скопировано в тикет S1 в `backlog.md`:

- [ ] ингресс-адаптер реализован: парсит JSON в `RegistrationStartRequest`, без бизнес-валидации
- [ ] конструкторы доменных структур (`NewHandle`, `NewRegistrationStartCommand`, `NewRegistrationSession`) реализованы: проверяют антецедент, при невалидных данных возвращают ошибку
- [ ] модули логики (`generateChallenge`, `generateRegistrationID`, `buildCreationOptions`, `buildResponse`) реализованы, контракты выполнены
- [ ] модуль I/O (`persistRegistrationSession`) реализован, оборачивает SQLITE_BUSY → `ErrDBLocked`, SQLITE_FULL → `ErrDiskFull`
- [ ] головной модуль `ProcessRegistrationStart` реализован: пайп из 7 шагов, ранний возврат при ошибке
- [ ] миграция `0001_registration_sessions.sql` создаёт таблицу `registration_sessions(id PRIMARY KEY, handle, challenge, expires_at)`
- [ ] слайс подключён в инфраструктурном модуле: HTTP-роут `POST /v1/registrations` ведёт на ингресс-адаптер
- [ ] юнит-тесты по формуле (14 тестов на модули логики), покрытие 100% по строкам и веткам логики
- [ ] компонентный сценарий `Сценарий: Создание challenge регистрации` зелёный
- [ ] локальный CI зелёный (`make test && ./component-tests/scripts/run-tests.sh`)
- [ ] PR создан, описание заполнено по шаблону
- [ ] PR смержен в main, CI на main зелёный
