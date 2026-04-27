#!/usr/bin/env bash
# Запуск компонентных тестов passkey-demo-api.
#
# Использование:
#   ./scripts/run-tests.sh                # профиль healthy (дефолт)
#   ./scripts/run-tests.sh healthy
#   ./scripts/run-tests.sh disk-full
#
# Профили:
#   healthy   — обычный SQLite на named volume; для happy-path и db_locked
#   disk-full — SQLite на tmpfs 2 МБ; для сценария db_disk_full
#
# Раннер запускается ВНУТРИ Docker (см. AGENTS.md / SKILL — обязательное
# требование изоляции). Никаких `go test` с хоста.

set -euo pipefail

# Перейти в директорию component-tests/ независимо от того, откуда вызвали.
cd "$(dirname "$0")/.."

PROFILE="${1:-healthy}"

case "$PROFILE" in
  healthy)
    COMPOSE_FILES=(-f docker-compose.test.yml)
    ;;
  disk-full)
    COMPOSE_FILES=(-f docker-compose.test.yml -f docker-compose.disk-full.yml)
    ;;
  *)
    echo "unknown profile: $PROFILE (expected: healthy | disk-full)" >&2
    exit 2
    ;;
esac

# Гарантированная очистка контейнеров и volumes на любом завершении.
cleanup() {
  docker compose "${COMPOSE_FILES[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "==> profile: $PROFILE"
echo "==> building images..."
docker compose "${COMPOSE_FILES[@]}" build

echo "==> running tests..."
# --exit-code-from runner возвращает exit-код именно раннера (не первого
# завершившегося контейнера). --abort-on-container-exit останавливает SUT,
# когда раннер закончил.
docker compose "${COMPOSE_FILES[@]}" up \
  --abort-on-container-exit \
  --exit-code-from runner
