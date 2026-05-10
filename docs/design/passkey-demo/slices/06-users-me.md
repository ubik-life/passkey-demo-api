# Slice 06 — `users-me`

## Идентификатор входа

`HTTP GET /v1/users/me`

## Что делает (в одну фразу)

Принимает access JWT в заголовке `Authorization: Bearer <jwt>`, верифицирует его (подпись, срок, issuer), извлекает `UserID` из claim `Subject`, читает строку `users` по этому id, возвращает `{id, handle}` JSON-ом со статусом 200.

## OpenAPI

`api-specification/openapi.yaml`, `paths./users/me.get`. Контракт:

- Header: `Authorization: Bearer <jwt>` (`security: BearerAuth`).
- Тело запроса: пусто.
- Ответ 200: `{ "id": "<uuid>", "handle": "<string>" }` (схема `User`).
- Возможные ошибки: 401 `UNAUTHORIZED` (нет/невалидный/истёкший токен), 503 `db_locked` (+ `Retry-After: 1`).
- 507 `db_disk_full` **не декларирован** — эндпоинт read-only, SQLite на read не возвращает `SQLITE_FULL`. 500 `INTERNAL_ERROR` явно не декларирован, но используется как маппинг для аномалии согласованности (см. ниже секцию о `ErrUserNotFound`).

## Gherkin-сценарии слайса

`component-tests/features/users.feature`:

- `Сценарий: Возвращает данные пользователя из токена` — happy path. **Then-шаги** этого слайса:
  1. `Тогда ответ 200`,
  2. `И ответ содержит непустое JSON-поле id`,
  3. `И ответ содержит JSON-поле handle со значением "alice"`.
  When-шаги используют слайсы 1–4 (полный цикл регистрации + входа), которые к моменту S6 уже зелёные.

> **Сознательное расхождение со Шагом 0 скилла.** Скилл требует на вход к проектированию happy path **+ сценарий на каждый различимый режим отказа** (401, 503). На S6 в `users.feature` сейчас только happy path — то же решение, что для S5: оператор сознательно выбрал вариант «отказы — отдельной задачей позже». Дизайн при этом всё равно фиксирует маппинг `ErrAccessTokenInvalid → 401`, `ErrDBLocked → 503 + Retry-After + db_locked`, `ErrUserNotFound → 500 INTERNAL_ERROR` — это часть декларированного OpenAPI-контракта. После того, как сценарии будут дописаны, обратная сверка (Шаг 8.4) пройдёт без переделки дизайна.

## Зависимости от слайсов 1–5

- **Импорт типов:**
  - из S1: `ErrDBLocked`;
  - из S2: `User`, `UserID`, `UserFromRow` (S3 экспорт), `UserIDFromString` (S3 экспорт), `VerifyAccessToken` (S5 экспорт), `AuthenticatedUserID` (S5 экспорт), `VerifyAccessTokenInput` (S5 экспорт), `ErrAccessTokenInvalid` (S5 экспорт), `JWTConfig`.
- **Аддитивных расширений S1/S2/S3/S4/S5 не требуется.** S6 — первый слайс, который собирается из существующих публичных API без новых экспортов в других слайсах. Это — следствие того, что S5 уже экспортировал всё необходимое для Bearer-auth, а S3 — рехидраторы пользовательских строк.
- **Чтение БД:** один SELECT — `SELECT id, handle, created_at FROM users WHERE id = ?`.
- **Запись БД:** —.

S6 не зависит от слайсов 3–5 структурно (login_sessions / refresh_tokens не трогаются). Зависит от S2 операционно: строка `users`, которую S6 читает, появляется в БД через `Store.FinishRegistration` (S2). Зависит от S5 концептуально: тот же шаблон Bearer-auth → 401, та же функция `VerifyAccessToken`, тот же подтип `AuthenticatedUserID`.

## Дерево модулей

```
ингресс-адаптер: HTTP handler GET /v1/users/me
    ├── parse Authorization header → UsersMeRequest
    └── (после головного модуля) format Response → 200 + UserProfileResponse JSON,
        либо error → 401 / 500 / 503 + Retry-After
        │
головной модуль слайса: ProcessUsersMe
    ├── (1) NewUsersMeCommand(req)            → UsersMeCommand          [конструктор; проверка непустоты Bearer]
    ├── (2) VerifyAccessToken(input)          → AuthenticatedUserID     [импорт S2; deps: PublicKey, JWTConfig.Issuer, clock]
    ├── (3) Store.LoadUser(input)             → User                    [I/O-метод — SQLite read]
    └── (4) buildResponse(user)               → UserProfileResponse     [чистая функция; маппинг в DTO]
```

Каждый узел — **один data-аргумент** (Шаг 3 скилла). Зависимости (`Store`, `PublicKey`, `JWTConfig`, `clock`) вынесены через `Deps` и не считаются стрелками графа.

**Автономный I/O-объект `Store` (Шаг 6 скилла).** Узел (3) — метод объекта `Store` слайса 6, инкапсулирующего `*sql.DB`. Головной модуль `ProcessUsersMe` знает только API объекта (метод `LoadUser`), но не его внутреннюю зависимость. В `Deps` слайса — поле типа `*Store`, **не** сырой `*sql.DB`. См. `messages.md` → секция «I/O-объект слайса 6» и `infrastructure.md` → «Подключение слайса 6 (S6)».

Применение **подправила «подтип, не guard»** (Шаг 3 скилла): инвариант «JWT успешно верифицирован этим процессом» унаследован из S5 — узел (2) возвращает `AuthenticatedUserID`. Шаг (3) `Store.LoadUser` принимает `LoadUserInput`, в котором `UserID` распакован из `AuthenticatedUserID.UserID()` — компилятор не даст обойти верификацию (нет другого способа получить `AuthenticatedUserID`, кроме как через `VerifyAccessToken`).

Никаких новых подтипов S6 не вводит — `User` уже доменная сущность с инвариантом «handle прошёл `NewHandle`», конструктор `NewUser` гарантирует это для свежесозданных, рехидратор `UserFromRow` (S3 экспорт) — для прочитанных из БД.

## Решения по дизайну

### Длина пайпа: 4 узла — сабфлор скилла

Скилл рекомендует **5–10 шагов** в пайпе головного модуля (`program-design.skill` Шаг 3). У S6 — 4 узла + 2 шага адаптера (parse header / format response) = 6 видимых шагов end-to-end, но **4 узла графа** в пайпе.

**Это сознательное расхождение** — то же решение, что в S5 (3 узла). Read-by-id — операционно простая операция: «верифицировать токен → загрузить user → сериализовать». Натянуть padding-узлы было бы нечестно. Альтернативные варианты, которые мы рассмотрели и **не выбрали**:

1. *Слить* `parseAccessToken` и `NewAuthenticatedUserID` в один узел — **выбрано** (`VerifyAccessToken` S2-экспорта делает оба этапа, как и в S5).
2. *Раздробить* `Store.LoadUser` на «check exists» + «load fields» — это два SELECT'а вместо одного, противоречит принципу «I/O — труба, минимальный эффект».
3. *Вынести `buildResponse` в адаптер* — формально допустимо (адаптер уже «format response»), но `buildResponse` — это маппинг доменного `User` → DTO `UserProfileResponse`, чистая функция логики. Держать её узлом графа делает явным, какие именно поля попадают в ответ (важно для будущих изменений — добавить `created_at`, `email` и т.д.).
4. *Считать рантайм-метод `AuthenticatedUserID.UserID()` отдельным узлом* — это method getter, не модуль (та же логика, что в S5).

Зафиксировано на Шаге 12 (хендофф) как явное замечание оператору: 4-узловой пайп — норма для read-by-id-семантики.

### Стандартный порядок: verify → load (не «load → verify»)

Развилка зафиксирована в `devlog/09-design-S6-users-me.md` (промпт оператора «сначала найди пользователя по id потом валидировать токен» был уточнён до промпта «придерживайся первоначального плана»).

Вариант «extract Subject без verify → `Store.LoadUser` → потом проверить подпись» **отклонён**. Минусы:

- (а) **leak existence через timing.** Без проверки подписи злоумышленник дёргает БД на любых правильно структурированных JWT с произвольным UUID в `Subject` — каждый запрос триггерит SELECT, по времени ответа можно отличить «user существует» от «user не существует» даже без получения 200.
- (б) **ломается подправило «подтип, не guard».** После `Store.LoadUser` `User` уже на руках, но JWT ещё не верифицирован. Чтобы система типов запретила вернуть его клиенту до verify, пришлось бы вводить промежуточный подтип «User до проверки подписи» — лишний тип ради переставленного порядка.
- (в) **расхождение с S5.** В S5 `VerifyAccessToken` — атомарный шаг, после него `AuthenticatedUserID` готов; в S6 — те же примитивы, симметрия по флоу облегчает сопровождение и ревью.

В текущей реализации сервер-ключ Ed25519 **глобальный** и не привязан к user, поэтому «загрузить user, чтобы взять его ключ» (мотивация для инверсии порядка в системах с per-user keypair) не применимо.

### Маппинг `ErrUserNotFound` → 500, не 401 и не 404

После успешной верификации JWT `AuthenticatedUserID` гарантирует «токен корректен, выдан этим сервером, не истёк». Если в этот момент `Store.LoadUser` возвращает `ErrUserNotFound` — это **аномалия согласованности**: токен был выдан для пользователя, которого сейчас нет в БД. На текущей кодовой базе (S2–S5) случай **невозможен** — нет операции удаления `users` (и нет каскадов, которые бы эту строку убрали). Это будущий case (например, гипотетический admin-эндпоинт `DELETE /users/{id}`).

**Выбрано: маппинг → 500 `INTERNAL_ERROR` + лог `slog.Error("auth/db inconsistency", "user_id", uid)`.**

**Не выбрано (обсуждено и отклонено):**

- (B) *Маппить в 401 `UNAUTHORIZED`* — клиенту удобнее (логинится заново и продолжает), но это **скрывает консистентность-баг** под обычной экспирацией. Оператор не получит сигнал в логах оперативного мониторинга 5xx, а только 401-spike, который выглядит как «забыл залогинеть пачку клиентов».
- (C) *Маппить в 404 `NOT_FOUND`* — OpenAPI не декларирует 404 на `/v1/users/me` (только 200/401/503). Введение нового кода ответа без правки OpenAPI — нарушение API-First (`AGENTS.md §2`).

500 — самый честный сигнал «сервер не справился с собственным состоянием»; 503 не подходит (это не временная блокировка БД, а согласованность данных).

Промпт оператора «оставляй допустимой аномалию ошибки» (`devlog/09-design-S6-users-me.md`) подтверждает: case явно допустим в дизайне, не игнорируется и не маскируется.

### Без новых миграций

Таблица `users` уже существует с S2 (`internal/db/migrations/0002_users.sql`). S6 делает один SELECT по PK, никаких ALTER TABLE / новых индексов / новых файлов в `internal/db/migrations/` для S6 не создавать.

Дополнительный индекс — **не нужен.** PK на `id` (TEXT PRIMARY KEY) уже даёт O(log n) lookup; индекс по `handle` (S2 `0002_users.sql`) на этом эндпоинте не используется.

### `ErrMissingBearer` — локальный, не импорт из S5

S5 ввёл `ErrMissingBearer = errors.New("authorization: missing bearer token")` в пакете `sessions_logout` для конструктора команды на пустой Bearer. S6 нуждается в такой же sentinel-ошибке.

**Выбрано: ввести свой `ErrMissingBearer` в пакете `users_me`** (одна строка `var ErrMissingBearer = errors.New("...")`). Альтернативы:

1. *Импортировать `sessions_logout.ErrMissingBearer`* — структурно вводит зависимость S6 → S5, которой архитектурно нет (оба слайса — листья, расширяющие S2). Циклов не создаёт, но усложняет картину зависимостей.
2. *Вынести в S2 как часть Bearer-auth* — потребует ретроспективной правки S5 (перенос константы в `registrations_finish`). Аддитивно, не break. Чище в долгой перспективе, но цена сейчас — расширение scope handoff'а S6 на S5.

Дублирование одной sentinel-строки в двух пакетах — приемлемая цена локальности. Если в проекте появится третий «authenticated» эндпоинт, имеет смысл вынести в общий `internal/auth/` модуль и сделать аддитивный рефакторинг; на двух — преждевременно.

### Без read-time проверки `revoked_at` или экспирации refresh

S6 верифицирует **access** JWT (подпись + `exp` claim). Refresh-токены S6 не использует — поэтому проверка `revoked_at IS NULL` или `expires_at > now` для `refresh_tokens` не входит в pipeline S6.

Это согласуется с архитектурным выбором проекта: access-токены короткоживущие (TTL 15m по дефолту, см. `JWTConfig.AccessTTL`), validation полностью на стороне ключа подписи, БД-чек не нужен. После logout (S5) старые refresh-токены отозваны, но access-токены, выданные до logout, остаются валидны до своего `exp` — это известный trade-off короткого `AccessTTL` и отсутствия revocation list.

## Псевдокод пайпа головного модуля

```
ProcessUsersMe(req: UsersMeRequest, deps: Deps)
    -> (UserProfileResponse, error):

    | NewUsersMeCommand(req)                                       -> UsersMeCommand
    | verifyInput := VerifyAccessTokenInput {
                       AccessTokenRaw: cmd.AccessTokenRaw(),
                       PublicKey:      deps.Verifier,
                       ExpectedIssuer: deps.JWT.Issuer,
                       Now:            deps.clock.Now() }
    | VerifyAccessToken(verifyInput)                                -> AuthenticatedUserID
    | loadInput  := LoadUserInput { UserID: authID.UserID() }
    | deps.Store.LoadUser(loadInput)                                -> User
    | buildResponse(user)                                            -> UserProfileResponse
```

Ошибки протекают через ранний `return UserProfileResponse{}, fmt.Errorf("step: %w", err)`. Сборка `verifyInput`, `loadInput` — Go-литералы структур, не отдельные узлы графа.

**Узлов графа — 4** (`NewUsersMeCommand`, `VerifyAccessToken`, `Store.LoadUser`, `buildResponse`). См. секцию «Решения по дизайну → Длина пайпа» о сабфлоре 5–10.

## Контракты модулей

### `NewUsersMeCommand`

- **Сигнатура:** `NewUsersMeCommand(req UsersMeRequest) -> (UsersMeCommand, error)`
- **Input (data):** `req UsersMeRequest`
- **Dependencies (deps):** —
- **Что делает:** проверяет, что адаптер положил в DTO непустой Bearer-токен; собирает доменную команду.
- **Антецедент:** `req.AccessTokenRaw` — строка, переданная адаптером после извлечения из заголовка `Authorization`.
- **Консеквент:**
  - Success: `UsersMeCommand { accessTokenRaw: req.AccessTokenRaw }` при `req.AccessTokenRaw != ""`.
  - Failure: `ErrMissingBearer` при `req.AccessTokenRaw == ""`.

### `VerifyAccessToken` (импорт S2)

Полный контракт — в карточке S5 (`docs/design/passkey-demo/slices/05-sessions-logout.md` секция «Контракты модулей → `VerifyAccessToken` (импорт S2)») и в `messages.md` секция «Аддитивные расширения слайса 2 для слайса 5». В S6 функция используется без изменений.

Сводка для S6:

- **Сигнатура:** `VerifyAccessToken(input VerifyAccessTokenInput) -> (AuthenticatedUserID, error)`
- **Input (data):** `input VerifyAccessTokenInput { AccessTokenRaw, PublicKey, ExpectedIssuer, Now }`
- **Dependencies (deps):** —
- **Failure:** `ErrAccessTokenInvalid` (общий класс: malformed compact serialization, signature mismatch, expired, issuer mismatch, Subject не UUID, неверный алгоритм подписи).

Юнит-тесты `VerifyAccessToken` посчитаны и реализованы в S5 (6 юнитов в пакете `internal/slice/registrations_finish/`). В S6 повторно не считаются.

### `Store.LoadUser` (метод I/O-объекта)

- **Сигнатура:** `(s *Store) LoadUser(input LoadUserInput) -> (User, error)`
- **Input (data):** `input LoadUserInput { UserID }`
- **Dependencies (deps):** — (зависимость `*sql.DB` инкапсулирована внутри `Store`; головной модуль её не видит)
- **Что делает:** один SELECT:
  ```
  SELECT id, handle, created_at
    FROM users
   WHERE id = ?
  ```
  Рехидрирует строку в `User` через `UserFromRow` (S3 экспорт из S2).
- **Антецедент:** `input.UserID` валиден (приходит из `AuthenticatedUserID.UserID()`); миграция `0002_users.sql` применена.
- **Консеквент:**
  - Success: `User` соответствует строке БД (`User.ID() == input.UserID`, `User.Handle()` валиден через `NewHandle`).
  - Failure:
    - `ErrUserNotFound` — `sql.ErrNoRows` (строки нет; аномалия согласованности после успешной verify, см. секцию «Решения по дизайну»).
    - `ErrDBLocked` — `SQLITE_BUSY` → 503 `db_locked` + `Retry-After: 1`.
    - другие — обёрнуты в общую внутреннюю ошибку (→ 500 `INTERNAL_ERROR`).
- **`ErrDiskFull` не различается** для read-операций SQLite (SELECT не возвращает `SQLITE_FULL`).

### `buildResponse`

- **Сигнатура:** `buildResponse(user User) -> UserProfileResponse`
- **Input (data):** `user User`
- **Dependencies (deps):** —
- **Что делает:** маппит доменную сущность `User` в DTO ответа OpenAPI.
- **Антецедент:** `user` собрана конструктором / рехидратором (поля валидны, `Handle()` непустой, `ID()` — UUID v4).
- **Консеквент:**
  - Success: `UserProfileResponse { ID: user.ID().String(), Handle: user.Handle().Value() }`.
  - Failure: — (чистая функция без условий ошибки; антецедент гарантирован предыдущим узлом).

### Ингресс-адаптер: HTTP handler `GET /v1/users/me`

- **Что делает:**
  1. Читает заголовок `Authorization`. Если пустой или не начинается с префикса `"Bearer "` (case-sensitive, RFC 6750 §2.1) — пишет `401 UNAUTHORIZED` напрямую (без вызова головного модуля), `error.code: UNAUTHORIZED`, `message: "missing or malformed Authorization header"`.
  2. Извлекает значение после `"Bearer "` в `accessTokenRaw`.
  3. Собирает `UsersMeRequest{ AccessTokenRaw: accessTokenRaw }`.
  4. Вызывает `ProcessUsersMe(req, deps)`.
  5. На Success: пишет `200 OK`, `Content-Type: application/json`, тело — JSON-сериализация `UserProfileResponse`.
  6. На Failure: маппит ошибки в HTTP-ответ (см. таблицу маппинга ниже).
- **Никакой бизнес-логики** — только парсинг заголовка, JSON-сериализация результата и маппинг ошибок.
- **Юнит-тестами не покрывается** — проверяется компонентным тестом слайса через реальный HTTP-вход.

**Граница «адаптер vs `NewUsersMeCommand`».** Та же, что в S5: адаптер ловит **синтаксические** случаи (заголовок отсутствует или не начинается с `"Bearer "`); `NewUsersMeCommand` ловит **доменный** случай (значение после префикса пустое). Оба → 401 с одинаковым телом. Разделение — ради того, чтобы команда имела непустой контракт.

### Маппинг ошибок в ингресс-адаптере

| Класс ошибки                                                  | HTTP-статус | Заголовки           | Тело (`error.code`)         |
|---------------------------------------------------------------|-------------|---------------------|-----------------------------|
| (адаптер: нет/malformed `Authorization`)                       | 401          | —                   | `UNAUTHORIZED`               |
| `ErrMissingBearer`                                             | 401          | —                   | `UNAUTHORIZED`               |
| `ErrAccessTokenInvalid`                                        | 401          | —                   | `UNAUTHORIZED`               |
| `ErrUserNotFound`                                              | 500          | —                   | `INTERNAL_ERROR`             |
| `ErrDBLocked`                                                  | 503          | `Retry-After: 1`    | `db_locked`                  |
| Любая другая (неожиданные SQLite, panic-recover)              | 500          | —                   | `INTERNAL_ERROR`             |

`ErrMissingBearer` и `ErrAccessTokenInvalid` оба маппятся в 401: для клиента поведение идентично («залогинься заново»). Различение остаётся только в логах адаптера.

`ErrUserNotFound` — особый случай: маппится в 500, не в 401/404 (см. секцию «Решения по дизайму → Маппинг `ErrUserNotFound` → 500»). Адаптер логирует через `slog.Error`, не `slog.Warn`, чтобы попасть в стандартные 5xx-алерты оперативного мониторинга.

`ErrDiskFull` не маппится — на эндпоинте read-only, эта ошибка теоретически не возникает (SELECT не растит файл и не пишет WAL).

## Gherkin-mapping

| Сценарий                                                | Then-шаг                                              | Кто обеспечивает (узел графа / маппинг адаптера)                                                            |
|---------------------------------------------------------|-------------------------------------------------------|-------------------------------------------------------------------------------------------------------------|
| Возвращает данные пользователя из токена                 | `Тогда ответ 200`                                     | Узлы (1)–(4) Success-путь → ингресс-адаптер: `200 OK`, `Content-Type: application/json`                     |
| Возвращает данные пользователя из токена                 | `И ответ содержит непустое JSON-поле id`              | Узел (4) `buildResponse`: `UserProfileResponse.ID = user.ID().String()` (UUID непуст по конструктору)        |
| Возвращает данные пользователя из токена                 | `И ответ содержит JSON-поле handle со значением "alice"` | Узел (4) `buildResponse`: `UserProfileResponse.Handle = user.Handle().Value()` (строка, прочитанная из БД)  |

### Чек-лист сверки 8.5

1. [x] **Узел существует.** Узлы (1)–(4) описаны в дереве и в контрактах выше; ингресс-адаптер описан с маппингом.
2. [x] **Ветка соответствует.** Все Then'ы — Success-путь.
3. [x] **Формат ответа адаптера согласован.** OpenAPI декларирует `200 + User { id, handle }` — адаптер пишет JSON-тело `UserProfileResponse` с полями `id`/`handle`. Маппинги `ErrAccessTokenInvalid → 401 + UNAUTHORIZED`, `ErrDBLocked → 503 + Retry-After + db_locked`, `ErrUserNotFound → 500 + INTERNAL_ERROR` соответствуют README «Карта режимов отказа» и `responses` в OpenAPI (`Unauthorized`, `ServiceUnavailable`).
4. [x] **Все Then покрыты.** В сценарии «Возвращает данные пользователя из токена» 3 Then-шага, все покрыты.

`[x] Gherkin-mapping сверен.`

### Замечание о других режимах отказа

OpenAPI декларирует на этом эндпоинте также 401 `UNAUTHORIZED` и 503 `db_locked`. 500 `INTERNAL_ERROR` явно не декларирован (там только 200/401/503), но используется как маппинг для аномалии согласованности `ErrUserNotFound` (см. секцию «Решения по дизайну»). Gherkin-сценариев на 401 / 500 / 503 **нет** в `users.feature` (по сознательному решению оператора, как в S5).

Адаптер обязан корректно маппить все четыре класса в декларированные / выбранные коды — это часть OpenAPI-контракта (плюс выбор по `ErrUserNotFound`); проверка адаптерным маппингом тестируется опосредованно через прохождение happy path (если маппинг сломается, тест happy-path упадёт на ошибке внутреннего вызова).

`ErrMissingBearer`, `ErrAccessTokenInvalid`, `ErrUserNotFound`, `ErrDBLocked` — все имеют декларированные классы маппинга в карточке выше; их юнит-проверка покрывается формулой ниже (только для модулей логики; маппинг адаптера юнитами не покрывается).

## Юнит-тесты по формуле

`N_юнит_тестов = 1 (happy path) + Σ (ветки антецедента)` — **только модули логики и конструкторы** (Шаг 8.1: «I/O — трубы, юнитами не покрываются»; ингресс-адаптер — тоже).

| Модуль                            | Happy | Ветки антецедента                               | Итого |
|-----------------------------------|-------|--------------------------------------------------|-------|
| `NewUsersMeCommand`               | 1     | пустая `AccessTokenRaw` → `ErrMissingBearer`     | 2     |
| `buildResponse`                   | 1     | — (чистая функция без ветвлений по входу)        | 1     |
| **Итого**                         |       |                                                  | **3** |

Что **не** в таблице (и почему):

- **`VerifyAccessToken`** (импорт S2) — юниты уже посчитаны и реализованы в карточке S5 (6 тестов в пакете `internal/slice/registrations_finish/`). S6 переиспользует функцию, формулу не пересчитывает (правило «у того, кто вводит модуль»).
- **`Store.LoadUser`** — метод I/O-объекта, труба. Юнитов нет. Success-путь проверяется компонентным сценарием **«Возвращает данные пользователя из токена»** (если SELECT не дойдёт или отдаст не ту строку — happy-path упадёт на проверке `handle` в Then). Failure-ветки `ErrUserNotFound`/`ErrDBLocked` — без отдельного компонентного сценария на этом эндпоинте (по решению оператора, как в S5).
- **Ингресс-адаптер** — парсинг заголовка, JSON-сериализация и маппинг ошибок, юнитов нет.
- **Головной модуль** `ProcessUsersMe` — оркестратор-труба: линейный пайп из 4 узлов, ошибки I/O пробрасываются без трансформации. Юнит-тест над ним был бы интеграционным тестом. Корректность пайпа доказывается компонентным сценарием.

Замечания по покрытию:

- 100% строк/веток модулей логики достигается этими 3 юнит-тестами.
- Юнит `buildResponse` тривиален (одна строка маппинга); проверяет, что DTO собран корректно при валидном `User`. Без него `buildResponse` остался бы непокрытым по строкам — формально нарушение метрики, хотя содержательно функция простейшая.
- Юниты `NewUsersMeCommand` симметричны S5 `NewSessionLogoutCommand` — те же два теста (happy + empty Bearer).

## Definition of Done слайса

Скопировано в тикет S6 в `backlog.md`:

- [ ] **аддитивные расширения других слайсов не требуются** — все нужные публичные API (`VerifyAccessToken`, `AuthenticatedUserID`, `VerifyAccessTokenInput`, `ErrAccessTokenInvalid`, `User`, `UserID`, `UserFromRow`, `UserIDFromString`, `JWTConfig`) уже экспортированы в S2/S3/S5. Юнит-тесты S1–S5 остаются зелёными без изменений.
- [ ] ингресс-адаптер реализован: парсит заголовок `Authorization: Bearer <jwt>` в `UsersMeRequest{AccessTokenRaw}`, без бизнес-валидации (HTTP handler в `internal/slice/users_me/`); пустой/malformed заголовок (нет префикса `"Bearer "`) → 401 `UNAUTHORIZED` напрямую без вызова головного модуля; на Success — `200 OK` + JSON `UserProfileResponse`.
- [ ] конструктор доменной структуры `NewUsersMeCommand` реализован; пустая `AccessTokenRaw` → `ErrMissingBearer`.
- [ ] модуль логики `buildResponse` реализован: маппит `User` в `UserProfileResponse` (поля `ID`, `Handle`).
- [ ] **I/O-объект `Store` реализован** как автономный объект, инкапсулирующий `*sql.DB`: тип `*Store` в пакете `internal/slice/users_me/`, конструктор `NewStore(db *sql.DB) *Store`, один метод:
  - `(s *Store) LoadUser(input LoadUserInput) (User, error)`: один `SELECT id, handle, created_at FROM users WHERE id = ?`; маппинг `sql.ErrNoRows` → `ErrUserNotFound`, `SQLITE_BUSY` → `ErrDBLocked`. Рехидрация через `UserFromRow` (S3 экспорт).
  - Голова `ProcessUsersMe` обращается к БД **только через этот метод**; `*sql.DB` нигде кроме `Store` не светится в slice-пакете.
- [ ] головной модуль `ProcessUsersMe` реализован: пайп из 4 узлов (см. псевдокод выше), ранний возврат при ошибке через `fmt.Errorf("…: %w", err)`. **4 узла — сабфлор скилла (5–10), осознанное расхождение, обоснование в карточке.**
- [ ] **новых миграций нет** — слайс использует существующую таблицу `users` (S2 миграция `0002`). Никаких ALTER TABLE / новых файлов в `internal/db/migrations/` для S6 не создавать.
- [ ] инфраструктурный модуль расширен: `Deps` слайса 6 (`Store *Store`, `Clock`, `Logger`, `Verifier ed25519.PublicKey`, `JWT JWTConfig` — **без** сырого `*sql.DB`); подключение `users_me.Register(mux, deps.UsersMe)` в `cmd/api/main.go`; в `wire.go` создаётся `users_me.NewStore(db)` и пробрасывается в `Deps.Store`. Поле `Verifier` — то же `signer.Public`, что у S5 (второе использование парного публичного ключа в проекте).
- [ ] слайс подключён через `users_me.Register(mux, deps)`: HTTP-роут `GET /v1/users/me` ведёт на ингресс-адаптер.
- [ ] **юнит-тесты по формуле написаны и зелёные** — `go test ./...` проходит. **3 новых теста** на модули логики и конструкторы: 2 на `NewUsersMeCommand` + 1 на `buildResponse`; головной модуль, I/O-модуль и ингресс-адаптер юнитами не покрываются. Юниты S1–S5 остаются зелёными (никаких аддитивных расширений S6 не вносит).
- [ ] **компонентные тесты, профиль `healthy`, зелёные** — `./component-tests/scripts/run-tests.sh healthy` проходит. Новый зелёный сценарий: `Сценарий: Возвращает данные пользователя из токена` (`component-tests/features/users.feature`). Ранее зелёные сценарии S1–S5 продолжают проходить.
- [ ] **компонентные тесты, профиль `disk-full`, зелёные** — техдолг: профиль регрессировал ещё в S4 и продолжает регрессировать в S5 (см. карточку S5 → DoD). Перенесён в техдолг с явным разрешением оператора.
- [ ] `backlog.md` обновлён по каждому подтверждённому пункту (правило `AGENTS.md §10`).
- [ ] `docs/design/passkey-demo/devlog.md` дополнен блоком S6.
- [ ] PR создан, описание заполнено по шаблону Шага 8 скилла sonnet'а.
- [ ] PR смержен в main, CI на main зелёный.

## Ссылки на источники

- Скилл реализации: `skills/program-implementation/SKILL.md`
- Граф вызовов: `docs/design/passkey-demo/contracts-graph.md` Slice 06
- Gherkin-mapping: раздел `## Gherkin-mapping` выше
- Импорт `VerifyAccessToken/AuthenticatedUserID/VerifyAccessTokenInput/ErrAccessTokenInvalid`: `docs/design/passkey-demo/messages.md` секция «Аддитивные расширения слайса 2 для слайса 5»
- Импорт `UserFromRow/UserIDFromString/User/UserID/JWTConfig`: `docs/design/passkey-demo/messages.md` секция «Структуры слайса 2» + «Аддитивные расширения слайса 2 для слайса 3»
- Подключение слайса 6: `docs/design/passkey-demo/infrastructure.md` → «Подключение слайса 6 (S6)»
- Подправило «подтип, не guard»: `skills/program-design/SKILL.md` Шаг 3 (унаследовано из S5: `AuthenticatedUserID` несёт инвариант «JWT успешно верифицирован» в типе)
- Devlog проектирования: `devlog/09-design-S6-users-me.md`
