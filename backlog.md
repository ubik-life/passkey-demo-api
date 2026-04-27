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

**Тикеты Шага 2.0:**

- [ ] **T2.0.1 — Каркас директории и `go.mod`.** Создать `component-tests/` как отдельный Go-модуль (`go mod init github.com/ubik-life/passkey-demo-api/component-tests`). Подтянуть зависимости: `github.com/cucumber/godog`, `github.com/go-webauthn/webauthn` (test helpers), `github.com/stretchr/testify` для матчеров.
- [ ] **T2.0.2 — `Dockerfile.runtime` для раннера.** Multi-stage Go-образ с godog и зависимостями. Раннер запускается через `docker compose run runner`, не с хоста. Это требование изоляции — окружение раннера должно совпадать с CI.
- [ ] **T2.0.3 — `compose/service.Dockerfile`.** Multi-stage build сервиса под тестом — на базе образа из основного репозитория (`go build`, потом slim runtime).
- [ ] **T2.0.4 — `docker-compose.test.yml`.** Один файл, описывающий: SUT с healthcheck, runner с `depends_on: { service: { condition: service_healthy } }`, общая внутренняя сеть. SQLite-mount параметризован через переменные окружения compose-файла, чтобы скрипт менял профиль (`healthy`, `disk-full`).
- [ ] **T2.0.5 — `steps/main_test.go` и `world.go`.** godog `TestMain` с CLI-флагами; `world` хранит базовый URL (читается из env, который пробрасывает compose), http-клиент, контекст текущего пользователя/токена/challenge-id для чейнинга между шагами.
- [ ] **T2.0.6 — Базовые HTTP-степы.** «клиент отправляет POST/GET/DELETE `<path>`», «тело запроса содержит ...», «ответ `<code>`», «ответ содержит JSON-поле ... со значением ...», «ответ содержит заголовок `Retry-After`».
- [ ] **T2.0.7 — WebAuthn-степы.** «у пользователя есть виртуальный аутентификатор», «клиент подписывает challenge валидной подписью» — через `go-webauthn` test helpers. Ключи аутентификатора генерируются в Background, хранятся в `world`.
- [ ] **T2.0.8 — DB-failure степы.** «БД заблокирована» / «диск переполнен» — два отдельных Before-хука. Реализация: `db_locked` — параллельный коннект из раннер-контейнера держит `BEGIN EXCLUSIVE` до конца сценария; `db_disk_full` — профиль `disk-full.env` с bind-mount tmpfs 1–2 МБ, заполненный junk-файлом перед сценарием. Адаптер сервиса должен мапить `SQLITE_BUSY` → `db_locked`, `SQLITE_FULL` → `db_disk_full` (требование к реализации в Шаге 3).
- [ ] **T2.0.9 — Auth-степы.** «пользователь зарегистрирован и залогинен» — макрошаг, выполняющий полный happy-path регистрации и сохраняющий токен в `world`. Используется как Background для T2.3, T2.4.
- [ ] **T2.0.10 — `scripts/run-tests.sh`.** Полный цикл: `docker compose build` → `docker compose up -d <SUT>` → `docker compose run --rm runner` → `docker compose down -v`. Возвращает exit-код раннер-контейнера.
- [ ] **T2.0.11 — `README.md` и `HOW-TO.md` в `component-tests/`.** Как запустить (`./scripts/run-tests.sh`), какие профили compose, как добавить новый степ. Явно: «не запускать `go test` с хоста».
- [ ] **T2.0.12 — Smoke-проверка.** Один dummy-сценарий «сервис отвечает на `GET /v1/users/me` 401 без токена» — гоняем end-to-end через docker compose, чтобы убедиться, что вся обвязка работает.
- [ ] **T2.0.13 — Devlog 06.** Зафиксировать в `devlog/06-component-tests-template.md`.
- [ ] **T2.0.14 — Merge в main.**

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

- [ ] **T2.1 — `registrations.feature`.** Прочитай `skills/component-tests/SKILL.md`, `api-specification/openapi.yaml` (пути `/registrations`, `/registrations/{id}/attestation`) и README раздел «Карта режимов отказа». Сгенерируй `component-tests/registrations.feature` на русском Gherkin с тремя сценариями: (1) создание challenge — `POST /v1/registrations` валидный handle → `201 {id, options}`, запись challenge в БД; (2) завершение регистрации — `POST /v1/registrations/{id}/attestation` валидный attestation → `200 TokenPair`, пользователь и credential сохранены, challenge помечен использованным; (3) `db_disk_full` — диск переполнен на `POST /v1/registrations/{id}/attestation` → `507 error.code=db_disk_full`. Не склеивать фазы в один сценарий.

- [ ] **T2.2 — `sessions.feature`.** По SKILL и OpenAPI (`/sessions`, `/sessions/{id}/assertion`) сгенерируй `component-tests/sessions.feature` с тремя сценариями: (1) happy-path `POST /v1/sessions` известный handle → `201 {id, options}`, challenge сохранён; (2) happy-path `POST /v1/sessions/{id}/assertion` валидная подпись → `200 TokenPair`, сессия создана; (3) `db_locked` — `SQLITE_BUSY` на `POST /v1/sessions/{id}/assertion` → `503` с заголовком `Retry-After` и `error.code=db_locked`.

- [ ] **T2.3 — `sessions-current.feature`.** По SKILL и OpenAPI (`DELETE /v1/sessions/current`) сгенерируй `component-tests/sessions-current.feature` с одним сценарием: запрос с валидным Bearer-токеном → `204`, refresh token инвалидирован в БД.

- [ ] **T2.4 — `users.feature`.** По SKILL и OpenAPI (`GET /v1/users/me`) сгенерируй `component-tests/users.feature` с одним сценарием: запрос с валидным Bearer-токеном → `200 User`, поля `id` и `handle` соответствуют пользователю из токена.

- [ ] **T2.5 — Сверка.** Прогнать чек-лист SKILL: `4 файла = 4 ресурса`, `6 happy-path = 6 эндпоинтов`, `2 сценария отказа = 2 режима`. Каждый failure-сценарий привязан к **одному** эндпоинту — не дублировать. Если расхождение — зафиксировать в комментарии PR.

- [ ] **T2.6 — Devlog.** Зафиксировать шаг в `devlog/02-gherkin.md` по формату `Промпт / Что сделал агент / Решения / Результат`. Промпт каждого тикета T2.1–T2.4 — отдельным пунктом.

- [ ] **T2.7 — Merge в main** (по флоу AGENTS.md §11).

**Промпт-шаблон для sonnet (одинаковый для T2.1–T2.4):**

> Прочитай `AGENTS.md`, `CLAUDE.md`, `skills/component-tests/SKILL.md`, `api-specification/openapi.yaml`, README раздел «Карта режимов отказа», `component-tests/README.md` и `component-tests/HOW-TO.md` (готовый раннер из Шага 2.0), список доступных степов в `component-tests/steps/`. Реши тикет {T2.X} из `backlog.md`. Не выходи за рамки SKILL: один сценарий = одно утверждение спецификации; не склеивать фазы; никаких сценариев валидации полей и бизнес-логики; failure-сценарии не дублировать между файлами — каждый режим отказа привязан к единственному эндпоинту по раскладке в backlog. **Используй уже существующие степы**, не выдумывай новые формулировки — если для сценария не хватает степа, останови работу и сообщи. Перед коммитом покажи дифф и жди подтверждения.

### Шаг 3 — Go-сервер (TDD-цикл)
- [ ] Инициализировать Go-модуль, структуру директорий
- [ ] Настроить конфигурацию через env (AGENTS.md §16)
- [ ] Миграции БД через goose (AGENTS.md §15)
- [ ] Модуль: хранилище пользователей и credential (SQLite)
- [ ] Модуль: WebAuthn регистрация — фаза 1 (challenge)
- [ ] Модуль: WebAuthn регистрация — фаза 2 (attestation)
- [ ] Модуль: WebAuthn вход — фаза 1 (challenge)
- [ ] Модуль: WebAuthn вход — фаза 2 (assertion)
- [ ] Модуль: JWT (Ed25519) — выдача и валидация
- [ ] Модуль: выход (инвалидация refresh token)
- [ ] Модуль: `/users/me`
- [ ] HTTP-роутер, middleware (auth, logging, trace_id)
- [ ] Structured logging (JSON, trace_id, span_id)
- [ ] Зафиксировать в `devlog/03-go-server.md`
- [ ] Компонентные тесты зелёные
- [ ] Merge в main

## Done

- [x] `devlog/00-intent.md` — намерение, API-контракт, решения по архитектуре
- [x] `CLAUDE.md` — контекст проекта
- [x] `AGENTS.md` — фрейм разработки
- [x] `backlog.md` — план работы
- [x] Шаг 1 — `api-specification/openapi.yaml` + `devlog/01-api-contract.md`
- [x] `skills/component-tests/SKILL.md` — процедура генерации компонентных тестов; AGENTS.md §4/§6/§19 обновлены; «Карта режимов отказа» в README; `devlog/04-component-tests-skill.md`
- [x] Режимы отказа на интеграции SQLite: в OpenAPI добавлены `503 db_locked` и `507 db_disk_full`, расширена «Карта режимов отказа» в README, в SKILL добавлен Шаг 0 «Спроектировать режимы отказа и подготовить раннер», добавлен Шаг 2.0 (шаблон тестов на godog в Docker), детализирован Шаг 2 для sonnet (8 сценариев); `devlog/05-failure-modes.md`
