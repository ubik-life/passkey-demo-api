package sessions_logout

import (
	"crypto/ed25519"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/ubik-life/passkey-demo-api/internal/clock"
	s2 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_finish"
)

// Deps — зависимости слайса 5. Инжектируются wire.go.
// Store автономен: head-модуль не знает про *sql.DB.
type Deps struct {
	Store    *Store
	Clock    clock.Clock
	Logger   *slog.Logger
	JWT      s2.JWTConfig
	Verifier ed25519.PublicKey
}

// Register подключает слайс к роутеру.
func Register(mux chi.Router, deps Deps) {
	h := newHTTPHandler(deps)
	mux.Delete("/v1/sessions/current", h.ServeHTTP)
}
