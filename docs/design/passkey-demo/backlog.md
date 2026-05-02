# Backlog — passkey-demo (design pack)

Тикеты для sonnet'а. Один тикет = один слайс = одна ветка = один PR (TBD).

S1 спроектирован и реализован (PR #17). S2 спроектирован и реализован (PR #21). S3 спроектирован и реализован (PR #26). Техдолг S1/S2 → Store-объект закрыт (PR #27). S4 спроектирован (PR #28), ожидает реализации sonnet'ом. S5–S6 проектируются отдельными итерациями opus'а.

---

## Хендофф-чеклист S4 (заполняет opus, проверяет оператор)

- [x] OpenAPI / AsyncAPI зафиксирован, эндпоинт `POST /v1/sessions/{id}/assertion` описан с 200/404/422/503/507
- [x] OpenAPI / AsyncAPI содержит 5xx-ответы с `error.code` для режимов отказа `db_locked` и `db_disk_full`
- [x] README содержит таблицу «Карта режимов отказа» с этими режимами
- [x] **Компонентные сценарии Gherkin для эндпоинта S4 написаны, закоммичены, стабильны (`Сценарий: Завершение входа` + `Сценарий: БД заблокирована при завершении входа` в `sessions.feature`)**
- [x] Папка docs/design/passkey-demo/ дополнена под S4
- [x] intent.md — задача в одну фразу (без изменений)
- [x] slices.md — статус S4 → спроектирован
- [x] messages.md — все структуры данных S4 описаны (включая аддитивные расширения S2: `GenerateTokenPair`, `BuildResponse`; S3: `LoginSessionIDFromString`, `LoginSessionFromRow`)
- [x] Для S4 есть отдельный файл с деревом модулей (`slices/04-sessions-finish.md`)
- [x] У S4 описан головной модуль `ProcessSessionFinish` (оркестратор пайпа)
- [x] У головного модуля S4 зафиксирован псевдокод пайпа (8 узлов; 10 строк с импортированными `GenerateTokenPair`/`BuildResponse`; в диапазоне 5–10)
- [x] У каждого модуля логики S4 описаны антецедент и консеквент
- [x] У каждого I/O-модуля S4 описан контракт и режимы отказа — методы автономного объекта `Store` (`Store.LoadLoginSession`, `Store.LoadAssertionTarget`, `Store.FinishLogin`); сырого `*sql.DB` в `Deps` головного модуля нет (Шаг 6 + `feedback_io_autonomous_store`)
- [x] **У каждого модуля S4 Input — одна доменная структура / DTO / void; deps вынесены отдельной строкой `Dependencies:` (Шаг 5). Узлов с 2+ data-аргументами в графе нет**
- [x] **Карточка S4 содержит таблицу `## Gherkin-mapping`: каждый Then-шаг (3 в happy + 3 в `db_locked`, всего 6) привязан к узлу графа или маппингу адаптера**
- [x] **contracts-graph.md дополнен Slice 04: граф согласован (все стрелки `[x]`, включая пункт 5 о покрытии Gherkin)**
- [x] **Применено подправило «подтип, не guard» (Шаг 3 скилла): `NewFreshLoginSession` — конструктор подтипа, инвариант «не истекла» закреплён в типе. Дополнительно: инвариант «credential принадлежит user'у» инкапсулирован в I/O-возврате `AssertionTarget` (то же решение, что в S3 для `UserWithCredentials`)**
- [x] Для каждого модуля логики S4 посчитаны юнит-тесты по формуле (11 тестов)
- [x] **В таблице юнит-тестов S4 нет головного модуля, нет методов I/O-объекта (`Store.LoadLoginSession`, `Store.LoadAssertionTarget`, `Store.FinishLogin`) и нет ингресс-адаптера: все три — трубы, проверяются только компонентными сценариями (Шаг 8.1). `GenerateTokenPair`/`BuildResponse` — импорт S2, юниты уже посчитаны там**
- [x] infrastructure.md дополнен: `Deps` слайса 4 (без сырого `*sql.DB`), подключение `sessions_finish.Register` в `cmd/api/main.go`, явная отметка «новых миграций S4 не вводит» (использует `0003`/`0004`/`0005`)
- [x] backlog.md — тикет S4 (см. ниже) с DoD из карточки слайса
- [ ] Оператор аппрувит пакет S4 — @<github-handle>, <YYYY-MM-DD>

> **Замечание о scope.** Хендофф-чеклист подтверждается **для слайса S4**. Для S5–S6 пункты не применимы — карточки будут спроектированы отдельными итерациями. Sonnet берёт из backlog только тот тикет, который в нём прописан.

---

## Хендофф-чеклист S3 (исторический — для аудита)

- [x] OpenAPI / AsyncAPI зафиксирован, эндпоинт `POST /v1/sessions` описан с 201/404/422/503/507
- [x] OpenAPI / AsyncAPI содержит 5xx-ответы с `error.code` для режимов отказа `db_locked` и `db_disk_full`
- [x] README содержит таблицу «Карта режимов отказа» с этими режимами
- [x] **Компонентный сценарий Gherkin для эндпоинта S3 написан, закоммичен, стабилен (`Сценарий: Создание challenge входа` в `sessions.feature`)**
- [x] Папка docs/design/passkey-demo/ дополнена под S3
- [x] intent.md — задача в одну фразу (без изменений)
- [x] slices.md — статус S3 → спроектирован
- [x] messages.md — все структуры данных S3 описаны (включая аддитивные расширения S1: `GenerateChallenge`; и S2: `UserFromRow`, `CredentialFromRow`, `UserIDFromString`)
- [x] Для S3 есть отдельный файл с деревом модулей (`slices/03-sessions-start.md`)
- [x] У S3 описан головной модуль `ProcessSessionStart` (оркестратор пайпа)
- [x] У головного модуля S3 зафиксирован псевдокод пайпа (8 шагов; в диапазоне 5–10)
- [x] У каждого модуля логики S3 описаны антецедент и консеквент
- [x] У каждого I/O-модуля S3 описан контракт и режимы отказа — методы автономного объекта `Store` (`Store.LoadUserCredentials`, `Store.PersistLoginSession`); сырого `*sql.DB` в `Deps` головного модуля нет (Шаг 6 + `feedback_io_autonomous_store`)
- [x] **У каждого модуля S3 Input — одна доменная структура / DTO / void; deps вынесены отдельной строкой `Dependencies:` (Шаг 5). Узлов с 2+ data-аргументами в графе нет**
- [x] **Карточка S3 содержит таблицу `## Gherkin-mapping`: единственный Then-шаг сценария «Создание challenge входа» привязан к Success-цепочке узлов (1)–(8) → ингресс-адаптер**
- [x] **contracts-graph.md дополнен Slice 03: граф согласован (все стрелки `[x]`, включая пункт 5 о покрытии Gherkin)**
- [x] **Подправило «подтип, не guard» (Шаг 3 скилла): неприменимо к S3 — нет инвариантов над свежезагруженной сущностью; инвариант непустого списка credentials инкапсулирован в I/O-возврате `UserWithCredentials` (см. секцию «Дерево модулей»)**
- [x] Для каждого модуля логики S3 посчитаны юнит-тесты по формуле (6 тестов)
- [x] **В таблице юнит-тестов S3 нет головного модуля, нет методов I/O-объекта (`Store.LoadUserCredentials`, `Store.PersistLoginSession`) и нет ингресс-адаптера: все три — трубы, проверяются только компонентным сценарием (Шаг 8.1)**
- [x] infrastructure.md дополнен: миграция `0005_login_sessions.sql`, `Deps` слайса 3, подключение `sessions_start.Register` в `cmd/api/main.go`
- [x] backlog.md — тикет S3 (см. ниже) с DoD из карточки слайса
- [x] Оператор аппрувит пакет S3 — @maxmorev, 2026-05-02

> **Замечание о scope.** Хендофф-чеклист подтверждается **для слайса S3**. Для S4–S6 пункты не применимы — карточки будут спроектированы отдельными итерациями. Sonnet берёт из backlog только тот тикет, который в нём прописан.

---

## Хендофф-чеклист S2 (исторический — для аудита)

- [x] OpenAPI / AsyncAPI зафиксирован, эндпоинт `POST /v1/registrations/{id}/attestation` описан с 200/404/422/503/507
- [x] OpenAPI / AsyncAPI содержит 5xx-ответы с `error.code` для режимов отказа `db_locked` и `db_disk_full`
- [x] README содержит таблицу «Карта режимов отказа» с этими режимами
- [x] **Компонентные сценарии Gherkin для эндпоинта S2 написаны, закоммичены, стабильны (`Сценарий: Завершение регистрации` + `Сценарий: Диск переполнен при завершении регистрации` в `registrations.feature`)**
- [x] Папка docs/design/passkey-demo/ дополнена под S2
- [x] intent.md — задача в одну фразу (без изменений)
- [x] slices.md — статус S1 → реализован, S2 → спроектирован; раскладка failure-режимов и замечание о scope для S2 дополнены
- [x] messages.md — все структуры данных S2 описаны (включая аддитивные расширения S1: `RegistrationSessionFromRow`, `ChallengeFromBytes`, `RegistrationIDFromString`, `RPConfig.Origin`)
- [x] Для S2 есть отдельный файл с деревом модулей (`slices/02-registrations-finish.md`)
- [x] У S2 описан головной модуль `ProcessRegistrationFinish` (оркестратор пайпа)
- [x] У головного модуля S2 зафиксирован псевдокод пайпа (9 шагов; в диапазоне 5–10)
- [x] У каждого модуля логики S2 описаны антецедент и консеквент
- [x] У каждого I/O-модуля S2 описан контракт и режимы отказа (методы автономного объекта `Store`: `Store.LoadRegistrationSession`, `Store.FinishRegistration` — переписано в карточке S2 после ретро-правки 2026-05-02; в исходной реализации PR #21 это пакетные функции, см. техдолг в root `backlog.md`)
- [x] **У каждого модуля S2 Input — одна доменная структура / DTO / void; deps вынесены отдельной строкой `Dependencies:` (Шаг 5). Узлов с 2+ data-аргументами в графе нет**
- [x] **Карточка S2 содержит таблицу `## Gherkin-mapping`: каждый Then-шаг (3 в happy + 2 в `db_disk_full`, всего 5) привязан к узлу графа или маппингу адаптера**
- [x] **contracts-graph.md дополнен Slice 02: граф согласован (все стрелки `[x]`, включая пункт 5 о покрытии Gherkin)**
- [x] **Применено подправило «подтип, не guard» (Шаг 3 скилла): `NewFreshRegistrationSession` — конструктор подтипа, инвариант «не истекла» закреплён в типе**
- [x] Для каждого модуля логики S2 посчитаны юнит-тесты по формуле (16 тестов)
- [x] **В таблице юнит-тестов S2 нет головного модуля, нет методов I/O-объекта (`Store.LoadRegistrationSession`, `Store.FinishRegistration`) и нет ингресс-адаптера: все три — трубы, проверяются только компонентными сценариями (Шаг 8.1)**
- [x] infrastructure.md дополнен: новые env (`PASSKEY_RP_ORIGIN`, `PASSKEY_JWT_*`), генерация Ed25519 keypair, миграции `0002_users.sql`, `0003_credentials.sql`, `0004_refresh_tokens.sql`, `Deps` слайса 2
- [x] backlog.md — тикет S2 (см. ниже) с DoD из карточки слайса
- [x] Оператор аппрувит пакет S2 — @maxmorev, 2026-05-01

> **Замечание о scope.** Хендофф-чеклист подтверждается **для слайса S2**. Для S3–S6 пункты не применимы — карточки будут спроектированы отдельными итерациями. Sonnet берёт из backlog только тот тикет, который в нём прописан.

---

## Хендофф-чеклист S1 (исторический — для аудита)

S1 закрыт PR #17. Чеклист сохранён ниже для трассировки.

<details>
<summary>Чеклист S1 (свернуть/развернуть)</summary>

- [x] OpenAPI / AsyncAPI зафиксирован, все эндпоинты slice'ов в нём описаны
- [x] OpenAPI / AsyncAPI содержит 5xx-ответы с `error.code` для каждого режима отказа
- [x] README содержит таблицу «Карта режимов отказа»
- [x] **Компонентные сценарии Gherkin для эндпоинтов всех slice'ов написаны, закоммичены, стабильны**
- [x] Папка docs/design/passkey-demo/ создана и полна
- [x] intent.md — задача в одну фразу
- [x] slices.md — таблица срезов с типом входа, идентификатором, назначением
- [x] messages.md — все структуры данных и Result<T, Error> (для S1)
- [x] Для S1 есть отдельный файл с деревом модулей
- [x] У S1 описан головной модуль (оркестратор пайпа)
- [x] У головного модуля S1 зафиксирован псевдокод пайпа исполнения (5–10 шагов)
- [x] У каждого модуля логики S1 описаны антецедент и консеквент
- [x] У каждого I/O-модуля S1 описан контракт и режимы отказа
- [x] **У каждого модуля Input S1 — одна доменная структура / DTO / void; deps вынесены отдельной строкой `Dependencies:` (Шаг 5)**
- [x] **Карточка S1 содержит таблицу `## Gherkin-mapping`**
- [x] **contracts-graph.md существует, граф S1 согласован**
- [x] Для каждого модуля логики S1 посчитаны юнит-тесты по формуле
- [x] **В таблице юнит-тестов S1 нет головного модуля, нет I/O-модулей и нет ингресс-адаптера**
- [x] infrastructure.md — описан инфраструктурный модуль приложения
- [x] backlog.md — тикеты по одному на slice, с зависимостями
- [x] Оператор аппрувит пакет — @maxmorev, 2026-05-01

</details>

---

## Тикеты

### S1 — slice `registrations-start`: HTTP POST /v1/registrations

**Спецификация:**
- `docs/design/passkey-demo/slices/01-registrations-start.md`
- `docs/design/passkey-demo/messages.md`
- `docs/design/passkey-demo/contracts-graph.md` (Slice 01)
- `docs/design/passkey-demo/infrastructure.md`

**Зависимости:** —

**Ветка:** `feat/slice-registrations-start`

**Definition of Done:**

- [x] ингресс-адаптер реализован: парсит JSON в `RegistrationStartRequest`, без бизнес-валидации (HTTP handler в `internal/slice/registrations_start/`)
- [x] конструкторы доменных структур (`NewHandle`, `NewRegistrationStartCommand`, `NewRegistrationSession`) реализованы: проверяют антецедент, при невалидных данных возвращают ошибку (структура не создаётся)
- [x] модули логики (`generateChallenge`, `generateRegistrationID`, `buildCreationOptions`, `buildResponse`) реализованы, контракты выполнены
- [x] модуль I/O (`persistRegistrationSession`) реализован, оборачивает SQLITE_BUSY → `ErrDBLocked`, SQLITE_FULL → `ErrDiskFull`
- [x] головной модуль `ProcessRegistrationStart` реализован: пайп из 7 шагов, ранний возврат при ошибке через `fmt.Errorf("…: %w", err)`
- [x] миграция `internal/db/migrations/0001_registration_sessions.sql` создаёт таблицу `registration_sessions(id PRIMARY KEY, handle, challenge, expires_at)`
- [x] инфраструктурный модуль (`cmd/api/main.go`, `internal/app/`, `internal/db/`, `internal/clock/`) собран по `infrastructure.md`; placeholder из `devlog/06` заменён на реальный сервер с одним рабочим эндпоинтом и `/health`
- [x] слайс подключён через `registrations_start.Register(mux, deps)`: HTTP-роут `POST /v1/registrations` ведёт на ингресс-адаптер
- [x] юнит-тесты по формуле — **11 тестов на модули логики и конструкторы** (см. таблицу в карточке слайса), покрытие 100% по строкам и веткам логики; головной модуль, I/O-модуль и ингресс-адаптер юнитами не покрываются
- [x] компонентный сценарий `Сценарий: Создание challenge регистрации` (`component-tests/features/registrations.feature`) зелёный
- [x] остальные сценарии в `registrations.feature`, `sessions.feature`, `sessions-current.feature`, `users.feature` остаются красными в их Then-частях, но **не должны** ломаться по фазе 1 регистрации (When «отправляет POST /v1/registrations» возвращает валидный `id` и `options`)
- [x] локальный CI зелёный (`go test ./...` для юнитов и `./component-tests/scripts/run-tests.sh` для компонентных)
- [x] `backlog.md` обновлён по каждому подтверждённому пункту (правило `AGENTS.md §10`)
- [x] `docs/design/passkey-demo/devlog.md` дополнен блоком S1 (формат: `## S1 — HTTP POST /v1/registrations (<YYYY-MM-DD>)` + что сделано / решения / что застряло / тесты)
- [x] PR создан, описание заполнено по шаблону Шага 8 скилла sonnet'а
- [x] PR смержен в main, CI на main зелёный

**Ссылки на источники:**

- Скилл реализации: `skills/program-implementation/SKILL.md`
- Граф вызовов: `docs/design/passkey-demo/contracts-graph.md` Slice 01 (главный источник истины о форме стрелок)
- Gherkin-mapping: раздел `## Gherkin-mapping` в `slices/01-registrations-start.md`

---

### S2 — slice `registrations-finish`: HTTP POST /v1/registrations/{id}/attestation

**Спецификация:**
- `docs/design/passkey-demo/slices/02-registrations-finish.md` (главный документ)
- `docs/design/passkey-demo/messages.md` — секция «Структуры слайса 2»
- `docs/design/passkey-demo/contracts-graph.md` — секция «Slice 02»
- `docs/design/passkey-demo/infrastructure.md` — env-переменные `PASSKEY_RP_ORIGIN`/`PASSKEY_JWT_*`, генерация Ed25519, миграции `0002`-`0004`, Deps слайса 2

**Зависимости:** S1 (реализован, PR #17). Аддитивно расширяет S1 рехидраторами и полем `RPConfig.Origin` (см. ниже DoD).

**Ветка:** `feat/slice-registrations-finish`

**Внешние зависимости (новые go.mod записи):**
- `github.com/go-webauthn/webauthn` — серверная верификация attestation (используется подпакет `protocol`)
- `github.com/golang-jwt/jwt/v5` — выдача JWT Ed25519
- `github.com/descope/virtualwebauthn` — **test-dep**, для honest юнит-теста `verifyAttestation` (генерация валидных attestation в `*_test.go`)

**Definition of Done:**

- [x] **аддитивные расширения S1**: экспортированы `RegistrationSessionFromRow`, `ChallengeFromBytes`, `RegistrationIDFromString`; `RPConfig` расширен полем `Origin`. Юнит-тесты S1 остаются зелёными (без изменения существующих тестов).
- [x] ингресс-адаптер реализован: парсит path-параметр `{id}` и тело в `RegistrationFinishRequest`, без бизнес-валидации (HTTP handler в `internal/slice/registrations_finish/`).
- [x] конструкторы доменных структур (`NewRegistrationFinishCommand`, `NewFreshRegistrationSession`, `NewUser`, `NewCredential`) реализованы; невалидные данные → структура не создаётся, возвращается доменная ошибка.
- [x] модули логики (`parseAttestation`, `verifyAttestation`, `generateUserID`, `generateTokenPair`, `buildResponse`) реализованы, контракты выполнены.
- [x] модули I/O реализованы:
  - `loadRegistrationSession`: SELECT по id, рехидратор `RegistrationSessionFromRow`; маппинг `sql.ErrNoRows → ErrSessionNotFound`, `SQLITE_BUSY → ErrDBLocked`.
  - `finishRegistration`: одна транзакция, 4 операции; маппинг `SQLITE_CONSTRAINT_UNIQUE` на `users.handle → ErrHandleTaken`, `SQLITE_BUSY → ErrDBLocked`, `SQLITE_FULL → ErrDiskFull`.
- [x] головной модуль `ProcessRegistrationFinish` реализован: пайп из 9 шагов, ранний возврат при ошибке через `fmt.Errorf("…: %w", err)`.
- [x] миграции `internal/db/migrations/0002_users.sql`, `0003_credentials.sql`, `0004_refresh_tokens.sql` созданы по `infrastructure.md`.
- [x] инфраструктурный модуль расширен: `PASSKEY_RP_ORIGIN`, `PASSKEY_JWT_ACCESS_TTL`, `PASSKEY_JWT_REFRESH_TTL`, `PASSKEY_JWT_ISSUER` загружаются в `AppConfig`; Ed25519 keypair генерируется в `wire.go` при старте; `Deps` слайса 2 содержит `Signer`, `JWTConfig`.
- [x] слайс подключён через `registrations_finish.Register(mux, deps)`: HTTP-роут `POST /v1/registrations/{id}/attestation` ведёт на ингресс-адаптер.
- [x] юнит-тесты по формуле — **16 тестов** на модули логики и конструкторы (головной модуль, I/O-модули и ингресс-адаптер юнитами не покрываются).
- [x] `verifyAttestation` honest-тестируется через `virtualwebauthn` (без моков; happy + ветка с побитой подписью).
- [x] компонентный сценарий `Сценарий: Завершение регистрации` (`component-tests/features/registrations.feature`) зелёный.
- [x] компонентный сценарий `Сценарий: Диск переполнен при завершении регистрации` зелёный.
- [x] остальные сценарии в `sessions.feature`, `sessions-current.feature`, `users.feature` остаются красными в их Then-частях, но **не** ломаются по фазам S1+S2 (When-шаги «отправляет POST /v1/registrations» и «собирает attestation и отправляет его» работают).
- [x] локальный CI зелёный (`go test ./...` для юнитов и `./component-tests/scripts/run-tests.sh` для компонентных, оба профиля `healthy` и `disk-full`).
- [x] `backlog.md` обновлён по каждому подтверждённому пункту (правило `AGENTS.md §10`).
- [x] `docs/design/passkey-demo/devlog.md` дополнен блоком S2.
- [x] PR создан, описание заполнено по шаблону Шага 8 скилла sonnet'а.
- [x] PR смержен в main, CI на main зелёный.

**Ссылки на источники:**

- Скилл реализации: `skills/program-implementation/SKILL.md`
- Граф вызовов: `docs/design/passkey-demo/contracts-graph.md` Slice 02
- Gherkin-mapping: раздел `## Gherkin-mapping` в `slices/02-registrations-finish.md`
- Подправило «подтип, не guard»: `skills/program-design/SKILL.md` Шаг 3 (применено в узле (3) `NewFreshRegistrationSession`)

---

### S3 — slice `sessions-start`: HTTP POST /v1/sessions

**Спецификация:**
- `docs/design/passkey-demo/slices/03-sessions-start.md` (главный документ)
- `docs/design/passkey-demo/messages.md` — секция «Структуры слайса 3» + аддитивные расширения S1/S2
- `docs/design/passkey-demo/contracts-graph.md` — секция «Slice 03»
- `docs/design/passkey-demo/infrastructure.md` — миграция `0005_login_sessions.sql`, `Deps` слайса 3, подключение в `cmd/api/main.go`

**Зависимости:** S1 (PR #17, реализован), S2 (PR #21, реализован). Аддитивно расширяет S1 экспортом `GenerateChallenge`; S2 — рехидраторами `UserFromRow`, `CredentialFromRow`, `UserIDFromString`.

**Ветка:** `feat/slice-sessions-start`

**Внешние зависимости (новые go.mod записи):** —

S3 не вводит новых внешних зависимостей: WebAuthn-options строятся вручную (без `go-webauthn`), JWT не выдаются (только в S4). `github.com/google/uuid` уже в проекте.

**Definition of Done:**

- [ ] **аддитивные расширения слайса 1**: экспортирована `GenerateChallenge() (Challenge, error)`. Юнит-тесты S1 остаются зелёными (без изменения существующих тестов).
- [ ] **аддитивные расширения слайса 2**: экспортированы `UserFromRow`, `CredentialFromRow`, `UserIDFromString`. Юнит-тесты S2 остаются зелёными.
- [ ] миграция `internal/db/migrations/0005_login_sessions.sql` создана: таблица `login_sessions(id PRIMARY KEY, user_id REFERENCES users, challenge BLOB, expires_at INTEGER)` + индексы по `user_id` и `expires_at`.
- [ ] ингресс-адаптер реализован: парсит JSON в `SessionStartRequest`, без бизнес-валидации (HTTP handler в `internal/slice/sessions_start/`).
- [ ] конструкторы доменных структур (`NewSessionStartCommand`, `NewLoginSession`) реализованы; невалидные данные → структура не создаётся, возвращается доменная ошибка.
- [ ] модули логики (`generateLoginSessionID`, `buildRequestOptions`, `buildResponse`) реализованы, контракты выполнены.
- [ ] **I/O-объект `Store` реализован** как автономный объект, инкапсулирующий `*sql.DB`: тип `*Store` в пакете `internal/slice/sessions_start/`, конструктор `NewStore(db *sql.DB) *Store`, два метода:
  - `(s *Store) LoadUserCredentials(h Handle) (UserWithCredentials, error)`: SELECT по handle → user; SELECT по user_id → credentials. Если `sql.ErrNoRows` на user или `len(credentials) == 0` — `ErrUserNotFound`. `SQLITE_BUSY` → `ErrDBLocked`. Возвращает агрегат `UserWithCredentials` с инвариантом непустого списка credentials. Рехидраторы — `UserFromRow`, `CredentialFromRow` (S2).
  - `(s *Store) PersistLoginSession(ls LoginSession) error`: одна INSERT-операция; маппинг `SQLITE_BUSY` → `ErrDBLocked`, `SQLITE_FULL` → `ErrDiskFull`.
  - Голова `ProcessSessionStart` обращается к БД **только** через эти два метода; `*sql.DB` нигде кроме `Store` не светится в slice-пакете.
- [ ] головной модуль `ProcessSessionStart` реализован: пайп из 8 шагов, ранний возврат при ошибке через `fmt.Errorf("…: %w", err)`.
- [ ] инфраструктурный модуль расширен: `Deps` слайса 3 (`Store *Store`, `Clock`, `Logger`, `RP`, `ChallengeTTL` — **без** сырого `*sql.DB`); в `wire.go` создаётся `sessions_start.NewStore(db)` и пробрасывается в `Deps.Store`; подключение `sessions_start.Register(mux, deps.SessionsStart)` в `cmd/api/main.go`.
- [ ] слайс подключён через `sessions_start.Register(mux, deps)`: HTTP-роут `POST /v1/sessions` ведёт на ингресс-адаптер.
- [ ] юнит-тесты по формуле — **6 тестов** на модули логики и конструкторы (головной модуль, I/O-модули и ингресс-адаптер юнитами не покрываются).
- [ ] компонентный сценарий `Сценарий: Создание challenge входа` (`component-tests/features/sessions.feature`) зелёный.
- [ ] остальные сценарии в `sessions.feature`, `sessions-current.feature`, `users.feature` остаются красными в их Then-частях, но **не** ломаются по фазам S1+S2+S3 (When-шаг «отправляет POST /v1/sessions» возвращает валидные `id`, `options.challenge`, `options.allowCredentials`).
- [ ] локальный CI зелёный (`go test ./...` для юнитов и `./component-tests/scripts/run-tests.sh` для компонентных, профиль `healthy`).
- [ ] `backlog.md` обновлён по каждому подтверждённому пункту (правило `AGENTS.md §10`).
- [ ] `docs/design/passkey-demo/devlog.md` дополнен блоком S3.
- [ ] PR создан, описание заполнено по шаблону Шага 8 скилла sonnet'а.
- [ ] PR смержен в main, CI на main зелёный.

**Ссылки на источники:**

- Скилл реализации: `skills/program-implementation/SKILL.md`
- Граф вызовов: `docs/design/passkey-demo/contracts-graph.md` Slice 03
- Gherkin-mapping: раздел `## Gherkin-mapping` в `slices/03-sessions-start.md`

---

### S4 — slice `sessions-finish`: HTTP POST /v1/sessions/{id}/assertion

**Спецификация:**
- `docs/design/passkey-demo/slices/04-sessions-finish.md` (главный документ)
- `docs/design/passkey-demo/messages.md` — секция «Структуры слайса 4» + аддитивные расширения S2/S3
- `docs/design/passkey-demo/contracts-graph.md` — секция «Slice 04»
- `docs/design/passkey-demo/infrastructure.md` — секция «Подключение слайса 4 (S4)» + раздел «S4 — без новых миграций»

**Зависимости:**
- S1 (PR #17, реализован) — `Challenge`, `ChallengeFromBytes`, `ErrDBLocked`, `ErrDiskFull`.
- S2 (PR #21, реализован) — `User`, `UserID`, `UserIDFromString`, `Credential`, `CredentialFromRow`, `UserFromRow`, `JWTConfig`, `AccessToken`, `IssuedRefreshToken`, `IssuedTokenPair`, `GenerateTokenPairInput`, `BuildTokenPairView`, `TokenPair` + аддитивные расширения `GenerateTokenPair`, `BuildResponse`.
- S3 (PR #26, реализован) — `LoginSession`, `LoginSessionID` + аддитивные расширения `LoginSessionIDFromString`, `LoginSessionFromRow`.
- Техдолг S1/S2 → Store-объект — закрыт (PR #27): `Deps` всех слайсов используют `*Store`, сырого `*sql.DB` нет. Реализация S4 ляжет на однородный стиль.

**Ветка:** `feat/slice-sessions-finish`

**Внешние зависимости (новые go.mod записи):** —

S4 не вводит новых внешних зависимостей: `github.com/go-webauthn/webauthn` (через `protocol`-подпакет) уже подключён в S2 и используется в S4 для `parseAssertion` и `verifyAssertion`. `github.com/golang-jwt/jwt/v5` подключён в S2 и переиспользуется через экспорт `GenerateTokenPair`. `github.com/descope/virtualwebauthn` уже в test-deps (для `verifyAttestation` в S2 и компонентных тестов).

**Definition of Done:**

- [ ] **аддитивные расширения слайса 2**: экспортированы `GenerateTokenPair(input GenerateTokenPairInput) (IssuedTokenPair, error)` и `BuildResponse(view BuildTokenPairView) TokenPair` (публичные обёртки над пакетными `generateTokenPair`/`buildResponse`). Юнит-тесты S2 остаются зелёными (без изменения существующих тестов; тесты вызывают публичные имена).
- [ ] **аддитивные расширения слайса 3**: экспортированы `LoginSessionIDFromString(s string) (LoginSessionID, error)` и `LoginSessionFromRow(rowID, rowUserID string, rowChallenge []byte, rowExpiresAtUnix int64) (LoginSession, error)`. Юнит-тесты S3 остаются зелёными.
- [x] **техдолг S1/S2 (Store-объект) закрыт** — `refactor/s1-s2-store` смержен в main (PR #27) до начала реализации S4. `Deps` всех реализованных слайсов используют `*Store`, сырого `*sql.DB` в `Deps` нигде нет.
- [ ] ингресс-адаптер реализован: парсит path-параметр и тело в `SessionFinishRequest`, без бизнес-валидации (HTTP handler в `internal/slice/sessions_finish/`).
- [ ] конструкторы доменных структур (`NewSessionFinishCommand`, `NewFreshLoginSession`) реализованы; невалидные данные → структура не создаётся, возвращается доменная ошибка.
- [ ] модули логики (`parseAssertion`, `verifyAssertion`) реализованы, контракты выполнены.
- [ ] **I/O-объект `Store` реализован** как автономный объект, инкапсулирующий `*sql.DB`: тип `*Store` в пакете `internal/slice/sessions_finish/`, конструктор `NewStore(db *sql.DB) *Store`, три метода:
  - `(s *Store) LoadLoginSession(id LoginSessionID) (LoginSession, error)`: SELECT по id, рехидратор `LoginSessionFromRow`; маппинг `sql.ErrNoRows` → `ErrLoginSessionNotFound`, `SQLITE_BUSY` → `ErrDBLocked`.
  - `(s *Store) LoadAssertionTarget(input LoadAssertionTargetInput) (AssertionTarget, error)`: SELECT credential по `credential_id` → in-memory проверка `user_id == input.UserID` → SELECT user по `id`; маппинг `sql.ErrNoRows`/mismatch → `ErrCredentialNotFound`, `SQLITE_BUSY` → `ErrDBLocked`. Рехидраторы — `CredentialFromRow`, `UserFromRow` (S2).
  - `(s *Store) FinishLogin(input FinishLoginInput) error`: атомарная транзакция (3 операции: UPDATE credentials + INSERT refresh_tokens + DELETE login_sessions), откат при любой ошибке; маппинг `SQLITE_BUSY` → `ErrDBLocked`, `SQLITE_FULL` → `ErrDiskFull`.
  - Голова `ProcessSessionFinish` обращается к БД **только через эти три метода**; `*sql.DB` нигде кроме `Store` не светится в slice-пакете.
- [ ] головной модуль `ProcessSessionFinish` реализован: пайп из 8 шагов (10 строк с импортированными `GenerateTokenPair` и `BuildResponse`), ранний возврат при ошибке через `fmt.Errorf("…: %w", err)`.
- [ ] **новых миграций нет** — слайс использует `0003_credentials.sql` (поле `sign_count`), `0004_refresh_tokens.sql` (INSERT), `0005_login_sessions.sql` (DELETE). Никаких ALTER TABLE / новых файлов в `internal/db/migrations/` для S4 не создавать.
- [ ] инфраструктурный модуль расширен: `Deps` слайса 4 (`Store *Store`, `Clock`, `Logger`, `RP` (S1, нужны `ID` и `Origin`), `JWT` (S2), `Signer` ed25519.PrivateKey — **без** сырого `*sql.DB`); подключение `sessions_finish.Register(mux, deps.SessionsFinish)` в `cmd/api/main.go`; в `wire.go` создаётся `sessions_finish.NewStore(db)` и пробрасывается в `Deps.Store`.
- [ ] слайс подключён через `sessions_finish.Register(mux, deps)`: HTTP-роут `POST /v1/sessions/{id}/assertion` ведёт на ингресс-адаптер.
- [ ] **юнит-тесты по формуле написаны и зелёные** — `go test ./...` проходит. **11 новых тестов** на модули логики и конструкторы S4 (`parseAssertion`, `NewSessionFinishCommand`, `NewFreshLoginSession`, `verifyAssertion`); головной модуль, I/O-модули и ингресс-адаптер юнитами не покрываются. `verifyAssertion` honest-тестируется через `virtualwebauthn`. Юниты S1/S2/S3 остаются зелёными после аддитивных расширений (`GenerateTokenPair`, `BuildResponse`, `LoginSessionIDFromString`, `LoginSessionFromRow`).
- [ ] **компонентные тесты, профиль `healthy`, зелёные** — `./component-tests/scripts/run-tests.sh healthy` проходит. Новые зелёные сценарии: `Сценарий: Завершение входа`, `Сценарий: БД заблокирована при завершении входа` (`sessions.feature`). Ранее зелёные сценарии S1/S2/S3 в `registrations.feature` (Создание challenge регистрации, Завершение регистрации) и `sessions.feature` (Создание challenge входа) продолжают проходить.
- [ ] **компонентные тесты, профиль `disk-full`, зелёные** — `./component-tests/scripts/run-tests.sh disk-full` проходит. Regression-проверка: `Сценарий: Диск переполнен при завершении регистрации` (`registrations.feature`) из S2 продолжает проходить — изменения S4 (новые экспорты `GenerateTokenPair`/`BuildResponse`, новый слайс) не должны ломать `db_disk_full` маппинг.
- [ ] сценарии в `sessions-current.feature`/`users.feature` остаются красными в их Then-частях (S5/S6 ещё не реализованы), но **не** ломаются на When-шагах S1–S4 — `POST /v1/sessions/{id}/assertion` возвращает валидные `access_token`/`refresh_token`, которые используются как Bearer-токен в S5/S6.
- [ ] `backlog.md` обновлён по каждому подтверждённому пункту (правило `AGENTS.md §10`).
- [ ] `docs/design/passkey-demo/devlog.md` дополнен блоком S4.
- [ ] PR создан, описание заполнено по шаблону Шага 8 скилла sonnet'а.
- [ ] PR смержен в main, CI на main зелёный.

**Ссылки на источники:**

- Скилл реализации: `skills/program-implementation/SKILL.md`
- Граф вызовов: `docs/design/passkey-demo/contracts-graph.md` Slice 04
- Gherkin-mapping: раздел `## Gherkin-mapping` в `slices/04-sessions-finish.md`
- Подправило «подтип, не guard»: `skills/program-design/SKILL.md` Шаг 3 (применено в узле (3) `NewFreshLoginSession`; инвариант `AssertionTarget` инкапсулирован в I/O-возврате)

---

## Следующие итерации (S5–S6)

S4 спроектирован, ожидает аппрува оператора и реализации sonnet'ом. После реализации S4 — следующая итерация S5. Каждый слайс — отдельная ветка `feat/design-<slice>`, отдельный PR, отдельный хендофф-чеклист, подписанный оператором.

Очерёдность:

- **S5** `slices/05-sessions-logout.md` — `DELETE /v1/sessions/current` (отзыв refresh token) — **следующая итерация после S4**
- **S6** `slices/06-users-me.md` — `GET /v1/users/me` (профиль пользователя по access token)

Каждая итерация расширяет `messages.md`, `contracts-graph.md` и добавляет тикет в `backlog.md`.
