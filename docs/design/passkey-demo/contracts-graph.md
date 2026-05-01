# Contracts Graph — passkey-demo

Граф вызовов модулей слайсов и сверка согласованности контрактов (Шаг 9 `program-design.skill`).

На текущей итерации (S1) граф нарисован для одного слайса — `registrations-start`. Слайсы 2-6 будут добавлены в следующих итерациях.

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

## I/O без юнитов (сверка с Шагом 8.1)

В таблице юнит-тестов карточки слайса 1 нет:
- `persistRegistrationSession` (I/O — труба);
- ингресс-адаптера (парсинг и маппинг — компонентным).

Это соответствует жёсткому правилу Шага 8.1: I/O проверяется только компонентными сценариями, формула юнит-тестов считается только для модулей логики и конструкторов.

---

## Каталог сообщений: транзитивная замкнутость (9.1)

Прошёл `messages.md`:

- `RegistrationStartRequest` — все поля примитивы.
- `Handle`, `Challenge`, `RegistrationID` — конструктор-валидируемые. У каждого описан конструктор `NewT(...) -> (T, error)` или генератор без ошибки.
- `RegistrationStartCommand`, `RegistrationSession` — собираются конструкторами из доменных значений.
- `NewRegistrationSessionInput`, `RegistrationStartView` — value-агрегаторы для соблюдения «один data-аргумент»; не имеют конструктора (Go-литерал структуры на месте сборки).
- `CreationOptions`, `RPInfo`, `UserInfo`, `PubKeyCredParam` — DTO-схемы по OpenAPI; собираются `buildCreationOptions`.
- `RegistrationStartResponse` — DTO ответа; собирается `buildResponse`.
- Sentinel-ошибки `ErrHandle*`, `ErrDBLocked`, `ErrDiskFull` — определены в `messages.md`.
- `RPConfig` — value-объект конфига, передаётся как dep.

Ни одного «потом доопределим». Каталог замкнут.
