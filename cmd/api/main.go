package main

import (
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ubik-life/passkey-demo-api/internal/app"
	appdb "github.com/ubik-life/passkey-demo-api/internal/db"
	"github.com/ubik-life/passkey-demo-api/internal/clock"
	registrations_start "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_start"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := app.LoadConfig()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}

	db, err := appdb.Open(cfg.DBPath)
	if err != nil {
		log.Error("db open", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	deps := app.Build(cfg, db, log, clock.System{})

	mux := chi.NewRouter()
	mux.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	registrations_start.Register(mux, deps.RegistrationsStart)

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
		<-quit
		log.Info("shutting down")
		_ = srv.Close()
	}()

	log.Info("listening", "addr", cfg.ListenAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("listen", "err", err)
		os.Exit(1)
	}
}
