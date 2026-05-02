# Contracts Graph — passkey-demo

Граф вызовов модулей слайсов и сверка согласованности контрактов (Шаг 9 `program-design.skill`).

На текущей итерации спроектированы графы слайсов 1 (`registrations-start`), 2 (`registrations-finish`), 3 (`sessions-start`) и 4 (`sessions-finish`). Слайсы 5-6 будут добавлены в следующих итерациях.

---

## Slice 01 — `registrations-start`

### Граф вызовов

```
[ HTTP POST /v1/registrations ]
        |
        | bytes (JSON body)
        v
+----------------------------------------+
| ингресс-адаптер: HTTPHandler           |
|   parseJSON: bytes -> RegistrationStartRequest |
+----------------------------------------+
        |
        | RegistrationStartRequest
        v
+----------------------------------------+
| головной модуль: ProcessRegistrationStart |
+----------------------------------------+
   |
   |-- (1) NewRegistrationStartCommand:
   |        in:  RegistrationStartRequest
   |        out: (RegistrationStartCommand, error)
   |        delegates: NewHandle(string) -> (Handle, error)
   |
   |-- (2) generateChallenge:
   |        in:  void
   |        out: (Challenge, error)        [error — теоретическая, crypto/rand]
   |
   |-- (3) generateRegistrationID:
   |        in:  void
   |        out: RegistrationID            [без error]
   |
   |-- (4) NewRegistrationSession:
   |        in:  NewRegistrationSessionInput
   |              { ID, Handle, Challenge, TTL, Now }
   |        out: RegistrationSession       [без error: инварианты на нижних уровнях]
   |
   |-- (5) Store.PersistRegistrationSession:                [метод I/O-объекта; *sql.DB инкапсулирован в Store]
   |        in:  RegistrationSession
   |        out: error                      [Success: () | Failure: ErrDBLocked, ErrDiskFull, ...]
   |
   |-- (6) buildCreationOptions:
   |        in:  RegistrationSession
   |        out: CreationOptions            [без error]
   |        deps: RPConfig
   |
   |-- (7) buildResponse:
   |        in:  RegistrationStartView { Session, Options }
   |        out: RegistrationStartResponse  [без error]
   |
   v
[ ингресс-адаптер: formatResponse ]
   |
   |  Success:  RegistrationStartResponse → 201 + JSON
   |  Failure:  ErrHandle*    → 422 VALIDATION_ERROR
   |            ErrDBLocked   → 503 + Retry-After: 1 + db_locked
   |            ErrDiskFull   → 507 + db_disk_full
   |            прочее         → 500 INTERNAL_ERROR
   v
[ HTTP response ]
```

### Таблица стрелок

| #  | Кто вызывает              | Кого вызывает                  | Передаёт (data)                       | Получает обратно                  | Классы ошибок                                  |
|----|---------------------------|--------------------------------|---------------------------------------|-----------------------------------|------------------------------------------------|
| A  | HTTP runtime              | ингресс-адаптер.parseJSON      | `[]byte` (тело запроса)               | `RegistrationStartRequest`         | парсинг JSON: 400 VALIDATION_ERROR             |
| B  | ингресс-адаптер           | `ProcessRegistrationStart`     | `RegistrationStartRequest`            | `(RegistrationStartResponse, error)` | вся цепочка ошибок ниже                        |
| 1  | `ProcessRegistrationStart`| `NewRegistrationStartCommand`  | `RegistrationStartRequest`            | `(RegistrationStartCommand, error)` | `ErrHandleEmpty`, `ErrHandleTooShort`, `ErrHandleTooLong` |
| 1a | `NewRegistrationStartCommand` | `NewHandle`               | `string` (raw handle)                  | `(Handle, error)`                  | те же `ErrHandle*`                             |
| 2  | `ProcessRegistrationStart`| `generateChallenge`            | void                                   | `(Challenge, error)`               | catastrophic (crypto/rand) — теоретическая    |
| 3  | `ProcessRegistrationStart`| `generateRegistrationID`       | void                                   | `RegistrationID`                   | —                                              |
| 4  | `ProcessRegistrationStart`| `NewRegistrationSession`       | `NewRegistrationSessionInput`         | `RegistrationSession`              | —                                              |
| 5  | `ProcessRegistrationStart`| `Store.PersistRegistrationSession` | `RegistrationSession`              | `error`                             | `ErrDBLocked`, `ErrDiskFull`, низкоуровневые SQLite (→ 500) |
| 6  | `ProcessRegistrationStart`| `buildCreationOptions`         | `RegistrationSession`                  | `CreationOptions`                  | —                                              |
| 7  | `ProcessRegistrationStart`| `buildResponse`                | `RegistrationStartView`               | `RegistrationStartResponse`        | —                                              |
| C  | ингресс-адаптер           | HTTP runtime (formatResponse)  | `RegistrationStartResponse` или `error`| HTTP-ответ                          | маппинг `error` → 4xx/5xx                      |

### Чек-лист сверки 9.3

Прохожу по каждой стрелке в шести пунктах. Где «—» — пункт не применим (нет ошибок / нет deps).

| # | (1) Тип на стрелке существует | (2) Имя сигнатуры совпадает | (3) Консеквент A ⊆ антецеденту B | (4) Тип ошибки согласован | (5) Покрытие Gherkin | (6) Один data-аргумент |
|---|-----|-----|-----|-----|-----|-----|
| A  | [x] `[]byte`, `RegistrationStartRequest` (messages.md) | [x] handler.parseJSON | [x] адаптер требует только синтаксически валидный JSON | [x] невалидный JSON → 400 (адаптер обрабатывает локально) | [x] неявно (Then 201 включает успешный парсинг) | [x] один аргумент: `[]byte` |
| B  | [x] `RegistrationStartRequest`, `RegistrationStartResponse`, `error` (messages.md) | [x] `ProcessRegistrationStart` | [x] head принимает любой `Request`; внутренняя валидация в (1) | [x] все ошибки ниже маппятся адаптером | [x] Then «ответ 201» лежит в Success-ветке B | [x] один data-аргумент `RegistrationStartRequest`; `Deps` — отдельно |
| 1  | [x] `RegistrationStartRequest`, `RegistrationStartCommand` | [x] `NewRegistrationStartCommand` | [x] head передаёт уже распарсенный req | [x] `ErrHandle*` маппится адаптером в 422 | [x] неявно (Then 201) | [x] один аргумент `req` |
| 1a | [x] `string`, `Handle` | [x] `NewHandle` | [x] handle — UTF-8 строка из JSON | [x] `ErrHandle*` пробрасывается через `%w` | [x] неявно (Then 201) | [x] один аргумент `raw` |
| 2  | [x] `Challenge` | [x] `generateChallenge` | [x] нет антецедента | [x] catastrophic → 500 (адаптер) | [x] неявно (Then 201) | [x] void |
| 3  | [x] `RegistrationID` | [x] `generateRegistrationID` | [x] нет антецедента | [x] нет ошибок | [x] неявно (Then 201) | [x] void |
| 4  | [x] `NewRegistrationSessionInput`, `RegistrationSession` | [x] `NewRegistrationSession` | [x] поля input — валидные доменные значения с предыдущих шагов | [x] нет ошибок (инварианты на нижних уровнях) | [x] неявно (Then 201) | [x] один аргумент `input` |
| 5  | [x] `RegistrationSession`, `error` | [x] `Store.PersistRegistrationSession` | [x] `rs` — валидная доменная сущность | [x] `ErrDBLocked` → 503, `ErrDiskFull` → 507, прочее → 500 | [x] Success-путь — Then 201; Failure-ветки — компонентные сценарии слайсов 4 и 2 | [x] один data-аргумент `rs` (`*sql.DB` инкапсулирован в `Store`, не виден стрелкой) |
| 6  | [x] `RegistrationSession`, `CreationOptions` | [x] `buildCreationOptions` | [x] `s` валидна; `RPConfig.Name`/`ID` непустые из конфига | [x] нет ошибок | [x] Then 201 (формат `options.challenge`, `options.user.id`) | [x] один data-аргумент `s` (RPConfig — dep) |
| 7  | [x] `RegistrationStartView`, `RegistrationStartResponse` | [x] `buildResponse` | [x] view собран из шагов (4) и (6) | [x] нет ошибок | [x] Then 201 (формат `id`, `options`) | [x] один аргумент `view` |
| C  | [x] `RegistrationStartResponse`, `error` | [x] handler.formatResponse | [x] head вернул либо Success, либо одну из известных ошибок | [x] полный маппинг в таблице ошибок карточки слайса | [x] Then 201 / 422 / 503+Retry-After / 507 | [x] один data-аргумент: либо `Response`, либо `error` |

**Все стрелки помечены `[x] согласовано`.**

### Покрытие Gherkin-сценариев графом (пункт 9.3.5)

В Gherkin для этого эндпоинта один Then-шаг — `Тогда ответ 201`. Он покрыт цепочкой узлов B → 1 → 2 → 3 → 4 → 5 (Success) → 6 → 7 → C.

Узлов графа, не упомянутых ни одним Then-шагом, **нет**. Узлы (5 Failure), (1 Failure), (2 Failure), маппинг ошибок в C — отвечают за пути, которые на этом эндпоинте Gherkin не проверяет (по сознательной раскладке режимов отказа: `db_locked` → слайс 4, `db_disk_full` → слайс 2; валидационные ошибки — задача юнит-тестов конструкторов и компонентного теста соответствующего эндпоинта). Это **не** мёртвая логика — это часть декларированного OpenAPI-контракта.

### Сверка по правилу «один аргумент» (пункт 9.3.6) и автономии I/O-объекта (Шаг 6)

Все стрелки графа несут **ровно одну** data-сущность. Зависимости (`RPConfig`) на стрелках не отображены — они в `Dependencies:` контракта модуля.

**Сырого `*sql.DB` ни на одной стрелке нет.** Узел (5) — метод I/O-объекта `Store`, инкапсулирующего `*sql.DB` (Шаг 6 + `feedback_io_autonomous_store`). В `Deps` головного модуля — поле `*Store`, не сырой `*sql.DB`.

---

## Slice 02 — `registrations-finish`

### Граф вызовов

```
[ HTTP POST /v1/registrations/{id}/attestation ]
        |
        | path-param {id} + body []byte
        v
+-------------------------------------------------------+
| ингресс-адаптер: HTTPHandler                          |
|   parsePathAndBody: (id string, body []byte)          |
|     -> RegistrationFinishRequest                      |
+-------------------------------------------------------+
        |
        | RegistrationFinishRequest
        v
+-------------------------------------------------------+
| головной модуль: ProcessRegistrationFinish            |
+-------------------------------------------------------+
   |
   |-- (1) NewRegistrationFinishCommand:
   |        in:  RegistrationFinishRequest
   |        out: (RegistrationFinishCommand, error)
   |        delegates: RegistrationIDFromString(string) -> (RegistrationID, error)
   |                   parseAttestation([]byte)         -> (ParsedAttestation, error)
   |
   |-- (2) Store.LoadRegistrationSession:                  [метод I/O-объекта; *sql.DB инкапсулирован в Store]
   |        in:  RegistrationID
   |        out: (RegistrationSession, error)
   |        Failure: ErrSessionNotFound, ErrDBLocked
   |
   |-- (3) NewFreshRegistrationSession:                  [конструктор подтипа — подправило «не guard»]
   |        in:  NewFreshSessionInput { Session, Now }
   |        out: (FreshRegistrationSession, error)
   |        Failure: ErrSessionExpired
   |
   |-- (4) verifyAttestation:
   |        in:  AttestationVerification { Fresh, Parsed }
   |        out: (VerifiedCredential, error)
   |        deps: RPConfig (ID + Origin)
   |        Failure: ErrAttestationInvalid
   |
   |-- (5) NewUser (с встроенным generateUserID):
   |        in:  NewUserInput { ID, Handle, CreatedAt }
   |        out: User                                    [без error]
   |
   |-- (6) NewCredential:
   |        in:  NewCredentialInput { User, Verified, CreatedAt }
   |        out: Credential                              [без error]
   |
   |-- (7) generateTokenPair:
   |        in:  GenerateTokenPairInput { User, Now }
   |        out: (IssuedTokenPair, error)
   |        deps: ed25519.PrivateKey, JWTConfig, io.Reader
   |        Failure: catastrophic (rand / sign)
   |
   |-- (8) Store.FinishRegistration:                       [метод I/O-объекта; *sql.DB инкапсулирован в Store]
   |        in:  FinishRegistrationInput { User, Credential, RefreshTokenHash,
   |                                       RefreshExpiresAt, RegistrationID }
   |        out: error
   |        Failure: ErrHandleTaken, ErrDBLocked, ErrDiskFull
   |
   |-- (9) buildResponse:
   |        in:  BuildTokenPairView { Access, Refresh }
   |        out: TokenPair                               [без error]
   |
   v
[ ингресс-адаптер: formatResponse ]
   |
   |  Success:  TokenPair → 200 + JSON
   |  Failure:  ErrInvalidRegID / ErrAttestationParse → 422 VALIDATION_ERROR
   |            ErrSessionNotFound / ErrSessionExpired → 404 NOT_FOUND
   |            ErrAttestationInvalid                  → 422 ATTESTATION_INVALID
   |            ErrHandleTaken                         → 422 HANDLE_TAKEN
   |            ErrDBLocked                            → 503 + Retry-After: 1 + db_locked
   |            ErrDiskFull                            → 507 + db_disk_full
   |            прочее                                  → 500 INTERNAL_ERROR
   v
[ HTTP response ]
```

### Таблица стрелок

| #  | Кто вызывает              | Кого вызывает                   | Передаёт (data)                          | Получает обратно                       | Классы ошибок                                       |
|----|---------------------------|---------------------------------|------------------------------------------|----------------------------------------|----------------------------------------------------|
| A  | HTTP runtime              | ингресс-адаптер.parsePathAndBody| `(string, []byte)` (path + body)         | `RegistrationFinishRequest`             | I/O чтения тела → 400 (адаптер)                    |
| B  | ингресс-адаптер           | `ProcessRegistrationFinish`     | `RegistrationFinishRequest`              | `(TokenPair, error)`                    | вся цепочка ниже                                    |
| 1  | `ProcessRegistrationFinish`| `NewRegistrationFinishCommand` | `RegistrationFinishRequest`              | `(RegistrationFinishCommand, error)`    | `ErrInvalidRegID`, `ErrAttestationParse`            |
| 1a | `NewRegistrationFinishCommand` | `RegistrationIDFromString`  | `string`                                 | `(RegistrationID, error)`               | `ErrInvalidRegID`                                   |
| 1b | `NewRegistrationFinishCommand` | `parseAttestation`           | `[]byte`                                 | `(ParsedAttestation, error)`            | `ErrAttestationParse`                               |
| 2  | `ProcessRegistrationFinish`| `Store.LoadRegistrationSession` | `RegistrationID`                          | `(RegistrationSession, error)`          | `ErrSessionNotFound`, `ErrDBLocked`, низкоуровневые |
| 3  | `ProcessRegistrationFinish`| `NewFreshRegistrationSession`  | `NewFreshSessionInput`                    | `(FreshRegistrationSession, error)`     | `ErrSessionExpired`                                  |
| 4  | `ProcessRegistrationFinish`| `verifyAttestation`             | `AttestationVerification`                | `(VerifiedCredential, error)`           | `ErrAttestationInvalid`                              |
| 5  | `ProcessRegistrationFinish`| `NewUser`                       | `NewUserInput`                            | `User`                                  | —                                                   |
| 5a | `ProcessRegistrationFinish`| `generateUserID`                | void                                      | `UserID`                                | —                                                   |
| 6  | `ProcessRegistrationFinish`| `NewCredential`                 | `NewCredentialInput`                      | `Credential`                            | —                                                   |
| 7  | `ProcessRegistrationFinish`| `generateTokenPair`             | `GenerateTokenPairInput`                  | `(IssuedTokenPair, error)`              | catastrophic (→ 500)                                |
| 8  | `ProcessRegistrationFinish`| `Store.FinishRegistration`      | `FinishRegistrationInput`                 | `error`                                 | `ErrHandleTaken`, `ErrDBLocked`, `ErrDiskFull`, низкоуровневые |
| 9  | `ProcessRegistrationFinish`| `buildResponse`                 | `BuildTokenPairView`                      | `TokenPair`                             | —                                                   |
| C  | ингресс-адаптер           | HTTP runtime (formatResponse)   | `TokenPair` или `error`                   | HTTP-ответ                               | маппинг `error` → 4xx/5xx                           |

### Чек-лист сверки 9.3

| # | (1) Тип на стрелке существует | (2) Имя сигнатуры совпадает | (3) Консеквент A ⊆ антецеденту B | (4) Тип ошибки согласован | (5) Покрытие Gherkin | (6) Один data-аргумент |
|---|-----|-----|-----|-----|-----|-----|
| A  | [x] `string`, `[]byte`, `RegistrationFinishRequest` | [x] handler.parsePathAndBody | [x] адаптер требует только синтаксически валидный путь и тело | [x] провал чтения тела → 400 (локально) | [x] неявно (Then 200/507 включают парсинг) | [x] один аргумент: `RegistrationFinishRequest` (path+body — поля одной DTO) |
| B  | [x] `RegistrationFinishRequest`, `TokenPair`, `error` | [x] `ProcessRegistrationFinish` | [x] head принимает любой Request; внутренняя валидация в (1) | [x] все ошибки ниже маппятся адаптером | [x] Then «ответ 200» Success-ветка B; Then «ответ 507» Failure-ветка B | [x] один data-аргумент `req`; `Deps` отдельно |
| 1  | [x] `RegistrationFinishRequest`, `RegistrationFinishCommand` | [x] `NewRegistrationFinishCommand` | [x] head передаёт уже распарсенный req | [x] `ErrInvalidRegID`/`ErrAttestationParse` маппятся в 422 | [x] неявно (Then 200) | [x] один аргумент `req` |
| 1a | [x] `string`, `RegistrationID` | [x] `RegistrationIDFromString` (S1 рехидратор) | [x] строка из path-параметра | [x] `ErrInvalidRegID` пробрасывается через `%w` | [x] неявно | [x] один аргумент |
| 1b | [x] `[]byte`, `ParsedAttestation` | [x] `parseAttestation` | [x] байты тела запроса | [x] `ErrAttestationParse` пробрасывается через `%w` | [x] неявно | [x] один аргумент |
| 2  | [x] `RegistrationID`, `RegistrationSession`, `error` | [x] `Store.LoadRegistrationSession` | [x] `id` валиден из (1); миграция 0001 применена | [x] `ErrSessionNotFound` → 404; `ErrDBLocked` → 503; прочее → 500 | [x] Success — Then 200; Failure-ветки — без отдельного Then на этом эндпоинте | [x] один data-аргумент `id` (`*sql.DB` инкапсулирован в `Store`, не виден стрелкой) |
| 3  | [x] `NewFreshSessionInput`, `FreshRegistrationSession`, `error` | [x] `NewFreshRegistrationSession` | [x] Session — валидная сущность из (2); Now — момент | [x] `ErrSessionExpired` → 404 | [x] неявно (Then 200) | [x] один аргумент `input` |
| 4  | [x] `AttestationVerification`, `VerifiedCredential`, `error` | [x] `verifyAttestation` | [x] Fresh — non-expired; Parsed — синтаксически распарсенный | [x] `ErrAttestationInvalid` → 422 ATTESTATION_INVALID | [x] неявно (Then 200) | [x] один data-аргумент `input` (RPConfig — dep) |
| 5  | [x] `NewUserInput`, `User` | [x] `NewUser` | [x] ID, Handle, CreatedAt — валидные | [x] нет ошибок | [x] неявно | [x] один аргумент `input` |
| 5a | [x] `UserID` | [x] `generateUserID` | [x] нет антецедента | [x] нет ошибок | [x] неявно | [x] void |
| 6  | [x] `NewCredentialInput`, `Credential` | [x] `NewCredential` | [x] User валиден; Verified — успех (4); CreatedAt — момент | [x] нет ошибок | [x] неявно | [x] один аргумент `input` |
| 7  | [x] `GenerateTokenPairInput`, `IssuedTokenPair`, `error` | [x] `generateTokenPair` | [x] User валиден; Now — момент; signer непустой; TTL > 0 | [x] catastrophic → 500 | [x] Then «непустое access_token»/«непустое refresh_token» — поля выхода | [x] один data-аргумент `input` (signer/jwtCfg/rand — deps) |
| 8  | [x] `FinishRegistrationInput`, `error` | [x] `Store.FinishRegistration` | [x] User, Credential, RefreshHash, RefreshExpiresAt, RegistrationID — валидны | [x] `ErrHandleTaken` → 422; `ErrDBLocked` → 503; `ErrDiskFull` → 507; прочее → 500 | [x] Then «507» — Failure: `ErrDiskFull`; Then «200» — Success | [x] один data-аргумент `input` (`*sql.DB` инкапсулирован в `Store`, не виден стрелкой) |
| 9  | [x] `BuildTokenPairView`, `TokenPair` | [x] `buildResponse` | [x] view собран из выходов (5)/(7) | [x] нет ошибок | [x] Then 200 (формат `access_token`/`refresh_token`) | [x] один аргумент `view` |
| C  | [x] `TokenPair`, `error` | [x] handler.formatResponse | [x] head вернул либо Success, либо одну из известных ошибок | [x] полный маппинг в таблице ошибок карточки слайса | [x] Then 200 / 507 + `code=db_disk_full` | [x] один data-аргумент: либо `Response`, либо `error` |

**Все стрелки помечены `[x] согласовано`.**

### Покрытие Gherkin-сценариев графом (пункт 9.3.5)

В Gherkin для эндпоинта S2 — 5 Then-шагов (3 в «Завершение регистрации», 2 в «Диск переполнен при завершении регистрации»). Все покрыты, см. таблицу `## Gherkin-mapping` в `slices/02-registrations-finish.md`.

Цепочки:
- Happy: B → 1 → 2 (Success) → 3 (Success) → 4 (Success) → 5 → 5a → 6 → 7 (Success) → 8 (Success) → 9 → C (200)
- `db_disk_full`: B → 1 → 2 (Success) → 3 (Success) → 4 (Success) → 5 → 5a → 6 → 7 (Success) → 8 (Failure: ErrDiskFull) → C (507)

Узлы графа, не упомянутые ни одним Then-шагом, **нет**. Failure-ветки (1)/(1a)/(1b)/(2 ErrSessionNotFound, ErrDBLocked)/(3)/(4)/(7)/(8 ErrHandleTaken, ErrDBLocked) — пути, которые на этом эндпоинте Gherkin не проверяет (по сознательной раскладке: `db_locked` → слайс 4; валидационные/доменные ошибки — задача юнит-тестов конструкторов и контракта OpenAPI). Это **не** мёртвая логика — часть декларированного OpenAPI-контракта.

### Сверка по правилу «один аргумент» (пункт 9.3.6) и автономии I/O-объекта (Шаг 6)

Все стрелки графа несут **ровно одну** data-сущность. Зависимости (`RPConfig`, `ed25519.PrivateKey`, `JWTConfig`, `io.Reader`) на стрелках не отображены — они в `Dependencies:` контракта модуля.

**Сырого `*sql.DB` ни на одной стрелке нет.** Узлы (2) и (8) — методы I/O-объекта `Store`, инкапсулирующего `*sql.DB` (Шаг 6 + `feedback_io_autonomous_store`). В `Deps` головного модуля — поле `*Store`, не сырой `*sql.DB`.

### Применение подправила «подтип, не guard» (Шаг 3 скилла)

Узел (3) `NewFreshRegistrationSession` — конструктор подтипа `FreshRegistrationSession`, инвариант «не истекла» закреплён в типе. Узлы (4) и (5) принимают `FreshRegistrationSession` (не сырой `RegistrationSession`), что гарантировано системой типов.

Нет узлов со «висящей» сигнатурой `(Domain) -> ()` или `(input) -> error` — все логические шаги либо возвращают новую доменную структуру (1, 3, 4, 5, 6, 7, 9), либо являются I/O-эффектом с `error` (2, 8, и узел C-маппинг). Правило соблюдено.

---

## Slice 03 — `sessions-start`

### Граф вызовов

```
[ HTTP POST /v1/sessions ]
        |
        | bytes (JSON body)
        v
+----------------------------------------+
| ингресс-адаптер: HTTPHandler           |
|   parseJSON: bytes -> SessionStartRequest |
+----------------------------------------+
        |
        | SessionStartRequest
        v
+----------------------------------------+
| головной модуль: ProcessSessionStart   |
+----------------------------------------+
   |
   |-- (1) NewSessionStartCommand:
   |        in:  SessionStartRequest
   |        out: (SessionStartCommand, error)
   |        delegates: NewHandle(string) -> (Handle, error)   [импорт S1]
   |
   |-- (2) Store.LoadUserCredentials:                        [метод I/O-объекта; *sql.DB инкапсулирован в Store]
   |        in:  Handle
   |        out: (UserWithCredentials, error)
   |        delegates: UserFromRow, CredentialFromRow         [импорты S2]
   |        Failure: ErrUserNotFound, ErrDBLocked
   |
   |-- (3) GenerateChallenge:                                 [импорт S1, аддитивное расширение]
   |        in:  void
   |        out: (Challenge, error)        [error — теоретическая, crypto/rand]
   |
   |-- (4) generateLoginSessionID:
   |        in:  void
   |        out: LoginSessionID            [без error]
   |
   |-- (5) NewLoginSession:
   |        in:  NewLoginSessionInput
   |              { ID, UserID, Challenge, TTL, Now }
   |        out: LoginSession              [без error: инварианты на нижних уровнях]
   |
   |-- (6) Store.PersistLoginSession:                        [метод I/O-объекта; *sql.DB инкапсулирован в Store]
   |        in:  LoginSession
   |        out: error                      [Success: () | Failure: ErrDBLocked, ErrDiskFull, ...]
   |
   |-- (7) buildRequestOptions:
   |        in:  BuildRequestOptionsInput { Session, Credentials }
   |        out: RequestOptions             [без error]
   |        deps: RPConfig.ID
   |
   |-- (8) buildResponse:
   |        in:  SessionStartView { Session, Options }
   |        out: SessionStartResponse       [без error]
   |
   v
[ ингресс-адаптер: formatResponse ]
   |
   |  Success:  SessionStartResponse → 201 + JSON
   |  Failure:  ErrHandle*    → 422 VALIDATION_ERROR
   |            ErrUserNotFound → 404 NOT_FOUND
   |            ErrDBLocked   → 503 + Retry-After: 1 + db_locked
   |            ErrDiskFull   → 507 + db_disk_full
   |            прочее         → 500 INTERNAL_ERROR
   v
[ HTTP response ]
```

### Таблица стрелок

| #  | Кто вызывает              | Кого вызывает                  | Передаёт (data)                       | Получает обратно                  | Классы ошибок                                  |
|----|---------------------------|--------------------------------|---------------------------------------|-----------------------------------|------------------------------------------------|
| A  | HTTP runtime              | ингресс-адаптер.parseJSON      | `[]byte` (тело запроса)               | `SessionStartRequest`             | парсинг JSON: 400 VALIDATION_ERROR             |
| B  | ингресс-адаптер           | `ProcessSessionStart`          | `SessionStartRequest`                 | `(SessionStartResponse, error)`    | вся цепочка ошибок ниже                        |
| 1  | `ProcessSessionStart`     | `NewSessionStartCommand`       | `SessionStartRequest`                 | `(SessionStartCommand, error)`     | `ErrHandleEmpty`, `ErrHandleTooShort`, `ErrHandleTooLong` |
| 1a | `NewSessionStartCommand`  | `NewHandle` [S1]               | `string` (raw handle)                 | `(Handle, error)`                  | те же `ErrHandle*`                             |
| 2  | `ProcessSessionStart`     | `Store.LoadUserCredentials`     | `Handle`                              | `(UserWithCredentials, error)`     | `ErrUserNotFound`, `ErrDBLocked`, низкоуровневые |
| 2a | `Store.LoadUserCredentials` | `UserFromRow` [S2]            | `(string, string, int64)`             | `(User, error)`                    | синтаксические (UUID, handle)                  |
| 2b | `Store.LoadUserCredentials` | `CredentialFromRow` [S2]      | `([]byte, string, []byte, uint32, string, int64)` | `(Credential, error)`     | синтаксические (UUID, поля)                    |
| 3  | `ProcessSessionStart`     | `GenerateChallenge` [S1]       | void                                   | `(Challenge, error)`               | catastrophic (crypto/rand) — теоретическая    |
| 4  | `ProcessSessionStart`     | `generateLoginSessionID`       | void                                   | `LoginSessionID`                   | —                                              |
| 5  | `ProcessSessionStart`     | `NewLoginSession`              | `NewLoginSessionInput`                | `LoginSession`                     | —                                              |
| 6  | `ProcessSessionStart`     | `Store.PersistLoginSession`     | `LoginSession`                         | `error`                             | `ErrDBLocked`, `ErrDiskFull`, низкоуровневые SQLite (→ 500) |
| 7  | `ProcessSessionStart`     | `buildRequestOptions`          | `BuildRequestOptionsInput`            | `RequestOptions`                   | —                                              |
| 8  | `ProcessSessionStart`     | `buildResponse`                | `SessionStartView`                    | `SessionStartResponse`             | —                                              |
| C  | ингресс-адаптер           | HTTP runtime (formatResponse)  | `SessionStartResponse` или `error`    | HTTP-ответ                          | маппинг `error` → 4xx/5xx                      |

### Чек-лист сверки 9.3

| # | (1) Тип на стрелке существует | (2) Имя сигнатуры совпадает | (3) Консеквент A ⊆ антецеденту B | (4) Тип ошибки согласован | (5) Покрытие Gherkin | (6) Один data-аргумент |
|---|-----|-----|-----|-----|-----|-----|
| A  | [x] `[]byte`, `SessionStartRequest` (messages.md) | [x] handler.parseJSON | [x] адаптер требует только синтаксически валидный JSON | [x] невалидный JSON → 400 (адаптер обрабатывает локально) | [x] неявно (Then 201 включает успешный парсинг) | [x] один аргумент: `[]byte` |
| B  | [x] `SessionStartRequest`, `SessionStartResponse`, `error` (messages.md) | [x] `ProcessSessionStart` | [x] head принимает любой `Request`; внутренняя валидация в (1) | [x] все ошибки ниже маппятся адаптером | [x] Then «ответ 201» лежит в Success-ветке B | [x] один data-аргумент `SessionStartRequest`; `Deps` — отдельно |
| 1  | [x] `SessionStartRequest`, `SessionStartCommand` | [x] `NewSessionStartCommand` | [x] head передаёт уже распарсенный req | [x] `ErrHandle*` маппится адаптером в 422 | [x] неявно (Then 201) | [x] один аргумент `req` |
| 1a | [x] `string`, `Handle` (S1) | [x] `NewHandle` | [x] handle — UTF-8 строка из JSON | [x] `ErrHandle*` пробрасывается через `%w` | [x] неявно (Then 201) | [x] один аргумент `raw` |
| 2  | [x] `Handle`, `UserWithCredentials`, `error` | [x] `Store.LoadUserCredentials` | [x] `h` валиден из (1); миграции 0002/0003 применены | [x] `ErrUserNotFound` → 404; `ErrDBLocked` → 503; прочее → 500 | [x] Success — Then 201; Failure — без отдельного Then на этом эндпоинте | [x] один data-аргумент `h` (`*sql.DB` инкапсулирован в `Store`, не виден стрелкой) |
| 2a | [x] `string`, `string`, `int64`, `User` (S2 рехидратор) | [x] `UserFromRow` | [x] поля — строка users(...) из БД | [x] синтаксические ошибки → 500 | [x] неявно (через Then 201) | [x] три значения — поля одной строки БД (логически один кортеж) |
| 2b | [x] поля `credentials`, `Credential` (S2 рехидратор) | [x] `CredentialFromRow` | [x] поля — строка credentials(...) из БД | [x] синтаксические → 500 | [x] неявно | [x] поля одной строки БД |
| 3  | [x] `Challenge` (S1) | [x] `GenerateChallenge` | [x] нет антецедента | [x] catastrophic → 500 (адаптер) | [x] неявно (Then 201) | [x] void |
| 4  | [x] `LoginSessionID` | [x] `generateLoginSessionID` | [x] нет антецедента | [x] нет ошибок | [x] неявно (Then 201) | [x] void |
| 5  | [x] `NewLoginSessionInput`, `LoginSession` | [x] `NewLoginSession` | [x] поля input — валидные доменные значения с предыдущих шагов | [x] нет ошибок (инварианты на нижних уровнях) | [x] неявно (Then 201) | [x] один аргумент `input` |
| 6  | [x] `LoginSession`, `error` | [x] `Store.PersistLoginSession` | [x] `ls` — валидная доменная сущность; user_id ссылается на существующую запись users (гарантировано шагом 2 — `Store.LoadUserCredentials` не вернул `ErrUserNotFound`) | [x] `ErrDBLocked` → 503, `ErrDiskFull` → 507, прочее → 500 | [x] Success-путь — Then 201; Failure-ветки — без отдельного Then на этом эндпоинте | [x] один data-аргумент `ls` (`*sql.DB` инкапсулирован в `Store`, не виден стрелкой) |
| 7  | [x] `BuildRequestOptionsInput`, `RequestOptions` | [x] `buildRequestOptions` | [x] Session валидна; Credentials непустой по инварианту `UserWithCredentials`; `RPConfig.ID` непустой из конфига | [x] нет ошибок | [x] Then 201 (формат `options.challenge`, `options.allowCredentials`, `options.rpId`) | [x] один data-аргумент `input` (RPConfig — dep) |
| 8  | [x] `SessionStartView`, `SessionStartResponse` | [x] `buildResponse` | [x] view собран из шагов (5) и (7) | [x] нет ошибок | [x] Then 201 (формат `id`, `options`) | [x] один аргумент `view` |
| C  | [x] `SessionStartResponse`, `error` | [x] handler.formatResponse | [x] head вернул либо Success, либо одну из известных ошибок | [x] полный маппинг в таблице ошибок карточки слайса | [x] Then 201 / 422 / 404 / 503+Retry-After / 507 | [x] один data-аргумент: либо `Response`, либо `error` |

**Все стрелки помечены `[x] согласовано`.**

### Покрытие Gherkin-сценариев графом (пункт 9.3.5)

В Gherkin для эндпоинта S3 один Then-шаг — `Тогда ответ 201`. Он покрыт цепочкой узлов B → 1 → 2 (Success) → 3 → 4 → 5 → 6 (Success) → 7 → 8 → C.

Узлов графа, не упомянутых ни одним Then-шагом, **нет**. Узлы (2 Failure: `ErrUserNotFound`, `ErrDBLocked`), (1 Failure), (3 Failure), (6 Failure), маппинг ошибок в C — отвечают за пути, которые на этом эндпоинте Gherkin не проверяет (по сознательной раскладке режимов отказа: `db_locked` → слайс 4, `db_disk_full` → слайс 2; `NOT_FOUND` для несуществующего user — без компонентного сценария, проверяется юнит-тестами I/O косвенно через тип `UserWithCredentials` и контрактом OpenAPI). Это **не** мёртвая логика — часть декларированного OpenAPI-контракта.

### Сверка по правилу «один аргумент» (пункт 9.3.6) и автономии I/O-объекта (Шаг 6)

Все стрелки графа несут **ровно одну** data-сущность (или void — для (3), (4)). Зависимости (`RPConfig`) на стрелках не отображены — они в `Dependencies:` контракта модуля.

**Сырого `*sql.DB` ни на одной стрелке нет.** Узлы (2) и (6) — методы I/O-объекта `Store`, инкапсулирующего `*sql.DB` (Шаг 6 + `feedback_io_autonomous_store`). В `Deps` головного модуля — поле `*Store`, не сырой `*sql.DB`. Головной модуль `ProcessSessionStart` обращается к БД исключительно через методы `Store`.

Стрелки 2a/2b — внутри метода `Store.LoadUserCredentials`. Поля строки БД формально передаются как несколько `string`/`[]byte`/`int64`/`uint32`, но логически это **один кортеж** одной строки таблицы — введение промежуточной структуры `UserRow`/`CredentialRow` дало бы ноль выгоды (инкапсуляция уже в I/O-объекте, тип потока — раз и навсегда из БД). Решение: оставить «многоколонную» сигнатуру рехидраторов как идиоматический Go-маппинг `Scan(&col1, &col2, ...)`. Это не нарушение правила «один data-аргумент», потому что рехидраторы — листья I/O-трубы внутри `Store`, не узлы бизнес-логики.

---

## Slice 04 — `sessions-finish`

### Граф вызовов

```
[ HTTP POST /v1/sessions/{id}/assertion ]
        |
        | path-param {id} + body []byte
        v
+-------------------------------------------------------+
| ингресс-адаптер: HTTPHandler                          |
|   parsePathAndBody: (id string, body []byte)          |
|     -> SessionFinishRequest                           |
+-------------------------------------------------------+
        |
        | SessionFinishRequest
        v
+-------------------------------------------------------+
| головной модуль: ProcessSessionFinish                 |
+-------------------------------------------------------+
   |
   |-- (1) NewSessionFinishCommand:
   |        in:  SessionFinishRequest
   |        out: (SessionFinishCommand, error)
   |        delegates: LoginSessionIDFromString(string) -> (LoginSessionID, error)   [S3 рехидратор]
   |                   parseAssertion([]byte)             -> (ParsedAssertion, error)
   |
   |-- (2) Store.LoadLoginSession:                          [метод I/O-объекта; *sql.DB инкапсулирован в Store]
   |        in:  LoginSessionID
   |        out: (LoginSession, error)
   |        delegates: LoginSessionFromRow                   [S3 рехидратор]
   |        Failure: ErrLoginSessionNotFound, ErrDBLocked
   |
   |-- (3) NewFreshLoginSession:                           [конструктор подтипа — подправило «не guard»]
   |        in:  NewFreshLoginSessionInput { Session, Now }
   |        out: (FreshLoginSession, error)
   |        Failure: ErrLoginSessionExpired
   |
   |-- (4) Store.LoadAssertionTarget:                       [метод I/O-объекта; *sql.DB инкапсулирован в Store]
   |        in:  LoadAssertionTargetInput { UserID, CredentialID }
   |        out: (AssertionTarget, error)
   |        delegates: CredentialFromRow, UserFromRow         [S2 рехидраторы]
   |        Failure: ErrCredentialNotFound, ErrDBLocked
   |
   |-- (5) verifyAssertion:
   |        in:  AssertionVerification { Fresh, Parsed, Target }
   |        out: (VerifiedAssertion, error)
   |        deps: RPConfig (ID + Origin)
   |        Failure: ErrAssertionInvalid
   |
   |-- (6) GenerateTokenPair [импорт S2]:
   |        in:  GenerateTokenPairInput { User, Now }
   |        out: (IssuedTokenPair, error)
   |        deps: ed25519.PrivateKey, JWTConfig
   |        Failure: catastrophic (rand / sign)
   |
   |-- (7) Store.FinishLogin:                                 [метод I/O-объекта; *sql.DB инкапсулирован в Store]
   |        in:  FinishLoginInput { Credential, NewSignCount, RefreshTokenHash,
   |                                 RefreshExpiresAt, LoginSessionID }
   |        out: error
   |        Failure: ErrDBLocked, ErrDiskFull
   |
   |-- (8) BuildResponse [импорт S2]:
   |        in:  BuildTokenPairView { Access, Refresh }
   |        out: TokenPair                                  [без error]
   |
   v
[ ингресс-адаптер: formatResponse ]
   |
   |  Success:  TokenPair → 200 + JSON
   |  Failure:  ErrInvalidLoginSessionID / ErrAssertionParse → 422 VALIDATION_ERROR
   |            ErrLoginSessionNotFound / ErrLoginSessionExpired
   |              / ErrCredentialNotFound                    → 404 NOT_FOUND
   |            ErrAssertionInvalid                          → 422 ASSERTION_INVALID
   |            ErrDBLocked                                  → 503 + Retry-After: 1 + db_locked
   |            ErrDiskFull                                  → 507 + db_disk_full
   |            прочее                                        → 500 INTERNAL_ERROR
   v
[ HTTP response ]
```

### Таблица стрелок

| #  | Кто вызывает              | Кого вызывает                   | Передаёт (data)                          | Получает обратно                       | Классы ошибок                                       |
|----|---------------------------|---------------------------------|------------------------------------------|----------------------------------------|----------------------------------------------------|
| A  | HTTP runtime              | ингресс-адаптер.parsePathAndBody| `(string, []byte)` (path + body)         | `SessionFinishRequest`                  | I/O чтения тела → 400 (адаптер)                    |
| B  | ингресс-адаптер           | `ProcessSessionFinish`          | `SessionFinishRequest`                   | `(TokenPair, error)`                    | вся цепочка ниже                                    |
| 1  | `ProcessSessionFinish`    | `NewSessionFinishCommand`       | `SessionFinishRequest`                   | `(SessionFinishCommand, error)`         | `ErrInvalidLoginSessionID`, `ErrAssertionParse`     |
| 1a | `NewSessionFinishCommand` | `LoginSessionIDFromString` [S3] | `string`                                 | `(LoginSessionID, error)`               | `ErrInvalidLoginSessionID`                          |
| 1b | `NewSessionFinishCommand` | `parseAssertion`                | `[]byte`                                 | `(ParsedAssertion, error)`              | `ErrAssertionParse`                                 |
| 2  | `ProcessSessionFinish`    | `Store.LoadLoginSession`        | `LoginSessionID`                          | `(LoginSession, error)`                 | `ErrLoginSessionNotFound`, `ErrDBLocked`, низкоуровневые |
| 3  | `ProcessSessionFinish`    | `NewFreshLoginSession`           | `NewFreshLoginSessionInput`               | `(FreshLoginSession, error)`            | `ErrLoginSessionExpired`                             |
| 4  | `ProcessSessionFinish`    | `Store.LoadAssertionTarget`      | `LoadAssertionTargetInput`                | `(AssertionTarget, error)`              | `ErrCredentialNotFound`, `ErrDBLocked`, низкоуровневые |
| 5  | `ProcessSessionFinish`    | `verifyAssertion`                | `AssertionVerification`                   | `(VerifiedAssertion, error)`            | `ErrAssertionInvalid`                                |
| 6  | `ProcessSessionFinish`    | `GenerateTokenPair` [S2]        | `GenerateTokenPairInput`                  | `(IssuedTokenPair, error)`              | catastrophic (→ 500)                                |
| 7  | `ProcessSessionFinish`    | `Store.FinishLogin`              | `FinishLoginInput`                        | `error`                                 | `ErrDBLocked`, `ErrDiskFull`, низкоуровневые        |
| 8  | `ProcessSessionFinish`    | `BuildResponse` [S2]            | `BuildTokenPairView`                      | `TokenPair`                             | —                                                   |
| C  | ингресс-адаптер           | HTTP runtime (formatResponse)   | `TokenPair` или `error`                   | HTTP-ответ                               | маппинг `error` → 4xx/5xx                           |

### Чек-лист сверки 9.3

| # | (1) Тип на стрелке существует | (2) Имя сигнатуры совпадает | (3) Консеквент A ⊆ антецеденту B | (4) Тип ошибки согласован | (5) Покрытие Gherkin | (6) Один data-аргумент |
|---|-----|-----|-----|-----|-----|-----|
| A  | [x] `string`, `[]byte`, `SessionFinishRequest` | [x] handler.parsePathAndBody | [x] адаптер требует только синтаксически валидный путь и тело | [x] провал чтения тела → 400 (локально) | [x] неявно (Then 200/503 включают парсинг) | [x] один аргумент: `SessionFinishRequest` (path+body — поля одной DTO) |
| B  | [x] `SessionFinishRequest`, `TokenPair`, `error` | [x] `ProcessSessionFinish` | [x] head принимает любой Request; внутренняя валидация в (1) | [x] все ошибки ниже маппятся адаптером | [x] Then «ответ 200» Success-ветка B; Then «ответ 503» Failure-ветка B | [x] один data-аргумент `req`; `Deps` отдельно |
| 1  | [x] `SessionFinishRequest`, `SessionFinishCommand` | [x] `NewSessionFinishCommand` | [x] head передаёт уже распарсенный req | [x] `ErrInvalidLoginSessionID`/`ErrAssertionParse` маппятся в 422 | [x] неявно (Then 200) | [x] один аргумент `req` |
| 1a | [x] `string`, `LoginSessionID` | [x] `LoginSessionIDFromString` (S3 рехидратор) | [x] строка из path-параметра | [x] `ErrInvalidLoginSessionID` пробрасывается через `%w` | [x] неявно | [x] один аргумент |
| 1b | [x] `[]byte`, `ParsedAssertion` | [x] `parseAssertion` | [x] байты тела запроса | [x] `ErrAssertionParse` пробрасывается через `%w` | [x] неявно | [x] один аргумент |
| 2  | [x] `LoginSessionID`, `LoginSession`, `error` | [x] `Store.LoadLoginSession` | [x] `id` валиден из (1); миграция 0005 применена | [x] `ErrLoginSessionNotFound` → 404; `ErrDBLocked` → 503; прочее → 500 | [x] Success — Then 200; Failure-ветка `ErrDBLocked` — один из путей Then «503 + Retry-After + db_locked» (см. ниже про `db_locked` в S4) | [x] один data-аргумент `id` (`*sql.DB` инкапсулирован в `Store`, не виден стрелкой) |
| 3  | [x] `NewFreshLoginSessionInput`, `FreshLoginSession`, `error` | [x] `NewFreshLoginSession` | [x] Session — валидная сущность из (2); Now — момент | [x] `ErrLoginSessionExpired` → 404 | [x] неявно (Then 200) | [x] один аргумент `input` |
| 4  | [x] `LoadAssertionTargetInput`, `AssertionTarget`, `error` | [x] `Store.LoadAssertionTarget` | [x] UserID валиден из (3); CredentialID — байты из распарсенного assertion (1b); миграции 0002/0003 применены | [x] `ErrCredentialNotFound` → 404; `ErrDBLocked` → 503; прочее → 500 | [x] Success — Then 200; Failure-ветка `ErrDBLocked` — один из путей Then «503» | [x] один data-аргумент `input` (`*sql.DB` инкапсулирован в `Store`, не виден стрелкой) |
| 5  | [x] `AssertionVerification`, `VerifiedAssertion`, `error` | [x] `verifyAssertion` | [x] Fresh — non-expired; Parsed — синтаксически распарсенный; Target — credential «свой» (инвариант в типе) | [x] `ErrAssertionInvalid` → 422 ASSERTION_INVALID | [x] неявно (Then 200) | [x] один data-аргумент `input` (RPConfig — dep) |
| 6  | [x] `GenerateTokenPairInput`, `IssuedTokenPair`, `error` | [x] `GenerateTokenPair` (S2 экспорт) | [x] User валиден (из target); Now — момент; signer непустой; TTL > 0 | [x] catastrophic → 500 | [x] Then «непустое access_token»/«непустое refresh_token» — поля выхода | [x] один data-аргумент `input` (signer/jwtCfg — deps) |
| 7  | [x] `FinishLoginInput`, `error` | [x] `Store.FinishLogin` | [x] Credential, NewSignCount, RefreshHash, RefreshExpiresAt, LoginSessionID — валидны | [x] `ErrDBLocked` → 503; `ErrDiskFull` → 507; прочее → 500 | [x] Success — Then 200; Failure-ветка `ErrDBLocked` — один из путей Then «503» | [x] один data-аргумент `input` (`*sql.DB` инкапсулирован в `Store`, не виден стрелкой) |
| 8  | [x] `BuildTokenPairView`, `TokenPair` | [x] `BuildResponse` (S2 экспорт) | [x] view собран из выходов (6) | [x] нет ошибок | [x] Then 200 (формат `access_token`/`refresh_token`) | [x] один аргумент `view` |
| C  | [x] `TokenPair`, `error` | [x] handler.formatResponse | [x] head вернул либо Success, либо одну из известных ошибок | [x] полный маппинг в таблице ошибок карточки слайса | [x] Then 200 / 503 + Retry-After + `code=db_locked` | [x] один data-аргумент: либо `Response`, либо `error` |

**Все стрелки помечены `[x] согласовано`.**

### Покрытие Gherkin-сценариев графом (пункт 9.3.5)

В Gherkin для эндпоинта S4 — 6 Then-шагов (3 в «Завершение входа», 3 в «БД заблокирована при завершении входа»). Все покрыты, см. таблицу `## Gherkin-mapping` в `slices/04-sessions-finish.md`.

Цепочки:
- Happy: B → 1 → 2 (Success) → 3 (Success) → 4 (Success) → 5 (Success) → 6 (Success) → 7 (Success) → 8 → C (200)
- `db_locked`: B → 1 → (одна из (2)/(4)/(7) Failure: ErrDBLocked) → C (503 + Retry-After + db_locked)

**Замечание про `db_locked` в S4.** Compose-сценарий «БД заблокирована при завершении входа» открывает `BEGIN EXCLUSIVE TRANSACTION` в раннере **до** прихода запроса в SUT. SQLite EXCLUSIVE-lock блокирует и read'ы, и write'ы — поэтому первый же I/O-узел S4 (`Store.LoadLoginSession`, шаг 2) упрётся в `SQLITE_BUSY` и вернёт `ErrDBLocked`. Шаги (4) и (7) в этом сценарии не достигаются. Тем не менее **все три I/O-узла** обязаны корректно маппить `SQLITE_BUSY → ErrDBLocked` — это часть декларированного OpenAPI-контракта (`db_locked` декларирован на этом эндпоинте). В таблице сверки выше Then «503» формально привязан к ветке `ErrDBLocked` любого из узлов (2)/(4)/(7); в карточке слайса Gherkin-mapping явно указывает обе альтернативы.

Узлов графа, не упомянутых ни одним Then-шагом, **нет**. Failure-ветки (1)/(1a)/(1b)/(2 ErrLoginSessionNotFound)/(3 ErrLoginSessionExpired)/(4 ErrCredentialNotFound)/(5)/(6)/(7 ErrDiskFull) — пути, которые на этом эндпоинте Gherkin не проверяет (по сознательной раскладке: `db_disk_full` → S2; валидационные/доменные ошибки — задача юнит-тестов конструкторов и контракта OpenAPI). Это **не** мёртвая логика — часть декларированного OpenAPI-контракта.

### Сверка по правилу «один аргумент» (пункт 9.3.6) и автономии I/O-объекта (Шаг 6)

Все стрелки графа несут **ровно одну** data-сущность. Зависимости (`RPConfig`, `ed25519.PrivateKey`, `JWTConfig`) на стрелках не отображены — они в `Dependencies:` контракта модуля.

**Сырого `*sql.DB` ни на одной стрелке нет.** Узлы (2), (4), (7) — методы I/O-объекта `Store`, инкапсулирующего `*sql.DB` (Шаг 6 + `feedback_io_autonomous_store`). В `Deps` головного модуля — поле `*Store`, не сырой `*sql.DB`.

### Применение подправила «подтип, не guard» (Шаг 3 скилла)

Узел (3) `NewFreshLoginSession` — конструктор подтипа `FreshLoginSession`, инвариант «не истекла» закреплён в типе. Узлы (4) и (5) принимают `FreshLoginSession`/`AssertionTarget` (с инвариантом «credential свой»), не сырые `LoginSession`/`Credential` — что гарантировано системой типов.

Нет узлов со «висящей» сигнатурой `(Domain) -> ()` или `(input) -> error` — все логические шаги либо возвращают новую доменную структуру (1, 3, 5, 6, 8), либо являются I/O-эффектом с `error` (2, 4, 7). Правило соблюдено.

---

## I/O без юнитов (сверка с Шагом 8.1)

В таблице юнит-тестов карточки слайса 1 нет:
- `Store.PersistRegistrationSession` (метод I/O-объекта — труба);
- ингресс-адаптера (парсинг и маппинг — компонентным).

В таблице юнит-тестов карточки слайса 2 нет:
- `Store.LoadRegistrationSession` (метод I/O-объекта — труба);
- `Store.FinishRegistration` (метод I/O-объекта — труба, write-tx);
- ингресс-адаптера (парсинг path/body и маппинг — компонентным).

В таблице юнит-тестов карточки слайса 3 нет:
- `Store.LoadUserCredentials` (метод I/O-объекта — труба, read);
- `Store.PersistLoginSession` (метод I/O-объекта — труба, write);
- ингресс-адаптера (парсинг и маппинг — компонентным).

В таблице юнит-тестов карточки слайса 4 нет:
- `Store.LoadLoginSession` (метод I/O-объекта — труба, read);
- `Store.LoadAssertionTarget` (метод I/O-объекта — труба, read с двумя SELECT'ами и инвариант-проверкой);
- `Store.FinishLogin` (метод I/O-объекта — труба, write-tx);
- ингресс-адаптера (парсинг path/body и маппинг — компонентным);
- `GenerateTokenPair`, `BuildResponse` (импорт S2; юнит-тесты уже посчитаны в карточке S2).

Это соответствует жёсткому правилу Шага 8.1: I/O проверяется только компонентными сценариями, формула юнит-тестов считается только для модулей логики и конструкторов.

---

## Каталог сообщений: транзитивная замкнутость (9.1)

Прошёл `messages.md` для слайсов 1-4:

**Слайс 1:**
- `RegistrationStartRequest` — все поля примитивы.
- `Handle`, `Challenge`, `RegistrationID` — конструктор-валидируемые. У каждого описан конструктор `NewT(...) -> (T, error)` или генератор без ошибки. **Дополнительно для S2** добавлены рехидраторы `ChallengeFromBytes`, `RegistrationIDFromString`.
- `RegistrationStartCommand`, `RegistrationSession` — собираются конструкторами. **Дополнительно для S2** добавлен рехидратор `RegistrationSessionFromRow`.
- `NewRegistrationSessionInput`, `RegistrationStartView` — value-агрегаторы.
- `CreationOptions`, `RPInfo`, `UserInfo`, `PubKeyCredParam` — DTO-схемы по OpenAPI.
- `RegistrationStartResponse` — DTO ответа.
- Sentinel-ошибки `ErrHandle*`, `ErrDBLocked`, `ErrDiskFull` — определены.
- `RPConfig` — value-объект конфига. **В S2 расширяется** полем `Origin`.

**Слайс 2:**
- `RegistrationFinishRequest` — поля `string` + `[]byte`.
- `ParsedAttestation` — обёртка над `*protocol.ParsedCredentialCreationData` (внешний тип go-webauthn). Конструктор `parseAttestation([]byte) -> (ParsedAttestation, error)` описан.
- `RegistrationFinishCommand` — собирается `NewRegistrationFinishCommand`.
- `FreshRegistrationSession` — конструктор подтипа `NewFreshRegistrationSession(input) -> (FreshRegistrationSession, error)`; поля неэкспортируемые.
- `NewFreshSessionInput`, `AttestationVerification`, `NewUserInput`, `NewCredentialInput`, `GenerateTokenPairInput`, `FinishRegistrationInput`, `BuildTokenPairView` — value-агрегаторы.
- `VerifiedCredential` — собирается только `verifyAttestation`; поля неэкспортируемые.
- `UserID` — генератор без ошибки.
- `User`, `Credential` — собираются конструкторами `NewUser`, `NewCredential` (без ошибки, инварианты на нижних уровнях).
- `AccessToken`, `IssuedRefreshToken`, `IssuedTokenPair` — value-объекты выхода `generateTokenPair`.
- `TokenPair` — DTO ответа; собирается `buildResponse`.
- `JWTConfig` — value-объект конфига.
- Sentinel-ошибки `ErrInvalidRegID`, `ErrAttestationParse`, `ErrSessionNotFound`, `ErrSessionExpired`, `ErrAttestationInvalid`, `ErrHandleTaken` — определены. `ErrDBLocked`, `ErrDiskFull` — переиспользуются из S1.

**Слайс 3:**
- `SessionStartRequest` — поле `string`.
- `LoginSessionID` — генератор без ошибки (`generateLoginSessionID`).
- `SessionStartCommand` — собирается `NewSessionStartCommand`, делегирует `NewHandle` (S1).
- `UserWithCredentials` — агрегат, создаётся только методом `Store.LoadUserCredentials` (поля неэкспортируемые); инвариант непустого списка credentials.
- `LoginSession` — собирается `NewLoginSession` (без ошибки, инварианты на нижних уровнях).
- `NewLoginSessionInput`, `BuildRequestOptionsInput`, `SessionStartView` — value-агрегаторы.
- `RequestOptions`, `AllowCredentialDescriptor`, `SessionStartResponse` — DTO-схемы по OpenAPI.
- Sentinel-ошибки: `ErrUserNotFound` определена. `ErrHandle*`, `ErrDBLocked`, `ErrDiskFull` — переиспользуются из S1.
- **Аддитивные расширения S1:** `GenerateChallenge` экспортируется (обёртка над `generateChallenge`).
- **Аддитивные расширения S2:** рехидраторы `UserFromRow`, `CredentialFromRow`, `UserIDFromString` экспортируются.

**Слайс 4:**
- `SessionFinishRequest` — поля `string` + `[]byte`.
- `ParsedAssertion` — обёртка над `*protocol.ParsedCredentialAssertionData` (внешний тип go-webauthn). Конструктор `parseAssertion([]byte) -> (ParsedAssertion, error)` описан.
- `SessionFinishCommand` — собирается `NewSessionFinishCommand`, делегирует `LoginSessionIDFromString` (S3) и `parseAssertion`.
- `FreshLoginSession` — конструктор подтипа `NewFreshLoginSession(input) -> (FreshLoginSession, error)`; поля неэкспортируемые. Инвариант «не истекла».
- `AssertionTarget` — агрегат, создаётся только методом `Store.LoadAssertionTarget`; поля неэкспортируемые. Инвариант «credential.UserID() == user.ID()».
- `VerifiedAssertion` — собирается только `verifyAssertion`; поле `newSignCount uint32` неэкспортируемое.
- `NewFreshLoginSessionInput`, `LoadAssertionTargetInput`, `AssertionVerification`, `FinishLoginInput` — value-агрегаторы.
- `TokenPair`, `BuildTokenPairView`, `IssuedTokenPair`, `AccessToken`, `IssuedRefreshToken`, `GenerateTokenPairInput`, `JWTConfig` — импортируются из S2; используются как есть, без новых конструкторов.
- Sentinel-ошибки: `ErrInvalidLoginSessionID`, `ErrAssertionParse`, `ErrLoginSessionNotFound`, `ErrLoginSessionExpired`, `ErrCredentialNotFound`, `ErrAssertionInvalid` — определены. `ErrDBLocked`, `ErrDiskFull` — переиспользуются из S1.
- **Аддитивные расширения S2:** `GenerateTokenPair`, `BuildResponse` экспортируются (обёртки над пакетными `generateTokenPair`/`buildResponse`).
- **Аддитивные расширения S3:** рехидраторы `LoginSessionIDFromString`, `LoginSessionFromRow` экспортируются.

Ни одного «потом доопределим». Каталог замкнут для слайсов 1-4.
