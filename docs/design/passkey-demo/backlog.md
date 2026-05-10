# Backlog — passkey-demo (design pack)

Тикеты для sonnet'а. Один тикет = один слайс = одна ветка = один PR (TBD).

S1 спроектирован и реализован (PR #17). S2 спроектирован и реализован (PR #21). S3 спроектирован и реализован (PR #26). Техдолг S1/S2 → Store-объект закрыт (PR #27). S4 спроектирован (PR #28) и реализован (PR #30). S5 спроектирован (PR #32) и реализован (PR #36). S6 спроектирован (ветка `feat/design-users-me`), ожидает реализации.

---

## Хендофф-чеклист S6 (заполняет opus, проверяет оператор)

- [x] OpenAPI / AsyncAPI зафиксирован, эндпоинт `GET /v1/users/me` описан с 200/401/503
- [x] OpenAPI / AsyncAPI содержит 5xx-ответы с `error.code` для режима отказа `db_locked` (на S6 это единственный декларированный 5xx; `db_disk_full` не применим к read-only)
- [x] README содержит таблицу «Карта режимов отказа» с этими режимами (общий контракт SQLite)
- [ ] **Компонентные сценарии Gherkin для эндпоинта S6 написаны, закоммичены, стабильны** — **сознательное расхождение со Шагом 0 скилла**: на момент дизайна в `users.feature` есть только happy path (`Сценарий: Возвращает данные пользователя из токена`); сценарии на `db_locked` (503), `UNAUTHORIZED` (401), `INTERNAL_ERROR` (500 consistency anomaly) **отложены отдельной задачей** по решению оператора (то же, что в S5). Дизайн S6 фиксирует маппинги всех декларированных в OpenAPI режимов отказа плюс выбранный маппинг 500 для `ErrUserNotFound`; обратная сверка (Шаг 8.4) на момент handoff'а ведётся только по happy path. После того, как сценарии отказа будут дописаны, повторный прогон Gherkin-mapping пройдёт без переделки дизайна. Этот пункт намеренно остаётся `[ ]` до закрытия отдельного тикета на компонентные тесты S6.
- [x] Папка docs/design/passkey-demo/ дополнена под S6
- [x] intent.md — задача в одну фразу (без изменений)
- [x] slices.md — статус S5 → реализован (PR #36), S6 → спроектирован; раскладка failure-режимов и замечание о scope для S6 дополнены
- [x] messages.md — все структуры данных S6 описаны (`UsersMeRequest`, `UsersMeCommand`, `LoadUserInput`, `UserProfileResponse`, `Store`, `ErrMissingBearer`, `ErrUserNotFound`); аддитивных расширений других слайсов S6 не требует — все нужные публичные API уже экспортированы предыдущими handoff'ами
- [x] Для S6 есть отдельный файл с деревом модулей (`slices/06-users-me.md`)
- [x] У S6 описан головной модуль `ProcessUsersMe` (оркестратор пайпа)
- [x] У головного модуля S6 зафиксирован псевдокод пайпа — **4 узла, сабфлор скилла (5–10)**, обоснование в `slices/06-users-me.md` секция «Решения по дизайну → Длина пайпа: 4 узла». Задокументированный сабфлор требует явного аппрува оператора (см. строку «Оператор аппрувит пакет S6» внизу).
- [x] У каждого модуля логики S6 описаны антецедент и консеквент
- [x] У каждого I/O-модуля S6 описан контракт и режимы отказа — метод автономного объекта `Store` (`Store.LoadUser`); сырого `*sql.DB` в `Deps` головного модуля нет (Шаг 6 + `feedback_io_autonomous_store`)
- [x] **У каждого модуля S6 Input — одна доменная структура / DTO / void; deps вынесены отдельной строкой `Dependencies:` (Шаг 5). Узлов с 2+ data-аргументами в графе нет**
- [x] **Карточка S6 содержит таблицу `## Gherkin-mapping`: все три Then-шага сценария «Возвращает данные пользователя из токена» (200, id-непуст, handle="alice") привязаны к Success-цепочке узлов (1)–(4) → ингресс-адаптер**
- [x] **contracts-graph.md дополнен Slice 06: граф согласован (все стрелки `[x]`, включая пункт 5 о покрытии Gherkin)**
- [x] **Применено подправило «подтип, не guard» (Шаг 3 скилла): `AuthenticatedUserID` унаследован из S5 (узел (2) `VerifyAccessToken`), инвариант «JWT успешно верифицирован» закреплён в типе. `Store.LoadUser` принимает `LoadUserInput` с `UserID`, распакованным из `AuthenticatedUserID.UserID()` — компилятор не даст обойти верификацию.** S6 не вводит новых подтипов: `User` уже доменная сущность с инвариантами от `NewUser` (свежесозданные) и `UserFromRow` (рехидрированные).
- [x] Для каждого модуля логики S6 посчитаны юнит-тесты по формуле (3 теста: 2 на `NewUsersMeCommand` + 1 на `buildResponse`)
- [x] **В таблице юнит-тестов S6 нет головного модуля, нет метода I/O-объекта (`Store.LoadUser`) и нет ингресс-адаптера: все три — трубы, проверяются компонентным сценарием happy path (Шаг 8.1).** `VerifyAccessToken` в таблице **нет** — функция импортирована из S2 (введена в S5), юнит-формула посчитана и реализована в карточке S5.
- [x] infrastructure.md дополнен: `Deps` слайса 6 (без сырого `*sql.DB`, с полем `Verifier ed25519.PublicKey` — второе использование того же `signer.Public` после S5), подключение `users_me.Register` в `cmd/api/main.go`, явная отметка «новых миграций S6 не вводит» (использует таблицу `users` из `0002_users.sql`)
- [x] backlog.md — тикет S6 (см. ниже) с DoD из карточки слайса
- [x] Оператор аппрувит пакет S6 — @maxmorev, 2026-05-10

> **Замечания по Шагу 0 скилла (для оператора).** Хендофф S6 идёт с одним пунктом `[ ]` (Gherkin-сценарии отказа) и одной просьбой об аппруве сабфлора длины пайпа (4 узла vs 5-10 рекомендации). Оба расхождения зафиксированы в дизайне явно:
> - **Gherkin-отказы отложены** — то же сознательное решение оператора, что в S5; carve-out зафиксирован в `slices/06-users-me.md` (секция «Gherkin-сценарии слайса»), в `slices.md` (раскладка отказов «Важно для слайса 6») и в карточке слайса (раздел «Замечание о других режимах отказа»). Реализация sonnet'ом ляжет на happy-path Gherkin; маппинги адаптера для 401/500/503 будут реализованы по контракту OpenAPI плюс выбранному маппингу `ErrUserNotFound → 500` без отдельной компонентной проверки в этом PR.
> - **4-узловой пайп** — обоснован в `slices/06-users-me.md` (секция «Решения по дизайну → Длина пайпа: 4 узла»); альтернативы рассмотрены и отвергнуты как padding. По смыслу read-by-id — операционно простая операция «верифицировать токен → загрузить user → сериализовать»; натянуть 5+ узлов содержательно нельзя.
>
> Если эти решения требуют пересмотра — вернуться на Шаг 0 (написать Gherkin-сценарии отказа), либо пересмотреть `slices/06-users-me.md` (расширить пайп). Иначе — поставить `[x]` на строке аппрува и передать sonnet'у.

> **Замечание о scope.** Хендофф-чеклист подтверждается **для слайса S6**. На момент его создания S1–S5 уже реализованы. S6 — последний слайс по бэклогу проекта; после его реализации сервис закрыт по контракту.

---

## Хендофф-чеклист S5 (исторический — для аудита)

S5 закрыт PR #36. Чеклист сохранён ниже для трассировки.

<details>
<summary>Чеклист S5 (свернуть/развернуть)</summary>

- [x] OpenAPI / AsyncAPI зафиксирован, эндпоинт `DELETE /v1/sessions/current` описан с 204/401/503/507
- [x] OpenAPI / AsyncAPI содержит 5xx-ответы с `error.code` для режимов отказа `db_locked` и `db_disk_full`
- [x] README содержит таблицу «Карта режимов отказа» с этими режимами (общий контракт SQLite)
- [ ] **Компонентные сценарии Gherkin для эндпоинта S5 написаны, закоммичены, стабильны** — **сознательное расхождение со Шагом 0 скилла**: на момент дизайна в `sessions-current.feature` есть только happy path (`Сценарий: Выход инвалидирует refresh token`); сценарии на `db_locked` (503), `db_disk_full` (507), `UNAUTHORIZED` (401) **отложены отдельной задачей** по решению оператора (2026-05-05). Дизайн S5 фиксирует маппинги всех декларированных в OpenAPI режимов отказа; обратная сверка (Шаг 8.4) на момент handoff'а ведётся только по happy path. После того, как сценарии отказа будут дописаны, повторный прогон Gherkin-mapping пройдёт без переделки дизайна. Этот пункт намеренно остаётся `[ ]` до закрытия отдельного тикета на компонентные тесты S5.
- [x] Папка docs/design/passkey-demo/ дополнена под S5
- [x] intent.md — задача в одну фразу (без изменений)
- [x] slices.md — статус S5 → спроектирован; раскладка failure-режимов и замечание о scope для S5 дополнены
- [x] messages.md — все структуры данных S5 описаны (включая аддитивные расширения S2: `VerifyAccessToken`, `AuthenticatedUserID`, `VerifyAccessTokenInput`, `ErrAccessTokenInvalid`)
- [x] Для S5 есть отдельный файл с деревом модулей (`slices/05-sessions-logout.md`)
- [x] У S5 описан головной модуль `ProcessSessionLogout` (оркестратор пайпа)
- [x] У головного модуля S5 зафиксирован псевдокод пайпа — **3 узла, сабфлор скилла (5–10)**, обоснование в `slices/05-sessions-logout.md` секция «Решения по дизайну → Длина пайпа: 3 узла». Задокументированный сабфлор требует явного аппрува оператора (см. строку «Оператор аппрувит пакет S5» внизу).
- [x] У каждого модуля логики S5 описаны антецедент и консеквент
- [x] У каждого I/O-модуля S5 описан контракт и режимы отказа — метод автономного объекта `Store` (`Store.RevokeUserSessions`); сырого `*sql.DB` в `Deps` головного модуля нет (Шаг 6 + `feedback_io_autonomous_store`)
- [x] **У каждого модуля S5 Input — одна доменная структура / DTO / void; deps вынесены отдельной строкой `Dependencies:` (Шаг 5). Узлов с 2+ data-аргументами в графе нет**
- [x] **Карточка S5 содержит таблицу `## Gherkin-mapping`: единственный Then-шаг сценария «Выход инвалидирует refresh token» привязан к Success-цепочке узлов (1)–(3) → ингресс-адаптер (1 строка маппинга)**
- [x] **contracts-graph.md дополнен Slice 05: граф согласован (все стрелки `[x]`, включая пункт 5 о покрытии Gherkin)**
- [x] **Применено подправило «подтип, не guard» (Шаг 3 скилла): `VerifyAccessToken` — конструктор подтипа `AuthenticatedUserID`, инвариант «JWT успешно верифицирован» закреплён в типе. `Store.RevokeUserSessions` принимает `RevokeUserSessionsInput` с `UserID`, распакованным из `AuthenticatedUserID.UserID()` — компилятор не даст обойти верификацию.**
- [x] Для каждого модуля логики S5 посчитаны юнит-тесты по формуле (8 тестов: 2 на `NewSessionLogoutCommand` + 6 на `VerifyAccessToken`)
- [x] **В таблице юнит-тестов S5 нет головного модуля, нет метода I/O-объекта (`Store.RevokeUserSessions`) и нет ингресс-адаптера: все три — трубы, проверяются компонентным сценарием happy path (Шаг 8.1).** `VerifyAccessToken` в таблице **есть** — функция вводится в S5, юнит-формула считается у того, кто вводит модуль; физически юнит-тесты живут в пакете `internal/slice/registrations_finish/` (где определена сама функция).
- [x] infrastructure.md дополнен: `Deps` слайса 5 (без сырого `*sql.DB`, с новым полем `Verifier ed25519.PublicKey`), подключение `sessions_logout.Register` в `cmd/api/main.go`, явная отметка «новых миграций S5 не вводит» (использует колонку `revoked_at` из `0004_refresh_tokens.sql`, декларированную ещё в S2 именно для S5)
- [x] backlog.md — тикет S5 (см. ниже) с DoD из карточки слайса
- [x] Оператор аппрувит пакет S5 — @maxmorev, 2026-05-10

> **Замечания по Шагу 0 скилла (для оператора).** Хендофф S5 идёт с одним пунктом `[ ]` (Gherkin-сценарии отказа) и одной просьбой об аппруве сабфлора длины пайпа (3 узла vs 5-10 рекомендации). Оба расхождения зафиксированы в дизайне явно:
> - **Gherkin-отказы отложены** — сознательное решение оператора 2026-05-05; carve-out зафиксирован в `slices/05-sessions-logout.md` (секция «Gherkin-сценарии слайса»), в `slices.md` (раскладка отказов) и в карточке слайса (раздел «Замечание о других режимах отказа»). Реализация sonnet'ом ляжет на happy-path Gherkin; маппинги адаптера для 401/503/507 будут реализованы по контракту OpenAPI без отдельной компонентной проверки в этом PR.
> - **3-узловой пайп** — обоснован в `slices/05-sessions-logout.md` (секция «Решения по дизайну → Длина пайпа»); альтернативы рассмотрены и отвергнуты как padding. По смыслу логаут — операционно простая операция «верифицировать токен → отозвать токены»; натянуть 5+ узлов содержательно нельзя.
>
> Если эти решения требуют пересмотра — вернуться на Шаг 0 (написать Gherkin-сценарии отказа), либо пересмотреть `slices/05-sessions-logout.md` (расширить пайп). Иначе — поставить `[x]` на строке аппрува и передать sonnet'у.

> **Замечание о scope.** Хендофф-чеклист подтверждается **для слайса S5**. Для S6 пункты не применимы — карточка будет спроектирована отдельной итерацией. Sonnet берёт из backlog только тот тикет, который в нём прописан.

</details>

---

## Хендофф-чеклист S4 (исторический — для аудита)

S4 закрыт PR #30. Чеклист сохранён ниже для трассировки.

<details>
<summary>Чеклист S4 (свернуть/развернуть)</summary>

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
- [x] Оператор аппрувит пакет S4 — @maxmorev, 2026-05-02

> **Замечание о scope.** Хендофф-чеклист подтверждается **для слайса S4**. Для S5–S6 пункты не применимы — карточки будут спроектированы отдельными итерациями. Sonnet берёт из backlog только тот тикет, который в нём прописан.

</details>

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

- [x] **аддитивные расширения слайса 2**: экспортированы `GenerateTokenPair(input GenerateTokenPairInput) (IssuedTokenPair, error)` и `BuildResponse(view BuildTokenPairView) TokenPair` (публичные обёртки над пакетными `generateTokenPair`/`buildResponse`). Юнит-тесты S2 остаются зелёными (без изменения существующих тестов; тесты вызывают публичные имена).
- [x] **аддитивные расширения слайса 3**: экспортированы `LoginSessionIDFromString(s string) (LoginSessionID, error)` и `LoginSessionFromRow(rowID, rowUserID string, rowChallenge []byte, rowExpiresAtUnix int64) (LoginSession, error)`. Юнит-тесты S3 остаются зелёными.
- [x] **техдолг S1/S2 (Store-объект) закрыт** — `refactor/s1-s2-store` смержен в main (PR #27) до начала реализации S4. `Deps` всех реализованных слайсов используют `*Store`, сырого `*sql.DB` в `Deps` нигде нет.
- [x] ингресс-адаптер реализован: парсит path-параметр и тело в `SessionFinishRequest`, без бизнес-валидации (HTTP handler в `internal/slice/sessions_finish/`).
- [x] конструкторы доменных структур (`NewSessionFinishCommand`, `NewFreshLoginSession`) реализованы; невалидные данные → структура не создаётся, возвращается доменная ошибка.
- [x] модули логики (`parseAssertion`, `verifyAssertion`) реализованы, контракты выполнены.
- [x] **I/O-объект `Store` реализован** как автономный объект, инкапсулирующий `*sql.DB`: тип `*Store` в пакете `internal/slice/sessions_finish/`, конструктор `NewStore(db *sql.DB) *Store`, три метода:
  - `(s *Store) LoadLoginSession(id LoginSessionID) (LoginSession, error)`: SELECT по id, рехидратор `LoginSessionFromRow`; маппинг `sql.ErrNoRows` → `ErrLoginSessionNotFound`, `SQLITE_BUSY` → `ErrDBLocked`.
  - `(s *Store) LoadAssertionTarget(input LoadAssertionTargetInput) (AssertionTarget, error)`: SELECT credential по `credential_id` → in-memory проверка `user_id == input.UserID` → SELECT user по `id`; маппинг `sql.ErrNoRows`/mismatch → `ErrCredentialNotFound`, `SQLITE_BUSY` → `ErrDBLocked`. Рехидраторы — `CredentialFromRow`, `UserFromRow` (S2).
  - `(s *Store) FinishLogin(input FinishLoginInput) error`: атомарная транзакция (3 операции: UPDATE credentials + INSERT refresh_tokens + DELETE login_sessions), откат при любой ошибке; маппинг `SQLITE_BUSY` → `ErrDBLocked`, `SQLITE_FULL` → `ErrDiskFull`.
  - Голова `ProcessSessionFinish` обращается к БД **только через эти три метода**; `*sql.DB` нигде кроме `Store` не светится в slice-пакете.
- [x] головной модуль `ProcessSessionFinish` реализован: пайп из 8 шагов (10 строк с импортированными `GenerateTokenPair` и `BuildResponse`), ранний возврат при ошибке через `fmt.Errorf("…: %w", err)`.
- [x] **новых миграций нет** — слайс использует `0003_credentials.sql` (поле `sign_count`), `0004_refresh_tokens.sql` (INSERT), `0005_login_sessions.sql` (DELETE). Никаких ALTER TABLE / новых файлов в `internal/db/migrations/` для S4 не создавать.
- [x] инфраструктурный модуль расширен: `Deps` слайса 4 (`Store *Store`, `Clock`, `Logger`, `RP` (S1, нужны `ID` и `Origin`), `JWT` (S2), `Signer` ed25519.PrivateKey — **без** сырого `*sql.DB`); подключение `sessions_finish.Register(mux, deps.SessionsFinish)` в `cmd/api/main.go`; в `wire.go` создаётся `sessions_finish.NewStore(db)` и пробрасывается в `Deps.Store`.
- [x] слайс подключён через `sessions_finish.Register(mux, deps)`: HTTP-роут `POST /v1/sessions/{id}/assertion` ведёт на ингресс-адаптер.
- [x] **юнит-тесты по формуле написаны и зелёные** — `go test ./...` проходит. **11 новых тестов** на модули логики и конструкторы S4 (`parseAssertion`, `NewSessionFinishCommand`, `NewFreshLoginSession`, `verifyAssertion`); головной модуль, I/O-модули и ингресс-адаптер юнитами не покрываются. `verifyAssertion` honest-тестируется через `virtualwebauthn`. Юниты S1/S2/S3 остаются зелёными после аддитивных расширений (`GenerateTokenPair`, `BuildResponse`, `LoginSessionIDFromString`, `LoginSessionFromRow`).
- [x] **компонентные тесты, профиль `healthy`, зелёные** — `./component-tests/scripts/run-tests.sh healthy` проходит. Новые зелёные сценарии: `Сценарий: Завершение входа`, `Сценарий: БД заблокирована при завершении входа` (`sessions.feature`). Ранее зелёные сценарии S1/S2/S3 в `registrations.feature` (Создание challenge регистрации, Завершение регистрации) и `sessions.feature` (Создание challenge входа) продолжают проходить.
- [ ] **компонентные тесты, профиль `disk-full`, зелёные** — отложено до S5/S6: сценарий `Диск переполнен при завершении регистрации` регрессировал из-за роста БД в S3 (миграция 0005); junk.bin-размер или tmpfs требуют корректировки.
- [x] сценарии в `sessions-current.feature`/`users.feature` остаются красными в их Then-частях (S5/S6 ещё не реализованы), но **не** ломаются на When-шагах S1–S4 — `POST /v1/sessions/{id}/assertion` возвращает валидные `access_token`/`refresh_token`, которые используются как Bearer-токен в S5/S6.
- [x] `backlog.md` обновлён по каждому подтверждённому пункту (правило `AGENTS.md §10`).
- [x] `docs/design/passkey-demo/devlog.md` дополнен блоком S4.
- [x] PR создан, описание заполнено по шаблону Шага 8 скилла sonnet'а.
- [x] PR смержен в main, CI на main зелёный.

**Ссылки на источники:**

- Скилл реализации: `skills/program-implementation/SKILL.md`
- Граф вызовов: `docs/design/passkey-demo/contracts-graph.md` Slice 04
- Gherkin-mapping: раздел `## Gherkin-mapping` в `slices/04-sessions-finish.md`
- Подправило «подтип, не guard»: `skills/program-design/SKILL.md` Шаг 3 (применено в узле (3) `NewFreshLoginSession`; инвариант `AssertionTarget` инкапсулирован в I/O-возврате)

---

### S5 — slice `sessions-logout`: HTTP DELETE /v1/sessions/current

**Спецификация:**
- `docs/design/passkey-demo/slices/05-sessions-logout.md` (главный документ)
- `docs/design/passkey-demo/messages.md` — секция «Структуры слайса 5» + аддитивные расширения S2 (`VerifyAccessToken`, `AuthenticatedUserID`, `VerifyAccessTokenInput`, `ErrAccessTokenInvalid`)
- `docs/design/passkey-demo/contracts-graph.md` — секция «Slice 05»
- `docs/design/passkey-demo/infrastructure.md` — секция «Подключение слайса 5 (S5)» + раздел «S5 — без новых миграций»

**Зависимости:**
- S1 (PR #17, реализован) — `ErrDBLocked`, `ErrDiskFull`.
- S2 (PR #21, реализован) — `UserID`, `JWTConfig`, существующий `generateTokenPair` (для honest-юнит-теста `VerifyAccessToken`) + аддитивные расширения `VerifyAccessToken`, типы `VerifyAccessTokenInput`, `AuthenticatedUserID`, sentinel `ErrAccessTokenInvalid`.
- S4 (PR #30, реализован) — операционно: refresh-токены, которые S5 отзывает, появляются в БД через `Store.FinishLogin`. Структурного импорта типов S4 нет.

**Ветка:** `feat/slice-sessions-logout`

**Внешние зависимости (новые go.mod записи):** —

S5 не вводит новых внешних зависимостей: `github.com/golang-jwt/jwt/v5` уже подключён в S2 и используется в S5 для `jwt.ParseWithClaims` внутри `VerifyAccessToken`. `github.com/google/uuid` уже в проекте (для парсинга `claims.Subject` через существующий `UserIDFromString` S2-рехидратор).

**Definition of Done:**

- [x] **аддитивные расширения слайса 2**: экспортированы `VerifyAccessToken(input VerifyAccessTokenInput) (AuthenticatedUserID, error)`, типы `VerifyAccessTokenInput`, `AuthenticatedUserID` (с методом `UserID()`), sentinel `ErrAccessTokenInvalid`. Юнит-тесты S2 остаются зелёными (без изменения существующих тестов; новые 6 юнитов на `VerifyAccessToken` живут в пакете `registrations_finish`).
- [x] ингресс-адаптер реализован: парсит заголовок `Authorization: Bearer <jwt>` в `SessionLogoutRequest{AccessTokenRaw}`, без бизнес-валидации (HTTP handler в `internal/slice/sessions_logout/`); пустой/malformed заголовок (нет префикса `"Bearer "`) → 401 `UNAUTHORIZED` напрямую без вызова головного модуля.
- [x] конструктор доменной структуры `NewSessionLogoutCommand` реализован; пустая `AccessTokenRaw` → `ErrMissingBearer`.
- [x] **I/O-объект `Store` реализован** как автономный объект, инкапсулирующий `*sql.DB`: тип `*Store` в пакете `internal/slice/sessions_logout/`, конструктор `NewStore(db *sql.DB) *Store`, один метод:
  - `(s *Store) RevokeUserSessions(input RevokeUserSessionsInput) error`: одна `UPDATE refresh_tokens SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`; маппинг `SQLITE_BUSY` → `ErrDBLocked`, `SQLITE_FULL` → `ErrDiskFull`. Идемпотентность HTTP DELETE: фильтр `revoked_at IS NULL` гарантирует success на повторном вызове (0 affected rows = nil error).
  - Голова `ProcessSessionLogout` обращается к БД **только через этот метод**; `*sql.DB` нигде кроме `Store` не светится в slice-пакете.
- [x] головной модуль `ProcessSessionLogout` реализован: пайп из 3 узлов (см. псевдокод в карточке слайса), ранний возврат при ошибке через `fmt.Errorf("…: %w", err)`. **3 узла — сабфлор скилла (5–10), осознанное расхождение, обоснование в карточке.**
- [x] **новых миграций нет** — слайс использует `0004_refresh_tokens.sql` (UPDATE поля `revoked_at`, колонка декларирована именно для S5 ещё в S2). Никаких ALTER TABLE / новых файлов в `internal/db/migrations/` для S5 не создавать. Композитный индекс `(user_id, revoked_at)` — не вводится.
- [x] инфраструктурный модуль расширен: `Deps` слайса 5 (`Store *Store`, `Clock`, `Logger`, `Verifier ed25519.PublicKey`, `JWT JWTConfig` — **без** сырого `*sql.DB`); подключение `sessions_logout.Register(mux, deps.SessionsLogout)` в `cmd/api/main.go`; в `wire.go` создаётся `sessions_logout.NewStore(db)` и пробрасывается в `Deps.Store`. Поле `Verifier` берётся из `signer.Public` (тот же `Signer` структуры из `wire.go`, сейчас передающий только `signer.Private` в S2/S4; S5 — первое использование парного публичного ключа).
- [x] слайс подключён через `sessions_logout.Register(mux, deps)`: HTTP-роут `DELETE /v1/sessions/current` ведёт на ингресс-адаптер.
- [x] **юнит-тесты по формуле написаны и зелёные** — `go test ./...` проходит. **8 новых тестов**: 2 на `NewSessionLogoutCommand` (в пакете `sessions_logout`) + 6 на `VerifyAccessToken` (в пакете `registrations_finish` — там же, где сам `VerifyAccessToken`); головной модуль, I/O-модуль и ингресс-адаптер юнитами не покрываются. `VerifyAccessToken` honest-тестируется через `generateTokenPair` + два честных Ed25519-keypair'а, без моков (`feedback_no_mocks`). Юниты S1/S2/S3/S4 остаются зелёными после аддитивных расширений S2.
- [x] **компонентные тесты, профиль `healthy`, зелёные** — `./component-tests/scripts/run-tests.sh healthy` проходит. Новый зелёный сценарий: `Сценарий: Выход инвалидирует refresh token` (`component-tests/features/sessions-current.feature`). Ранее зелёные сценарии S1-S4 продолжают проходить.
- [ ] **компонентные тесты, профиль `disk-full`, зелёные** — техдолг: профиль регрессировал ещё в S4 (tmpfs 2 МБ не вмещает БД с S3-миграцией). Перенесён в техдолг с явным разрешением оператора (как в S4).
- [x] сценарии в `users.feature` остаются красными в их Then-частях (S6 ещё не реализован), но **не** ломаются на When-шагах S1–S5.
- [x] `backlog.md` обновлён по каждому подтверждённому пункту (правило `AGENTS.md §10`).
- [ ] `docs/design/passkey-demo/devlog.md` дополнен блоком S5.
- [ ] PR создан, описание заполнено по шаблону Шага 8 скилла sonnet'а.
- [ ] PR смержен в main, CI на main зелёный.

**Ссылки на источники:**

- Скилл реализации: `skills/program-implementation/SKILL.md`
- Граф вызовов: `docs/design/passkey-demo/contracts-graph.md` Slice 05
- Gherkin-mapping: раздел `## Gherkin-mapping` в `slices/05-sessions-logout.md`
- Подправило «подтип, не guard»: `skills/program-design/SKILL.md` Шаг 3 (применено в узле (2) `VerifyAccessToken` → `AuthenticatedUserID`)

---

### S6 — slice `users-me`: HTTP GET /v1/users/me

**Спецификация:**
- `docs/design/passkey-demo/slices/06-users-me.md` (главный документ)
- `docs/design/passkey-demo/messages.md` — секция «Структуры слайса 6»
- `docs/design/passkey-demo/contracts-graph.md` — секция «Slice 06»
- `docs/design/passkey-demo/infrastructure.md` — секция «Подключение слайса 6 (S6)» + раздел «S6 — без новых миграций»

**Зависимости:**
- S1 (PR #17, реализован) — `ErrDBLocked`.
- S2 (PR #21, реализован) — `User`, `UserID`, `JWTConfig`.
- S3 (PR #26, реализован) — рехидраторы `UserFromRow`, `UserIDFromString` (S3-аддитивные расширения S2).
- S5 (PR #36, реализован) — `VerifyAccessToken`, `AuthenticatedUserID`, `VerifyAccessTokenInput`, `ErrAccessTokenInvalid` (S5-аддитивные расширения S2).
- **Аддитивных расширений других слайсов S6 не требует** — все нужные публичные API уже экспортированы.

**Ветка:** `feat/slice-users-me`

**Внешние зависимости (новые go.mod записи):** —

S6 не вводит новых внешних зависимостей: `github.com/golang-jwt/jwt/v5` уже подключён в S2 и используется в S6 транзитивно через `VerifyAccessToken`. `github.com/google/uuid` уже в проекте (через `UserIDFromString`).

**Definition of Done:**

- [ ] **аддитивных расширений других слайсов не требуется** — все нужные публичные API (`VerifyAccessToken`, `AuthenticatedUserID`, `VerifyAccessTokenInput`, `ErrAccessTokenInvalid`, `User`, `UserID`, `UserFromRow`, `UserIDFromString`, `JWTConfig`) уже экспортированы в S2/S3/S5. Юнит-тесты S1–S5 остаются зелёными без изменений.
- [ ] ингресс-адаптер реализован: парсит заголовок `Authorization: Bearer <jwt>` в `UsersMeRequest{AccessTokenRaw}`, без бизнес-валидации (HTTP handler в `internal/slice/users_me/`); пустой/malformed заголовок (нет префикса `"Bearer "`) → 401 `UNAUTHORIZED` напрямую без вызова головного модуля; на Success — `200 OK` + JSON `UserProfileResponse` с `Content-Type: application/json`.
- [ ] конструктор доменной структуры `NewUsersMeCommand` реализован; пустая `AccessTokenRaw` → `ErrMissingBearer` (локальный sentinel в пакете `users_me`, не импорт из `sessions_logout`).
- [ ] модуль логики `buildResponse` реализован: маппит `User` в `UserProfileResponse` (поля `ID = user.ID().String()`, `Handle = user.Handle().Value()`).
- [ ] **I/O-объект `Store` реализован** как автономный объект, инкапсулирующий `*sql.DB`: тип `*Store` в пакете `internal/slice/users_me/`, конструктор `NewStore(db *sql.DB) *Store`, один метод:
  - `(s *Store) LoadUser(input LoadUserInput) (User, error)`: один `SELECT id, handle, created_at FROM users WHERE id = ?`; маппинг `sql.ErrNoRows` → `ErrUserNotFound` (consistency anomaly, → 500), `SQLITE_BUSY` → `ErrDBLocked`. Рехидрация через `UserFromRow` (S3 экспорт из S2). `ErrDiskFull` для read не различается.
  - Голова `ProcessUsersMe` обращается к БД **только через этот метод**; `*sql.DB` нигде кроме `Store` не светится в slice-пакете.
- [ ] головной модуль `ProcessUsersMe` реализован: пайп из 4 узлов (см. псевдокод в карточке слайса), ранний возврат при ошибке через `fmt.Errorf("…: %w", err)`. **4 узла — сабфлор скилла (5–10), осознанное расхождение, обоснование в карточке.**
- [ ] **новых миграций нет** — слайс использует существующую таблицу `users` (S2 миграция `0002`). Никаких ALTER TABLE / новых индексов / новых файлов в `internal/db/migrations/` для S6 не создавать.
- [ ] инфраструктурный модуль расширен: `Deps` слайса 6 (`Store *Store`, `Clock`, `Logger`, `Verifier ed25519.PublicKey`, `JWT JWTConfig` — **без** сырого `*sql.DB`); подключение `users_me.Register(mux, deps.UsersMe)` в `cmd/api/main.go`; в `wire.go` создаётся `users_me.NewStore(db)` и пробрасывается в `Deps.Store`. Поле `Verifier` — то же `signer.Public`, что у S5 (второе использование парного публичного ключа в проекте).
- [ ] слайс подключён через `users_me.Register(mux, deps)`: HTTP-роут `GET /v1/users/me` ведёт на ингресс-адаптер.
- [ ] **юнит-тесты по формуле написаны и зелёные** — `go test ./...` проходит. **3 новых теста**: 2 на `NewUsersMeCommand` (happy + пустая `AccessTokenRaw`) + 1 на `buildResponse` (чистый маппинг `User` → DTO). Головной модуль, I/O-метод и ингресс-адаптер юнитами не покрываются. Юниты S1–S5 остаются зелёными.
- [ ] **компонентные тесты, профиль `healthy`, зелёные** — `./component-tests/scripts/run-tests.sh healthy` проходит. Новый зелёный сценарий: `Сценарий: Возвращает данные пользователя из токена` (`component-tests/features/users.feature`). Ранее зелёные сценарии S1–S5 продолжают проходить.
- [ ] **компонентные тесты, профиль `disk-full`, зелёные** — техдолг: профиль регрессировал ещё в S4 и продолжает регрессировать в S5. Перенесён в техдолг с явным разрешением оператора (как S4/S5).
- [ ] `backlog.md` обновлён по каждому подтверждённому пункту (правило `AGENTS.md §10`).
- [ ] `docs/design/passkey-demo/devlog.md` дополнен блоком S6.
- [ ] PR создан, описание заполнено по шаблону Шага 8 скилла sonnet'а.
- [ ] PR смержен в main, CI на main зелёный.

**Ссылки на источники:**

- Скилл реализации: `skills/program-implementation/SKILL.md`
- Граф вызовов: `docs/design/passkey-demo/contracts-graph.md` Slice 06
- Gherkin-mapping: раздел `## Gherkin-mapping` в `slices/06-users-me.md`
- Подправило «подтип, не guard»: `skills/program-design/SKILL.md` Шаг 3 (унаследовано из S5: `AuthenticatedUserID` несёт инвариант «JWT успешно верифицирован» в типе)
- Devlog проектирования: `devlog/09-design-S6-users-me.md`

---

## Следующие итерации

S6 спроектирован (ветка `feat/design-users-me`), ожидает мержа дизайн-PR и реализации sonnet'ом по тикету выше на ветке `feat/slice-users-me`. После мержа S6-impl сервис закрыт по контракту: все 6 эндпоинтов OpenAPI реализованы и покрыты компонентными сценариями (happy path для всех; сценарии отказа на эндпоинтах S5/S6 — отдельной задачей, см. carve-outs в `slices.md` и хендофф-чеклистах).

Дальнейшие направления (вне текущего бэклога):

- Дописать Gherkin-сценарии отказа для S5 (`db_locked`/`db_disk_full`/`UNAUTHORIZED`) и S6 (`db_locked`/`UNAUTHORIZED`/`INTERNAL_ERROR` consistency).
- Архитектурное ретро по итогам шести слайсов (Шаг 5 проекта, см. `CLAUDE.md` → «Статус модулей»).
- ADR по Ed25519 keypair (генерация при старте, не персистится) — TODO из `CLAUDE.md`.
- Refresh-flow (`POST /v1/sessions/refresh`) — выходит за пределы текущего демо-контракта; потребует расширения OpenAPI и нового слайса S7.
