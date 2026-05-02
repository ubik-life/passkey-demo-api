package sessions_finish

import (
	"crypto/ed25519"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/ubik-life/passkey-demo-api/internal/clock"
	s1 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_start"
	s2 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_finish"
)

// Deps — зависимости слайса 4. Инжектируются wire.go.
// Store автономен: head-модуль не знает про *sql.DB.
type Deps struct {
	Store  *Store
	Clock  clock.Clock
	Logger *slog.Logger
	RP     s1.RPConfig
	JWT    s2.JWTConfig
	Signer ed25519.PrivateKey
}

// Register подключает слайс к роутеру.
func Register(mux chi.Router, deps Deps) {
	h := newHTTPHandler(deps)
	mux.Post("/v1/sessions/{id}/assertion", h.ServeHTTP)
}
