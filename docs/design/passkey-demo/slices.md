# Slices — passkey-demo

Полный план слайсов сервиса. По одному слайсу на эндпоинт OpenAPI (правило `program-design.skill` Шаг 2: один внешний вход = один vertical slice).

| # | Тип входа | Идентификатор | Slice (имя) | Краткое описание | Статус |
|---|-----------|---------------|-------------|------------------|--------|
| 1 | HTTP | `POST /v1/registrations` | `registrations-start` | Фаза 1 регистрации: принять handle, создать challenge + регистрационную сессию, вернуть `{id, options}` | реализован (PR #17) |
| 2 | HTTP | `POST /v1/registrations/{id}/attestation` | `registrations-finish` | Фаза 2 регистрации: верифицировать attestation, создать пользователя и credential, выдать JWT-пару | спроектирован |
| 3 | HTTP | `POST /v1/sessions` | `sessions-start` | Фаза 1 входа: принять handle, найти пользователя и credential, создать challenge, вернуть `{id, options}` | спроектирован |
| 4 | HTTP | `POST /v1/sessions/{id}/assertion` | `sessions-finish` | Фаза 2 входа: верифицировать assertion, обновить счётчик, выдать JWT-пару | спроектирован |
| 5 | HTTP | `DELETE /v1/sessions/current` | `sessions-logout` | Инвалидация refresh token текущей сессии | todo |
| 6 | HTTP | `GET /v1/users/me` | `users-me` | Возврат данных пользователя из access token | todo |

## Зависимости между слайсами

Слайсы независимы по коду (каждый — самостоятельный пакет под `internal/slice/<name>/`). Зависимости — только через **общие данные в SQLite** и **общий инфраструктурный модуль**:

- 2 (`registrations-finish`) читает регистрационную сессию, созданную 1
- 3 (`sessions-start`) читает credential, созданный 2
- 4 (`sessions-finish`) читает сессию входа, созданную 3
- 4 после успеха создаёт JWT-пару, которую читают 5 и 6
- 5 пишет в blacklist refresh token (или удаляет запись о сессии)

## Раскладка режимов отказа SQLite по слайсам (из backlog.md)

`db_locked` (503 + `Retry-After`) и `db_disk_full` (507) — оба на одной интеграции (SQLite). По правилу различимости (`skills/component-tests/SKILL.md`) каждый различимый режим тестируется **одним** компонентным сценарием. Раскладка:

| Режим | Gherkin-сценарий | Эндпоинт | Слайс |
|-------|-----------------|----------|-------|
| `db_locked` | `Сценарий: SQLITE_BUSY на завершении входа` (`sessions.feature`) | `POST /v1/sessions/{id}/assertion` | 4 (`sessions-finish`) |
| `db_disk_full` | `Сценарий: Диск переполнен при завершении регистрации` (`registrations.feature`) | `POST /v1/registrations/{id}/attestation` | 2 (`registrations-finish`) |

**Важно для слайса 1.** В Gherkin-файле `registrations.feature` нет сценариев отказа на эндпоинте `POST /v1/registrations` — режимы отказа SQLite привязаны к фазе 2 (`registrations-finish`) и к фазе 2 входа (`sessions-finish`). Однако OpenAPI декларирует 503/507 на каждом write-эндпоинте, поэтому код слайса 1 **должен** маппить ошибки SQLite в эти статусы (без отдельного Gherkin-сценария на этом эндпоинте). Сверка покрытия Gherkin для слайса 1 ведётся только по happy path.

**Важно для слайса 2.** На эндпоинте `POST /v1/registrations/{id}/attestation` Gherkin покрывает happy + `db_disk_full`. `db_locked` декларирован OpenAPI, но не имеет компонентного сценария именно здесь (привязан к слайсу 4). Доменные ошибки (`HANDLE_TAKEN` race, `ATTESTATION_INVALID`, `NOT_FOUND` для истёкшей/отсутствующей сессии) — без компонентных сценариев, проверяются юнит-тестами модулей логики и контрактом OpenAPI.
