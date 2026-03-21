# 01 — API-контракт (OpenAPI)

## Промпт

1. «загрузи CLAUDE.md»
2. «загруги AGENTS.md»
3. «смотри на devlog/00-intent.md»
4. «ведем лог моих промптов и делаем первый шаг»
5. «нужно тебе описать для себя обновлять файл backlog по окончанию каждого шага, где это сделать лучше?»
6. «да»

## Что сделал агент

- Создал `backlog.md` с планом всех трёх шагов разработки
- Добавил правило обновления backlog в `CLAUDE.md`
- Создал `api-specification/openapi.yaml` — OpenAPI 3.1.0 спецификация

Покрыты все шесть эндпоинтов из `00-intent.md`:

| Метод | Ресурс | Ответ |
|-------|--------|-------|
| POST | /v1/registrations | 201 {id, options} |
| POST | /v1/registrations/{id}/attestation | 200 TokenPair |
| POST | /v1/sessions | 201 {id, options} |
| POST | /v1/sessions/{id}/assertion | 200 TokenPair |
| DELETE | /v1/sessions/current | 204 |
| GET | /v1/users/me | 200 User |

## Решения

**WebAuthn-структуры в схеме.** Описаны как подмножество Level 2 стандарта:
`PublicKeyCredentialCreationOptions` и `PublicKeyCredentialRequestOptions`.
Передаются клиенту as-is для передачи в `navigator.credentials.create/get()`.

**AttestationRequest / AssertionRequest** — структура взята из W3C WebAuthn:
поля `id`, `rawId`, `response`, `type`. Всё в Base64URL — браузер отдаёт именно так.

**TokenPair** — `access_token` (JWT Ed25519, короткоживущий) + `refresh_token`
(opaque, долгоживущий). Формат access_token не раскрывается в схеме — внутренняя деталь реализации.

**Единый формат ошибки** `{code, message, details}` — по AGENTS.md §14.

**Версионирование через URL** `/v1/...` — по AGENTS.md §1.

**attestation: none** (default) — для demo не нужна проверка цепочки сертификатов аутентификатора.

## Результат

`api-specification/openapi.yaml`
