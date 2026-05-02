# Slice 04 — `sessions-finish`

## Идентификатор входа

`HTTP POST /v1/sessions/{id}/assertion`

## Что делает (в одну фразу)

Принимает assertion от аутентификатора, верифицирует подпись против challenge свежей login-сессии и публичного ключа credential'а пользователя, обновляет счётчик `sign_count`, выдаёт новую пару JWT (access + refresh) и удаляет login-сессию.

## OpenAPI

`api-specification/openapi.yaml`, `paths./sessions/{id}/assertion.post`. Контракт:

- Path: `id` — UUID login-сессии (создаётся слайсом 3).
- Запрос: `application/json` `AssertionRequest` (форма WebAuthn `AuthenticatorAssertionResponse`).
- Ответ 200: `TokenPair` (`access_token`, `refresh_token`).
- Возможные ошибки: 404 `NOT_FOUND` (сессия / credential не найдены, сессия истекла), 422 `VALIDATION_ERROR` / `ASSERTION_INVALID`, 503 `db_locked` (+ `Retry-After: 1`), 507 `db_disk_full`.

## Gherkin-сценарии слайса

`component-tests/features/sessions.feature`:

- `Сценарий: Завершение входа` — happy path. **Then-шаги** этого слайса: ответ 200, непустые `access_token` и `refresh_token`. When-шаги используют слайс 3 (создание challenge входа) — Then-шаги слайса 3 в карточке S3 уже зафиксированы.
- `Сценарий: БД заблокирована при завершении входа` — failure-режим `db_locked` на этой интеграции. Then-шаги: 503, заголовок `Retry-After`, JSON-поле `code` со значением `db_locked`.

Сценарий «Создание challenge входа» — слайс 3, в этой карточке не учитывается.

## Зависимости от слайсов 1–3

- **Импорт типов:**
  - из S1: `Challenge`, `ChallengeFromBytes`, `ErrDBLocked`, `ErrDiskFull`;
  - из S2: `User`, `UserID`, `UserIDFromString`, `Credential`, `CredentialFromRow`, `UserFromRow`, `JWTConfig`, `AccessToken`, `IssuedRefreshToken`, `IssuedTokenPair`, `GenerateTokenPairInput`, `BuildTokenPairView`, `TokenPair`;
  - из S3: `LoginSession`, `LoginSessionID`.
- **Аддитивное расширение слайса 2** (см. `messages.md` → «Аддитивные расширения слайса 2 для слайса 4»):
  - экспортировать `GenerateTokenPair(input GenerateTokenPairInput) (IssuedTokenPair, error)` — публичная обёртка над `generateTokenPair`;
  - экспортировать `BuildResponse(view BuildTokenPairView) TokenPair` — публичная обёртка над `buildResponse`.
- **Аддитивное расширение слайса 3** (см. `messages.md` → «Аддитивные расширения слайса 3 для слайса 4»):
  - экспортировать `LoginSessionIDFromString(s string) (LoginSessionID, error)` — рехидратор для path-параметра;
  - экспортировать `LoginSessionFromRow(rowID, rowUserID string, rowChallenge []byte, rowExpiresAtUnix int64) (LoginSession, error)` — рехидратор для метода `Store.LoadLoginSession`.
- **Чтение БД:** строки `login_sessions` (по `id`), `credentials` (по `credential_id`), `users` (по `id`) — все три таблицы созданы предыдущими миграциями.
- **Запись БД:** одна транзакция: `UPDATE credentials SET sign_count`, `INSERT INTO refresh_tokens`, `DELETE FROM login_sessions`.

## Дерево модулей

```
ингресс-адаптер: HTTP handler POST /v1/sessions/{id}/assertion
    ├── parse path-param + body → SessionFinishRequest
    └── (после головного модуля) format Response → 200 + JSON,
        либо error → 404 / 422 / 503 + Retry-After / 507 / 500
        │
головной модуль слайса: ProcessSessionFinish
    ├── (1) NewSessionFinishCommand(req)              → SessionFinishCommand
    │       ├── LoginSessionIDFromString(raw)         → LoginSessionID    [рехидратор S3]
    │       └── parseAssertion(body)                   → ParsedAssertion   [логика-лист; deps: protocol]
    ├── (2) Store.LoadLoginSession(id)                → LoginSession      [I/O-метод — SQLite read]
    ├── (3) NewFreshLoginSession(input)               → FreshLoginSession [конструктор подтипа]
    ├── (4) Store.LoadAssertionTarget(input)          → AssertionTarget   [I/O-метод — SQLite read, два SELECT'а]
    ├── (5) verifyAssertion(input)                    → VerifiedAssertion [логика; deps: rpConfig]
    ├── (6) GenerateTokenPair(input)                  → IssuedTokenPair   [импорт S2; deps: signer, jwtCfg]
    ├── (7) Store.FinishLogin(input)                  → error             [I/O-метод — SQLite write tx]
    └── (8) BuildResponse(view)                       → TokenPair         [импорт S2]
```

Каждый узел — **один data-аргумент** (Шаг 3 скилла). Зависимости (`Store`, `rpConfig`, `signer`, `jwtCfg`, `clock`) вынесены через `Deps` и не считаются стрелками графа.

**Автономный I/O-объект `Store` (Шаг 6 скилла).** Узлы (2), (4) и (7) — методы объекта `Store` слайса 4, инкапсулирующего `*sql.DB`. Головной модуль `ProcessSessionFinish` знает только API объекта (методы `LoadLoginSession`, `LoadAssertionTarget`, `FinishLogin`), но не его внутреннюю зависимость. В `Deps` слайса — поле типа `*Store`, **не** сырой `*sql.DB`. См. `messages.md` → секция «I/O-объект слайса 4» и `infrastructure.md` → «Подключение слайса 4 (S4)».

> **Технический долг S1/S2.** Реализованные слайсы 1 и 2 держат `*sql.DB` напрямую в `Deps` — нарушение этого же правила и `feedback_io_autonomous_store`. Тикет `refactor/s1-s2-store` (root `backlog.md`) **должен быть закрыт до начала реализации S4**: иначе после мержа S4 в `wire.go` окажется смешанный стиль (S3/S4 — через `*Store`, S1/S2 — через сырой `*sql.DB`). Альтернатива — закрыть техдолг одним PR вместе с реализацией S4 — нарушает правило «связанные правки — одна ветка», поэтому не предпочтительно.

Применение **подправила «подтип, не guard»** (Шаг 3 скилла): инвариант «сессия не истекла» оформлен как конструктор подтипа `NewFreshLoginSession` (узел 3), не как guard `checkSessionFresh(...) -> ()`. Дальнейшие шаги (4, 5) принимают `FreshLoginSession`, не сырой `LoginSession`. Дополнительно, инвариант «credential принадлежит пользователю из login-сессии» инкапсулирован в I/O-возврате `AssertionTarget` (см. контракт `Store.LoadAssertionTarget`): если credential не найден или принадлежит другому user — метод возвращает `ErrCredentialNotFound`, и `AssertionTarget` с «чужим» credential не существует по построению. Это правильнее guard-функции `checkCredentialOwnership(target) -> error`: код выше по пайпу (узел 5) не видит «не свой» credential.

## Псевдокод пайпа головного модуля

```
ProcessSessionFinish(req: SessionFinishRequest, deps: Deps)
    -> (TokenPair, error):

    | NewSessionFinishCommand(req)                                 -> SessionFinishCommand
    | deps.Store.LoadLoginSession(cmd.LoginSessionID())             -> LoginSession
    | freshInput := NewFreshLoginSessionInput{ Session: session, Now: deps.clock.Now() }
    | NewFreshLoginSession(freshInput)                              -> FreshLoginSession
    | targetInput := LoadAssertionTargetInput{ UserID: fresh.UserID(),
                                                CredentialID: cmd.Parsed().CredentialID() }
    | deps.Store.LoadAssertionTarget(targetInput)                   -> AssertionTarget
    | verifyInput := AssertionVerification{ Fresh: fresh,
                                             Parsed: cmd.Parsed(),
                                             Target: target }
    | verifyAssertion(verifyInput)                                  -> VerifiedAssertion    [dep: deps.cfg.RP]
    | tokenInput := GenerateTokenPairInput{ User: target.User(),
                                              Now: deps.clock.Now() }
    | GenerateTokenPair(tokenInput)                                 -> IssuedTokenPair      [deps: deps.signer, deps.jwt]
    | finishInput := FinishLoginInput{ Credential:       target.Credential(),
                                         NewSignCount:    verified.NewSignCount(),
                                         RefreshTokenHash: issued.Refresh.Hash(),
                                         RefreshExpiresAt: issued.Refresh.ExpiresAt(),
                                         LoginSessionID:  fresh.ID() }
    | deps.Store.FinishLogin(finishInput)                           -> error
    | view := BuildTokenPairView{ Access: issued.Access, Refresh: issued.Refresh }
    | BuildResponse(view)                                            -> TokenPair
```

Ошибки протекают через ранний `return TokenPair{}, fmt.Errorf("step: %w", err)`. Сборка `freshInput`, `targetInput`, `verifyInput`, `tokenInput`, `finishInput`, `view` — Go-литералы структур, не отдельные узлы графа. Шагов в пайпе — 10 (с учётом `BuildResponse`); фактических узлов графа — 8 (1)–(8). Это в диапазоне 5–10 (Шаг 3 скилла).

## Контракты модулей

### `parseAssertion`

- **Сигнатура:** `parseAssertion(raw []byte) -> (ParsedAssertion, error)`
- **Input (data):** `raw []byte` — сырое тело запроса.
- **Dependencies (deps):** — (`go-webauthn/protocol` — компилируемая зависимость, не runtime dep)
- **Что делает:** парсит JSON-тело `AssertionRequest` через `protocol.ParseCredentialRequestResponseBody`. Заполняет распарсенную структуру; **не верифицирует** ни challenge, ни origin, ни подпись.
- **Антецедент:** `raw` — синтаксически валидный JSON, соответствующий схеме `AssertionRequest` (`id`, `rawId`, `type`, `response.{clientDataJSON, authenticatorData, signature, userHandle?}`).
- **Консеквент:**
  - Success: `ParsedAssertion`, обёртка над `*protocol.ParsedCredentialAssertionData`.
  - Failure: `ErrAssertionParse` (некорректный JSON, отсутствуют обязательные поля, `authenticatorData` не парсится, `clientDataJSON` не base64url).

### `NewSessionFinishCommand`

- **Сигнатура:** `NewSessionFinishCommand(req SessionFinishRequest) -> (SessionFinishCommand, error)`
- **Input (data):** `req SessionFinishRequest`
- **Dependencies (deps):** —
- **Что делает:** собирает доменную команду из DTO. Делегирует `LoginSessionIDFromString` (S3 рехидратор) и `parseAssertion`.
- **Антецедент:** `req.LoginSessionIDRaw` — UUID-строка из path-параметра; `req.AssertionBody` — соответствует антецеденту `parseAssertion`.
- **Консеквент:**
  - Success: команда с валидным `LoginSessionID` и распарсенным `ParsedAssertion`.
  - Failure: `ErrInvalidLoginSessionID` (UUID не парсится) или `ErrAssertionParse` (тело не парсится), обёрнутые через `fmt.Errorf("...: %w", err)`.

### `Store.LoadLoginSession` (метод I/O-объекта)

- **Сигнатура:** `(s *Store) LoadLoginSession(id LoginSessionID) -> (LoginSession, error)`
- **Input (data):** `id LoginSessionID`
- **Dependencies (deps):** — (зависимость `*sql.DB` инкапсулирована внутри `Store`; головной модуль её не видит)
- **Что делает:** `SELECT id, user_id, challenge, expires_at FROM login_sessions WHERE id = ?`. Если строка найдена — рехидрирует через `LoginSessionFromRow` (S3 экспорт). Если нет — `ErrLoginSessionNotFound`.
- **Антецедент:** миграция `0005_login_sessions.sql` применена; `id` валиден.
- **Консеквент:**
  - Success: `LoginSession`, восстановленная из БД.
  - Failure: `ErrLoginSessionNotFound` (no row), `ErrDBLocked` (`SQLITE_BUSY`), низкоуровневые SQLite-ошибки (→ 500). `ErrDiskFull` для read-операции не различается.

### `NewFreshLoginSession`

- **Сигнатура:** `NewFreshLoginSession(input NewFreshLoginSessionInput) -> (FreshLoginSession, error)`
- **Input (data):** `input NewFreshLoginSessionInput { Session, Now }`
- **Dependencies (deps):** —
- **Что делает:** конструктор подтипа, инвариант «сессия не истекла на момент конструкции».
- **Антецедент:** `input.Session` — валидная сущность из I/O; `input.Now` — момент.
- **Консеквент:**
  - Success: `FreshLoginSession{ session: input.Session }` при `input.Now < input.Session.ExpiresAt()`.
  - Failure: `ErrLoginSessionExpired` при `input.Now >= input.Session.ExpiresAt()`.

### `Store.LoadAssertionTarget` (метод I/O-объекта)

- **Сигнатура:** `(s *Store) LoadAssertionTarget(input LoadAssertionTargetInput) -> (AssertionTarget, error)`
- **Input (data):** `input LoadAssertionTargetInput { UserID, CredentialID }`
- **Dependencies (deps):** — (зависимость `*sql.DB` инкапсулирована внутри `Store`; головной модуль её не видит)
- **Что делает:** двумя SELECT'ами и одной in-memory проверкой собирает агрегат «`User` + его `Credential` по предъявленному `credential_id`».
  ```
  SELECT credential_id, user_id, public_key, sign_count, transports, created_at
    FROM credentials WHERE credential_id = ?
    -> если нет строки → ErrCredentialNotFound
    -> если row.user_id != input.UserID → ErrCredentialNotFound (логируется как credential mismatch)
  SELECT id, handle, created_at FROM users WHERE id = ?
    -> если нет строки → ErrCredentialNotFound (user удалён, credential «осиротел»)
  ```
  Рехидрирует через `CredentialFromRow` и `UserFromRow` (S2 экспорты).
- **Антецедент:** миграции `0002`/`0003` применены; `input.UserID` валиден (приходит из `FreshLoginSession.UserID()`); `input.CredentialID` — байты из распарсенного assertion (без проверки длины — реальный credentialID из аутентификатора может быть переменной длины).
- **Консеквент:**
  - Success: `AssertionTarget { user, credential }` с инвариантом `credential.UserID() == user.ID()` — гарантирован тем, что `Store.LoadAssertionTarget` единственный конструктор `AssertionTarget` и проверяет принадлежность перед сборкой. Структура с «чужим» credential или с user ≠ owner_of_credential по построению не существует.
  - Failure: `ErrCredentialNotFound` (нет credential, или credential принадлежит другому user, или user удалён), `ErrDBLocked` (`SQLITE_BUSY`), низкоуровневые SQLite-ошибки (→ 500). `ErrDiskFull` для read-операции не различается.

**Решение — оба SELECT в одном методе.** По правилу Шага 6 «один режим работы с одной зависимостью» оба SELECT — чтения из одной БД, логически одно действие («достать credential и его владельца, проверив принадлежность»). Дробить на два метода смысла нет: оба возвращают одну ошибку (`ErrCredentialNotFound` / `ErrDBLocked`), компонентный сценарий покрывает их как одну точку.

**Решение — единый класс ошибки `ErrCredentialNotFound` для трёх случаев.** «Нет credential», «credential чужой» и «user удалён» — для клиента поведение идентично (404 NOT_FOUND, нужно начать фазу 1 заново). Различение остаётся только в логах метода (`logger.Warn("credential mismatch", "user_id", uid, "cred_user_id", cuid)` / `logger.Warn("orphan credential", "user_id", uid)`).

### `verifyAssertion`

- **Сигнатура:** `verifyAssertion(input AssertionVerification) -> (VerifiedAssertion, error)`
- **Input (data):** `input AssertionVerification { Fresh, Parsed, Target }`
- **Dependencies (deps):** `RPConfig` (нужны `ID` и `Origin`)
- **Что делает:** делегирует `protocol.ParsedCredentialAssertionData.Verify(storedChallenge, rpID, rpOrigins, ..., requireUV, credentialPublicKey, ...)` из go-webauthn. Передаёт challenge из `Fresh.Challenge()`, публичный ключ — из `Target.Credential().PublicKey()`, ожидаемый `rpID` и `rpOrigins` — из `RPConfig`. На успехе извлекает обновлённый `signCount` из распарсенной структуры (`AuthenticatorData.Counter` после Verify) и собирает `VerifiedAssertion`.
- **Антецедент:** `input.Fresh` — non-expired сессия (инвариант в типе); `input.Parsed` — синтаксически распарсенный assertion; `input.Target` — credential «свой» (инвариант в типе); `RPConfig.ID` и `RPConfig.Origin` непустые.
- **Консеквент:**
  - Success: `VerifiedAssertion { newSignCount }` — счётчик из authenticatorData после прохождения Verify.
  - Failure: `ErrAssertionInvalid` — любой провал `Verify` (challenge mismatch, RP ID hash mismatch, origin mismatch, signature invalid, sign-count clone-warning, user-verification-required not satisfied, неподдерживаемая алгоритмика).

### `GenerateTokenPair` (импорт S2)

- **Сигнатура:** `GenerateTokenPair(input GenerateTokenPairInput) -> (IssuedTokenPair, error)` (S2 экспорт; идентичная семантика с пакетной `generateTokenPair`).
- Контракт описан в `slices/02-registrations-finish.md` и `messages.md` → секция «Структуры слайса 2». Юнит-теста в S4 нет — он уже в S2.

### `Store.FinishLogin` (метод I/O-объекта)

- **Сигнатура:** `(s *Store) FinishLogin(input FinishLoginInput) -> error`
- **Input (data):** `input FinishLoginInput { Credential, NewSignCount, RefreshTokenHash, RefreshExpiresAt, LoginSessionID }`
- **Dependencies (deps):** — (зависимость `*sql.DB` инкапсулирована внутри `Store`; головной модуль её не видит)
- **Что делает:** одна транзакция:
  ```
  BEGIN
    UPDATE credentials
       SET sign_count = ?
     WHERE credential_id = ?
    INSERT INTO refresh_tokens (token_hash, user_id, expires_at)
      VALUES (?, ?, ?)
    DELETE FROM login_sessions WHERE id = ?
  COMMIT
  ```
  При любой ошибке — ROLLBACK. `user_id` для INSERT берётся из `input.Credential.UserID()` — `Credential` хранит свою связь с `User` (см. messages.md). `credential_id` для UPDATE — из `input.Credential.CredentialID()`.
- **Антецедент:** все доменные значения валидны (`input.Credential` — успех `Store.LoadAssertionTarget`; `input.NewSignCount` — успех `verifyAssertion`; `input.RefreshTokenHash`/`input.RefreshExpiresAt` — успех `GenerateTokenPair`; `input.LoginSessionID` — успех `NewFreshLoginSession`); миграции `0003`-`0005` применены.
- **Консеквент:**
  - Success: tx закоммичена, все 3 операции применены: `sign_count` обновлён, refresh-токен хранится, login-сессия удалена (повторное использование `id` невозможно).
  - Failure:
    - `ErrDBLocked` — `SQLITE_BUSY` → 503 `db_locked` + `Retry-After: 1`. Tx **не** закоммичена; ни одно из трёх изменений не применено (включая `sign_count`).
    - `ErrDiskFull` — `SQLITE_FULL` → 507 `db_disk_full`. Tx не закоммичена.
    - другие — обёрнуты в общую внутреннюю ошибку (→ 500 `INTERNAL_ERROR`).

`UPDATE credentials SET sign_count` атомарно с DELETE login-сессии — критично для replay-защиты: если бы UPDATE прошёл без DELETE, при перезапросе тот же assertion вернул бы 404 `ErrLoginSessionNotFound`, но `sign_count` уже инкрементирован, и следующая нормальная попытка входа пройдёт Verify (новый assertion даст бóльший counter). Если бы DELETE прошёл без UPDATE — следующая попытка входа использует устаревший `sign_count`, и Verify провалится с clone-warning при ретрае.

### `BuildResponse` (импорт S2)

- **Сигнатура:** `BuildResponse(view BuildTokenPairView) -> TokenPair` (S2 экспорт; идентичная семантика с пакетной `buildResponse`).
- Контракт описан в `slices/02-registrations-finish.md` и `messages.md` → секция «Структуры слайса 2». Юнит-теста в S4 нет — он уже в S2.

### Ингресс-адаптер: HTTP handler `POST /v1/sessions/{id}/assertion`

- **Что делает:**
  1. Извлекает path-параметр `id` (chi router `chi.URLParam`).
  2. Читает тело запроса в `[]byte`.
  3. Собирает `SessionFinishRequest{ LoginSessionIDRaw, AssertionBody }`.
  4. Вызывает `ProcessSessionFinish(req, deps)`.
  5. На Success: пишет `200 OK`, `Content-Type: application/json`, тело — `TokenPair`.
  6. На Failure: маппит ошибки в HTTP-ответ (см. таблицу маппинга ниже).
- **Никакой бизнес-логики** — только парсинг path/body и маппинг.
- **Юнит-тестами не покрывается** — проверяется компонентным тестом слайса через реальный HTTP-вход.

### Маппинг ошибок в ингресс-адаптере

| Класс ошибки                                                  | HTTP-статус | Заголовки           | Тело (`error.code`)         |
|---------------------------------------------------------------|-------------|---------------------|-----------------------------|
| `ErrInvalidLoginSessionID`                                    | 422          | —                   | `VALIDATION_ERROR`           |
| `ErrAssertionParse`                                            | 422          | —                   | `VALIDATION_ERROR`           |
| `ErrLoginSessionNotFound`                                      | 404          | —                   | `NOT_FOUND`                  |
| `ErrLoginSessionExpired`                                       | 404          | —                   | `NOT_FOUND`                  |
| `ErrCredentialNotFound`                                        | 404          | —                   | `NOT_FOUND`                  |
| `ErrAssertionInvalid`                                          | 422          | —                   | `ASSERTION_INVALID`          |
| `ErrDBLocked`                                                  | 503          | `Retry-After: 1`    | `db_locked`                  |
| `ErrDiskFull`                                                  | 507          | —                   | `db_disk_full`               |
| Любая другая (catastrophic из `GenerateTokenPair`, неожиданные SQLite) | 500 | —                   | `INTERNAL_ERROR`             |

Истёкшая, несуществующая login-сессия и не найденный credential **все три** маппятся в 404: для клиента поведение идентично — нужно начать фазу 1 заново. Различение остаётся только в логах адаптера.

`ErrAssertionInvalid` и `ErrInvalidLoginSessionID`/`ErrAssertionParse` все маппятся в 422, но с разными `error.code`: `ASSERTION_INVALID` vs `VALIDATION_ERROR`. По схеме `ErrorResponse` в OpenAPI оба валидны.

## Gherkin-mapping

| Сценарий                                          | Then-шаг                                                         | Кто обеспечивает (узел графа / маппинг адаптера)                                                            |
|---------------------------------------------------|------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------|
| Завершение входа                                   | `Тогда ответ 200`                                                 | Узлы (1)–(8) Success-путь → ингресс-адаптер: `200 OK` + JSON-сериализация `TokenPair`                       |
| Завершение входа                                   | `И ответ содержит непустое JSON-поле access_token`               | (6) `GenerateTokenPair` (поле `Access.Value()`) → (8) `BuildResponse` → ингресс-адаптер                     |
| Завершение входа                                   | `И ответ содержит непустое JSON-поле refresh_token`              | (6) `GenerateTokenPair` (поле `Refresh.Plaintext()`) → (8) `BuildResponse` → ингресс-адаптер                |
| БД заблокирована при завершении входа              | `Тогда ответ 503`                                                 | I/O-узел (2) `Store.LoadLoginSession` Failure: `ErrDBLocked` (первый I/O в пайпе — упирается в SQLite EXCLUSIVE-lock первым); либо (4)/(7) Failure: `ErrDBLocked`, если бы lock возник позже → ингресс-адаптер: маппинг `ErrDBLocked` → 503 |
| БД заблокирована при завершении входа              | `И ответ содержит заголовок Retry-After`                          | ингресс-адаптер: маппинг `ErrDBLocked` → `Retry-After: 1`                                                    |
| БД заблокирована при завершении входа              | `И ответ содержит JSON-поле code со значением "db_locked"`        | ингресс-адаптер: маппинг `ErrDBLocked` → тело `{"code":"db_locked",...}`                                     |

### Чек-лист сверки 8.5

1. [x] **Узел существует.** Узлы (1)–(8) описаны в дереве и в контрактах выше; ингресс-адаптер описан с маппингом.
2. [x] **Ветка соответствует.** Then `200` — Success-путь; Then `503` — Failure-путь любого из I/O-узлов (2)/(4)/(7) с `ErrDBLocked`. `access_token`/`refresh_token` — Success-выход шага (6), сериализованный шагом (8).
3. [x] **Формат ответа адаптера согласован.** OpenAPI декларирует 200 + `TokenPair`; адаптер сериализует `TokenPair{access_token, refresh_token}` с теми же полями. 503 + `Retry-After: 1` + `{"code":"db_locked",...}` совпадает с README «Карта режимов отказа».
4. [x] **Все Then покрыты.** В сценарии «Завершение входа» 3 Then-шага, в «БД заблокирована при завершении входа» — 3; всего 6, все покрыты.

`[x] Gherkin-mapping сверен.`

### Какой именно I/O-узел «ловит» SQLITE_BUSY в db_locked

Compose-сценарий «БД заблокирована при завершении входа» открывает `BEGIN EXCLUSIVE TRANSACTION` в раннере **до** прихода запроса в SUT. SQLite EXCLUSIVE-lock блокирует и read'ы, и write'ы — поэтому первый же I/O-узел S4 (`Store.LoadLoginSession`, шаг 2) упрётся в `SQLITE_BUSY` и вернёт `ErrDBLocked`. Шаги (4) и (7) в этом сценарии не достигаются. Тем не менее **все три I/O-узла** обязаны корректно маппить `SQLITE_BUSY → ErrDBLocked` — это часть декларированного OpenAPI-контракта (`db_locked` декларирован на этом эндпоинте).

В таблице Gherkin-mapping выше Then «503» формально привязан к ветке `ErrDBLocked` любого из узлов (2)/(4)/(7). В компонентном сценарии фактически срабатывает (2). Если в будущем лок будет выставляться позднее (например, между read'ом и write'ом другим раннером — что в текущем шаблоне не реализовано), сценарий по-прежнему останется зелёным благодаря симметричному маппингу в (4) и (7).

### Замечание о других режимах отказа

OpenAPI декларирует на этом эндпоинте также 507 `db_disk_full`, но Gherkin-сценария на `db_disk_full` именно здесь нет (по раскладке в `slices.md` `db_disk_full` — слайс 2). Адаптер обязан корректно маппить `ErrDiskFull` → 507, но проверяется это компонентным тестом слайса 2 (через тот же путь маппинга в общем хелпере, либо параллельной реализацией).

`ErrLoginSessionNotFound`, `ErrLoginSessionExpired`, `ErrCredentialNotFound`, `ErrAssertionInvalid`, `ErrInvalidLoginSessionID`, `ErrAssertionParse` — доменные ошибки, проверяются юнит-тестами модулей логики (`NewSessionFinishCommand`, `NewFreshLoginSession`, `verifyAssertion`) и контрактом OpenAPI; компонентных сценариев на них нет (по раскладке: `sessions.feature` содержит только happy + `db_locked` для эндпоинта S4).

## Юнит-тесты по формуле

`N_юнит_тестов = 1 (happy path) + Σ (ветки антецедента)` — **только модули логики и конструкторы** (Шаг 8.1: «I/O — трубы, юнитами не покрываются»; ингресс-адаптер — тоже).

| Модуль                            | Happy | Ветки антецедента                                                                  | Итого |
|-----------------------------------|-------|------------------------------------------------------------------------------------|-------|
| `parseAssertion`                  | 1     | невалидный JSON, отсутствует поле `response`, `authenticatorData` не парсится       | 4     |
| `NewSessionFinishCommand`         | 1     | `ErrInvalidLoginSessionID` (UUID), `ErrAssertionParse` (склейка)                    | 3     |
| `NewFreshLoginSession`            | 1     | `now >= expiresAt` → `ErrLoginSessionExpired`                                        | 2     |
| `verifyAssertion`                 | 1     | `Verify` вернул ошибку (мутация подписи в virtualwebauthn-сценарии)                  | 2     |
| **Итого**                         |       |                                                                                    | **11**|

Для honest-теста `verifyAssertion` happy path используется `github.com/descope/virtualwebauthn` (генерирует валидные assertion-данные на стороне теста и подписывает приватным ключом тестового аутентификатора). Та же библиотека уже используется в `component-tests/` и в test-deps S2 — повторно подключать не нужно. Альтернатива «1 happy через мок Verify» нарушает принцип «без моков в тестах» (`feedback_no_mocks`), поэтому отвергнута.

Что **не** в таблице (и почему):

- `Store.LoadLoginSession` — метод I/O-объекта, труба. Юнитов нет. Success-путь проверяется компонентным сценарием **«Завершение входа»** (если read не дойдёт — пайп упадёт на шаге 2, Then 200 не выполнится). Failure-ветка `ErrDBLocked` проверяется компонентным сценарием **«БД заблокирована при завершении входа»** (фактический ловец SQLITE_BUSY в этом сценарии). `ErrLoginSessionNotFound` — без отдельного компонентного сценария на этом эндпоинте; маппинг проверяется через адаптер.
- `Store.LoadAssertionTarget` — метод I/O-объекта, труба. Success — happy-сценарий. Failure-ветки `ErrCredentialNotFound`/`ErrDBLocked` — без отдельного компонентного сценария на этом эндпоинте; в `db_locked` сценарии до этого узла обычно не доходит (см. выше).
- `Store.FinishLogin` — метод I/O-объекта. Success — happy-сценарий (если запись не дойдёт, токены не вернутся). Failure-ветки `ErrDBLocked`/`ErrDiskFull` — без отдельного компонентного сценария на этом эндпоинте (`db_disk_full` отдан S2; `db_locked` фактически ловится узлом 2).
- `GenerateTokenPair`, `BuildResponse` — импорт S2; юнит-тесты уже посчитаны в карточке S2 (1 для `generateTokenPair`, 1 для `buildResponse`). Дублирование не нужно.
- **Ингресс-адаптер** — парсинг path/body и маппинг, юнитов нет. Реальные HTTP-вызовы в обоих компонентных сценариях.
- **Головной модуль** `ProcessSessionFinish` — оркестратор-труба: линейный пайп из 8 узлов, ошибки I/O пробрасываются без трансформации. Юнит-тест над ним был бы интеграционным тестом. Корректность пайпа и все ветки ошибок доказываются компонентными сценариями через реальный HTTP-вход.

Замечания по покрытию:

- 100% строк/веток модулей логики достигается этими 11 юнит-тестами.
- Honest-тест `verifyAssertion` использует `virtualwebauthn` для генерации валидных assertion-данных — не мок.

## Definition of Done слайса

Скопировано в тикет S4 в `backlog.md`:

- [ ] **аддитивные расширения слайса 2**: экспортированы `GenerateTokenPair`, `BuildResponse`. Юнит-тесты S2 остаются зелёными (без изменения существующих тестов; внутренние пакетные функции переименованы в публичные обёртки или сохранены, тесты вызывают публичные имена).
- [ ] **аддитивные расширения слайса 3**: экспортированы `LoginSessionIDFromString`, `LoginSessionFromRow`. Юнит-тесты S3 остаются зелёными.
- [ ] **техдолг S1/S2 (Store-объект) закрыт** — `refactor/s1-s2-store` (тикет в root `backlog.md`) смержен в main **до** начала реализации S4. Иначе `wire.go` будет содержать смешанный стиль `Deps` (S3/S4 — через `*Store`, S1/S2 — через сырой `*sql.DB`).
- [ ] ингресс-адаптер реализован: парсит path-параметр и тело в `SessionFinishRequest`, без бизнес-валидации (HTTP handler в `internal/slice/sessions_finish/`).
- [ ] конструкторы доменных структур (`NewSessionFinishCommand`, `NewFreshLoginSession`) реализованы; невалидные данные → структура не создаётся, возвращается доменная ошибка.
- [ ] модули логики (`parseAssertion`, `verifyAssertion`) реализованы, контракты выполнены.
- [ ] **I/O-объект `Store` реализован** как автономный объект, инкапсулирующий `*sql.DB`: тип `*Store` в пакете `internal/slice/sessions_finish/`, конструктор `NewStore(db *sql.DB) *Store`, три метода:
  - `(s *Store) LoadLoginSession(id LoginSessionID) (LoginSession, error)`: SELECT по id, рехидратор `LoginSessionFromRow`; маппинг `sql.ErrNoRows` → `ErrLoginSessionNotFound`, `SQLITE_BUSY` → `ErrDBLocked`.
  - `(s *Store) LoadAssertionTarget(input LoadAssertionTargetInput) (AssertionTarget, error)`: SELECT credential по `credential_id` → in-memory проверка `user_id == input.UserID` → SELECT user по `id`; маппинг `sql.ErrNoRows` (на любом из SELECT'ов) или mismatch → `ErrCredentialNotFound`, `SQLITE_BUSY` → `ErrDBLocked`. Рехидраторы — `CredentialFromRow`, `UserFromRow` (S2).
  - `(s *Store) FinishLogin(input FinishLoginInput) error`: атомарная транзакция (3 операции: UPDATE credentials + INSERT refresh_tokens + DELETE login_sessions), откат при любой ошибке; маппинг `SQLITE_BUSY` → `ErrDBLocked`, `SQLITE_FULL` → `ErrDiskFull`.
  - Голова `ProcessSessionFinish` обращается к БД **только через эти три метода**; `*sql.DB` нигде кроме `Store` не светится в slice-пакете.
- [ ] головной модуль `ProcessSessionFinish` реализован: пайп из 8 шагов (10 строк с импортированными `GenerateTokenPair` и `BuildResponse`), ранний возврат при ошибке через `fmt.Errorf("…: %w", err)`.
- [ ] инфраструктурный модуль расширен: `Deps` слайса 4 (`Store *Store`, `Clock`, `Logger`, `RP` (S1, нужны `ID` и `Origin`), `JWT` (S2), `Signer` ed25519.PrivateKey — **без** сырого `*sql.DB`); подключение `sessions_finish.Register(mux, deps.SessionsFinish)` в `cmd/api/main.go`; в `wire.go` создаётся `sessions_finish.NewStore(db)` и пробрасывается в `Deps.Store`.
- [ ] слайс подключён через `sessions_finish.Register(mux, deps)`: HTTP-роут `POST /v1/sessions/{id}/assertion` ведёт на ингресс-адаптер.
- [ ] **юнит-тесты по формуле написаны и зелёные** — `go test ./...` проходит. **11 новых тестов** на модули логики и конструкторы S4 (`parseAssertion`, `NewSessionFinishCommand`, `NewFreshLoginSession`, `verifyAssertion`); головной модуль, I/O-модули и ингресс-адаптер юнитами не покрываются. `verifyAssertion` honest-тестируется через `virtualwebauthn`. Юниты S1/S2/S3 остаются зелёными после аддитивных расширений (`GenerateTokenPair`, `BuildResponse`, `LoginSessionIDFromString`, `LoginSessionFromRow`).
- [ ] **компонентные тесты, профиль `healthy`, зелёные** — `./component-tests/scripts/run-tests.sh healthy` проходит. Новые зелёные сценарии: `Сценарий: Завершение входа`, `Сценарий: БД заблокирована при завершении входа` (`component-tests/features/sessions.feature`). Ранее зелёные сценарии S1/S2/S3 продолжают проходить.
- [ ] **компонентные тесты, профиль `disk-full`, зелёные** — `./component-tests/scripts/run-tests.sh disk-full` проходит. Regression-проверка: `Сценарий: Диск переполнен при завершении регистрации` (`registrations.feature`) из S2 продолжает проходить — изменения S4 не должны ломать `db_disk_full` маппинг.
- [ ] сценарии в `sessions-current.feature`/`users.feature` остаются красными в их Then-частях (S5/S6 ещё не реализованы), но **не** ломаются на When-шагах S1–S4 — `POST /v1/sessions/{id}/assertion` возвращает валидные `access_token`/`refresh_token`, которые используются как Bearer-токен в S5/S6.
- [ ] `backlog.md` обновлён по каждому подтверждённому пункту (правило `AGENTS.md §10`).
- [ ] `docs/design/passkey-demo/devlog.md` дополнен блоком S4.
- [ ] PR создан, описание заполнено по шаблону Шага 8 скилла sonnet'а.
- [ ] PR смержен в main, CI на main зелёный.

## Ссылки на источники

- Скилл реализации: `skills/program-implementation/SKILL.md`
- Граф вызовов: `docs/design/passkey-demo/contracts-graph.md` Slice 04
- Gherkin-mapping: раздел `## Gherkin-mapping` выше
- Аддитивные расширения S2/S3: `docs/design/passkey-demo/messages.md` («Аддитивные расширения слайса 2 для слайса 4», «Аддитивные расширения слайса 3 для слайса 4»)
- Подключение слайса 4: `docs/design/passkey-demo/infrastructure.md` → «Подключение слайса 4 (S4)»
- Подправило «подтип, не guard»: `skills/program-design/SKILL.md` Шаг 3 (применено в узле (3) `NewFreshLoginSession`; инвариант `AssertionTarget` инкапсулирован в I/O-возврате — то же решение, что в S3 для `UserWithCredentials`)
