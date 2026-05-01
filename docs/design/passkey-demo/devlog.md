# Devlog — passkey-demo (реализация)

## S1 — HTTP POST /v1/registrations (2026-05-01)

**Что сделано:** Реализован слайс `registrations-start` по дереву модулей из карточки S1. Заменён placeholder `cmd/api/main.go` на реальный сервер с chi-роутером, goose-миграциями, JSON-логгером (slog) и graceful shutdown. Создана вся инфраструктура: `internal/app/`, `internal/db/`, `internal/clock/`.

**Решения, принятые по ходу:**

- **Env-переменные.** Infrastructure.md называл `PASSKEY_LISTEN_ADDR`/`PASSKEY_DB_PATH`, но compose-инфра (Шаг 2.0) уже использовала `SERVICE_ADDR`/`SQLITE_PATH`. Решение: использовать `SERVICE_ADDR`/`SQLITE_PATH` (как в compose), `PASSKEY_RP_NAME`/`PASSKEY_RP_ID`/`PASSKEY_CHALLENGE_TTL` — со значениями по умолчанию для demo.

- **Мокирование I/O в юнит-тестах.** Spec требовала "тестируется с моками I/O" для головного модуля, но `Deps` по design-doc имел `*sql.DB`. Добавлен `Persist func(RegistrationSession) error` в `Deps`, который устанавливается через `NewDeps` (замыкание над `*sql.DB`). Юниты подменяют `Persist` напрямую без SQLite-процесса.

- **Права на volume.** Docker-volume `/var/lib/passkey` монтируется как root:root. Исправление: в Dockerfile добавлен `mkdir -p /var/lib/passkey && chown app:app` перед `USER app:app`, плюс `os.MkdirAll` в `db.Open`.

- **DSN SQLite.** Использован `file:` URI-префикс для поддержки query-параметров (`_journal=WAL`, `_busy_timeout=5000`).

- **Smoke test.** Старый `smoke.feature` проверял 501 от placeholder — обновлён на `GET /health → 200`, что является постоянным контрактом сервиса.

**Тесты:** юнитов 11 (4+2+1+1+1+1+1), 100% покрытие конструкторов и модулей логики. Головной модуль юнитами не покрывается — склейка уже протестированных частей. Компонентные сценарии: `Создание challenge регистрации` зелёный, `Smoke` зелёный. Остальные 7 сценариев красные ожидаемо (слайсы 2–6 не реализованы), все падают на фазе 2, не на фазе 1.

**Стоимость сессии:** $4.07 (claude-sonnet-4-6). Время API: 19m 12s, wall: 42m 19s. Изменений: +926 / -73 строк.
