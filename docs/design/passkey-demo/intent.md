# Intent — passkey-demo

## Задача в одну фразу

Сервер на Go, обслуживающий полный цикл passwordless-авторизации (двухфазная регистрация, двухфазный вход, выход, текущий пользователь) через WebAuthn + JWT (Ed25519) с хранением в SQLite.

## Контекст

Первый кирпич ноды [Ubik](https://github.com/ubik-life/concept). UI-часть — `passkey-demo-ui`. Контракт API зафиксирован в `api-specification/openapi.yaml`. Карта режимов отказа — в `README.md`. Компонентные сценарии Gherkin — в `component-tests/features/`.

Этот пакет — артефакт Шага 3 проекта (Go-сервер). До него:

- Шаг 1 — OpenAPI (`devlog/01-api-contract.md`)
- Шаг 2.0 — шаблон компонентных тестов (`devlog/06-component-tests-template.md`)
- Шаг 2 — 8 Gherkin-сценариев (`devlog/02-gherkin.md`)

## Метод

`skills/program-design/SKILL.md` — vertical slice architecture, контракты модулей с антецедентами/консеквентами, обратная сверка дизайна с Gherkin (таблица `## Gherkin-mapping` в карточке слайса), формула юнит-тестов. Реализация — `skills/program-implementation/SKILL.md` (TBD: один тикет = один slice = одна ветка = один PR).

## Итерационный режим

Пакет наполняется **по одному слайсу за раз**. В этой первой итерации:

- `slices.md` — таблица всех 6 слайсов (полный план)
- `messages.md` — структуры данных для первого слайса
- `slices/01-registrations-start.md` — полная карточка слайса 1
- `infrastructure.md` — инфраструктурный модуль с подключением только слайса 1
- `contracts-graph.md` — граф вызовов слайса 1
- `backlog.md` — один тикет S1 + хендофф-чеклист

После реализации S1 sonnet'ом и мержа в main — возвращаемся в скилл opus'а на следующий слайс.

## Ссылки

- OpenAPI: `api-specification/openapi.yaml`
- Карта режимов отказа: `README.md` раздел «Карта режимов отказа»
- Gherkin: `component-tests/features/*.feature`
- Скиллы: `skills/program-design/SKILL.md`, `skills/program-implementation/SKILL.md`, `skills/component-tests/SKILL.md`
