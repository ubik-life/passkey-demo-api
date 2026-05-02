# Backlog — Passkey Demo API

## In Progress

## Todo

### Шаг 2.0 — Шаблон компонентных тестов на Go

**Зачем:** Sonnet не может писать `.feature`-файлы в воздухе. Нужен раннер (godog), степ-фреймворк, docker-compose с SUT и фикстурами для каждого режима отказа SQLite, и набор «общих» степов (HTTP-клиент, виртуальный WebAuthn-аутентификатор, чейнинг токенов). Всё это разово готовится один раз перед T2.1.

**Аналог:** [`codemonstersteam/mq-rest-sync-adapter/component-tests`](https://github.com/codemonstersteam/mq-rest-sync-adapter/tree/main/component-tests) — JVM-версия (Cucumber + Gradle + Docker Compose + WireMock + IBM MQ). Делаем структурный аналог на Go.

**Ключевое отличие от JVM-примера:** SQLite встроенная — нет внешних контейнеров-стабов. «Стабы» сводятся к вариантам bind-mount/env для постановки SQLite в нужное состояние.

**Структура (предлагаемая):**

```
component-tests/
├── README.md, HOW-TO.md
├── go.mod, go.sum
├── Dockerfile.runtime                 # godog в контейнере (ОБЯЗАТЕЛЬНО, не запускается с хоста)
├── docker-compose.test.yml            # SUT + runner + bind-mount варианты
├── compose/
│   ├── service.Dockerfile             # multi-stage Go build сервиса
│   └── envs/                          # healthy.env, disk-full.env
├── scripts/
│   ├── run-tests.sh                   # build → docker compose run runner → down
│   ├── make-disk-full.sh              # tmpfs-fill до отказа
│   └── hold-lock.go                   # утилита для db_locked
├── features/                          # пишет sonnet в Шаге 2
├── steps/
│   ├── main_test.go                   # godog entry, suite hooks
│   ├── world.go                       # TestContext: HTTP-клиент, БД, токены
│   ├── http_steps.go
│   ├── webauthn_steps.go              # go-webauthn test helpers
│   ├── db_failure_steps.go            # два режима SQLite (locked, disk-full)
│   ├── auth_steps.go                  # «пользователь зарегистрирован и залогинен»
│   └── hooks.go                       # Before/After Scenario
└── fixtures/webauthn/
```

**Изоляция: раннер в контейнере.** Godog запускается **внутри Docker** через `docker compose run runner`, не с хоста. Это снимает класс «у меня работает, в CI падает», устраняет зависимость от локального Go-toolchain, переводит сеть раннер↔SUT на внутреннюю docker-сеть.

**Failure-режимы SQLite — как ставятся в compose/тестах:**

| Режим | Реализация |
|---|---|
| `db_locked` | Раннер-контейнер открывает свой коннект к тому же SQLite-файлу (общий bind-mount), стартует `BEGIN EXCLUSIVE` без коммита; SUT при попытке записи получает `SQLITE_BUSY` после `busy_timeout` |
| `db_disk_full` | bind-mount `tmpfs` размером 1–2 МБ, заполненный junk-файлом до отказа перед сценарием; SUT при `INSERT` получает `SQLITE_FULL` |

**Почему нет `db_unavailable`:** для встроенной SQLite это синтетический режим — нет «сетевой недоступности». Реальные runtime-отказы покрыты двумя режимами выше; неподнявшийся при старте сервис не входит в runtime-контракт. При миграции на сетевую БД режим вернётся.

**Bootstrap: placeholder-сервис.** Шаг 2.0 нужно закрыть одним PR, но сервис из Шага 3 ещё не написан, и smoke-сценарию (T2.0.12) не во что стрелять. Решение: в `compose/service.Dockerfile` собирается **временный заглушечный `main.go`** — HTTP-сервер, который биндится на порт и отдаёт `501 Not Implemented` на всё, кроме `/health` (он отвечает 200 для healthcheck). Он живёт в `cmd/api/main.go` и постепенно вытесняется реальной логикой в Шаге 3 (TDD-цикл первый модуль за модулем заменит хендлеры). 501 Not Implemented выбран намеренно — даёт чёткую границу «эндпоинт ещё не реализован» и не путается с реальным 401 (отсутствует токен), который появится в Шаге 3 для protected-эндпоинтов. Это сохраняет принцип «контракт и тесты первичны»: тесты красные не потому, что нет сервера, а потому, что сервер не реализует контракт.

**Тикеты Шага 2.0:**

- [x] **T2.0.1 — Каркас директории и `go.mod`.** Создан `component-tests/` как отдельный Go-модуль (Go 1.26). Зависимости (godog, go-webauthn, testify) подтягиваются в Stage B вместе со степ-дефинициями.
- [x] **T2.0.2 — `Dockerfile.runtime` для раннера.** Multi-stage Go-образ на базе `golang:1.26-alpine`. Раннер запускается через `docker compose run runner`, не с хоста.
- [x] **T2.0.3 — `compose/service.Dockerfile` + placeholder `cmd/api/main.go`.** Создан placeholder: `/health` → 200, всё остальное → **501 Not Implemented** с JSON `{"code":"NOT_IMPLEMENTED",...}`. Dockerfile собирает с `CGO_ENABLED=1` (выбран `mattn/go-sqlite3` как стандарт), статически линкует, runtime — `alpine:3` с wget для healthcheck.
- [x] **T2.0.4 — `docker-compose.test.yml` + `docker-compose.disk-full.yml`.** Базовый compose: SUT с HEALTHCHECK через wget на `/health`, runner с `depends_on: { service: { condition: service_healthy } }`, шаринг volume `sqlite-data` между service и runner для сценария `db_locked`. Override `docker-compose.disk-full.yml` подменяет volume на tmpfs 2 МБ для сценария `db_disk_full`.
- [x] **T2.0.5 — `steps/main_test.go` и `world.go`.** godog `TestMain`, регистрация всех степ-семейств в `InitializeScenario`. `world.go` хранит базовый URL и путь SQLite (из env), HTTP-клиент, последний ответ, фикстуры (handle/токены/challenge-id), указатели на виртуальный аутентификатор и SQLite lock-коннект. Lifecycle-хуки `beforeScenario`/`afterScenario` сбрасывают состояние и освобождают ресурсы.
- [x] **T2.0.6 — Базовые HTTP-степы (`steps/http_steps.go`).** Реализованы: `клиент отправляет (GET\|POST\|PUT\|DELETE) <path>` (с/без DocString-тела), `ответ <code>`, `ответ содержит заголовок <header>`, `ответ содержит JSON-поле <key> со значением "<value>"`. Если `accessToken` уже сохранён в `world` — добавляется `Authorization: Bearer`.
- [x] **T2.0.7 — WebAuthn-степы (`steps/webauthn_steps.go`).** Виртуальный аутентификатор через `github.com/descope/virtualwebauthn` (готовая библиотека для тестов WebAuthn-флоу). Степы: «у пользователя `<handle>` есть виртуальный аутентификатор», «клиент собирает attestation для challenge с id `<id>` и отправляет его», «клиент собирает assertion ... и отправляет его». Ключи генерируются `KeyTypeEC2`, attestation/assertion строятся из options, полученных в фазе 1.
- [x] **T2.0.8 — DB-failure степы (`steps/db_failure_steps.go`).** «БД заблокирована» — раннер открывает свой коннект к общему SQLite-файлу (volume `sqlite-data`), берёт `BEGIN EXCLUSIVE TRANSACTION`, держит до конца сценария; `afterScenario` делает `ROLLBACK`. «диск переполнен» — раннер пишет junk-файл 1.5 МБ в каталог БД (volume на этом профиле — tmpfs 2 МБ), забивая место. Адаптер сервиса (Шаг 3) обязан мапить `SQLITE_BUSY` → `db_locked`, `SQLITE_FULL` → `db_disk_full`.
- [x] **T2.0.9 — Auth-степы (`steps/auth_steps.go`).** Макрошаг «пользователь `<handle>` зарегистрирован и залогинен» — последовательно: `POST /v1/registrations` → создаётся виртуальный аутентификатор → `POST /v1/registrations/{id}/attestation` → токены извлекаются и сохраняются в `world.accessToken`/`refreshToken`. Используется как Background для тестов protected-эндпоинтов (T2.3, T2.4).
- [x] **T2.0.10 — `scripts/run-tests.sh`.** Принимает профиль (`healthy` / `disk-full`), собирает образы, делает `docker compose up --abort-on-container-exit --exit-code-from runner` (один вызов вместо up/run/down), гарантированно убирает контейнеры и volumes через trap. Файл `chmod +x`.
- [x] **T2.0.11 — `README.md` и `HOW-TO.md` в `component-tests/`.** README — тактический (как запустить, профили, список степов с регулярками, troubleshooting). HOW-TO — методология (зачем тесты, формула, почему Docker, почему mattn, что НЕ проверять, TDD-red phase). В обоих документах явно: `go test` с хоста не запускается никогда.
- [x] **T2.0.12 — Smoke-проверка (`features/smoke.feature`).** Один сценарий: `GET /v1/users/me` → `501 Not Implemented` + `code=NOT_IMPLEMENTED`. Использует только HTTP-степы (не зависит от WebAuthn/auth/db_failure). End-to-end прогнан через `./scripts/run-tests.sh`: 1 scenarios passed, 3 steps passed, exit 0. Обвязка работает.
- [x] **T2.0.13 — Devlog 06.** Зафиксирован в `devlog/06-component-tests-template.md`: 6 промптов сессии, 7 архитектурных решений Q1–Q7, что построено в Stages A/B/C, итоговое решение по `AGENTS.md §10`.
- [x] **T2.0.14 — Merge в main** (PR создаётся в финальном коммите).

**Важно:** Шаг 2.0 выполняется **до** T2.1. Без него sonnet будет писать `.feature` под несуществующие степы.

**Кто делает Шаг 2.0:** opus или человек. Sonnet — на T2.1+.

### Шаг 2 — Компонентные тесты (Gherkin) — план для sonnet

**Контекст:** контракт зафиксирован, шаблон тестов готов (Шаг 2.0 завершён). OpenAPI содержит 503 `db_locked` и 507 `db_disk_full`; README — таблица из двух режимов (синтетический `db_unavailable` для встроенной SQLite убран). Шаг 0 SKILL пройден заранее, sonnet сразу начинает с Шага 1.

**Ожидаемый итог:** 4 файла, 8 сценариев = 6 happy-path + 2 отказа SQLite. Запуск: Docker Compose с реальным SQLite, без моков.

**Раскладка failure-сценариев по эндпоинтам.** Оба режима — на одной интеграции (SQLite), по одному сценарию на режим. Распределяем по эндпоинтам, чьё поведение наиболее показательно для режима:

| Режим | Эндпоинт | Почему здесь |
|---|---|---|
| `db_locked` | `POST /v1/sessions/{id}/assertion` | Под нагрузкой это самый горячий путь записи (создание сессии) |
| `db_disk_full` | `POST /v1/registrations/{id}/attestation` | Самая «тяжёлая» запись (user + credential), естественное место для disk-full |

**Раскладка по файлам:**

```
component-tests/
├── registrations.feature       # 2 happy + db_disk_full (POST /registrations/{id}/attestation)
├── sessions.feature            # 2 happy + db_locked (POST /sessions/{id}/assertion)
├── sessions-current.feature    # 1 happy (DELETE /v1/sessions/current)
└── users.feature               # 1 happy (GET /v1/users/me)
```

**Тикеты для sonnet** (каждый — самодостаточный промпт, исполняется отдельной сессией):

- [x] **T2.1 — `registrations.feature`.** Прочитай `skills/component-tests/SKILL.md`, `api-specification/openapi.yaml` (пути `/registrations`, `/registrations/{id}/attestation`) и README раздел «Карта режимов отказа». Сгенерируй `component-tests/registrations.feature` на русском Gherkin с тремя сценариями: (1) создание challenge — `POST /v1/registrations` валидный handle → `201 {id, options}`, запись challenge в БД; (2) завершение регистрации — `POST /v1/registrations/{id}/attestation` валидный attestation → `200 TokenPair`, пользователь и credential сохранены, challenge помечен использованным; (3) `db_disk_full` — диск переполнен на `POST /v1/registrations/{id}/attestation` → `507 error.code=db_disk_full`. Не склеивать фазы в один сценарий.

- [x] **T2.2 — `sessions.feature`.** По SKILL и OpenAPI (`/sessions`, `/sessions/{id}/assertion`) сгенерируй `component-tests/sessions.feature` с тремя сценариями: (1) happy-path `POST /v1/sessions` известный handle → `201 {id, options}`, challenge сохранён; (2) happy-path `POST /v1/sessions/{id}/assertion` валидная подпись → `200 TokenPair`, сессия создана; (3) `db_locked` — `SQLITE_BUSY` на `POST /v1/sessions/{id}/assertion` → `503` с заголовком `Retry-After` и `error.code=db_locked`.

- [x] **T2.3 — `sessions-current.feature`.** По SKILL и OpenAPI (`DELETE /v1/sessions/current`) сгенерируй `component-tests/sessions-current.feature` с одним сценарием: запрос с валидным Bearer-токеном → `204`, refresh token инвалидирован в БД.

- [x] **T2.4 — `users.feature`.** По SKILL и OpenAPI (`GET /v1/users/me`) сгенерируй `component-tests/users.feature` с одним сценарием: запрос с валидным Bearer-токеном → `200 User`, поля `id` и `handle` соответствуют пользователю из токена.

- [x] **T2.5 — Сверка.** Прогнать чек-лист SKILL: `4 файла = 4 ресурса`, `6 happy-path = 6 эндпоинтов`, `2 сценария отказа = 2 режима`. Каждый failure-сценарий привязан к **одному** эндпоинту — не дублировать. Если расхождение — зафиксировать в комментарии PR.

- [x] **T2.6 — Devlog.** Зафиксировать шаг в `devlog/02-gherkin.md` по формату `Промпт / Что сделал агент / Решения / Результат`. Промпт каждого тикета T2.1–T2.4 — отдельным пунктом.

- [x] **T2.7 — Merge в main** (по флоу AGENTS.md §11).

**Промпт-шаблон для sonnet (одинаковый для T2.1–T2.4):**

> Прочитай `AGENTS.md`, `CLAUDE.md`, `skills/component-tests/SKILL.md`, `api-specification/openapi.yaml`, README раздел «Карта режимов отказа», `component-tests/README.md` и `component-tests/HOW-TO.md` (готовый раннер из Шага 2.0), список доступных степов в `component-tests/steps/`. Реши тикет {T2.X} из `backlog.md`. Не выходи за рамки SKILL: один сценарий = одно утверждение спецификации; не склеивать фазы; никаких сценариев валидации полей и бизнес-логики; failure-сценарии не дублировать между файлами — каждый режим отказа привязан к единственному эндпоинту по раскладке в backlog. **Используй уже существующие степы**, не выдумывай новые формулировки — если для сценария не хватает степа, останови работу и сообщи. Перед коммитом покажи дифф и жди подтверждения.

### Шаг 3 — Go-сервер (slice-by-slice)

Реализация идёт срезами (slice), а не плоским списком модулей. Проектирование каждого slice'а — итерация opus'а на скилле `program-design`, реализация — sonnet на скилле `program-implementation`. Источник истины по слайсу — `docs/design/passkey-demo/slices/NN-<slice>.md` + `contracts-graph.md` + `messages.md` + `infrastructure.md`.

Инфраструктурный модуль (Go-модуль, env-конфигурация, goose-миграции, HTTP-роутер, structured logging с trace_id, `/health`) собран в S1 по `infrastructure.md` — placeholder из `devlog/06` заменён на реальный сервер.

| Slice | Эндпоинт | Failure-режим | Статус |
|---|---|---|---|
| S1 — registrations-start | `POST /v1/registrations` | — | done (PR #17) |
| S2 — registrations-finish | `POST /v1/registrations/{id}/attestation` | `db_disk_full` | done (PR #XX) |
| S3 — sessions-start | `POST /v1/sessions` | — | todo (дизайн) |
| S4 — sessions-finish | `POST /v1/sessions/{id}/assertion` | `db_locked` | todo (дизайн) |
| S5 — sessions-logout | `DELETE /v1/sessions/current` | — | todo (дизайн) |
| S6 — users-me | `GET /v1/users/me` | — | todo (дизайн) |

S2 вводит JWT (Ed25519), сущности `User` и `Credential`. S5–S6 опираются на auth-middleware, который появится в S2 вместе с JWT.

**Цикл одного slice'а:**
1. Opus на ветке `feat/design-<slice>` расширяет `messages.md`, `contracts-graph.md`, добавляет `slices/NN-<slice>.md`, тикет в `docs/design/passkey-demo/backlog.md`. PR в main.
2. Sonnet на ветке `feat/slice-<slice>` реализует по тикету. PR в main.
3. Devlog `docs/design/passkey-demo/devlog.md` дополняется блоком после каждого slice'а.

**Definition of Done Шага 3:** все 6 slice'ов закрыты, все компонентные сценарии зелёные, `devlog/03-go-server.md` зафиксирован, CI на main зелёный.

### Шаг 4 — CI на PR

Запускается **после Шагов 1–3** (контракт, тесты, реализация). Цель — каждый PR в `main` автоматически проверяется на:

1. Валидность OpenAPI-спецификации.
2. Сборку сервиса (`go build ./cmd/api`).
3. Прогон компонентных тестов (`./scripts/run-tests.sh`).

Аналог из JVM-мира — [`.gitlab-ci.yml` mq-rest-sync-adapter](https://github.com/codemonstersteam/mq-rest-sync-adapter/blob/main/.gitlab-ci.yml): три стадии `test-api-specification → unit-tests → component-tests` с `asyncapi validate` для контракта. У нас REST → валидируем `openapi.yaml`. У нас GitHub, не GitLab → пишем GitHub Actions в `.github/workflows/`.

**Тикеты:**

- [ ] **T4.1 — Валидация OpenAPI на PR.** GitHub Actions workflow `.github/workflows/validate-openapi.yml`. Тригер: `pull_request` + изменения в `api-specification/**`. Инструмент: `redocly/cli` (npx или Docker-образ) — стандарт для OpenAPI 3.x. Проверки: lint + spec validity. PR с битой спекой не мержится.
- [ ] **T4.2 — Сборка сервиса в CI.** Workflow `.github/workflows/build-service.yml`: `go build ./cmd/api` на каждый PR. Защита от поломки сборки — компонентным тестам на сломанной сборке нет смысла гоняться.
- [ ] **T4.3 — Прогон компонентных тестов в CI.** Workflow `.github/workflows/component-tests.yml`: запускает `./scripts/run-tests.sh` (профиль `healthy` всегда; `disk-full` — если в фиче пишется в БД). Зависит от T4.2 (сборка зелёная). Артефакты — лог godog для разбора падений.
- [ ] **T4.4 — TODO: валидация документации (исследование).** Отдельная сессия: что значит «валидная документация»? Проверки на полноту README по требованиям AGENTS.md §19? Markdown-lint? Битые ссылки? Соответствие структуры devlog шаблону `docs/templates/devlog.md`? Связь Pipe-описаний эндпоинтов с реальными путями в OpenAPI? Результат исследования — отдельный план в backlog (T4.4.1+).
- [ ] **T4.5 — Devlog 07.** Зафиксировать в `devlog/07-ci.md`.
- [ ] **T4.6 — Merge в main.**

**Зачем после Шага 3.** До Шага 3 реализации сервиса нет — компонентные тесты в CI проверяли бы только smoke (placeholder отдаёт 501), а это и так есть локально через `./scripts/run-tests.sh`. Реальная защита от регрессий включается, когда в Шаге 3 модули по одному вытесняют placeholder и сценарии становятся зелёными.

## Done

- [x] `devlog/00-intent.md` — намерение, API-контракт, решения по архитектуре
- [x] `CLAUDE.md` — контекст проекта
- [x] `AGENTS.md` — фрейм разработки
- [x] `backlog.md` — план работы
- [x] Шаг 1 — `api-specification/openapi.yaml` + `devlog/01-api-contract.md`
- [x] `skills/component-tests/SKILL.md` — процедура генерации компонентных тестов; AGENTS.md §4/§6/§19 обновлены; «Карта режимов отказа» в README; `devlog/04-component-tests-skill.md`
- [x] Режимы отказа на интеграции SQLite: в OpenAPI добавлены `503 db_locked` и `507 db_disk_full`, расширена «Карта режимов отказа» в README, в SKILL добавлен Шаг 0 «Спроектировать режимы отказа и подготовить раннер», добавлен Шаг 2.0 (шаблон тестов на godog в Docker), детализирован Шаг 2 для sonnet (8 сценариев); `devlog/05-failure-modes.md`
- [x] Улучшения шаблона по итогам хендоффа sonnet'у: добавлен степ `ответ содержит JSON-поле <key>` (просто наличие, без проверки значения); подровнен `component-tests/README.md` под текущий вид WebAuthn-степов и оба JSON-степа; в `HOW-TO.md` зафиксировано правило «каждый степ из шаблона должен быть прокачен через `.feature`-сценарий до хендоффа» (не только через Go-вызов); в SKILL добавлен второй чек-лист для шаблона с этим правилом; smoke расширен до использования всех трёх JSON-степов
