# Slice 03 — `sessions-start`

## Идентификатор входа

`HTTP POST /v1/sessions`

## Что делает (в одну фразу)

Принимает handle пользователя, находит его и его credentials в БД, создаёт сессию входа с challenge, возвращает `id` сессии и `PublicKeyCredentialRequestOptions` (включая `allowCredentials`) для передачи в `navigator.credentials.get()` на клиенте.

## OpenAPI

`api-specification/openapi.yaml`, `paths./sessions.post`. Контракт:

- Запрос: `application/json` `SessionInitRequest { handle: string, 3..64 }`
- Ответ 201: `SessionInitResponse { id: uuid, options: PublicKeyCredentialRequestOptions }`
- Возможные ошибки: 404 `NOT_FOUND` (handle не зарегистрирован), 422 `VALIDATION_ERROR`, 503 `db_locked` (+ `Retry-After`), 507 `db_disk_full`.

## Gherkin-сценарии слайса

`component-tests/features/sessions.feature`:

- `Сценарий: Создание challenge входа` — единственный сценарий, **проверяющий результат** этого слайса напрямую (`Тогда ответ 201`). Background-шаг «пользователь `alice` зарегистрирован и залогинен» обеспечивает, что `users` и `credentials` уже содержат запись для `alice`.
- `Сценарий: Завершение входа` и `Сценарий: БД заблокирована при завершении входа` — используют `POST /v1/sessions` в **When**-шагах, но их **Then**-шаги относятся к слайсу 4 (`sessions-finish`). Корректность работы слайса 3 в них проверяется неявно: чтобы сценарии прошли, фаза 1 должна вернуть валидные `id` и `options` (включая `challenge` и `allowCredentials` с правильным credential ID).

В таблице `## Gherkin-mapping` ниже ведётся учёт **только Then-шагов первого сценария**.

## Зависимости от слайсов 1–2

- **Импорт типов:** `Handle`, `Challenge`, `ErrDBLocked`, `ErrDiskFull` (из S1); `User`, `Credential`, `UserID` (из S2).
- **Аддитивное расширение слайса 1** (см. `messages.md` → «Аддитивные расширения слайса 1 для слайса 3»):
  - экспортировать `GenerateChallenge() (Challenge, error)` — обёртка над уже существующей `generateChallenge`. Реиспользуется в S3 и далее в S4 без дублирования кода. Юнит-тесты S1 на `generateChallenge` остаются прежними и продолжают покрывать новую публичную функцию.
- **Аддитивное расширение слайса 2** (см. `messages.md` → «Аддитивные расширения слайса 2 для слайса 3»):
  - экспортировать `UserFromRow(rowID, rowHandle string, rowCreatedAtUnix int64) (User, error)` и `CredentialFromRow(rowCredentialID []byte, rowUserID string, rowPublicKey []byte, rowSignCount uint32, rowTransports string, rowCreatedAtUnix int64) (Credential, error)` — рехидраторы для метода `Store.LoadUserCredentials`;
  - экспортировать `UserIDFromString(s string) (UserID, error)` — рехидратор `UserID` (нужен `CredentialFromRow`, чтобы восстановить связь credential→user).
- **Чтение БД:** строки `users` (по `handle`) и `credentials` (по `user_id`) — обе таблицы созданы миграциями `0002` и `0003` слайса 2.
- **Запись БД:** строка в новую таблицу `login_sessions` (миграция `0005` — см. `infrastructure.md`).

## Дерево модулей

```
ингресс-адаптер: HTTP handler POST /v1/sessions
    ├── parse JSON body → SessionStartRequest
    └── (после головного модуля) format Response → 201 + JSON,
        либо error → 404 / 422 / 503 + Retry-After / 507 / 500
        │
головной модуль слайса: ProcessSessionStart
    ├── (1) NewSessionStartCommand(req)              → SessionStartCommand    [конструктор; делегирует NewHandle из S1]
    ├── (2) Store.LoadUserCredentials(handle)        → UserWithCredentials   [I/O-метод — SQLite read]
    ├── (3) GenerateChallenge()                       → Challenge              [реиспользуем S1]
    ├── (4) generateLoginSessionID()                  → LoginSessionID         [логика-лист]
    ├── (5) NewLoginSession(input)                    → LoginSession           [конструктор-сборщик]
    ├── (6) Store.PersistLoginSession(s)              → error                  [I/O-метод — SQLite write]
    ├── (7) buildRequestOptions(input)                → RequestOptions         [логика; dep: rpConfig.ID]
    └── (8) buildResponse(view)                        → SessionStartResponse   [логика]
```

Каждый узел — **один data-аргумент** (Шаг 3 скилла opus'а). Зависимости (`rpConfig`, `clock`, `Store`) вынесены через `Deps` и не считаются стрелками графа.

**Автономный I/O-объект `Store` (Шаг 6 скилла).** Узлы (2) и (6) — методы объекта `Store` слайса 3, инкапсулирующего `*sql.DB`. Головной модуль `ProcessSessionStart` знает только API объекта (методы `LoadUserCredentials`, `PersistLoginSession`), но не его внутреннюю зависимость. В `Deps` слайса — поле типа `*Store`, **не** сырой `*sql.DB`. См. `messages.md` → секция «I/O-объект слайса 3» и `infrastructure.md` → «Подключение слайса 3».

> **Технический долг S1/S2.** Реализованные слайсы 1 и 2 держат `*sql.DB` напрямую в `Deps` — нарушение этого же правила и `feedback_io_autonomous_store`. В scope S3 это **не правится** (правило «связанные правки — одна ветка» не позволяет расширять ветку дизайна S3 на рефакторинг S1/S2). Долг фиксируется в backlog отдельным тикетом и закрывается одним PR — либо вместе с дизайном S4 (когда S4 потребует свой `Store`), либо отдельной refactor-сессией.

**Подправило «подтип, не guard» (Шаг 3 скилла) — не применяется в S3.** В фазе 1 входа нет инварианта над свежезагруженной сущностью, который требовал бы конструктор подтипа: проверки «не истекла» появятся только в S4 (`FreshLoginSession`). Инвариант «у пользователя есть хотя бы один credential» инкапсулирован в I/O-возврате `UserWithCredentials` (см. контракт `Store.LoadUserCredentials`): если при загрузке credentials пуст — метод возвращает `ErrUserNotFound`, и `UserWithCredentials` с пустым списком не существует по построению. Это правильнее guard'а: код выше по пайпу не видит User'а без credentials.

## Псевдокод пайпа головного модуля

```
ProcessSessionStart(req: SessionStartRequest, deps: Deps)
    -> (SessionStartResponse, error):

    | NewSessionStartCommand(req)                              -> SessionStartCommand
    | deps.Store.LoadUserCredentials(cmd.Handle())              -> UserWithCredentials
    | GenerateChallenge()                                       -> Challenge
    | generateLoginSessionID()                                  -> LoginSessionID
    | input := NewLoginSessionInput{ ID, UserID: uwc.User().ID(),
                                       Challenge, TTL: deps.cfg.ChallengeTTL,
                                       Now: deps.clock.Now() }
    | NewLoginSession(input)                                    -> LoginSession
    | deps.Store.PersistLoginSession(session)                   -> error
    | optInput := BuildRequestOptionsInput{ Session: session,
                                             Credentials: uwc.Credentials() }
    | buildRequestOptions(optInput)                             -> RequestOptions        [dep: deps.cfg.RP.ID]
    | view := SessionStartView{ Session, Options }
    | buildResponse(view)                                        -> SessionStartResponse
```

Ошибки протекают через ранний `return SessionStartResponse{}, fmt.Errorf("step: %w", err)`. Сборка `input`, `optInput`, `view` — Go-литералы структур, не отдельные узлы графа.

## Контракты модулей

### `NewSessionStartCommand`

- **Сигнатура:** `NewSessionStartCommand(req SessionStartRequest) -> (SessionStartCommand, error)`
- **Input (data):** `req SessionStartRequest`
- **Dependencies (deps):** —
- **Что делает:** собирает доменную команду из DTO. Делегирует валидацию `NewHandle` (S1).
- **Антецедент:** `req.Handle` соответствует антецеденту `NewHandle`.
- **Консеквент:**
  - Success: команда с валидным `Handle`.
  - Failure: ошибка из `NewHandle`, обёрнутая в `fmt.Errorf("handle: %w", err)` — `ErrHandleEmpty`, `ErrHandleTooShort`, `ErrHandleTooLong`.

### `Store.LoadUserCredentials` (метод I/O-объекта)

- **Сигнатура:** `(s *Store) LoadUserCredentials(h Handle) -> (UserWithCredentials, error)`
- **Input (data):** `h Handle`
- **Dependencies (deps):** — (зависимость `*sql.DB` инкапсулирована внутри `Store`; головной модуль её не видит)
- **Что делает:** двумя последовательными SELECT (см. ниже) находит пользователя по handle и его credentials.
  ```
  SELECT id, handle, created_at FROM users WHERE handle = ?
    -> если нет строки → ErrUserNotFound
  SELECT credential_id, user_id, public_key, sign_count, transports, created_at
    FROM credentials WHERE user_id = ?
    -> если нет строк → ErrUserNotFound (см. ниже инвариант)
  ```
  Рехидрирует строки через `UserFromRow` и `CredentialFromRow` (S2 экспорты).
- **Антецедент:** миграции `0002`/`0003` применены; `h` валиден.
- **Консеквент:**
  - Success: `UserWithCredentials { user, credentials }`, где `len(credentials) >= 1` — инвариант агрегата (см. `messages.md`). Если credentials пуст — метод возвращает `ErrUserNotFound` вместо собирания агрегата с пустым списком.
  - Failure: `ErrUserNotFound` (нет user или нет credentials), `ErrDBLocked` (SQLITE_BUSY), низкоуровневые SQLite-ошибки (→ 500). `ErrDiskFull` для read не различается.

**Решение — оба SELECT в одном методе одного I/O-объекта.** По правилу Шага 6 «один режим работы с одной зависимостью» оба SELECT — чтения из одной БД, логически одно действие («достать пользователя со всеми его credentials»). Дробить на два метода смысла нет: оба возвращают одну ошибку (`ErrUserNotFound` / `ErrDBLocked`), компонентный сценарий покрывает их как одну точку. Альтернатива — JOIN одним запросом — даёт тот же контракт, но усложняет рехидратацию (смешанные строки `users × credentials`). Берём двухзапросный вариант ради простоты ремонтов.

**Решение — пустой credentials-список → ErrUserNotFound.** Для клиента «нет такого user» и «user есть, но войти нечем» неотличимы — оба требуют пройти регистрацию заново. Различение остаётся только в логах метода (`logger.Warn("user has no credentials", "user_id", id)`).

### `GenerateChallenge` (реиспользуем из S1)

- **Сигнатура:** `GenerateChallenge() -> (Challenge, error)` (экспорт `generateChallenge` из S1; идентичная семантика).
- Контракт описан в `slices/01-registrations-start.md`. Юнит-теста в S3 нет — он уже в S1.

### `generateLoginSessionID`

- **Сигнатура:** `generateLoginSessionID() -> LoginSessionID`
- **Input (data):** void
- **Dependencies (deps):** —
- **Что делает:** возвращает свежий UUID v4.
- **Антецедент:** —
- **Консеквент:** валидный UUID v4. Без `error` (`uuid.New()` не падает).

### `NewLoginSession`

- **Сигнатура:** `NewLoginSession(input NewLoginSessionInput) -> LoginSession`
- **Input (data):** `input NewLoginSessionInput { ID, UserID, Challenge, TTL, Now }` (см. `messages.md`)
- **Dependencies (deps):** —
- **Что делает:** собирает доменную сущность сессии входа.
- **Антецедент:**
  - `input.ID` — валидный UUID v4 (гарантировано `generateLoginSessionID`);
  - `input.UserID` — валидный UUID v4 (приходит из `UserWithCredentials.User().ID()`);
  - `input.Challenge` — 32 байта (гарантировано конструктором/`GenerateChallenge`);
  - `input.TTL > 0` (гарантируется конфигом `PASSKEY_CHALLENGE_TTL`);
  - `input.Now` — момент создания.
- **Консеквент:**
  - Success: `LoginSession` с `expiresAt = input.Now + input.TTL`.
  - Failure не возвращается — все доменные инварианты проверены конструкторами вложенных сущностей; `TTL > 0` гарантируется конфигом, `Now` не «проваливается».

### `Store.PersistLoginSession` (метод I/O-объекта)

- **Сигнатура:** `(s *Store) PersistLoginSession(ls LoginSession) -> error`
- **Input (data):** `ls LoginSession`
- **Dependencies (deps):** — (зависимость `*sql.DB` инкапсулирована внутри `Store`; головной модуль её не видит)
- **Что делает:** одной операцией INSERT добавляет запись в `login_sessions(id, user_id, challenge, expires_at)`.
- **Антецедент:**
  - `ls` — валидная доменная сущность (гарантировано `NewLoginSession`);
  - `ls.UserID()` соответствует существующей строке в `users` (гарантировано предшествующим шагом `Store.LoadUserCredentials`, который не вернул `ErrUserNotFound`);
  - миграция `0005` применена.
- **Консеквент:**
  - Success: запись в БД, `expires_at = ls.ExpiresAt()`.
  - Failure:
    - `ErrDBLocked` — `SQLITE_BUSY`, запись **не** создана. Маппится в 503 + `Retry-After: 1`.
    - `ErrDiskFull` — `SQLITE_FULL`, запись **не** создана. Маппится в 507.
    - другие низкоуровневые SQLite-ошибки оборачиваются в общую внутреннюю ошибку (→ 500). Не различаются по контракту.

### `buildRequestOptions`

- **Сигнатура:** `buildRequestOptions(input BuildRequestOptionsInput) -> RequestOptions`
- **Input (data):** `input BuildRequestOptionsInput { Session LoginSession, Credentials []Credential }`
- **Dependencies (deps):** `RPConfig` (нужно только поле `ID`)
- **Что делает:** собирает `PublicKeyCredentialRequestOptions` по схеме OpenAPI из доменной сессии и списка credentials пользователя.
- **Антецедент:** `input.Session` валидна; `input.Credentials` непустой (гарантировано инвариантом `UserWithCredentials`); `RPConfig.ID` непустой.
- **Консеквент:**
  - `Challenge = base64url(input.Session.Challenge())` (без padding).
  - `RpID = rpConfig.ID`.
  - `AllowCredentials` — массив `{type:"public-key", id: base64url(c.CredentialID())}` для каждого `c` из `input.Credentials`. Порядок — порядок строк из БД (детерминирован для теста: ORDER BY created_at в I/O).
  - `UserVerification = "preferred"` (по умолчанию OpenAPI).
  - `Timeout` — не задаётся (`omitempty`).
- Падений нет — чистая функция без `error`.

### `buildResponse`

- **Сигнатура:** `buildResponse(view SessionStartView) -> SessionStartResponse`
- **Input (data):** `view SessionStartView { Session LoginSession; Options RequestOptions }` — value-агрегатор для соблюдения «один data-аргумент».
- **Dependencies (deps):** —
- **Что делает:** упаковывает доменное представление ответа в DTO.
- **Антецедент:** аргументы валидны (Session — доменная сущность, Options — собран `buildRequestOptions`).
- **Консеквент:** `Response.ID = view.Session.ID().String()`, `Response.Options = view.Options`. Без `error`.

### Ингресс-адаптер: HTTP handler `POST /v1/sessions`

- **Что делает:**
  1. Парсит тело JSON в `SessionStartRequest`. Если JSON синтаксически невалиден — 400 с `error.code: VALIDATION_ERROR` (без обращения к головному модулю).
  2. Вызывает `ProcessSessionStart(req, deps)`.
  3. На Success: пишет `201 Created`, `Content-Type: application/json`, тело — `SessionStartResponse`.
  4. На Failure: маппит ошибки в HTTP-ответ (см. таблицу маппинга ниже).
- **Никакой бизнес-логики, никакой валидации полей** — только парсинг и маппинг.
- **Юнит-тестами не покрывается** — проверяется компонентным тестом слайса через реальный HTTP-вход (Шаг 8.1 скилла opus'а).

### Маппинг ошибок в ингресс-адаптере

| Класс ошибки                                                  | HTTP-статус | Заголовки           | Тело (`error.code`)         |
|---------------------------------------------------------------|-------------|---------------------|-----------------------------|
| Невалидный JSON (`json.Decoder.Decode` failed)                | 400         | —                   | `VALIDATION_ERROR`           |
| `ErrHandleEmpty` / `ErrHandleTooShort` / `ErrHandleTooLong`   | 422         | —                   | `VALIDATION_ERROR`           |
| `ErrUserNotFound`                                              | 404         | —                   | `NOT_FOUND`                  |
| `ErrDBLocked`                                                  | 503         | `Retry-After: 1`    | `db_locked`                  |
| `ErrDiskFull`                                                  | 507         | —                   | `db_disk_full`               |
| Любая другая (catastrophic из `GenerateChallenge`, неожиданные SQLite) | 500 | —                   | `INTERNAL_ERROR`             |

`ErrUserNotFound` маппится в **404 NOT_FOUND** (не отдельный код `UNKNOWN_USER`). Согласовано с маппингом `ErrSessionNotFound`/`ErrSessionExpired` в S2: для клиента поведение идентично — нужно пройти регистрацию или предъявить корректный handle. Различение в логах адаптера остаётся.

## Gherkin-mapping

Только Then-шаги, проверяющие **результат** этого слайса. Сценарии 2 и 3 (`Завершение входа`, `БД заблокирована при завершении входа`) проверяют слайс 4 (`sessions-finish`) и не входят в эту таблицу.

| Сценарий                          | Then-шаг       | Кто обеспечивает (узел графа / маппинг адаптера)                                                                               |
|-----------------------------------|----------------|--------------------------------------------------------------------------------------------------------------------------------|
| Создание challenge входа          | `Тогда ответ 201` | Узлы (1)–(8) головного модуля (Success-путь) → ингресс-адаптер: `201 Created` + JSON-сериализация `SessionStartResponse`        |

### Чек-лист сверки 8.5

1. [x] **Узел существует.** Узлы (1)–(8) описаны в дереве и в контрактах выше; ингресс-адаптер описан с маппингом.
2. [x] **Ветка соответствует.** Then `ответ 201` — Success-ветка пайпа; ингресс-адаптер пишет 201.
3. [x] **Формат ответа адаптера согласован.** OpenAPI декларирует 201 + `SessionInitResponse`; адаптер сериализует `SessionStartResponse` с теми же полями (`id`, `options`). Структура `RequestOptions` в `messages.md` соответствует схеме `PublicKeyCredentialRequestOptions` в OpenAPI.
4. [x] **Все Then покрыты.** В сценарии «Создание challenge входа» один Then-шаг, он покрыт.

`[x] Gherkin-mapping сверен.`

### Замечание о сценариях 2 и 3

Сценарии «Завершение входа» и «БД заблокирована при завершении входа» используют этот слайс через When-шаги. Чтобы они прошли:

- ингресс-адаптер должен возвращать **сериализуемый** `SessionStartResponse` (godog WebAuthn-степ парсит тело и собирает assertion из `options.challenge` и `options.allowCredentials[].id`);
- метод `Store.PersistLoginSession` должен **реально записать** challenge в БД, чтобы фаза 2 (слайс 4) могла его прочитать;
- метод `Store.LoadUserCredentials` должен **реально вернуть** хотя бы один credential, чтобы клиентский WebAuthn-степ нашёл свой ключ в `allowCredentials`.

Эти неявные требования покрыты контрактом (`buildResponse` строит правильный JSON; I/O имеет Success-ветку, гарантирующую запись и чтение), но **не** добавляют строк в таблицу Gherkin-mapping этого слайса — они формализуются как стрелки графа в `contracts-graph.md` и проверяются интеграционно через сценарии слайса 4.

### Замечание о других режимах отказа

OpenAPI декларирует на этом эндпоинте также 503 `db_locked` и 507 `db_disk_full`, но Gherkin-сценариев на эти режимы именно здесь **нет** (по сознательной раскладке в `slices.md` `db_locked` → слайс 4, `db_disk_full` → слайс 2). Адаптер обязан корректно маппить `ErrDBLocked` → 503 + `Retry-After: 1` и `ErrDiskFull` → 507, но проверяется это компонентными сценариями других слайсов (через тот же путь маппинга в общем хелпере или параллельной реализацией).

`ErrUserNotFound` (`NOT_FOUND`) — без компонентного сценария на этом эндпоинте; декларирован OpenAPI и проверяется юнит-тестами I/O косвенно (через тип `UserWithCredentials` с инвариантом непустого списка credentials) и контрактом OpenAPI.

## Юнит-тесты по формуле

`N_юнит_тестов = 1 (happy path) + Σ (ветки антецедента)` — **только модули логики и конструкторы** (Шаг 8.1 скилла opus'а: «I/O — трубы, юнитами не покрываются»; ингресс-адаптер — тоже).

| Модуль                            | Happy | Ветки антецедента                                              | Итого |
|-----------------------------------|-------|----------------------------------------------------------------|-------|
| `NewSessionStartCommand`          | 1     | (склейка; ветки `ErrHandle*` покрыты в S1.`NewHandle`, тут одна ошибка-обёртка) | 2 |
| `generateLoginSessionID`          | 1     | —                                                              | 1     |
| `NewLoginSession`                 | 1     | —                                                              | 1     |
| `buildRequestOptions`             | 1     | —                                                              | 1     |
| `buildResponse`                   | 1     | —                                                              | 1     |
| **Итого**                         |       |                                                                | **6** |

Что **не** в таблице (и почему):

- `GenerateChallenge` — реиспользуется из S1, юнит уже посчитан в карточке S1 (1 happy). Дублировать в S3 не нужно.
- `Store.LoadUserCredentials` — метод I/O-объекта, по сути труба. Юнитов нет. Success-путь проверяется компонентным сценарием **«Создание challenge входа»** (если read не дойдёт — фаза 1 не вернёт `allowCredentials`, и сценарии S4 «Завершение входа»/«БД заблокирована» упадут; именно эти сценарии S4 — фактический индикатор корректности read'а в S3). Failure-ветки `ErrUserNotFound`, `ErrDBLocked` — без отдельного компонентного сценария на этом эндпоинте; маппинг проверяется через адаптер.
- `Store.PersistLoginSession` — метод I/O-объекта. Success — happy-сценарий (если запись не дойдёт, S4 не сможет загрузить сессию по `id`). Failure-ветки `ErrDBLocked`/`ErrDiskFull` — без компонентного сценария на этом эндпоинте.
- **Ингресс-адаптер** — парсинг и маппинг ошибок, юнитов нет. Проверяется реальным HTTP-вызовом в компонентном сценарии.
- **Головной модуль** (`ProcessSessionStart`) — оркестратор-труба: каждый шаг вызывает ровно один дочерний модуль, ошибки I/O пробрасываются без трансформации. Юнит-тест над ним был бы интеграционным тестом. Корректность пайпа и все ветки ошибок доказываются компонентными сценариями через реальный HTTP-вход.

Замечания по покрытию:

- Покрытие 100% строк/веток модулей логики достигается этими 6 юнит-тестами.
- Головной модуль, I/O-модули и ингресс-адаптер не входят в метрику покрытия юнитов; их корректность доказана компонентными сценариями.

## Definition of Done слайса

Скопировано в тикет S3 в `backlog.md`:

- [ ] **аддитивные расширения слайса 1**: экспортирована `GenerateChallenge() (Challenge, error)`. Юнит-тесты S1 остаются зелёными (без изменения существующих тестов).
- [ ] **аддитивные расширения слайса 2**: экспортированы `UserFromRow`, `CredentialFromRow`, `UserIDFromString`. Юнит-тесты S2 остаются зелёными.
- [ ] миграция `internal/db/migrations/0005_login_sessions.sql` создана по `infrastructure.md`.
- [ ] ингресс-адаптер реализован: парсит тело в `SessionStartRequest`, без бизнес-валидации (HTTP handler в `internal/slice/sessions_start/`).
- [ ] конструкторы доменных структур (`NewSessionStartCommand`, `NewLoginSession`) реализованы; невалидные данные → структура не создаётся, возвращается доменная ошибка.
- [ ] модули логики (`generateLoginSessionID`, `buildRequestOptions`, `buildResponse`) реализованы, контракты выполнены.
- [ ] **I/O-объект `Store` реализован** как автономный объект, инкапсулирующий `*sql.DB`: тип `*Store` в пакете `internal/slice/sessions_start/`, конструктор `NewStore(db *sql.DB) *Store`, два метода:
  - `(s *Store) LoadUserCredentials(h Handle) (UserWithCredentials, error)`: два SELECT (или один JOIN) с маппингом `sql.ErrNoRows` / `len(credentials) == 0` → `ErrUserNotFound`, `SQLITE_BUSY` → `ErrDBLocked`. Возвращает агрегат `UserWithCredentials` с инвариантом непустого списка credentials. Рехидраторы — `UserFromRow`, `CredentialFromRow` (S2).
  - `(s *Store) PersistLoginSession(ls LoginSession) error`: INSERT в `login_sessions`; маппинг `SQLITE_BUSY` → `ErrDBLocked`, `SQLITE_FULL` → `ErrDiskFull`.
  - Голова `ProcessSessionStart` обращается к БД **только через эти два метода**; `*sql.DB` нигде кроме `Store` не светится в slice-пакете.
- [ ] головной модуль `ProcessSessionStart` реализован: пайп из 8 шагов, ранний возврат при ошибке через `fmt.Errorf("…: %w", err)`.
- [ ] инфраструктурный модуль расширен: `Deps` слайса 3 (`Store *Store`, `Clock`, `Logger`, `RP`, `ChallengeTTL` — **без** сырого `*sql.DB`); подключение `sessions_start.Register(mux, deps.SessionsStart)` в `cmd/api/main.go`; в `wire.go` создаётся `sessions_start.NewStore(db)` и пробрасывается в `Deps.Store`.
- [ ] слайс подключён через `sessions_start.Register(mux, deps)`: HTTP-роут `POST /v1/sessions` ведёт на ингресс-адаптер.
- [ ] юнит-тесты по формуле — **6 тестов** на модули логики и конструкторы (головной модуль, I/O-модули и ингресс-адаптер юнитами не покрываются).
- [ ] компонентный сценарий `Сценарий: Создание challenge входа` (`component-tests/features/sessions.feature`) зелёный.
- [ ] остальные сценарии в `sessions.feature`, `sessions-current.feature`, `users.feature` остаются красными в их Then-частях, но **не** ломаются по фазам S1+S2+S3 (When-шаги «отправляет POST /v1/sessions» работают и возвращают валидные `id`/`options.challenge`/`options.allowCredentials`).
- [ ] локальный CI зелёный (`go test ./...` для юнитов и `./component-tests/scripts/run-tests.sh` для компонентных, профиль `healthy`).
- [ ] `backlog.md` обновлён по каждому подтверждённому пункту (правило `AGENTS.md §10`).
- [ ] `docs/design/passkey-demo/devlog.md` дополнен блоком S3.
- [ ] PR создан, описание заполнено по шаблону Шага 8 скилла sonnet'а.
- [ ] PR смержен в main, CI на main зелёный.

## Ссылки на источники

- Скилл реализации: `skills/program-implementation/SKILL.md`
- Граф вызовов: `docs/design/passkey-demo/contracts-graph.md` Slice 03
- Gherkin-mapping: раздел `## Gherkin-mapping` выше
- Аддитивные расширения S1/S2: `docs/design/passkey-demo/messages.md` («Аддитивные расширения слайса 1 для слайса 3», «Аддитивные расширения слайса 2 для слайса 3»)
- Миграция `0005_login_sessions.sql`: `docs/design/passkey-demo/infrastructure.md`
