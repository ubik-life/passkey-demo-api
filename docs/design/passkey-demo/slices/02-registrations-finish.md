# Slice 02 — `registrations-finish`

## Идентификатор входа

`HTTP POST /v1/registrations/{id}/attestation`

## Что делает (в одну фразу)

Принимает attestation от аутентификатора, верифицирует против сохранённой challenge регистрационной сессии, создаёт пользователя и credential, выдаёт пару JWT (access + refresh).

## OpenAPI

`api-specification/openapi.yaml`, `paths./registrations/{id}/attestation.post`. Контракт:

- Path: `id` — UUID регистрационной сессии (создаётся слайсом 1).
- Запрос: `application/json` `AttestationRequest` (форма WebAuthn `AuthenticatorAttestationResponse`).
- Ответ 200: `TokenPair` (`access_token`, `refresh_token`).
- Возможные ошибки: 404 `NOT_FOUND` (сессия не найдена / истекла), 422 `VALIDATION_ERROR` / `HANDLE_TAKEN` / `ATTESTATION_INVALID`, 503 `db_locked` (+ `Retry-After`), 507 `db_disk_full`.

422 `HANDLE_TAKEN` — доменная ошибка (race на UNIQUE handle), не режим отказа интеграции; в README «Карту режимов отказа» не входит.

## Gherkin-сценарии слайса

`component-tests/features/registrations.feature`:

- `Сценарий: Завершение регистрации` — happy path. **Then-шаги** этого слайса: ответ 200, непустые `access_token` и `refresh_token`. When-шаги используют слайс 1 (создание challenge) — это уже зафиксировано в карточке слайса 1.
- `Сценарий: Диск переполнен при завершении регистрации` — failure-режим `db_disk_full` на этой интеграции. Then-шаги: 200 ответ заменён на 507 + `code=db_disk_full`.

Сценарий «Создание challenge регистрации» — слайс 1, в этой карточке не учитывается.

## Зависимости от слайса 1

- **Импорт типов:** `Handle`, `RegistrationID`, `Challenge`, `RegistrationSession`, `RPConfig`, `ErrDBLocked`, `ErrDiskFull`, конструкторы `NewHandle`, `RegistrationIDFromString`, `ChallengeFromBytes`, `RegistrationSessionFromRow`.
- **Аддитивное расширение слайса 1** (см. `messages.md` → «Аддитивные расширения слайса 1 (рехидратор)»):
  - экспортировать `RegistrationSessionFromRow(rowID, rowHandle string, rowChallenge []byte, rowExpiresAtUnix int64) (RegistrationSession, error)`;
  - экспортировать `ChallengeFromBytes(b []byte) (Challenge, error)`;
  - экспортировать `RegistrationIDFromString(s string) (RegistrationID, error)`;
  - расширить `RPConfig` полем `Origin string` (для верификации `clientDataJSON.origin`).
- **Чтение БД:** строки таблицы `registration_sessions` (создана миграцией `0001`).
- **Удаление БД:** строка из `registration_sessions` после успешной фазы 2 (часть финального tx).

## Дерево модулей

```
ингресс-адаптер: HTTP handler POST /v1/registrations/{id}/attestation
    ├── parse path-param + body → RegistrationFinishRequest
    └── (после головного модуля) format Response → 200 + JSON,
        либо error → 404 / 422 / 503 + Retry-After / 507 / 500
        │
головной модуль слайса: ProcessRegistrationFinish
    ├── (1) NewRegistrationFinishCommand(req)              → RegistrationFinishCommand
    │       ├── RegistrationIDFromString(raw)              → RegistrationID    [рехидратор S1]
    │       └── parseAttestation(body)                     → ParsedAttestation [логика-лист; deps: protocol]
    ├── (2) loadRegistrationSession(cmd.RegID)             → RegistrationSession  [I/O — SQLite read; dep: db]
    ├── (3) NewFreshRegistrationSession(input)             → FreshRegistrationSession [конструктор подтипа]
    ├── (4) verifyAttestation(input)                       → VerifiedCredential [логика; deps: rpConfig]
    ├── (5) NewUser(input)                                 → User              [конструктор-сборщик]
    │       └── generateUserID()                           → UserID            [логика-лист]
    ├── (6) NewCredential(input)                           → Credential        [конструктор-сборщик]
    ├── (7) generateTokenPair(input)                       → IssuedTokenPair   [логика; deps: signer, jwtCfg, rand]
    ├── (8) finishRegistration(input)                      → error             [I/O — SQLite write tx; dep: db]
    └── (9) buildResponse(view)                            → TokenPair         [логика]
```

Каждый узел — **один data-аргумент** (Шаг 3 скилла). Зависимости (`db`, `rpConfig`, `signer`, `jwtCfg`, `clock`, `rand`) вынесены через `Deps` и не считаются стрелками графа.

Применение **подправила «подтип, не guard»** (Шаг 3 скилла): инвариант «сессия не истекла» оформлен как конструктор подтипа `NewFreshRegistrationSession` (узел 3), не как guard `checkSessionFresh(...) -> ()`. Дальнейшие шаги (4, 5) принимают `FreshRegistrationSession`, не сырой `RegistrationSession` — система типов гарантирует, что верификация и создание пользователя могут произойти только из non-expired сессии.

## Псевдокод пайпа головного модуля

```
ProcessRegistrationFinish(req: RegistrationFinishRequest, deps: Deps)
    -> (TokenPair, error):

    | NewRegistrationFinishCommand(req)                              -> RegistrationFinishCommand
    | loadRegistrationSession(cmd.RegID())                            -> RegistrationSession   [dep: deps.db]
    | input := NewFreshSessionInput{ Session: session, Now: deps.clock.Now() }
    | NewFreshRegistrationSession(input)                              -> FreshRegistrationSession
    | verifyInput := AttestationVerification{ Fresh: fresh, Parsed: cmd.Parsed() }
    | verifyAttestation(verifyInput)                                  -> VerifiedCredential    [dep: deps.cfg.RP]
    | userInput := NewUserInput{ ID: generateUserID(),
                                  Handle: fresh.Handle(),
                                  CreatedAt: deps.clock.Now() }
    | NewUser(userInput)                                              -> User
    | credInput := NewCredentialInput{ User: user, Verified: verified, CreatedAt: deps.clock.Now() }
    | NewCredential(credInput)                                        -> Credential
    | tokenInput := GenerateTokenPairInput{ User: user, Now: deps.clock.Now() }
    | generateTokenPair(tokenInput)                                   -> IssuedTokenPair       [deps: deps.signer, deps.jwtCfg, deps.rand]
    | finishInput := FinishRegistrationInput{ User: user,
                                               Credential: credential,
                                               RefreshTokenHash: issued.Refresh.Hash(),
                                               RefreshExpiresAt: issued.Refresh.ExpiresAt(),
                                               RegistrationID: fresh.ID() }
    | finishRegistration(finishInput)                                 -> error                  [dep: deps.db]
    | view := BuildTokenPairView{ Access: issued.Access, Refresh: issued.Refresh }
    | buildResponse(view)                                              -> TokenPair
```

Ошибки протекают через ранний `return TokenPair{}, fmt.Errorf("step: %w", err)`. Сборка `userInput`, `credInput`, `tokenInput`, `verifyInput`, `finishInput`, `view` — Go-литералы структур, не отдельные узлы графа.

## Контракты модулей

### `parseAttestation`

- **Сигнатура:** `parseAttestation(raw []byte) -> (ParsedAttestation, error)`
- **Input (data):** `raw []byte` — сырое тело запроса.
- **Dependencies (deps):** — (`go-webauthn/protocol` — компилируемая зависимость, не runtime dep)
- **Что делает:** парсит JSON-тело `AttestationRequest` через `protocol.ParseCredentialCreationResponseBody`. Заполняет распарсенную структуру; **не верифицирует** ни challenge, ни origin, ни подпись.
- **Антецедент:** `raw` — синтаксически валидный JSON, соответствующий схеме `AttestationRequest`.
- **Консеквент:**
  - Success: `ParsedAttestation`, обёртка над `*protocol.ParsedCredentialCreationData`.
  - Failure: `ErrAttestationParse` (некорректный JSON, отсутствуют обязательные поля, CBOR `attestationObject` не парсится).

### `NewRegistrationFinishCommand`

- **Сигнатура:** `NewRegistrationFinishCommand(req RegistrationFinishRequest) -> (RegistrationFinishCommand, error)`
- **Input (data):** `req RegistrationFinishRequest`
- **Dependencies (deps):** —
- **Что делает:** собирает доменную команду из DTO. Делегирует `RegistrationIDFromString` и `parseAttestation`.
- **Антецедент:** `req.RegistrationIDRaw` — UUID-строка; `req.AttestationBody` — соответствует антецеденту `parseAttestation`.
- **Консеквент:**
  - Success: команда с валидным `RegistrationID` и распарсенным `ParsedAttestation`.
  - Failure: `ErrInvalidRegID` (UUID не парсится) или `ErrAttestationParse` (тело не парсится), обёрнутые через `fmt.Errorf("...: %w", err)`.

### `loadRegistrationSession`

- **Сигнатура:** `loadRegistrationSession(id RegistrationID) -> (RegistrationSession, error)`
- **Input (data):** `id RegistrationID`
- **Dependencies (deps):** `*sql.DB`
- **Что делает:** SELECT id, handle, challenge, expires_at FROM registration_sessions WHERE id = ?. Если строка найдена — рехидрирует через `RegistrationSessionFromRow`. Если нет — `ErrSessionNotFound`.
- **Антецедент:** миграция `0001` применена; `id` валиден.
- **Консеквент:**
  - Success: `RegistrationSession`, восстановленная из БД.
  - Failure: `ErrSessionNotFound` (no row), `ErrDBLocked` (SQLITE_BUSY), низкоуровневые SQLite-ошибки (→ 500). `ErrDiskFull` для read-операции не различается (диск переполнен — write-тема; read возвращает строку или нет).

### `NewFreshRegistrationSession`

- **Сигнатура:** `NewFreshRegistrationSession(input NewFreshSessionInput) -> (FreshRegistrationSession, error)`
- **Input (data):** `input NewFreshSessionInput { Session, Now }`
- **Dependencies (deps):** —
- **Что делает:** конструктор подтипа, инвариант «сессия не истекла на момент конструкции».
- **Антецедент:** `input.Session` — валидная сущность из I/O; `input.Now` — момент.
- **Консеквент:**
  - Success: `FreshRegistrationSession{ session: input.Session }` при `input.Now < input.Session.ExpiresAt()`.
  - Failure: `ErrSessionExpired` при `input.Now >= input.Session.ExpiresAt()`.

### `verifyAttestation`

- **Сигнатура:** `verifyAttestation(input AttestationVerification) -> (VerifiedCredential, error)`
- **Input (data):** `input AttestationVerification { Fresh, Parsed }`
- **Dependencies (deps):** `RPConfig` (нужны `ID` и `Origin`)
- **Что делает:** делегирует `protocol.ParsedCredentialCreationData.Verify(challenge, false, rpID, origins, ...)` из go-webauthn. На успехе извлекает `credentialID`, `publicKey`, `signCount`, `transports` из распарсенной структуры и собирает `VerifiedCredential`.
- **Антецедент:** `input.Fresh` — non-expired сессия; `input.Parsed` — синтаксически распарсенный attestation; `RPConfig.ID` и `RPConfig.Origin` непустые.
- **Консеквент:**
  - Success: `VerifiedCredential` с публичным ключом и метаданными аутентификатора.
  - Failure: `ErrAttestationInvalid` — любой провал `Verify` (challenge mismatch, RP ID hash mismatch, origin mismatch, signature invalid, неподдерживаемая алгоритмика).

### `generateUserID`

- **Сигнатура:** `generateUserID() -> UserID`
- **Input (data):** void
- **Dependencies (deps):** —
- **Что делает:** возвращает свежий UUID v4.
- **Антецедент:** —
- **Консеквент:** валидный UUID v4. Без `error` (`uuid.New()` не падает).

### `NewUser`

- **Сигнатура:** `NewUser(input NewUserInput) -> User`
- **Input (data):** `input NewUserInput { ID, Handle, CreatedAt }`
- **Dependencies (deps):** —
- **Что делает:** собирает доменную сущность пользователя.
- **Антецедент:** `input.ID` — UUID v4 (гарантировано `generateUserID`); `input.Handle` — валидное доменное значение (приходит из `FreshRegistrationSession.Handle()`); `input.CreatedAt` — момент.
- **Консеквент:** `User{ id, handle, createdAt }`. Падений нет.

### `NewCredential`

- **Сигнатура:** `NewCredential(input NewCredentialInput) -> Credential`
- **Input (data):** `input NewCredentialInput { User, Verified, CreatedAt }`
- **Dependencies (deps):** —
- **Что делает:** собирает credential из верифицированного результата + пользователя.
- **Антецедент:** `input.User` — валидная сущность; `input.Verified` — успех `verifyAttestation`; `input.CreatedAt` — момент.
- **Консеквент:** `Credential` с публичным ключом, signCount, transports, `userID = input.User.ID()`. Падений нет.

### `generateTokenPair`

- **Сигнатура:** `generateTokenPair(input GenerateTokenPairInput) -> (IssuedTokenPair, error)`
- **Input (data):** `input GenerateTokenPairInput { User, Now }`
- **Dependencies (deps):** `signer ed25519.PrivateKey`, `jwtCfg JWTConfig`, `rand io.Reader`
- **Что делает:** выдаёт пару (access JWT + refresh opaque):
  - Access: `golang-jwt/jwt/v5` Ed25519, claims `{ iss, sub: user.ID(), iat: now, exp: now + AccessTTL }`. Подпись `signer`.
  - Refresh: 32 байта из `rand` → base64url(plaintext); hash = hex(sha256(plaintext)); expiresAt = now + RefreshTTL.
- **Антецедент:** `input.User` — валидная сущность; `input.Now` — момент; `signer` непустой; `jwtCfg.AccessTTL > 0`, `jwtCfg.RefreshTTL > 0`.
- **Консеквент:**
  - Success: `IssuedTokenPair { Access: AccessToken{ value, expiresAt }, Refresh: IssuedRefreshToken{ plaintext, hash, expiresAt } }`.
  - Failure: catastrophic — ошибка `rand.Read` или `signer.Sign` (теоретическая; обрабатываем как 500).

### `finishRegistration`

- **Сигнатура:** `finishRegistration(input FinishRegistrationInput) -> error`
- **Input (data):** `input FinishRegistrationInput { User, Credential, RefreshTokenHash, RefreshExpiresAt, RegistrationID }`
- **Dependencies (deps):** `*sql.DB`
- **Что делает:** одна транзакция:
  ```
  BEGIN
    INSERT INTO users (id, handle, created_at)
    INSERT INTO credentials (credential_id, user_id, public_key, sign_count, transports, created_at)
    INSERT INTO refresh_tokens (token_hash, user_id, expires_at)
    DELETE FROM registration_sessions WHERE id = ?
  COMMIT
  ```
  При любой ошибке — ROLLBACK.
- **Антецедент:** все доменные значения валидны; миграции `0001`-`0004` применены.
- **Консеквент:**
  - Success: tx закоммичена, все 4 операции применены.
  - Failure:
    - `ErrHandleTaken` — UNIQUE violation на `users.handle` (race с другой регистрацией). Маппится в 422 `HANDLE_TAKEN`.
    - `ErrDBLocked` — SQLITE_BUSY → 503 `db_locked`.
    - `ErrDiskFull` — SQLITE_FULL → 507 `db_disk_full`.
    - другие — обёрнуты в общую внутреннюю ошибку (→ 500 `INTERNAL_ERROR`).

### `buildResponse`

- **Сигнатура:** `buildResponse(view BuildTokenPairView) -> TokenPair`
- **Input (data):** `view BuildTokenPairView { Access, Refresh }`
- **Dependencies (deps):** —
- **Что делает:** упаковывает выданные токены в DTO.
- **Антецедент:** аргументы валидны.
- **Консеквент:** `TokenPair{ AccessToken: view.Access.Value(), RefreshToken: view.Refresh.Plaintext() }`. Без `error`.

### Ингресс-адаптер: HTTP handler `POST /v1/registrations/{id}/attestation`

- **Что делает:**
  1. Извлекает path-параметр `id` (chi router `chi.URLParam`).
  2. Читает тело запроса в `[]byte`.
  3. Собирает `RegistrationFinishRequest{ RegistrationIDRaw, AttestationBody }`.
  4. Вызывает `ProcessRegistrationFinish(req, deps)`.
  5. На Success: пишет `200 OK`, `Content-Type: application/json`, тело — `TokenPair`.
  6. На Failure: маппит ошибки в HTTP-ответ (см. таблицу маппинга ниже).
- **Никакой бизнес-логики** — только парсинг path/body и маппинг.
- **Юнит-тестами не покрывается** — проверяется компонентным тестом слайса через реальный HTTP-вход.

### Маппинг ошибок в ингресс-адаптере

| Класс ошибки                                                  | HTTP-статус | Заголовки           | Тело (`error.code`)         |
|---------------------------------------------------------------|-------------|---------------------|-----------------------------|
| `ErrInvalidRegID`                                             | 422          | —                   | `VALIDATION_ERROR`           |
| `ErrAttestationParse`                                          | 422          | —                   | `VALIDATION_ERROR`           |
| `ErrSessionNotFound`                                           | 404          | —                   | `NOT_FOUND`                  |
| `ErrSessionExpired`                                            | 404          | —                   | `NOT_FOUND`                  |
| `ErrAttestationInvalid`                                        | 422          | —                   | `ATTESTATION_INVALID`        |
| `ErrHandleTaken`                                               | 422          | —                   | `HANDLE_TAKEN`               |
| `ErrDBLocked`                                                  | 503          | `Retry-After: 1`    | `db_locked`                  |
| `ErrDiskFull`                                                  | 507          | —                   | `db_disk_full`               |
| Любая другая (catastrophic из `generateTokenPair`, неожиданные SQLite) | 500 | —                   | `INTERNAL_ERROR`             |

Истёкшая и несуществующая сессии **обе** маппятся в 404: для клиента поведение идентично — нужно начать фазу 1 заново. Различение остаётся только в логах адаптера.

## Gherkin-mapping

| Сценарий                                          | Then-шаг                                            | Кто обеспечивает (узел графа / маппинг адаптера)                                              |
|---------------------------------------------------|-----------------------------------------------------|-----------------------------------------------------------------------------------------------|
| Завершение регистрации                            | `Тогда ответ 200`                                    | Узлы (1)–(9) Success-путь → ингресс-адаптер: `200 OK` + JSON-сериализация `TokenPair`         |
| Завершение регистрации                            | `И ответ содержит непустое JSON-поле access_token`   | (7) `generateTokenPair` (поле `Access.Value()`) → (9) `buildResponse` → ингресс-адаптер       |
| Завершение регистрации                            | `И ответ содержит непустое JSON-поле refresh_token`  | (7) `generateTokenPair` (поле `Refresh.Plaintext()`) → (9) `buildResponse` → ингресс-адаптер  |
| Диск переполнен при завершении регистрации        | `Тогда ответ 507`                                    | (8) `finishRegistration` Failure: `ErrDiskFull` → ингресс-адаптер: маппинг `ErrDiskFull` → 507 |
| Диск переполнен при завершении регистрации        | `И ответ содержит JSON-поле code со значением "db_disk_full"` | ингресс-адаптер: маппинг `ErrDiskFull` → тело `{"code":"db_disk_full",...}`            |

### Чек-лист сверки 8.5

1. [x] **Узел существует.** Узлы (1)–(9) описаны в дереве и в контрактах выше; ингресс-адаптер описан с маппингом.
2. [x] **Ветка соответствует.** Then `200` — Success-путь; Then `507` — Failure-путь шага (8) `ErrDiskFull`. `access_token`/`refresh_token` — Success-выход шага (7), сериализованный шагом (9).
3. [x] **Формат ответа адаптера согласован.** OpenAPI декларирует 200 + `TokenPair`; адаптер сериализует `TokenPair{access_token, refresh_token}` с теми же полями. 507 + `{"code":"db_disk_full",...}` совпадает с README «Карта режимов отказа».
4. [x] **Все Then покрыты.** В сценарии «Завершение регистрации» 3 Then-шага, в «Диск переполнен» — 2; всего 5, все покрыты.

`[x] Gherkin-mapping сверен.`

### Замечание о других режимах отказа

OpenAPI декларирует на этом эндпоинте также 503 `db_locked`, но Gherkin-сценария на `db_locked` именно здесь нет (по раскладке в `slices.md` `db_locked` — слайс 4). Адаптер обязан корректно маппить `ErrDBLocked` → 503 + `Retry-After: 1`, но проверяется это компонентным тестом слайса 4 (через тот же путь маппинга в общем хелпере, либо параллельной реализацией).

`ErrSessionNotFound`, `ErrSessionExpired`, `ErrAttestationInvalid`, `ErrHandleTaken` — доменные ошибки, проверяются юнит-тестами модулей логики и контрактом OpenAPI; компонентных сценариев на них нет (по раскладке: `registrations.feature` содержит только happy + `db_disk_full`).

## Юнит-тесты по формуле

`N_юнит_тестов = 1 (happy path) + Σ (ветки антецедента)` — **только модули логики и конструкторы** (Шаг 8.1: «I/O — трубы, юнитами не покрываются»; ингресс-адаптер — тоже).

| Модуль                            | Happy | Ветки антецедента                                                  | Итого |
|-----------------------------------|-------|--------------------------------------------------------------------|-------|
| `parseAttestation`                | 1     | невалидный JSON, отсутствует поле `response`, невалидный CBOR `attestationObject` | 4 |
| `NewRegistrationFinishCommand`    | 1     | `ErrInvalidRegID` (UUID), `ErrAttestationParse` (склейка)          | 3     |
| `NewFreshRegistrationSession`     | 1     | `now >= expiresAt` → `ErrSessionExpired`                           | 2     |
| `verifyAttestation`               | 1     | `Verify` вернул ошибку (мутация подписи в virtualwebauthn-сценарии) | 2     |
| `generateUserID`                  | 1     | —                                                                  | 1     |
| `NewUser`                         | 1     | —                                                                  | 1     |
| `NewCredential`                   | 1     | —                                                                  | 1     |
| `generateTokenPair`               | 1     | (catastrophic `rand.Read` / `Sign` — теоретическое; пропускаем в MVP) | 1   |
| `buildResponse`                   | 1     | —                                                                  | 1     |
| `ProcessRegistrationFinish` (head)| 1     | ошибка из (1), (2), (3), (4), (7) catastrophic, (8)                | 7     |
| **Итого**                         |       |                                                                    | **23** |

Для honest-теста `verifyAttestation` happy path используется `github.com/descope/virtualwebauthn` (генерирует валидные attestation-данные). Та же библиотека используется в `component-tests/`, для unit-теста S2 добавляется в основной `go.mod` как **test-dep** (импорт только из `*_test.go`). Альтернатива «1 happy через мок» нарушает принцип «без моков в тестах» (см. `feedback_no_mocks`), поэтому отвергнута.

Что **не** в таблице (и почему):

- `loadRegistrationSession` — I/O-модуль, труба. Юнитов нет. Success-путь проверяется компонентным сценарием **«Завершение регистрации»** (если read не дойдёт — фаза 2 не получит challenge, сценарий красный). Failure-ветки `ErrSessionNotFound`, `ErrDBLocked` — без отдельного компонентного сценария на этом эндпоинте; маппинг проверяется через адаптер.
- `finishRegistration` — I/O-модуль. Success — happy-сценарий; Failure `ErrDiskFull` — компонентный сценарий «Диск переполнен при завершении регистрации». `ErrDBLocked`, `ErrHandleTaken` — без компонентного сценария на этом эндпоинте.
- **Ингресс-адаптер** — парсинг path/body и маппинг, юнитов нет. Реальные HTTP-вызовы в обоих компонентных сценариях.
- `головной модуль` `ProcessRegistrationFinish` — оркестратор, **есть** в таблице. Тестируется с реальными зависимостями (in-memory SQLite через `:memory:`) — без mock-функций, в соответствии с `feedback_no_mocks`.

Замечания по покрытию:

- 100% строк/веток модулей логики достигается этими 23 юнит-тестами.
- Honest-тесты для `verifyAttestation` и `ProcessRegistrationFinish` используют real libraries (`virtualwebauthn` для генерации, `mattn/go-sqlite3 :memory:` для БД), не моки.

## Definition of Done слайса

Скопировано в тикет S2 в `backlog.md`:

- [ ] **аддитивные расширения слайса 1**: экспортированы `RegistrationSessionFromRow`, `ChallengeFromBytes`, `RegistrationIDFromString`; `RPConfig` расширен полем `Origin`. Юнит-тесты S1 остаются зелёными.
- [ ] ингресс-адаптер реализован: парсит path-параметр и тело в `RegistrationFinishRequest`, без бизнес-валидации.
- [ ] конструкторы доменных структур (`NewRegistrationFinishCommand`, `NewFreshRegistrationSession`, `NewUser`, `NewCredential`) реализованы; невалидные данные → структура не создаётся.
- [ ] модули логики (`parseAttestation`, `verifyAttestation`, `generateUserID`, `generateTokenPair`, `buildResponse`) реализованы, контракты выполнены.
- [ ] модули I/O (`loadRegistrationSession`, `finishRegistration`) реализованы; `finishRegistration` — атомарная транзакция с откатом при любой ошибке; маппинг `SQLITE_BUSY` → `ErrDBLocked`, `SQLITE_FULL` → `ErrDiskFull`, UNIQUE на `users.handle` → `ErrHandleTaken`.
- [ ] головной модуль `ProcessRegistrationFinish` реализован: пайп из 9 шагов, ранний возврат при ошибке через `fmt.Errorf("…: %w", err)`.
- [ ] миграции `0002_users.sql`, `0003_credentials.sql`, `0004_refresh_tokens.sql` созданы по `infrastructure.md`.
- [ ] инфраструктурный модуль расширен: `PASSKEY_RP_ORIGIN`, `PASSKEY_JWT_ACCESS_TTL`, `PASSKEY_JWT_REFRESH_TTL`, `PASSKEY_JWT_ISSUER` загружаются в `AppConfig`; Ed25519 keypair генерируется в `wire.go` при старте; `Deps` слайса 2 содержит `Signer`, `JWTConfig`, `Rand`.
- [ ] слайс подключён через `registrations_finish.Register(mux, deps)`: HTTP-роут `POST /v1/registrations/{id}/attestation` ведёт на ингресс-адаптер.
- [ ] юнит-тесты по формуле — **23 теста на модули логики, конструкторы и головной модуль**; покрытие 100% по строкам и веткам логики; I/O-модули и ингресс-адаптер юнитами не покрываются. `verifyAttestation` honest-тестируется через `virtualwebauthn`.
- [ ] компонентный сценарий `Сценарий: Завершение регистрации` (`component-tests/features/registrations.feature`) зелёный.
- [ ] компонентный сценарий `Сценарий: Диск переполнен при завершении регистрации` зелёный.
- [ ] остальные сценарии в `sessions.feature`, `sessions-current.feature`, `users.feature` остаются красными в их Then-частях, но **не должны** ломаться по фазам слайсов 1–2 (When-шаги «отправляет POST /v1/registrations» и «собирает attestation и отправляет его» работают).
- [ ] локальный CI зелёный (`go test ./...` для юнитов и `./component-tests/scripts/run-tests.sh` для компонентных, оба профиля `healthy` и `disk-full`).
- [ ] `backlog.md` обновлён по каждому подтверждённому пункту (правило `AGENTS.md §10`).
- [ ] `docs/design/passkey-demo/devlog.md` дополнен блоком S2.
- [ ] PR создан, описание заполнено по шаблону Шага 8 скилла sonnet'а.
- [ ] PR смержен в main, CI на main зелёный.
