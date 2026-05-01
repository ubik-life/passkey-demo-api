# Backlog — passkey-demo (design pack)

Тикеты для sonnet'а. Один тикет = один слайс = одна ветка = один PR (TBD).

S1 спроектирован и реализован (PR #17). В **этой итерации** спроектирован S2 (`registrations-finish`); S3–S6 будут добавлены в следующих итерациях opus'ом после мержа S2.

---

## Хендофф-чеклист S2 (заполняет opus, проверяет оператор)

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
- [x] У каждого I/O-модуля S2 описан контракт и режимы отказа (`loadRegistrationSession`, `finishRegistration`)
- [x] **У каждого модуля S2 Input — одна доменная структура / DTO / void; deps вынесены отдельной строкой `Dependencies:` (Шаг 5). Узлов с 2+ data-аргументами в графе нет**
- [x] **Карточка S2 содержит таблицу `## Gherkin-mapping`: каждый Then-шаг (3 в happy + 2 в `db_disk_full`, всего 5) привязан к узлу графа или маппингу адаптера**
- [x] **contracts-graph.md дополнен Slice 02: граф согласован (все стрелки `[x]`, включая пункт 5 о покрытии Gherkin)**
- [x] **Применено подправило «подтип, не guard» (Шаг 3 скилла): `NewFreshRegistrationSession` — конструктор подтипа, инвариант «не истекла» закреплён в типе**
- [x] Для каждого модуля логики S2 посчитаны юнит-тесты по формуле (23 теста)
- [x] **В таблице юнит-тестов S2 нет I/O-модулей (`loadRegistrationSession`, `finishRegistration`) и нет ингресс-адаптера: I/O — трубы, проверяются только компонентными сценариями (Шаг 8.1)**
- [x] infrastructure.md дополнен: новые env (`PASSKEY_RP_ORIGIN`, `PASSKEY_JWT_*`), генерация Ed25519 keypair, миграции `0002_users.sql`, `0003_credentials.sql`, `0004_refresh_tokens.sql`, `Deps` слайса 2
- [x] backlog.md — тикет S2 (см. ниже) с DoD из карточки слайса
- [ ] Оператор аппрувит пакет S2 — @<github-handle>, <YYYY-MM-DD>

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
- [x] **В таблице юнит-тестов S1 нет I/O-модулей и нет ингресс-адаптера**
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
- [x] юнит-тесты по формуле — **14 тестов на модули логики и головной модуль** (см. таблицу в карточке слайса), покрытие 100% по строкам и веткам логики; I/O-модуль и ингресс-адаптер юнитами не покрываются
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

- [ ] **аддитивные расширения S1**: экспортированы `RegistrationSessionFromRow`, `ChallengeFromBytes`, `RegistrationIDFromString`; `RPConfig` расширен полем `Origin`. Юнит-тесты S1 остаются зелёными (без изменения существующих тестов).
- [ ] ингресс-адаптер реализован: парсит path-параметр `{id}` и тело в `RegistrationFinishRequest`, без бизнес-валидации (HTTP handler в `internal/slice/registrations_finish/`).
- [ ] конструкторы доменных структур (`NewRegistrationFinishCommand`, `NewFreshRegistrationSession`, `NewUser`, `NewCredential`) реализованы; невалидные данные → структура не создаётся, возвращается доменная ошибка.
- [ ] модули логики (`parseAttestation`, `verifyAttestation`, `generateUserID`, `generateTokenPair`, `buildResponse`) реализованы, контракты выполнены.
- [ ] модули I/O реализованы:
  - `loadRegistrationSession`: SELECT по id, рехидратор `RegistrationSessionFromRow`; маппинг `sql.ErrNoRows → ErrSessionNotFound`, `SQLITE_BUSY → ErrDBLocked`.
  - `finishRegistration`: одна транзакция, 4 операции; маппинг `SQLITE_CONSTRAINT_UNIQUE` на `users.handle → ErrHandleTaken`, `SQLITE_BUSY → ErrDBLocked`, `SQLITE_FULL → ErrDiskFull`.
- [ ] головной модуль `ProcessRegistrationFinish` реализован: пайп из 9 шагов, ранний возврат при ошибке через `fmt.Errorf("…: %w", err)`.
- [ ] миграции `internal/db/migrations/0002_users.sql`, `0003_credentials.sql`, `0004_refresh_tokens.sql` созданы по `infrastructure.md`.
- [ ] инфраструктурный модуль расширен: `PASSKEY_RP_ORIGIN`, `PASSKEY_JWT_ACCESS_TTL`, `PASSKEY_JWT_REFRESH_TTL`, `PASSKEY_JWT_ISSUER` загружаются в `AppConfig`; Ed25519 keypair генерируется в `wire.go` при старте; `Deps` слайса 2 содержит `Signer`, `JWTConfig`, `Rand`.
- [ ] слайс подключён через `registrations_finish.Register(mux, deps)`: HTTP-роут `POST /v1/registrations/{id}/attestation` ведёт на ингресс-адаптер.
- [ ] юнит-тесты по формуле — **23 теста** (см. таблицу в карточке слайса); покрытие 100% по строкам и веткам логики; I/O-модули и ингресс-адаптер юнитами не покрываются.
- [ ] `verifyAttestation` honest-тестируется через `virtualwebauthn` (без моков; happy + ветка с побитой подписью).
- [ ] `ProcessRegistrationFinish` honest-тестируется с in-memory SQLite (`mattn/go-sqlite3 :memory:`); без моков по `feedback_no_mocks`.
- [ ] компонентный сценарий `Сценарий: Завершение регистрации` (`component-tests/features/registrations.feature`) зелёный.
- [ ] компонентный сценарий `Сценарий: Диск переполнен при завершении регистрации` зелёный.
- [ ] остальные сценарии в `sessions.feature`, `sessions-current.feature`, `users.feature` остаются красными в их Then-частях, но **не** ломаются по фазам S1+S2 (When-шаги «отправляет POST /v1/registrations» и «собирает attestation и отправляет его» работают).
- [ ] локальный CI зелёный (`go test ./...` для юнитов и `./component-tests/scripts/run-tests.sh` для компонентных, оба профиля `healthy` и `disk-full`).
- [ ] `backlog.md` обновлён по каждому подтверждённому пункту (правило `AGENTS.md §10`).
- [ ] `docs/design/passkey-demo/devlog.md` дополнен блоком S2 (формат: `## S2 — HTTP POST /v1/registrations/{id}/attestation (<YYYY-MM-DD>)` + что сделано / решения / что застряло / тесты).
- [ ] PR создан, описание заполнено по шаблону Шага 8 скилла sonnet'а.
- [ ] PR смержен в main, CI на main зелёный.

**Ссылки на источники:**

- Скилл реализации: `skills/program-implementation/SKILL.md`
- Граф вызовов: `docs/design/passkey-demo/contracts-graph.md` Slice 02
- Gherkin-mapping: раздел `## Gherkin-mapping` в `slices/02-registrations-finish.md`
- Подправило «подтип, не guard»: `skills/program-design/SKILL.md` Шаг 3 (применено в узле (3) `NewFreshRegistrationSession`)

---

## Заметка для следующих итераций (S3–S6)

Когда S2 смержен и зелёный, opus возвращается на `program-design.skill` и наполняет:

- `slices/03-sessions-start.md` (фаза 1 входа)
- `slices/04-sessions-finish.md` (режим `db_locked` на этом эндпоинте; здесь же интеграция со счётчиком signCount)
- `slices/05-sessions-logout.md`
- `slices/06-users-me.md`

Каждая итерация — отдельный `feat/design-<slice>` PR с расширением `messages.md`, `contracts-graph.md` и `backlog.md`. Хендофф-чеклист переподписывается оператором на каждой итерации.
