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

## S2 — HTTP POST /v1/registrations/{id}/attestation (2026-05-01)

**Что сделано:** Реализован слайс `registrations-finish` — фаза 2 регистрации. WebAuthn attestation verification (go-webauthn/webauthn), создание User+Credential, выдача JWT Ed25519 (access token) + opaque refresh token (sha256-хеш в БД). 3 новые миграции (users, credentials, refresh_tokens), расширения S1 (ChallengeFromBytes, RegistrationIDFromString, RegistrationSessionFromRow, RPConfig.Origin). Подтип FreshRegistrationSession (сессия не истекла). HTTP-обработчик со всеми кодами ошибок по контракту.

**Решения, принятые по ходу:**

- **API go-webauthn Verify.** Сигнатура `ParsedCredentialCreationData.Verify` в v0.17.0 имеет 11 параметров и отличается от старых примеров. Найдена по исходнику в GOMODCACHE. Параметр `credParams` обязателен — при `nil` верификация падает с "algorithm not supported".

- **virtualwebauthn API.** Пакет не имеет методов `auth.NewCredential()`, `auth.GetAttestationFor()`. Правильный вызов: `vwa.NewCredential(vwa.KeyTypeEC2)` (функция), `vwa.CreateAttestationResponse(rp, auth, cred, options)` (функция, не метод).

- **Честные юниты с virtualwebauthn.** Вместо моков WebAuthn-слоя используется virtualwebauthn прямо в юнит-тестах `logic_test.go` + in-memory SQLite в тестах головного модуля. Никаких stub-интерфейсов в Deps.

- **Изоляция компонентных сценариев.** SQLite-файл шарится между всеми сценариями (общий Docker volume). Без очистки: "bob" создаётся в "Завершение регистрации", затем background-шаг "sessions" пытается создать "bob" снова → 422 HANDLE_TAKEN. Исправление: `beforeScenario` открывает SQLite и делает DELETE FROM всех 4 таблиц + удаляет junk.bin от disk-full сценария.

- **AppConfig.JWT.** В S1 не было JWT-конфигурации. Добавлен `rf.JWTConfig` в `AppConfig` с тремя полями (AccessTTL, RefreshTTL, Issuer) и env-переменными PASSKEY_JWT_ACCESS_TTL/REFRESH_TTL/ISSUER. Signer (Ed25519 пара) генерируется при старте и не персистируется.

**Тесты:** юнитов 23 (3+2+1+1+1 domain + 4+2+1+1+7 logic/head). Компонентные сценарии: `Завершение регистрации` зелёный. Background-шаг "пользователь зарегистрирован и залогинен" зелёный. `Диск переполнен` красный ожидаемо (нужен профиль disk-full с tmpfs). Слайсы S3–S6 красные ожидаемо (не реализованы).
