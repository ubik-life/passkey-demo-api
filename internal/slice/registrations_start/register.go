package registrations_start

import (
	"database/sql"
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ubik-life/passkey-demo-api/internal/clock"
)

// Deps — зависимости слайса. Инжектируются wire.go.
// Store автономен: head-модуль не знает про *sql.DB.
type Deps struct {
	Store        Store
	Clock        clock.Clock
	Logger       *slog.Logger
	RP           RPConfig
	ChallengeTTL time.Duration
}

// NewDeps собирает Deps из общих зависимостей приложения.
func NewDeps(db *sql.DB, clk clock.Clock, log *slog.Logger, rp RPConfig, ttl time.Duration) Deps {
	return Deps{
		Store:        NewStore(db),
		Clock:        clk,
		Logger:       log,
		RP:           rp,
		ChallengeTTL: ttl,
	}
}

// Register подключает слайс к роутеру.
func Register(mux chi.Router, deps Deps) {
	h := newHTTPHandler(deps)
	mux.Post("/v1/registrations", h.ServeHTTP)
}
