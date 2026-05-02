package app

import (
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"fmt"
	"io"
	"log/slog"

	"github.com/ubik-life/passkey-demo-api/internal/clock"
	rf "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_finish"
	s1 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_start"
)

// Signer — пара Ed25519 ключей, генерируется при старте процесса.
type Signer struct {
	Private ed25519.PrivateKey
	Public  ed25519.PublicKey
}

// GenerateSigner создаёт новую пару Ed25519. Не персистируется.
func GenerateSigner() (Signer, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return Signer{}, fmt.Errorf("generate ed25519 keypair: %w", err)
	}
	return Signer{Private: priv, Public: pub}, nil
}

// WiredDeps — собранные зависимости всех слайсов.
type WiredDeps struct {
	RegistrationsStart  s1.Deps
	RegistrationsFinish rf.Deps
}

// Build собирает зависимости для всех слайсов.
func Build(cfg AppConfig, db *sql.DB, log *slog.Logger, clk clock.Clock, signer Signer, rnd io.Reader) WiredDeps {
	return WiredDeps{
		RegistrationsStart: s1.NewDeps(db, clk, log, cfg.RP, cfg.ChallengeTTL),
		RegistrationsFinish: rf.Deps{
			Store:  rf.NewStore(db),
			Clock:  clk,
			Logger: log,
			RP:     cfg.RP,
			JWT:    cfg.JWT,
			Signer: signer.Private,
			Rand:   rnd,
		},
	}
}
