# Backlog — passkey-demo (design pack)

Тикеты для sonnet'а. Один тикет = один слайс = одна ветка = один PR (TBD).

В этой итерации спроектирован один слайс (S1). Слайсы S2–S6 будут добавлены в следующих итерациях opus'ом после мержа S1.

---

## Хендофф-чеклист (заполняет opus, проверяет оператор)

- [x] OpenAPI / AsyncAPI зафиксирован, все эндпоинты slice'ов в нём описаны
- [x] OpenAPI / AsyncAPI содержит 5xx-ответы с `error.code` для каждого режима отказа
- [x] README содержит таблицу «Карта режимов отказа» (HTTP-статус / тип события / заголовки, действие клиента, действие оператора)
- [x] **Компонентные сценарии Gherkin для эндпоинтов всех slice'ов написаны, закоммичены, стабильны (один happy + сценарий на каждый различимый режим отказа)**
- [x] Папка docs/design/passkey-demo/ создана и полна
- [x] intent.md — задача в одну фразу
- [x] slices.md — таблица срезов с типом входа, идентификатором, назначением (все 6 слайсов перечислены; в текущей итерации спроектирован только S1, S2–S6 — статус «todo»)
- [x] messages.md — все структуры данных и Result<T, Error> (для S1)
- [x] Для каждого slice'а есть отдельный файл с деревом модулей *(на текущей итерации — только slices/01-registrations-start.md; S2–S6 добавятся в следующих итерациях по одному)*
- [x] У каждого slice'а описан головной модуль (оркестратор пайпа)
- [x] У головного модуля каждого slice'а зафиксирован псевдокод пайпа исполнения (5–10 шагов)
- [x] У каждого модуля логики описаны антецедент и консеквент
- [x] У каждого I/O-модуля slice'а описан контракт и режимы отказа
- [x] **У каждого модуля Input — одна доменная структура / DTO / void; deps вынесены отдельной строкой `Dependencies:` (Шаг 5). Узлов с 2+ data-аргументами в графе нет**
- [x] **Карточка каждого slice'а содержит таблицу `## Gherkin-mapping`: каждый Then-шаг каждого сценария slice'а привязан к узлу графа или маппингу адаптера (Шаг 8.4)**
- [x] **contracts-graph.md существует, граф каждого slice'а согласован (все стрелки помечены `[x]`, в т.ч. пункт 5 о покрытии Gherkin-сценариев)**
- [x] Для каждого модуля логики посчитаны юнит-тесты по формуле
- [x] **В таблице юнит-тестов каждой карточки слайса нет I/O-модулей и нет ингресс-адаптера: I/O — трубы, проверяются только компонентными сценариями (Шаг 8.1)**
- [x] infrastructure.md — описан инфраструктурный модуль приложения
- [x] backlog.md — тикеты по одному на slice, с зависимостями
- [x] Оператор аппрувит пакет — @maxmorev, 2026-05-01

> **Замечание о scope.** Хендофф-чеклист подтверждается **для слайса S1**. Для S2–S6 в этой итерации соответствующие пункты не применимы — их карточки будут спроектированы и согласованы отдельными итерациями. Это явный итеративный режим (см. `intent.md` «Итерационный режим»). Sonnet берёт из backlog только тот тикет, который в нём прописан.

---

## Тикеты

### S1 — slice `registrations-start`: HTTP POST /v1/registrations

**Спецификация:**
- `docs/design/passkey-demo/slices/01-registrations-start.md`
- `docs/design/passkey-demo/messages.md`
- `docs/design/passkey-demo/contracts-graph.md` (Slice 01)
- `docs/design/passkey-demo/infrastructure.md`

**Зависимости:** —

**Ветка:** `feat/slice-registrations-start`

**Definition of Done:**

- [x] ингресс-адаптер реализован: парсит JSON в `RegistrationStartRequest`, без бизнес-валидации (HTTP handler в `internal/slice/registrations_start/`)
- [x] конструкторы доменных структур (`NewHandle`, `NewRegistrationStartCommand`, `NewRegistrationSession`) реализованы: проверяют антецедент, при невалидных данных возвращают ошибку (структура не создаётся)
- [x] модули логики (`generateChallenge`, `generateRegistrationID`, `buildCreationOptions`, `buildResponse`) реализованы, контракты выполнены
- [x] модуль I/O (`persistRegistrationSession`) реализован, оборачивает SQLITE_BUSY → `ErrDBLocked`, SQLITE_FULL → `ErrDiskFull`
- [x] головной модуль `ProcessRegistrationStart` реализован: пайп из 7 шагов, ранний возврат при ошибке через `fmt.Errorf("…: %w", err)`
- [x] миграция `internal/db/migrations/0001_registration_sessions.sql` создаёт таблицу `registration_sessions(id PRIMARY KEY, handle, challenge, expires_at)`
- [x] инфраструктурный модуль (`cmd/api/main.go`, `internal/app/`, `internal/db/`, `internal/clock/`) собран по `infrastructure.md`; placeholder из `devlog/06` заменён на реальный сервер с одним рабочим эндпоинтом и `/health`
- [x] слайс подключён через `registrations_start.Register(mux, deps)`: HTTP-роут `POST /v1/registrations` ведёт на ингресс-адаптер
- [x] юнит-тесты по формуле — **14 тестов на модули логики и головной модуль** (см. таблицу в карточке слайса), покрытие 100% по строкам и веткам логики; I/O-модуль и ингресс-адаптер юнитами не покрываются
- [x] компонентный сценарий `Сценарий: Создание challenge регистрации` (`component-tests/features/registrations.feature`) зелёный
- [x] остальные сценарии в `registrations.feature`, `sessions.feature`, `sessions-current.feature`, `users.feature` остаются красными в их Then-частях, но **не должны** ломаться по фазе 1 регистрации (When «отправляет POST /v1/registrations» возвращает валидный `id` и `options`)
- [x] локальный CI зелёный (`go test ./...` для юнитов и `./component-tests/scripts/run-tests.sh` для компонентных)
- [x] `backlog.md` обновлён по каждому подтверждённому пункту (правило `AGENTS.md §10`)
- [x] `docs/design/passkey-demo/devlog.md` дополнен блоком S1 (формат: `## S1 — HTTP POST /v1/registrations (<YYYY-MM-DD>)` + что сделано / решения / что застряло / тесты)
- [x] PR создан, описание заполнено по шаблону Шага 8 скилла sonnet'а
- [x] PR смержен в main, CI на main зелёный

**Ссылки на источники:**

- Скилл реализации: `skills/program-implementation/SKILL.md`
- Граф вызовов: `docs/design/passkey-demo/contracts-graph.md` Slice 01 (главный источник истины о форме стрелок)
- Gherkin-mapping: раздел `## Gherkin-mapping` в `slices/01-registrations-start.md`

---

## Заметка для следующих итераций (S2–S6)

Когда S1 смержен и зелёный, opus возвращается на `program-design.skill` и наполняет:

- `slices/02-registrations-finish.md` (фаза 2 регистрации; вводит модули JWT, сущность User и Credential, режим `db_disk_full` на этом эндпоинте)
- `slices/03-sessions-start.md`
- `slices/04-sessions-finish.md` (режим `db_locked` на этом эндпоинте; здесь же интеграция со счётчиком signCount)
- `slices/05-sessions-logout.md`
- `slices/06-users-me.md`

Каждая итерация — отдельный `feat/design-<slice>` PR с расширением `messages.md`, `contracts-graph.md` и `backlog.md`. Хендофф-чеклист переподписывается оператором на каждой итерации.
