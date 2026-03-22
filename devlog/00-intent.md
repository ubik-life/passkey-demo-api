# 00 — Намерение

## Что строим

**Passkey Demo API** — Go-сервер, реализующий полный цикл авторизации без паролей.

Стек: Go + WebAuthn + JWT (Ed25519) + SQLite.

Это первый кирпич ноды Ubik — открытой децентрализованной платформы для распространения знаний. UI-часть живёт в репо `passkey-demo-ui`.

## Зачем этот devlog

Здесь фиксируется весь процесс разработки: дословные промпты, решения, альтернативы. Цель — чтобы любой студент смог повторить путь шаг за шагом, работая с ИИ-агентом так же, как это делал автор.

## Что умеет сервис

- Регистрация пользователя по handle + биометрия
- Вход по handle + биометрия
- Выход (инвалидация сессии)
- Проверка текущей сессии

## REST API

Регистрация и вход — двухфазные: первый POST создаёт challenge и возвращает ресурс с `id`, второй завершает процесс. Термины `attestation` и `assertion` взяты напрямую из WebAuthn.

| Метод | Ресурс | Действие |
|-------|--------|----------|
| `POST` | `/registrations` | Создать challenge → `201 {id, options}` |
| `POST` | `/registrations/{id}/attestation` | Завершить регистрацию → JWT |
| `POST` | `/sessions` | Создать challenge → `201 {id, options}` |
| `POST` | `/sessions/{id}/assertion` | Завершить вход → JWT |
| `DELETE` | `/sessions/current` | Выход — инвалидация refresh token |
| `GET` | `/users/me` | Текущий пользователь — требует валидный access token |

### Почему не PUT

PUT по семантике HTTP означает «положи ресурс по этому адресу целиком» и должен быть идемпотентным. Завершение регистрации и входа — продолжение процесса, а не замена ресурса. POST с вложенным ресурсом (`/attestation`, `/assertion`) честнее отражает природу действия.

## Как работает WebAuthn

Все бинарные данные между браузером и сервером передаются в **base64url**.

### Регистрация

**Фаза 1** — клиент → сервер:
```json
{ "handle": "alice" }
```

Сервер создаёт challenge, сохраняет сессию, возвращает `201`:
```json
{
  "id": "uuid",
  "options": {
    "challenge": "base64url",
    "rp": { "name": "Passkey Demo", "id": "localhost" },
    "user": { "id": "base64url", "name": "alice", "displayName": "alice" },
    "pubKeyCredParams": [{ "type": "public-key", "alg": -7 }],
    "timeout": 60000,
    "attestation": "none"
  }
}
```

**Фаза 2** — браузер вызывает `navigator.credentials.create(options)`, аутентификатор создаёт ключевую пару. Клиент → сервер:
```json
{
  "id": "base64url",
  "rawId": "base64url",
  "type": "public-key",
  "response": {
    "clientDataJSON": "base64url",
    "attestationObject": "base64url"
  }
}
```

- `clientDataJSON` (после decode): `{"type":"webauthn.create","challenge":"...","origin":"http://localhost"}`
- `attestationObject` — CBOR: содержит публичный ключ в COSE-формате, счётчик, rpIdHash

Сервер проверяет challenge, origin, декодирует публичный ключ и сохраняет credential. Возвращает пару JWT.

---

### Вход

**Фаза 1** — клиент → сервер:
```json
{ "handle": "alice" }
```

Сервер находит credentials пользователя, создаёт challenge, возвращает `201`:
```json
{
  "id": "uuid",
  "options": {
    "challenge": "base64url",
    "rpId": "localhost",
    "allowCredentials": [{ "type": "public-key", "id": "base64url" }],
    "userVerification": "preferred",
    "timeout": 60000
  }
}
```

**Фаза 2** — браузер вызывает `navigator.credentials.get(options)`, аутентификатор подписывает данные. Клиент → сервер:
```json
{
  "id": "base64url",
  "rawId": "base64url",
  "type": "public-key",
  "response": {
    "clientDataJSON": "base64url",
    "authenticatorData": "base64url",
    "signature": "base64url",
    "userHandle": "base64url"
  }
}
```

- `clientDataJSON` (после decode): `{"type":"webauthn.get","challenge":"...","origin":"http://localhost"}`
- `authenticatorData`: rpIdHash + flags + counter
- `signature`: подпись над `authenticatorData + SHA256(clientDataJSON)`

Сервер проверяет: challenge совпадает, origin совпадает, подпись валидна, счётчик вырос. Возвращает пару JWT.

---

### JWT

Сервер возвращает два токена:
```json
{
  "access_token": "JWT (Ed25519, TTL 15 мин)",
  "refresh_token": "opaque string (TTL 30 дней)"
}
```

`access_token` содержит claims: `sub` (user_id), `handle`, `exp`.
`refresh_token` хранится в БД, инвалидируется при выходе.

## Шаги разработки

1. `01-api-contract.md` — OpenAPI-спека
2. `02-gherkin.md` — компонентные тесты
3. `03-go-server.md` — TDD-цикл: Go-сервер

## Фрейм работы с агентом

См. `AGENTS.md` — системный контекст для всех сессий разработки.
