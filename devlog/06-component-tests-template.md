# 06 — Шаблон компонентных тестов на Go (godog в Docker)

## Промпты

1. «что делаем по плану?» — старт Шага 2.0.

2. «добавь B в бэклог» — закрепить решение «placeholder-сервис в Шаге 2.0 как bootstrap для smoke-теста, постепенно вытесняется реальной логикой в Шаге 3».

3. «что мне нужно провалидировать?» / «задай вопросы по одному» — разобрать 7 архитектурных решений по одному, чтобы я не катил их в бэклог явочным порядком.

4. Семь точечных ответов на вопросы Q1–Q7 (см. ниже).

5. «обновляй backlog после каждого шага» / «пропиши его для всех агентов» — правило про сразу-обновление `backlog.md` после каждого тикета. Сохранено в `AGENTS.md §10` для всех агентов.

6. «запусти End-to-end smoke я запустил докер» — финальная проверка обвязки.

## Контекст

После Шага 2 шаблона у нас уже была карта режимов отказа SQLite (`db_locked`, `db_disk_full`), ранее зафиксированная в `OpenAPI` и `README`. Тимлид-ревью (см. `devlog/05`) показало: sonnet нельзя выпускать на `.feature`-файлы, пока нет раннера, степов и compose-инфраструктуры. Шаг 2.0 — про то, чтобы построить эту обвязку до того, как sonnet начнёт работу.

Аналог из JVM-мира: [`codemonstersteam/mq-rest-sync-adapter/component-tests`](https://github.com/codemonstersteam/mq-rest-sync-adapter/tree/main/component-tests). Перенесли структурно на Go.

## 7 архитектурных решений (Q1–Q7)

| # | Вопрос | Решение | Почему |
|---|---|---|---|
| Q1 | Версия Go | **1.26** | Установлена у разработчика, актуальная стабильная |
| Q2 | Layout: один или два модуля | **Два** (`/go.mod` + `/component-tests/go.mod`) | Тест-зависимости (godog, virtualwebauthn, mattn/go-sqlite3) не пачкают прод-`go.mod`. Совпадает с JVM-аналогом (отдельный gradle-проект) |
| Q3 | Драйвер SQLite | **`mattn/go-sqlite3`** (не `modernc.org/sqlite`) | Стандарт де-факто, важно для обучающего материала. Цена — CGO + alpine runtime вместо distroless/static |
| Q4 | Поведение placeholder | **`501 Not Implemented`** на всё, кроме `/health` (200) | Чёткая граница «эндпоинт не реализован», не путается с реальным `401` (отсутствует токен) в Шаге 3 |
| Q5 | Как ставить `db_locked` | **Раннер шарит volume с SUT, открывает свой коннект, делает `BEGIN EXCLUSIVE`** | Прямой путь, никаких sidecar/test-endpoint. Runner и SQLite — оба тестовый код, шеринг файла приемлем |
| Q6 | HEALTHCHECK SUT | **В `service.Dockerfile`** (`HEALTHCHECK CMD wget -qO- /health`) + `depends_on: condition: service_healthy` | Стандартный compose-флоу, работает прозрачно. После Q3 (alpine, не distroless) — wget доступен |
| Q7 | Структура compose-файлов | **Один базовый + override** (`docker-compose.test.yml` + `docker-compose.disk-full.yml`) | Стандартный путь Docker Compose, видно явно, что меняется между профилями |

## Что построено

### Stage A — каркас + Dockerfiles + compose

```
.gitignore                                               (новый)
go.mod                                                    (новый, root)
cmd/api/main.go                                           placeholder
component-tests/.gitignore
component-tests/go.mod
component-tests/Dockerfile.runtime
component-tests/compose/service.Dockerfile               multi-stage CGO build, статика, alpine runtime
component-tests/compose/envs/healthy.env
component-tests/compose/envs/disk-full.env
component-tests/docker-compose.test.yml
component-tests/docker-compose.disk-full.yml             override: tmpfs 2 МБ
```

Placeholder локально проверен: `/health → 200`, всё остальное → `501 Not Implemented`. Compose-файлы валидированы через `docker compose config`.

### Stage B — степ-дефиниции

```
component-tests/steps/main_test.go         godog runner entry, регистрация всех степов
component-tests/steps/world.go             TestContext: URL/SQLite-path/HTTP-клиент/токены/lock
component-tests/steps/http_steps.go        универсальные HTTP-степы (5 шт)
component-tests/steps/webauthn_steps.go    виртуальный аутентификатор через descope/virtualwebauthn
component-tests/steps/db_failure_steps.go  db_locked (BEGIN EXCLUSIVE) + db_disk_full (junk-файл)
component-tests/steps/auth_steps.go        макрошаг «зарегистрирован и залогинен»
```

Зависимости: `cucumber/godog v0.14.1`, `descope/virtualwebauthn v1.0.3`, `mattn/go-sqlite3 v1.14.22`. `go build ./steps/...` зелёный.

При первой попытке использовать `descope/virtualwebauthn` API оказался не такой, как я предположил: `Authenticator` — value-тип, отдельного `User`-типа нет (user-данные внутри `AttestationOptions`). Поправил без накладок.

### Stage C — scripts + docs + smoke

```
component-tests/scripts/run-tests.sh      build → up --abort-on-container-exit --exit-code-from runner → cleanup через trap
component-tests/features/smoke.feature    GET /v1/users/me → 501 + code=NOT_IMPLEMENTED
component-tests/README.md                 тактический (как запустить, профили, степы)
component-tests/HOW-TO.md                 методология (зачем, формула, что НЕ проверять)
```

End-to-end smoke запущен после старта Docker Desktop. На первом прогоне раннер упал с `missing go.sum`: в `Dockerfile.runtime` я забыл скопировать `component-tests/go.sum` (оставил TODO с момента Stage A). Поправил, второй прогон зелёный:

```
Сценарий: Placeholder-сервис отдаёт 501 на любой нереализованный эндпоинт
  Когда клиент отправляет GET /v1/users/me                            # passed
  Тогда ответ 501                                                     # passed
  И ответ содержит JSON-поле code со значением "NOT_IMPLEMENTED"      # passed

1 scenarios (1 passed), 3 steps (3 passed)
runner exited with code 0
```

### AGENTS.md §10 — обновлено

Добавлено правило: `backlog.md` обновляется после каждого тикета, не одним батчем в конце PR. Это правило универсальное для всех агентов, версионируется в репо.

## Решения

**Раннер в Docker, не с хоста — без компромиссов.** Обнаружилось два раза: в `Dockerfile.runtime` нужны `gcc`/`musl-dev`/`sqlite-dev` для CGO-сборки `mattn/go-sqlite3`; готовая alpine-база не имела всего нужного, пришлось `apk add` в одну строку. Это плата за изоляцию.

**Один volume шарится — runner и SUT видят один SQLite-файл.** Без шеринга `db_locked` нереализуем чисто. Принято: тестовая инвазия в файловую систему SUT приемлема, при условии что HTTP-контракт остаётся чёрным ящиком.

**Sonnet работает по готовым степам, не выдумывает свои.** Промпт-шаблон из `backlog.md` Шаг 2 теперь явно требует читать `component-tests/README.md` и список степов в `steps/`. Если степа не хватает — sonnet останавливается и сообщает.

**Placeholder отдаёт 501, не 401.** Чёткая граница «не реализовано» / «не авторизован» — критично, когда в Шаге 3 будут вытесняться эндпоинты по одному, и часть из них действительно начнёт отдавать 401 для protected.

**Bootstrap-цикл `service.Dockerfile` + placeholder `cmd/api/main.go`.** Без минимального сервера smoke-сценарию не во что стрелять. Placeholder будет постепенно заменяться реальной логикой в Шаге 3 — единый `cmd/api/main.go` эволюционирует, Dockerfile остаётся.

## Результат

- `component-tests/` — полноценный шаблон Go-раннера в Docker.
- `cmd/api/main.go` — placeholder, готов к эволюции в Шаге 3.
- `AGENTS.md §10` — правило про обновление backlog.
- Smoke `./scripts/run-tests.sh` — зелёный end-to-end.
- Шаги T2.0.1–T2.0.12 в `backlog.md` отмечены `[x]`.
- Ветка `feat/component-tests-template` готова к PR.

После мержа sonnet может стартовать на T2.1 — все обещания шаблона держатся.
