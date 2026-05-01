# 08 — Дизайн пакета S2 (`registrations-finish`)

## Задача

Спроектировать второй слайс сервиса — `POST /v1/registrations/{id}/attestation` — по дисциплине `program-design.skill`. На входе: контракт OpenAPI, карта режимов отказа в README, два Gherkin-сценария (`Завершение регистрации`, `Диск переполнен при завершении регистрации`). На выходе: пакет проектной документации в `docs/design/passkey-demo/`, готовый к подхвату sonnet'ом.

## Контекст и ограничения

- S1 уже реализован (PR #17). Слайс 2 читает регистрационную сессию, созданную S1, и удаляет её после успеха. Это требует аддитивного расширения S1: рехидраторы `RegistrationSessionFromRow`/`ChallengeFromBytes`/`RegistrationIDFromString` и поле `RPConfig.Origin`.
- Скилл `program-design.skill` обновлён в этом же цикле (PR #18) подправилом «подтип, не guard» — сразу применил его в проектировании S2.
- Принципы памяти: «без моков в тестах» (`feedback_no_mocks`), «I/O — трубы, без юнитов» (`feedback_io_no_units`), «I/O модули — один аргумент-структура» (`feedback_io_one_arg`), «головной модуль без юнит-тестов» (отменён в текущей версии скилла — `ProcessRegistrationFinish` тестируется юнитами с реальной in-memory БД).

## Принятое решение

**Пайп головного модуля — 9 шагов.** В диапазоне 5–10, который требует скилл. Применение подправила «подтип, не guard»: проверка свежести регистрационной сессии оформлена как конструктор `NewFreshRegistrationSession(input) -> (FreshRegistrationSession, error)`, не как guard `checkSessionFresh(...) -> ()`. Дальнейшие шаги (`verifyAttestation`, `NewUser`) принимают `FreshRegistrationSession` — система типов гарантирует, что верификация attestation не может произойти из просроченной сессии.

**Технологический стек:**
- `github.com/go-webauthn/webauthn` (подпакет `protocol`) — серверная верификация attestation. Используются примитивы `ParseCredentialCreationResponseBody` + `Verify`, без полного «engine» библиотеки.
- `github.com/golang-jwt/jwt/v5` — выдача JWT Ed25519. Ключ генерируется при старте процесса, не персистится.
- `github.com/descope/virtualwebauthn` в основном `go.mod` как **test-dep** — для honest юнит-теста `verifyAttestation` (генерация валидных attestation в `*_test.go`).

**Refresh token** — 32 байта `crypto/rand` → base64url. В БД хранится `hex(sha256(plaintext))`. Plaintext отдаётся клиенту. Валидация в S5 — по hash.

**Финальная I/O — одна транзакция.** `finishRegistration` делает 4 операции (`INSERT users`, `INSERT credentials`, `INSERT refresh_tokens`, `DELETE registration_sessions`) одной tx. Атомарность критична: иначе при, например, успехе INSERT users + провале INSERT credentials получим «осиротевшего» пользователя без credential.

**Маппинг ошибок:**
- `ErrSessionNotFound` и `ErrSessionExpired` — оба в 404. Для клиента поведение идентично («начни фазу 1 заново»); различение остаётся только в логах.
- `ErrAttestationInvalid` → 422 `ATTESTATION_INVALID` (не 401 — это provided-by-user данные).
- `ErrHandleTaken` (UNIQUE race) → 422 `HANDLE_TAKEN`.

**Миграции** — три отдельных файла (`0002_users.sql`, `0003_credentials.sql`, `0004_refresh_tokens.sql`), один на таблицу. Лучше читается история и легче откатывать частично.

## Отклонённые варианты

- **`checkSessionFresh(session, now) -> ()` как отдельный шаг пайпа.** Нарушает подправило «подтип, не guard»: инвариант не закреплён в типе, после проверки сырой `RegistrationSession` живёт в пайпе как ни в чём не бывало. Заменён конструктором подтипа.
- **Свернуть `NewUser` и `NewCredential` в одну сущность `AuthenticatedIdentity`.** Это два отдельных доменных понятия (`users` и `credentials` — отдельные таблицы, у credential может быть много на пользователя в будущих слайсах). Слипание вредит расширяемости.
- **Pre-check `findUserByHandle(handle)` перед `finishRegistration`** для избежания UNIQUE-провала. Лишний раунд-трип, не закрывает race (всегда останется окно между check и insert). UNIQUE-констрейнт — единственная честная защита.
- **Ed25519 keypair, персистимый в файл.** По `CLAUDE.md` решение: ключ генерируется при старте, не персистится. Перезапуск процесса инвалидирует все access-токены — сознательный выбор demo-сервиса.
- **Свернуть `loadRegistrationSession` и `NewFreshRegistrationSession` в один I/O `loadFreshSession(id, now)`.** Это запихивает бизнес-логику (проверка инварианта) в I/O-модуль, нарушая Шаг 6 («I/O — только эффект, без бизнес-логики»). Раздельно чище.
- **Юнит-тест `verifyAttestation` через mock `protocol.Verify`.** Нарушает `feedback_no_mocks`. Заменено на honest-тест с `virtualwebauthn` (генерирует валидные attestation, побитие подписи в failure-ветке).

## Ключевые промпты

- «приступай к проектированию 2 эндпоинта» — после чего opus проработал Шаг 0 скилла (читать OpenAPI, README, Gherkin, S1-реализацию).
- «checkSessionFresh замени на создание фреш сессии и если сессия не фреш ошибка в зависимости от логики» — родил подправило «подтип, не guard»; затем «доработай скилл согласно этой корректировки, чтобы самостоятельно принимать решения» — закрепил правило в `program-design.skill` (PR #18).

## Времязатраты

- **API duration:** 24m 44s (только активное время модели; ожидание ответов оператора не входит).
- **Стоимость:** $7.91.
- **Скоуп:** Шаг 0 → Шаг 12 скилла `program-design` для одного слайса + аддитивный апдейт скилла подправилом «подтип, не guard» (отдельный PR #18, его время в этой цифре учтено).
- **Распределение по фазам (на глаз):** ≈30% чтение входа (OpenAPI, Gherkin, S1-реализация, инфраструктура), ≈30% проектирование (дерево модулей, пайп, контракты, граф), ≈30% написание документов, ≈10% обновление скилла + housekeeping PR.

## Результат

Файлы пакета (все на ветке `feat/design-registrations-finish`):

- `docs/design/passkey-demo/slices/02-registrations-finish.md` — карточка слайса (новый файл)
- `docs/design/passkey-demo/messages.md` — секция «Структуры слайса 2», аддитивные расширения S1
- `docs/design/passkey-demo/contracts-graph.md` — секция «Slice 02», граф из 14 стрелок, чек-лист 9.3
- `docs/design/passkey-demo/infrastructure.md` — новые env, генерация Ed25519, миграции 0002-0004, Deps слайса 2
- `docs/design/passkey-demo/slices.md` — статус S1 → реализован, S2 → спроектирован
- `docs/design/passkey-demo/backlog.md` — тикет S2 с DoD, обновлённый хендофф-чеклист

PR: TBD (создаётся в финале сессии).

После аппрува пакета оператором — sonnet берёт тикет S2 на скилле `program-implementation.skill`.
