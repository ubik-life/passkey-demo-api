# Multi-stage build SUT (passkey-demo-api).
# Build stage: тащим исходники из корня репозитория, собираем cmd/api
# с CGO=1 (mattn/go-sqlite3 — стандарт де-факто, требует C-тулчейна).
# Линкуем статически: runtime-образ на голом alpine, без sqlite-libs.
# По мере реализации Шага 3 содержимое cmd/api эволюционирует — Dockerfile
# уже готов к появлению sqlite-импортов и не требует пересборки инфры.

FROM golang:1.26-alpine@sha256:91eda9776261207ea25fd06b5b7fed8d397dd2c0a283e77f2ab6e91bfa71079d AS build

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /src

# Контекст сборки — корень репозитория (см. docker-compose.test.yml).
COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

ENV CGO_ENABLED=1
ENV GOOS=linux

RUN go build \
      -tags 'sqlite_omit_load_extension' \
      -ldflags '-s -w -linkmode external -extldflags "-static"' \
      -o /out/api ./cmd/api

FROM alpine:3.21@sha256:48b0309ca019d89d40f670aa1bc06e426dc0931948452e8491e3d65087abc07d

RUN apk add --no-cache ca-certificates wget && \
    addgroup -S app && adduser -S -G app app && \
    mkdir -p /var/lib/passkey && chown app:app /var/lib/passkey

WORKDIR /app
COPY --from=build /out/api /app/api

EXPOSE 8080

HEALTHCHECK --interval=2s --timeout=2s --start-period=5s --retries=10 \
    CMD wget -qO- http://localhost:8080/health || exit 1

USER app:app

ENTRYPOINT ["/app/api"]
