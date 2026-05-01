# Contracts Graph — passkey-demo

Граф вызовов модулей слайсов и сверка согласованности контрактов (Шаг 9 `program-design.skill`).

На текущей итерации спроектированы графы слайсов 1 (`registrations-start`) и 2 (`registrations-finish`). Слайсы 3-6 будут добавлены в следующих итерациях.

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
   |-- (5) persistRegistrationSession:
   |        in:  RegistrationSession
   |        out: error                      [Success: () | Failure: ErrDBLocked, ErrDiskFull, ...]
   |        deps: *sql.DB
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
| 5  | `ProcessRegistrationStart`| `persistRegistrationSession`   | `RegistrationSession`                  | `error`                             | `ErrDBLocked`, `ErrDiskFull`, низкоуровневые SQLite (→ 500) |
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
| 5  | [x] `RegistrationSession`, `error` | [x] `persistRegistrationSession` | [x] `s` — валидная доменная сущность | [x] `ErrDBLocked` → 503, `ErrDiskFull` → 507, прочее → 500 | [x] Success-путь — Then 201; Failure-ветки — компонентные сценарии слайсов 4 и 2 | [x] один data-аргумент `s` (db — dep) |
| 6  | [x] `RegistrationSession`, `CreationOptions` | [x] `buildCreationOptions` | [x] `s` валидна; `RPConfig.Name`/`ID` непустые из конфига | [x] нет ошибок | [x] Then 201 (формат `options.challenge`, `options.user.id`) | [x] один data-аргумент `s` (RPConfig — dep) |
| 7  | [x] `RegistrationStartView`, `RegistrationStartResponse` | [x] `buildResponse` | [x] view собран из шагов (4) и (6) | [x] нет ошибок | [x] Then 201 (формат `id`, `options`) | [x] один аргумент `view` |
| C  | [x] `RegistrationStartResponse`, `error` | [x] handler.formatResponse | [x] head вернул либо Success, либо одну из известных ошибок | [x] полный маппинг в таблице ошибок карточки слайса | [x] Then 201 / 422 / 503+Retry-After / 507 | [x] один data-аргумент: либо `Response`, либо `error` |

**Все стрелки помечены `[x] согласовано`.**

### Покрытие Gherkin-сценариев графом (пункт 9.3.5)

В Gherkin для этого эндпоинта один Then-шаг — `Тогда ответ 201`. Он покрыт цепочкой узлов B → 1 → 2 → 3 → 4 → 5 (Success) → 6 → 7 → C.

Узлов графа, не упомянутых ни одним Then-шагом, **нет**. Узлы (5 Failure), (1 Failure), (2 Failure), маппинг ошибок в C — отвечают за пути, которые на этом эндпоинте Gherkin не проверяет (по сознательной раскладке режимов отказа: `db_locked` → слайс 4, `db_disk_full` → слайс 2; валидационные ошибки — задача юнит-тестов конструкторов и компонентного теста соответствующего эндпоинта). Это **не** мёртвая логика — это часть декларированного OpenAPI-контракта.

### Сверка по правилу «один аргумент» (пункт 9.3.6)

Все стрелки графа несут **ровно одну** data-сущность. Зависимости (`*sql.DB`, `RPConfig`) на стрелках не отображены — они входят в `Dependencies:` контракта модуля и инжектируются на уровне инфраструктурного модуля.

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
   |-- (2) loadRegistrationSession:
   |        in:  RegistrationID
   |        out: (RegistrationSession, error)
   |        deps: *sql.DB
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
   |-- (8) finishRegistration:
   |        in:  FinishRegistrationInput { User, Credential, RefreshTokenHash,
   |                                       RefreshExpiresAt, RegistrationID }
   |        out: error
   |        deps: *sql.DB
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
| 2  | `ProcessRegistrationFinish`| `loadRegistrationSession`      | `RegistrationID`                          | `(RegistrationSession, error)`          | `ErrSessionNotFound`, `ErrDBLocked`, низкоуровневые |
| 3  | `ProcessRegistrationFinish`| `NewFreshRegistrationSession`  | `NewFreshSessionInput`                    | `(FreshRegistrationSession, error)`     | `ErrSessionExpired`                                  |
| 4  | `ProcessRegistrationFinish`| `verifyAttestation`             | `AttestationVerification`                | `(VerifiedCredential, error)`           | `ErrAttestationInvalid`                              |
| 5  | `ProcessRegistrationFinish`| `NewUser`                       | `NewUserInput`                            | `User`                                  | —                                                   |
| 5a | `ProcessRegistrationFinish`| `generateUserID`                | void                                      | `UserID`                                | —                                                   |
| 6  | `ProcessRegistrationFinish`| `NewCredential`                 | `NewCredentialInput`                      | `Credential`                            | —                                                   |
| 7  | `ProcessRegistrationFinish`| `generateTokenPair`             | `GenerateTokenPairInput`                  | `(IssuedTokenPair, error)`              | catastrophic (→ 500)                                |
| 8  | `ProcessRegistrationFinish`| `finishRegistration`            | `FinishRegistrationInput`                 | `error`                                 | `ErrHandleTaken`, `ErrDBLocked`, `ErrDiskFull`, низкоуровневые |
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
| 2  | [x] `RegistrationID`, `RegistrationSession`, `error` | [x] `loadRegistrationSession` | [x] `id` валиден из (1); миграция 0001 применена | [x] `ErrSessionNotFound` → 404; `ErrDBLocked` → 503; прочее → 500 | [x] Success — Then 200; Failure-ветки — без отдельного Then на этом эндпоинте | [x] один data-аргумент `id` (db — dep) |
| 3  | [x] `NewFreshSessionInput`, `FreshRegistrationSession`, `error` | [x] `NewFreshRegistrationSession` | [x] Session — валидная сущность из (2); Now — момент | [x] `ErrSessionExpired` → 404 | [x] неявно (Then 200) | [x] один аргумент `input` |
| 4  | [x] `AttestationVerification`, `VerifiedCredential`, `error` | [x] `verifyAttestation` | [x] Fresh — non-expired; Parsed — синтаксически распарсенный | [x] `ErrAttestationInvalid` → 422 ATTESTATION_INVALID | [x] неявно (Then 200) | [x] один data-аргумент `input` (RPConfig — dep) |
| 5  | [x] `NewUserInput`, `User` | [x] `NewUser` | [x] ID, Handle, CreatedAt — валидные | [x] нет ошибок | [x] неявно | [x] один аргумент `input` |
| 5a | [x] `UserID` | [x] `generateUserID` | [x] нет антецедента | [x] нет ошибок | [x] неявно | [x] void |
| 6  | [x] `NewCredentialInput`, `Credential` | [x] `NewCredential` | [x] User валиден; Verified — успех (4); CreatedAt — момент | [x] нет ошибок | [x] неявно | [x] один аргумент `input` |
| 7  | [x] `GenerateTokenPairInput`, `IssuedTokenPair`, `error` | [x] `generateTokenPair` | [x] User валиден; Now — момент; signer непустой; TTL > 0 | [x] catastrophic → 500 | [x] Then «непустое access_token»/«непустое refresh_token» — поля выхода | [x] один data-аргумент `input` (signer/jwtCfg/rand — deps) |
| 8  | [x] `FinishRegistrationInput`, `error` | [x] `finishRegistration` | [x] User, Credential, RefreshHash, RefreshExpiresAt, RegistrationID — валидны | [x] `ErrHandleTaken` → 422; `ErrDBLocked` → 503; `ErrDiskFull` → 507; прочее → 500 | [x] Then «507» — Failure: `ErrDiskFull`; Then «200» — Success | [x] один data-аргумент `input` (db — dep) |
| 9  | [x] `BuildTokenPairView`, `TokenPair` | [x] `buildResponse` | [x] view собран из выходов (5)/(7) | [x] нет ошибок | [x] Then 200 (формат `access_token`/`refresh_token`) | [x] один аргумент `view` |
| C  | [x] `TokenPair`, `error` | [x] handler.formatResponse | [x] head вернул либо Success, либо одну из известных ошибок | [x] полный маппинг в таблице ошибок карточки слайса | [x] Then 200 / 507 + `code=db_disk_full` | [x] один data-аргумент: либо `Response`, либо `error` |

**Все стрелки помечены `[x] согласовано`.**

### Покрытие Gherkin-сценариев графом (пункт 9.3.5)

В Gherkin для эндпоинта S2 — 5 Then-шагов (3 в «Завершение регистрации», 2 в «Диск переполнен при завершении регистрации»). Все покрыты, см. таблицу `## Gherkin-mapping` в `slices/02-registrations-finish.md`.

Цепочки:
- Happy: B → 1 → 2 (Success) → 3 (Success) → 4 (Success) → 5 → 5a → 6 → 7 (Success) → 8 (Success) → 9 → C (200)
- `db_disk_full`: B → 1 → 2 (Success) → 3 (Success) → 4 (Success) → 5 → 5a → 6 → 7 (Success) → 8 (Failure: ErrDiskFull) → C (507)

Узлы графа, не упомянутые ни одним Then-шагом, **нет**. Failure-ветки (1)/(1a)/(1b)/(2 ErrSessionNotFound, ErrDBLocked)/(3)/(4)/(7)/(8 ErrHandleTaken, ErrDBLocked) — пути, которые на этом эндпоинте Gherkin не проверяет (по сознательной раскладке: `db_locked` → слайс 4; валидационные/доменные ошибки — задача юнит-тестов конструкторов и контракта OpenAPI). Это **не** мёртвая логика — часть декларированного OpenAPI-контракта.

### Сверка по правилу «один аргумент» (пункт 9.3.6)

Все стрелки графа несут **ровно одну** data-сущность. Зависимости (`*sql.DB`, `RPConfig`, `ed25519.PrivateKey`, `JWTConfig`, `io.Reader`) на стрелках не отображены — они в `Dependencies:` контракта модуля.

### Применение подправила «подтип, не guard» (Шаг 3 скилла)

Узел (3) `NewFreshRegistrationSession` — конструктор подтипа `FreshRegistrationSession`, инвариант «не истекла» закреплён в типе. Узлы (4) и (5) принимают `FreshRegistrationSession` (не сырой `RegistrationSession`), что гарантировано системой типов.

Нет узлов со «висящей» сигнатурой `(Domain) -> ()` или `(input) -> error` — все логические шаги либо возвращают новую доменную структуру (1, 3, 4, 5, 6, 7, 9), либо являются I/O-эффектом с `error` (2, 8, и узел C-маппинг). Правило соблюдено.

---

## I/O без юнитов (сверка с Шагом 8.1)

В таблице юнит-тестов карточки слайса 1 нет:
- `persistRegistrationSession` (I/O — труба);
- ингресс-адаптера (парсинг и маппинг — компонентным).

В таблице юнит-тестов карточки слайса 2 нет:
- `loadRegistrationSession` (I/O — труба);
- `finishRegistration` (I/O — труба, write-tx);
- ингресс-адаптера (парсинг path/body и маппинг — компонентным).

Это соответствует жёсткому правилу Шага 8.1: I/O проверяется только компонентными сценариями, формула юнит-тестов считается только для модулей логики и конструкторов.

---

## Каталог сообщений: транзитивная замкнутость (9.1)

Прошёл `messages.md` для слайсов 1 и 2:

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

Ни одного «потом доопределим». Каталог замкнут для слайсов 1-2.
