package app

import (
	"database/sql"
	"log/slog"

	"github.com/ubik-life/passkey-demo-api/internal/clock"
	"github.com/ubik-life/passkey-demo-api/internal/slice/registrations_start"
)

// WiredDeps — собранные зависимости всех слайсов.
type WiredDeps struct {
	RegistrationsStart registrations_start.Deps
}

// Build собирает зависимости для всех слайсов.
func Build(cfg AppConfig, db *sql.DB, log *slog.Logger, clk clock.Clock) WiredDeps {
	return WiredDeps{
		RegistrationsStart: registrations_start.NewDeps(db, clk, log, cfg.RP, cfg.ChallengeTTL),
	}
}
