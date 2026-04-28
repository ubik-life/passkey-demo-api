# 02 — Gherkin-сценарии компонентных тестов

## Промпты

**T2.1 — `registrations.feature`:**
> «начинаем T2.0.14 + T2.1»

**T2.2 — `sessions.feature`:**
> «перейди в main забери обновления, продолжай по плану в новой ветке»

**T2.3 + T2.4 — `sessions-current.feature`, `users.feature`:**
> «вмерджил — перейди в мэйн, забери обновления и продолжай»

## Что сделал агент

Написал 4 `.feature`-файла, 8 сценариев по плану T2.1–T2.4.

До написания первого файла обнаружил разрыв в WebAuthn-степах: `sendAttestation(challengeID string)` и `sendAssertion(challengeID string)` принимали UUID аргументом из Gherkin, но UUID генерируется сервером динамически — вписать его в `.feature`-файл статически невозможно. Поле `phase1.ID` из `w.lastBody` уже парсилось, но игнорировалось. Исправил: оба степа теперь читают ID из `w.lastBody`, аргумент убран. Обновлён `auth_steps.go`. Добавлен степ `ответ содержит непустое JSON-поле <field>` — для проверки динамических полей (JWT, opaque token).

Каждый PR прогонялся через `./scripts/run-tests.sh` перед созданием.

## Решения

**Разные handle на каждый сценарий (alice/bob/carol).**
SQLite volume общий на весь прогон (не пересоздаётся между сценариями, только при `docker compose down -v`). Если все сценарии регистрируют одного пользователя — UNIQUE constraint ломает тесты после первого успешного сценария. Разные handle — простейшее решение без инфраструктурных изменений.

**`пользователь "X" зарегистрирован и залогинен` как Given для sessions.**
`sendAssertion()` требует `w.credential` и `w.authenticator` в World — они устанавливаются только через `sendAttestation()`. Единственный путь подготовить их через существующие степы — макрошаг `userIsLoggedIn`. Пользователь регистрируется как precondition; сам тест проверяет только sessions-контракт.

**Для `db_locked` — lock ПОСЛЕ phase 1, ПЕРЕД phase 2.**
Phase 1 тоже пишет в DB (сохраняет challenge). Если заблокировать DB до phase 1, сценарий упадёт раньше нужного. Правильный порядок: phase 1 завершена → `БД заблокирована` → phase 2 проверяет 503.

**Для `db_disk_full` — junk ПОСЛЕ phase 1, ПЕРЕД phase 2.**
Та же логика: phase 1 регистрации записывает challenge на диск. Junk пишется после phase 1, чтобы именно attestation (более тяжёлая запись: user + credential) получила `SQLITE_FULL`.

**`ответ 200` + `непустое JSON-поле` вместо точного значения JWT.**
Приватный ключ Ed25519 генерируется при старте процесса (не персистируется), timestamp `iat`/`exp` меняется каждый раз — JWT принципиально нединамичен. Добавлен новый степ `responseJSONFieldNonEmpty` для проверки присутствия токенов без проверки значения.

## Результат

```
component-tests/features/
├── registrations.feature      # 3 сценария: challenge, attestation, db_disk_full
├── sessions.feature           # 3 сценария: challenge, assertion, db_locked
├── sessions-current.feature   # 1 сценарий: DELETE /sessions/current → 204
└── users.feature              # 1 сценарий: GET /users/me → 200 + handle
```

Итого: 8 сценариев = 6 happy-path + 2 отказа SQLite. Формула SKILL: `6 эндпоинтов + 2 режима = 8`. ✅

Smoke зелёный на каждом прогоне. Все 8 новых сценариев красные — placeholder отдаёт 501. Нет ни одного `undefined step`.

PR: #10 (T2.1), #11 (T2.2), #12 (T2.3+T2.4).
