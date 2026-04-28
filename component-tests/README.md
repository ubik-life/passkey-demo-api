# Component Tests — passkey-demo-api

Компонентные тесты сервиса в полной изоляции: SUT и раннер godog
поднимаются в Docker Compose, тесты гоняют чёрный ящик через HTTP,
без моков. Контракт описан в `../api-specification/openapi.yaml`,
методология — в [`../skills/component-tests/SKILL.md`](../skills/component-tests/SKILL.md),
философия — в [`HOW-TO.md`](HOW-TO.md).

## Требования

- `docker` 24+ с `docker compose` v2 (`compose v1` устарел и не работает с тегами `!override` в YAML).
- Свободные ~2 ГБ под образы и volume.
- Go на хосте **не нужен** — раннер в контейнере.

## Запуск

```bash
./scripts/run-tests.sh                  # профиль healthy (дефолт)
./scripts/run-tests.sh disk-full        # профиль для сценария db_disk_full
```

Скрипт собирает образы, поднимает сервис, ждёт health, запускает раннер,
возвращает его exit-код, гарантированно убирает контейнеры и volumes.

## Структура

```
component-tests/
├── go.mod / go.sum                 модуль раннера
├── Dockerfile.runtime              godog-контейнер
├── docker-compose.test.yml         базовый стек (SUT + runner)
├── docker-compose.disk-full.yml    override: SQLite на tmpfs 2 МБ
├── compose/
│   ├── service.Dockerfile          multi-stage build SUT (CGO + статика)
│   └── envs/                       env-профили для compose
├── scripts/run-tests.sh            точка входа
├── features/                       .feature-файлы (пишутся в Шаге 2 sonnet)
│   └── smoke.feature               проверка обвязки (Шаг 2.0)
└── steps/                          реализация степов на Go
    ├── main_test.go                godog-runner
    ├── world.go                    TestContext + lifecycle hooks
    ├── http_steps.go               универсальные HTTP-степы
    ├── webauthn_steps.go           виртуальный аутентификатор
    ├── db_failure_steps.go         db_locked + db_disk_full
    └── auth_steps.go               макрошаг «зарегистрирован и залогинен»
```

## Профили compose

| Профиль | Назначение | Volume для SQLite |
|---|---|---|
| `healthy` (дефолт) | happy-path сценарии и `db_locked` | named volume `sqlite-data` |
| `disk-full` | сценарий `db_disk_full` | tmpfs 2 МБ (override-файл) |

В `db_locked` раннер шарит volume с SUT, открывает свой коннект к тому же
SQLite-файлу и берёт `BEGIN EXCLUSIVE TRANSACTION`. SUT ловит `SQLITE_BUSY`
и должен отдать 503 `db_locked`.

В `db_disk_full` раннер пишет junk-файл 1.5 МБ в каталог БД на tmpfs,
оставляя SQLite без места. Первая же запись SUT получает `SQLITE_FULL`,
сервис должен отдать 507 `db_disk_full`.

## Доступные степы

Полный список — в `steps/*.go`. Самые важные:

**HTTP** (`http_steps.go`)
- `клиент отправляет (GET|POST|PUT|DELETE) <path>`
- `клиент отправляет (GET|POST|PUT|DELETE) <path> с телом:` (DocString JSON)
- `ответ <code>`
- `ответ содержит заголовок <header>`
- `ответ содержит JSON-поле <key> со значением "<value>"` — точное равенство
- `ответ содержит непустое JSON-поле <key>` — присутствует и не пустое (для JWT, refresh token и других динамических значений)
- `ответ содержит JSON-поле <key>` — просто присутствует, значение любое (для опциональных полей вроде `details: []`)

**WebAuthn** (`webauthn_steps.go`) — через [descope/virtualwebauthn](https://github.com/descope/virtualwebauthn)
- `у пользователя "<handle>" есть виртуальный аутентификатор`
- `клиент собирает attestation и отправляет его` — challenge id читается из последнего ответа сервера (фаза 1 регистрации)
- `клиент собирает assertion и отправляет его` — то же для фазы 2 входа

**Failure** (`db_failure_steps.go`)
- `БД заблокирована` — раннер берёт EXCLUSIVE-транзакцию
- `диск переполнен` — раннер заполняет tmpfs

**Auth** (`auth_steps.go`)
- `пользователь "<handle>" зарегистрирован и залогинен` — макрошаг
  для Background protected-эндпоинтов

## Не запускайте `go test` с хоста

Раннер в контейнере — это требование изоляции (см. `AGENTS.md`).
`go test ./steps/...` локально даст разное поведение в CI и у разработчика.
Всегда через `./scripts/run-tests.sh`.

## Добавление нового степа

1. Регулярка степа в `steps/<тема>_steps.go`, регистрация в
   `register<Тема>Steps`.
2. Реализация-метод на `*World`.
3. Если нужны новые сторонние пакеты — `go mod tidy` локально для
   обновления `go.sum` (не для запуска тестов с хоста).
4. Smoke-проверка: `./scripts/run-tests.sh`.

## Troubleshooting

- **`service` не становится healthy**: проверь `docker compose -f docker-compose.test.yml logs service`. Чаще всего — упавший билд из-за изменений в `cmd/api`.
- **Раннер не видит сервис**: проверь, что внутри сети compose ходит `wget -qO- http://service:8080/health` из раннер-контейнера.
- **`db_locked` тест проходит зелёным даже без сервиса**: значит SUT не мапит `SQLITE_BUSY` корректно. Проверь адаптер БД в Шаге 3.
