package sessions_start

import (
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ubik-life/passkey-demo-api/internal/clock"
	s1 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_start"
)

// Deps — зависимости слайса 3. Инжектируются wire.go.
// Store автономен: head-модуль не знает про *sql.DB.
type Deps struct {
	Store        Store
	Clock        clock.Clock
	Logger       *slog.Logger
	RP           s1.RPConfig
	ChallengeTTL time.Duration
}

// Register подключает слайс к роутеру.
func Register(mux chi.Router, deps Deps) {
	h := newHTTPHandler(deps)
	mux.Post("/v1/sessions", h.ServeHTTP)
}
