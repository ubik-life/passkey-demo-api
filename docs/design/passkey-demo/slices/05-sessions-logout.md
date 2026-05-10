# Slice 05 — `sessions-logout`

## Идентификатор входа

`HTTP DELETE /v1/sessions/current`

## Что делает (в одну фразу)

Принимает access JWT в заголовке `Authorization: Bearer <jwt>`, верифицирует его (подпись, срок, issuer), отзывает все активные refresh-токены аутентифицированного пользователя (UPDATE `refresh_tokens.revoked_at`), возвращает 204.

## OpenAPI

`api-specification/openapi.yaml`, `paths./sessions/current.delete`. Контракт:

- Header: `Authorization: Bearer <jwt>` (`security: BearerAuth`).
- Тело запроса: пусто.
- Ответ 204: пусто.
- Возможные ошибки: 401 `UNAUTHORIZED` (нет/невалидный/истёкший токен), 503 `db_locked` (+ `Retry-After: 1`), 507 `db_disk_full`.

## Gherkin-сценарии слайса

`component-tests/features/sessions-current.feature`:

- `Сценарий: Выход инвалидирует refresh token` — happy path. **Then-шаги** этого слайса: ответ 204. When-шаги используют слайсы 1-4 (полный цикл регистрации + входа), которые к моменту S5 уже зелёные.

> **Сознательное расхождение со Шагом 0 скилла.** Скилл требует на вход к проектированию happy path **+ сценарий на каждый различимый режим отказа** (`db_locked`, `db_disk_full`). На S5 в `sessions-current.feature` сейчас только happy path: оператор сознательно выбрал вариант «отказы — отдельной задачей позже» (см. backlog хендофф S5 ниже, ветку «Замечания по Шагу 0»). Дизайн при этом всё равно фиксирует маппинг `ErrDBLocked → 503 + Retry-After + db_locked` и `ErrDiskFull → 507 + db_disk_full` — это часть декларированного OpenAPI-контракта. После того, как сценарии будут дописаны, обратная сверка (Шаг 8.4) пройдёт без переделки дизайна.

## Зависимости от слайсов 1-4

- **Импорт типов:**
  - из S1: `ErrDBLocked`, `ErrDiskFull`;
  - из S2: `UserID`.
- **Аддитивное расширение слайса 2** (см. `messages.md` → «Аддитивные расширения слайса 2 для слайса 5»):
  - экспортировать `VerifyAccessToken(input VerifyAccessTokenInput) (AuthenticatedUserID, error)` — публичная функция верификации access JWT;
  - экспортировать типы `VerifyAccessTokenInput`, `AuthenticatedUserID`;
  - экспортировать `ErrAccessTokenInvalid`.
- **Чтение БД:** —.
- **Запись БД:** одна операция: `UPDATE refresh_tokens SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`.

S5 не зависит от слайсов 3 и 4 структурно (логин-сессии S3 не трогаются — login_sessions DELETE при логауте не нужен; они либо уже удалены S4 при успешном входе, либо протухнут по `expires_at`). Но S5 зависит от S2/S4 операционно: refresh-токены, которые он отзывает, появляются в БД через `Store.FinishRegistration` (S2) и `Store.FinishLogin` (S4).

## Дерево модулей

```
ингресс-адаптер: HTTP handler DELETE /v1/sessions/current
    ├── parse Authorization header → SessionLogoutRequest
    └── (после головного модуля) format Response → 204 (без тела),
        либо error → 401 / 503 + Retry-After / 507 / 500
        │
головной модуль слайса: ProcessSessionLogout
    ├── (1) NewSessionLogoutCommand(req)             → SessionLogoutCommand     [конструктор; проверка непустоты Bearer]
    ├── (2) VerifyAccessToken(input)                  → AuthenticatedUserID      [импорт S2; deps: PublicKey, JWTConfig.Issuer, clock]
    └── (3) Store.RevokeUserSessions(input)           → error                    [I/O-метод — SQLite write]
```

Каждый узел — **один data-аргумент** (Шаг 3 скилла). Зависимости (`Store`, `PublicKey`, `JWTConfig`, `clock`) вынесены через `Deps` и не считаются стрелками графа.

**Автономный I/O-объект `Store` (Шаг 6 скилла).** Узел (3) — метод объекта `Store` слайса 5, инкапсулирующего `*sql.DB`. Головной модуль `ProcessSessionLogout` знает только API объекта (метод `RevokeUserSessions`), но не его внутреннюю зависимость. В `Deps` слайса — поле типа `*Store`, **не** сырой `*sql.DB`. См. `messages.md` → секция «I/O-объект слайса 5» и `infrastructure.md` → «Подключение слайса 5 (S5)».

Применение **подправила «подтип, не guard»** (Шаг 3 скилла): инвариант «JWT успешно верифицирован этим процессом» оформлен как конструктор подтипа `VerifyAccessToken` (узел 2), возвращающий `AuthenticatedUserID`. Шаг (3) `Store.RevokeUserSessions` принимает `RevokeUserSessionsInput`, в котором `UserID` распакован из `AuthenticatedUserID.UserID()` — компилятор не даст обойти верификацию (нет другого способа получить `AuthenticatedUserID`, кроме как через `VerifyAccessToken`).

## Решения по дизайну

### Длина пайпа: 3 узла — сабфлор скилла

Скилл рекомендует **5–10 шагов** в пайпе головного модуля (`program-design.skill` Шаг 3). У S5 — 3 узла + 2 шага адаптера (parse header / format response) = 5 видимых шагов end-to-end, но **3 узла графа** в пайпе.

**Это сознательное расхождение.** Логаут — операционно простая операция: «верифицировать токен → отозвать токены». Натянуть padding-узлы было бы нечестно: каждый дополнительный узел должен делать содержательную работу, а не существовать ради счётчика. Альтернативные варианты, которые мы рассмотрели и **не выбрали:**

1. *Слить* `parseAccessToken` и `NewAuthenticatedUserID` в один узел — **выбрано** (`VerifyAccessToken` S2-экспорта делает оба этапа). Дробление на два не давало нового инварианта в типе и удлиняло пайп ради счёта.
2. *Добавить узел `buildResponse`* — для 204 ответа без тела это литеральная заглушка `return nil`. Не добавляет узел графа.
3. *Добавить узел `parseAuthorizationHeader`* — это ответственность ингресс-адаптера (синтаксис заголовка), не пайпа. Адаптер уже делает.
4. *Считать рантайм-метод `AuthenticatedUserID.UserID()` отдельным узлом* — это method getter, не модуль.

Зафиксировано на Шаге 12 (хендофф) как явное замечание оператору: 3-узловой пайп — норма для logout-семантики, при пересмотре скилла можно понизить нижнюю границу для слайсов «отзыв-инвалидация» до 3.

### Семантика «logout-from-all», а не «текущая сессия»

OpenAPI говорит: «Инвалидирует refresh token текущей сессии». Однако access JWT в текущей реализации (`internal/slice/registrations_finish/logic.go`, S2) несёт только `RegisteredClaims` с `Subject = user_id` — связи «этот access ↔ конкретный refresh» в системе типов нет.

**Выбрано: `UPDATE refresh_tokens SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`** — отзывает все активные refresh-токены аутентифицированного пользователя.

**Не выбрано (зафиксировано в DoD):**

- (B) Связать access ↔ refresh через `jti`-claim: новое поле в JWT, новая колонка `refresh_tokens.access_jti`, миграция, возврат к S2 за изменением `generateTokenPair` и юнит-тестам. Точнее, но дороже на ~1.5 слайса работы.
- (C) Ввести таблицу `sessions` и `session_id` в JWT: ещё дороже.

Для демо-сервиса достаточно «logout-from-all»: типичный пользователь — один аккаунт, одно устройство, понятие «текущей сессии» = «этот пользователь сейчас». Вынести (B) в отдельный тикет, если понадобится multi-device-aware logout.

### Без новых миграций

`refresh_tokens.revoked_at` уже существует с S2 (`internal/db/migrations/0004_refresh_tokens.sql`, колонка `revoked_at INTEGER NULL`, см. `infrastructure.md`). Никаких ALTER TABLE / новых файлов в `internal/db/migrations/` для S5 не создавать.

Дополнительный индекс по `(user_id, revoked_at)` — **не нужен.** Cardinality `user_id` низкая, активных refresh на user — единицы (типично 1-2). Индекс по `user_id` уже есть (`idx_refresh_tokens_user_id`); фильтр `revoked_at IS NULL` обрабатывается scan'ом по строкам user'а (≤ 2 строки в типичном случае). Композитный индекс — нет выгоды.

### Один общий класс ошибки `ErrAccessTokenInvalid`

Все под-причины отказа верификации access-токена (parse fail, signature mismatch, expired, issuer mismatch, claim Subject не парсится) маппятся в один sentinel `ErrAccessTokenInvalid` → 401 `UNAUTHORIZED`. Различение нужно только в логах (`logger.Warn("access token rejected", "reason", "signature")` / `"reason", "expired"` и т.д.).

Это — то же решение, что для `ErrAssertionInvalid` (S4: challenge mismatch / sig fail / clone-warning все в один класс) и `ErrCredentialNotFound` (S4: нет credential / credential чужой / user удалён в один класс). Клиент эти под-причины не различает.

`ErrMissingBearer` (адаптер не нашёл `Authorization: Bearer ...`) — отдельный sentinel, но тоже маппится в 401. Различение существует ради логов («клиент вообще не пытался» vs «пытался, но токен битый») и ради того, чтобы конструктор `NewSessionLogoutCommand` имел осмысленную failure-ветку.

## Псевдокод пайпа головного модуля

```
ProcessSessionLogout(req: SessionLogoutRequest, deps: Deps)
    -> error:

    | NewSessionLogoutCommand(req)                                 -> SessionLogoutCommand
    | verifyInput := VerifyAccessTokenInput {
                       AccessTokenRaw: cmd.AccessTokenRaw(),
                       PublicKey:      deps.Verifier,
                       ExpectedIssuer: deps.JWT.Issuer,
                       Now:            deps.clock.Now() }
    | VerifyAccessToken(verifyInput)                                -> AuthenticatedUserID
    | revInput := RevokeUserSessionsInput {
                    UserID: authID.UserID(),
                    Now:    deps.clock.Now() }
    | deps.Store.RevokeUserSessions(revInput)                       -> error
    | return nil
```

Ошибки протекают через ранний `return fmt.Errorf("step: %w", err)`. Сборка `verifyInput`, `revInput` — Go-литералы структур, не отдельные узлы графа.

**Узлов графа — 3** (`NewSessionLogoutCommand`, `VerifyAccessToken`, `Store.RevokeUserSessions`). См. секцию «Решения по дизайну → Длина пайпа» о сабфлоре 5–10.

## Контракты модулей

### `NewSessionLogoutCommand`

- **Сигнатура:** `NewSessionLogoutCommand(req SessionLogoutRequest) -> (SessionLogoutCommand, error)`
- **Input (data):** `req SessionLogoutRequest`
- **Dependencies (deps):** —
- **Что делает:** проверяет, что адаптер положил в DTO непустой Bearer-токен; собирает доменную команду.
- **Антецедент:** `req.AccessTokenRaw` — строка, переданная адаптером после извлечения из заголовка `Authorization`.
- **Консеквент:**
  - Success: `SessionLogoutCommand { accessTokenRaw: req.AccessTokenRaw }` при `req.AccessTokenRaw != ""`.
  - Failure: `ErrMissingBearer` при `req.AccessTokenRaw == ""` (адаптер не нашёл Bearer-токен в заголовке или нашёл с пустым value).

### `VerifyAccessToken` (импорт S2)

- **Сигнатура:** `VerifyAccessToken(input VerifyAccessTokenInput) -> (AuthenticatedUserID, error)` (S2 экспорт; новая функция, см. `messages.md` → «Аддитивные расширения слайса 2 для слайса 5»).
- **Input (data):** `input VerifyAccessTokenInput { AccessTokenRaw, PublicKey, ExpectedIssuer, Now }`
- **Dependencies (deps):** — (всё нужное приходит в `input` как value-объект; `clock`/`PublicKey`/`JWTConfig` — зависимости *вызывающего*, не самой функции)
- **Что делает:**
  1. `jwt.ParseWithClaims(input.AccessTokenRaw, &jwt.RegisteredClaims{}, keyFunc)` с `keyFunc`, возвращающим `input.PublicKey`. Алгоритм — `jwt.SigningMethodEdDSA` (как в `generateTokenPair`).
  2. Проверка ошибки парсинга/верификации подписи.
  3. Проверка `claims.Issuer == input.ExpectedIssuer`.
  4. Проверка `claims.ExpiresAt > input.Now` (jwt-библиотека делает это сама при парсинге через `jwt.WithLeeway(0)` / по умолчанию; повторно — для согласованности с инъекцией `clock`).
  5. Парсинг `claims.Subject` как UUID → `UserID` через существующий `UserIDFromString` (S2 рехидратор).
- **Антецедент:** `input.AccessTokenRaw` — синтаксически JWT compact serialization (3 base64url-сегмента через `.`); `input.PublicKey` — корректный `ed25519.PublicKey`; `input.ExpectedIssuer` непустой; `input.Now` — момент.
- **Консеквент:**
  - Success: `AuthenticatedUserID { userID: UserID(claims.Subject) }` — токен корректен, не истёк, выдан этим issuer'ом.
  - Failure: `ErrAccessTokenInvalid` — любая из под-причин: malformed compact serialization, signature mismatch (другой ключ), expired, issuer mismatch, Subject не UUID, неверный алгоритм подписи в header'е.

Юнит-тесты считаются в карточке S5 (см. ниже): функция вводится в S5, юнит-формула считается там же.

### `Store.RevokeUserSessions` (метод I/O-объекта)

- **Сигнатура:** `(s *Store) RevokeUserSessions(input RevokeUserSessionsInput) -> error`
- **Input (data):** `input RevokeUserSessionsInput { UserID, Now }`
- **Dependencies (deps):** — (зависимость `*sql.DB` инкапсулирована внутри `Store`; головной модуль её не видит)
- **Что делает:** одна `UPDATE`-операция:
  ```
  UPDATE refresh_tokens
     SET revoked_at = ?
   WHERE user_id    = ?
     AND revoked_at IS NULL
  ```
  Single UPDATE в SQLite атомарен — обёртка в `BEGIN/COMMIT` не нужна. Возвращает `nil` независимо от числа затронутых строк (0+ строк: пользователь без активных refresh-токенов — это нормально, идемпотентность HTTP DELETE).
- **Антецедент:** `input.UserID` валиден (приходит из `AuthenticatedUserID.UserID()`); миграция `0004_refresh_tokens.sql` применена; `input.Now` — момент.
- **Консеквент:**
  - Success: все строки `refresh_tokens` с `user_id = input.UserID` и `revoked_at IS NULL` теперь имеют `revoked_at = input.Now`. Новые входы `POST /v1/sessions/{id}/assertion` (S4) этого пользователя выдадут новые refresh-токены, не затронутые этим UPDATE'ом.
  - Failure:
    - `ErrDBLocked` — `SQLITE_BUSY` → 503 `db_locked` + `Retry-After: 1`.
    - `ErrDiskFull` — `SQLITE_FULL` → 507 `db_disk_full`. (Технически UPDATE не растит файл, но WAL-лог растит; SQLite может вернуть `SQLITE_FULL` на UPDATE если diskspace = 0 и WAL не помещается.)
    - другие — обёрнуты в общую внутреннюю ошибку (→ 500 `INTERNAL_ERROR`).

`AND revoked_at IS NULL` в `WHERE` обеспечивает идемпотентность: повторный вызов logout с уже отозванными токенами вернёт 0 affected rows и `nil` ошибку. HTTP DELETE по REST идемпотентен — это инвариант обеспечивается на уровне SQL-запроса.

### Ингресс-адаптер: HTTP handler `DELETE /v1/sessions/current`

- **Что делает:**
  1. Читает заголовок `Authorization`. Если пустой или не начинается с префикса `"Bearer "` (case-sensitive, по RFC 6750 §2.1) — пишет `401 UNAUTHORIZED` напрямую (без вызова головного модуля), `error.code: UNAUTHORIZED`, `message: "missing or malformed Authorization header"`.
  2. Извлекает значение после `"Bearer "` в `accessTokenRaw`. Если после префикса пустая строка — то же 401 (или передать в `NewSessionLogoutCommand`, который вернёт `ErrMissingBearer`; см. ниже про границу).
  3. Собирает `SessionLogoutRequest{ AccessTokenRaw: accessTokenRaw }`.
  4. Вызывает `ProcessSessionLogout(req, deps)`.
  5. На Success: пишет `204 No Content`, без тела.
  6. На Failure: маппит ошибки в HTTP-ответ (см. таблицу маппинга ниже).
- **Никакой бизнес-логики** — только парсинг заголовка и маппинг.
- **Юнит-тестами не покрывается** — проверяется компонентным тестом слайса через реальный HTTP-вход.

**Граница «адаптер vs `NewSessionLogoutCommand`».** Адаптер ловит **синтаксические** случаи (заголовок отсутствует или не начинается с `"Bearer "`); `NewSessionLogoutCommand` ловит **доменный** случай (значение после префикса пустое). На практике оба приводят к 401 с одинаковым телом. Разделение — ради того, чтобы команда имела непустой контракт (Шаг 5: «Если консеквент не удаётся обосновать — модуль спроектирован неправильно»).

### Маппинг ошибок в ингресс-адаптере

| Класс ошибки                                                  | HTTP-статус | Заголовки           | Тело (`error.code`)         |
|---------------------------------------------------------------|-------------|---------------------|-----------------------------|
| (адаптер: нет/malformed `Authorization`)                       | 401          | —                   | `UNAUTHORIZED`               |
| `ErrMissingBearer`                                             | 401          | —                   | `UNAUTHORIZED`               |
| `ErrAccessTokenInvalid`                                        | 401          | —                   | `UNAUTHORIZED`               |
| `ErrDBLocked`                                                  | 503          | `Retry-After: 1`    | `db_locked`                  |
| `ErrDiskFull`                                                  | 507          | —                   | `db_disk_full`               |
| Любая другая (неожиданные SQLite, panic-recover)              | 500          | —                   | `INTERNAL_ERROR`             |

`ErrMissingBearer` и `ErrAccessTokenInvalid` оба маппятся в 401: для клиента поведение идентично («залогинься заново»). Различение остаётся только в логах адаптера.

## Gherkin-mapping

| Сценарий                                          | Then-шаг                                                         | Кто обеспечивает (узел графа / маппинг адаптера)                                                            |
|---------------------------------------------------|------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------|
| Выход инвалидирует refresh token                   | `Тогда ответ 204`                                                 | Узлы (1)–(3) Success-путь → ингресс-адаптер: `204 No Content` без тела                                       |

### Чек-лист сверки 8.5

1. [x] **Узел существует.** Узлы (1)–(3) описаны в дереве и в контрактах выше; ингресс-адаптер описан с маппингом.
2. [x] **Ветка соответствует.** Then `204` — Success-путь.
3. [x] **Формат ответа адаптера согласован.** OpenAPI декларирует 204 без тела; адаптер пишет `204 No Content` и пустое тело. Маппинги `ErrDBLocked → 503 + Retry-After + db_locked`, `ErrDiskFull → 507 + db_disk_full`, `ErrAccessTokenInvalid → 401 + UNAUTHORIZED` соответствуют README «Карта режимов отказа» и `responses` в OpenAPI (`Unauthorized`, `ServiceUnavailable`, `InsufficientStorage`).
4. [x] **Все Then покрыты.** В сценарии «Выход инвалидирует refresh token» 1 Then-шаг, покрыт.

`[x] Gherkin-mapping сверен.`

### Замечание о других режимах отказа

OpenAPI декларирует на этом эндпоинте также 401 `UNAUTHORIZED`, 503 `db_locked` и 507 `db_disk_full`, но Gherkin-сценариев на них **нет** в `sessions-current.feature` (по сознательному решению оператора: «отказы — отдельной задачей позже», см. секцию «Gherkin-сценарии слайса» выше про расхождение со Шагом 0).

Адаптер обязан корректно маппить все три класса в декларированные коды — это часть OpenAPI-контракта, проверка адаптерным маппингом тестируется опосредованно через прохождение happy path (если маппинг сломается, тест happy-path упадёт на ошибке внутреннего вызова).

`ErrMissingBearer`, `ErrAccessTokenInvalid`, `ErrDBLocked`, `ErrDiskFull` — все имеют декларированные классы маппинга в карточке выше; их юнит-проверка покрывается формулой ниже.

## Юнит-тесты по формуле

`N_юнит_тестов = 1 (happy path) + Σ (ветки антецедента)` — **только модули логики и конструкторы** (Шаг 8.1: «I/O — трубы, юнитами не покрываются»; ингресс-адаптер — тоже).

| Модуль                            | Happy | Ветки антецедента                                                                  | Итого |
|-----------------------------------|-------|------------------------------------------------------------------------------------|-------|
| `NewSessionLogoutCommand`         | 1     | пустая `AccessTokenRaw` → `ErrMissingBearer`                                        | 2     |
| `VerifyAccessToken` (импорт S2)   | 1     | malformed compact serialization, signature mismatch, expired, issuer mismatch, Subject не UUID | 6     |
| **Итого**                         |       |                                                                                    | **8** |

Для honest-теста `VerifyAccessToken` happy path и веток нужны:

- **valid token** — генерация: вызов того же `generateTokenPair` (S2) на тестовом user'е с известным `signer.Private`, потом `VerifyAccessToken` с парным `signer.Public`. Никаких моков, реальный JWT.
- **malformed**: `"not.a.jwt"` или одиночная строка без точек.
- **signature mismatch**: токен подписан одним ключом, верифицируется другим (генерим два keypair'а в тесте).
- **expired**: токен с `ExpiresAt` в прошлом (передаём `Now` в будущем после генерации).
- **issuer mismatch**: токен с `Issuer = "passkey-demo"`, передаём `ExpectedIssuer = "other"`.
- **Subject не UUID**: на этой ветке нужно вручную собрать JWT с `Subject = "not-a-uuid"`. Делается через тот же `jwt.NewWithClaims` в тесте, минуя `generateTokenPair`.

Что **не** в таблице (и почему):

- `Store.RevokeUserSessions` — метод I/O-объекта, труба. Юнитов нет. Success-путь проверяется компонентным сценарием **«Выход инвалидирует refresh token»** (если UPDATE не дойдёт, refresh-токен не отзовётся; следующий запрос на `POST /v1/sessions/{id}/assertion` или использование refresh для refresh-flow всё ещё пройдёт — но refresh-flow в этом проекте не реализован, поэтому проверка эффекта косвенная: повторный logout вернёт 204 идемпотентно). Failure-ветки `ErrDBLocked`/`ErrDiskFull` — без отдельного компонентного сценария на этом эндпоинте (по решению оператора).
- **Ингресс-адаптер** — парсинг заголовка и маппинг, юнитов нет.
- **Головной модуль** `ProcessSessionLogout` — оркестратор-труба: линейный пайп из 3 узлов, ошибки I/O пробрасываются без трансформации. Юнит-тест над ним был бы интеграционным тестом. Корректность пайпа доказывается компонентным сценарием.

Замечания по покрытию:

- 100% строк/веток модулей логики достигается этими 8 юнит-тестами.
- Honest-тест `VerifyAccessToken` использует `generateTokenPair` + два честных keypair'а — не мок.
- Юнит-тесты `VerifyAccessToken` живут в **`internal/slice/registrations_finish/`** (где определён сам `VerifyAccessToken`), не в `sessions_logout`. Это согласуется с правилом «юниты — у того, кто вводит модуль» (Шаг 8.1) и «без моков» (`feedback_no_mocks`).

## Definition of Done слайса

Скопировано в тикет S5 в `backlog.md`:

- [ ] **аддитивные расширения слайса 2**: экспортированы `VerifyAccessToken(input VerifyAccessTokenInput) (AuthenticatedUserID, error)`, типы `VerifyAccessTokenInput`, `AuthenticatedUserID` (с методом `UserID()`), sentinel `ErrAccessTokenInvalid`. Юнит-тесты S2 остаются зелёными (без изменения существующих тестов; новые 6 юнитов на `VerifyAccessToken` живут в пакете `registrations_finish`, как и сам `VerifyAccessToken`).
- [ ] ингресс-адаптер реализован: парсит заголовок `Authorization: Bearer <jwt>` в `SessionLogoutRequest{AccessTokenRaw}`, без бизнес-валидации (HTTP handler в `internal/slice/sessions_logout/`); пустой/malformed заголовок → 401 `UNAUTHORIZED` без вызова головного модуля.
- [ ] конструктор доменной структуры `NewSessionLogoutCommand` реализован; пустая `AccessTokenRaw` → `ErrMissingBearer`.
- [ ] **I/O-объект `Store` реализован** как автономный объект, инкапсулирующий `*sql.DB`: тип `*Store` в пакете `internal/slice/sessions_logout/`, конструктор `NewStore(db *sql.DB) *Store`, один метод:
  - `(s *Store) RevokeUserSessions(input RevokeUserSessionsInput) error`: одна `UPDATE refresh_tokens SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`; маппинг `SQLITE_BUSY` → `ErrDBLocked`, `SQLITE_FULL` → `ErrDiskFull`.
  - Голова `ProcessSessionLogout` обращается к БД **только через этот метод**; `*sql.DB` нигде кроме `Store` не светится в slice-пакете.
- [ ] головной модуль `ProcessSessionLogout` реализован: пайп из 3 узлов (см. псевдокод выше), ранний возврат при ошибке через `fmt.Errorf("…: %w", err)`.
- [ ] **новых миграций нет** — слайс использует `0004_refresh_tokens.sql` (UPDATE поля `revoked_at`). Никаких ALTER TABLE / новых файлов в `internal/db/migrations/` для S5 не создавать.
- [ ] инфраструктурный модуль расширен: `Deps` слайса 5 (`Store *Store`, `Clock`, `Logger`, `Verifier ed25519.PublicKey`, `JWT JWTConfig` — **без** сырого `*sql.DB`); подключение `sessions_logout.Register(mux, deps.SessionsLogout)` в `cmd/api/main.go`; в `wire.go` создаётся `sessions_logout.NewStore(db)` и пробрасывается в `Deps.Store`. Поле `Verifier` берётся из `signer.Public` (тот же `Signer` структуры из `wire.go`, сейчас передающий только `signer.Private` в S2/S4; S5 использует парный публичный ключ).
- [ ] слайс подключён через `sessions_logout.Register(mux, deps)`: HTTP-роут `DELETE /v1/sessions/current` ведёт на ингресс-адаптер.
- [ ] **юнит-тесты по формуле написаны и зелёные** — `go test ./...` проходит. **8 новых тестов**: 2 на `NewSessionLogoutCommand` (в пакете `sessions_logout`) + 6 на `VerifyAccessToken` (в пакете `registrations_finish` — там же, где сам `VerifyAccessToken`); головной модуль, I/O-модули и ингресс-адаптер юнитами не покрываются. `VerifyAccessToken` honest-тестируется через `generateTokenPair` + два честных Ed25519-keypair'а, без моков. Юниты S1/S2/S3/S4 остаются зелёными после аддитивных расширений S2 (`VerifyAccessToken`, типы, ошибки).
- [ ] **компонентные тесты, профиль `healthy`, зелёные** — `./component-tests/scripts/run-tests.sh healthy` проходит. Новый зелёный сценарий: `Сценарий: Выход инвалидирует refresh token` (`component-tests/features/sessions-current.feature`). Ранее зелёные сценарии S1-S4 в `registrations.feature`, `sessions.feature` продолжают проходить.
- [ ] **компонентные тесты, профиль `disk-full`, зелёные** — `./component-tests/scripts/run-tests.sh disk-full` проходит. Regression-проверка: `Сценарий: Диск переполнен при завершении регистрации` (`registrations.feature`) из S2. Если этот профиль продолжает регрессировать (как в S4), отметка переносится в техдолг и DoD-пункт пропускается с явным разрешением оператора (как S4 сделал для `db_disk_full`-сценария).
- [ ] сценарии в `users.feature` остаются красными в их Then-частях (S6 ещё не реализован), но **не** ломаются на When-шагах S1–S5.
- [ ] `backlog.md` обновлён по каждому подтверждённому пункту (правило `AGENTS.md §10`).
- [ ] `docs/design/passkey-demo/devlog.md` дополнен блоком S5.
- [ ] PR создан, описание заполнено по шаблону Шага 8 скилла sonnet'а.
- [ ] PR смержен в main, CI на main зелёный.

## Ссылки на источники

- Скилл реализации: `skills/program-implementation/SKILL.md`
- Граф вызовов: `docs/design/passkey-demo/contracts-graph.md` Slice 05
- Gherkin-mapping: раздел `## Gherkin-mapping` выше
- Аддитивные расширения S2: `docs/design/passkey-demo/messages.md` («Аддитивные расширения слайса 2 для слайса 5»)
- Подключение слайса 5: `docs/design/passkey-demo/infrastructure.md` → «Подключение слайса 5 (S5)»
- Подправило «подтип, не guard»: `skills/program-design/SKILL.md` Шаг 3 (применено в узле (2) `VerifyAccessToken` → `AuthenticatedUserID`)
