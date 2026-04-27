# Backlog — Passkey Demo API

## In Progress

## Todo

### Шаг 2 — Компонентные тесты (Gherkin)

Генерировать по процедуре `skills/component-tests/SKILL.md`. Ожидаемое число — 7 сценариев (6 happy-path + 1 отказ SQLite по `error.code=db_unavailable`).

- [ ] Описать сценарии регистрации (`POST /registrations`, `POST /registrations/{id}/attestation`)
- [ ] Описать сценарии входа (`POST /sessions`, `POST /sessions/{id}/assertion`)
- [ ] Описать сценарий выхода (`DELETE /sessions/current`)
- [ ] Описать сценарий `/users/me`
- [ ] Описать сценарии отказа на интеграции с БД
- [ ] Разместить в `/component-tests` (один `.feature` на ресурс)
- [ ] Зафиксировать в `devlog/02-gherkin.md`
- [ ] Merge в main

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
