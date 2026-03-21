# Passkey Demo API

> **Учебный проект.** Это простой MVP, созданный как демонстрация процесса разработки с ИИ-агентом. Не предназначен для использования в продакшне.

Go-сервер, реализующий полный цикл авторизации без паролей: **WebAuthn + JWT (Ed25519) + SQLite**.

Первый кирпич ноды [Ubik](https://github.com/ubik-life/concept) — открытой децентрализованной платформы для распространения знаний.

---

## Что умеет сервис

- Регистрация пользователя по handle + биометрия (passkey)
- Вход по handle + биометрия
- Выход с инвалидацией сессии
- Проверка текущей сессии

## Стек

| Компонент | Технология |
|-----------|-----------|
| Язык | Go |
| Аутентификация | WebAuthn (FIDO2) |
| Токены | JWT, подпись Ed25519 |
| База данных | SQLite |

## API

Регистрация и вход — двухфазные: первый `POST` создаёт challenge, второй завершает процесс.

| Метод | Ресурс | Действие |
|-------|--------|----------|
| `POST` | `/v1/registrations` | Создать challenge → `201 {id, options}` |
| `POST` | `/v1/registrations/{id}/attestation` | Завершить регистрацию → JWT |
| `POST` | `/v1/sessions` | Создать challenge → `201 {id, options}` |
| `POST` | `/v1/sessions/{id}/assertion` | Завершить вход → JWT |
| `DELETE` | `/v1/sessions/current` | Выход — инвалидация refresh token |
| `GET` | `/v1/users/me` | Текущий пользователь |

Полная спецификация: [`api-specification/openapi.yaml`](api-specification/openapi.yaml)

## Структура репозитория

```
api-specification/   ← OpenAPI 3.1 спека
component-tests/     ← Gherkin-сценарии (компонентные тесты)
devlog/              ← Журнал разработки: промпты, решения, результаты
migrations/          ← Миграции БД (goose)
```

## Devlog

Весь процесс разработки задокументирован в [`devlog/`](devlog/). Цель — чтобы любой мог повторить путь шаг за шагом, работая с ИИ-агентом так же, как это делал автор.

| Файл | Содержание |
|------|-----------|
| [`00-intent.md`](devlog/00-intent.md) | Намерение, API-контракт, архитектурные решения |
| [`01-api-contract.md`](devlog/01-api-contract.md) | OpenAPI-спека |
| `02-gherkin.md` | Компонентные тесты |
| `03-go-server.md` | TDD-цикл: Go-сервер |

## Связанные репозитории

- UI: `git@github.com:ubik-life/passkey-demo-ui.git`
- Концепция платформы: `git@github.com:ubik-life/concept.git`
