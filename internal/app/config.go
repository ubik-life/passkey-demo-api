package app

import (
	"fmt"
	"os"
	"time"

	"github.com/ubik-life/passkey-demo-api/internal/slice/registrations_start"
)

// AppConfig — конфигурация приложения из переменных окружения.
type AppConfig struct {
	ListenAddr   string
	DBPath       string
	RP           registrations_start.RPConfig
	ChallengeTTL time.Duration
	JWT          JWTConfig
}

// JWTConfig — заглушка; используется в слайсах 2/4.
type JWTConfig struct {
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// LoadConfig читает конфиг из env; паникует при отсутствии обязательных переменных.
func LoadConfig() (AppConfig, error) {
	var errs []string

	dbPath := os.Getenv("SQLITE_PATH")
	if dbPath == "" {
		errs = append(errs, "SQLITE_PATH is required")
	}

	ttl, err := parseDuration("PASSKEY_CHALLENGE_TTL", "5m")
	if err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return AppConfig{}, fmt.Errorf("config: %v", errs)
	}

	return AppConfig{
		ListenAddr: envOr("SERVICE_ADDR", ":8080"),
		DBPath:     dbPath,
		RP: registrations_start.RPConfig{
			Name: envOr("PASSKEY_RP_NAME", "Passkey Demo"),
			ID:   envOr("PASSKEY_RP_ID", "localhost"),
		},
		ChallengeTTL: ttl,
	}, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseDuration(key, def string) (time.Duration, error) {
	s := envOr(key, def)
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("%s=%q: invalid duration", key, s)
	}
	return d, nil
}
