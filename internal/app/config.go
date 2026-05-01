package app

import (
	"fmt"
	"os"
	"time"

	rf "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_finish"
	s1 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_start"
)

// AppConfig — конфигурация приложения из переменных окружения.
type AppConfig struct {
	ListenAddr   string
	DBPath       string
	RP           s1.RPConfig
	ChallengeTTL time.Duration
	JWT          rf.JWTConfig
}

// LoadConfig читает конфиг из env; возвращает ошибку при отсутствии обязательных переменных.
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

	accessTTL, err := parseDuration("PASSKEY_JWT_ACCESS_TTL", "15m")
	if err != nil {
		errs = append(errs, err.Error())
	}

	refreshTTL, err := parseDuration("PASSKEY_JWT_REFRESH_TTL", "720h")
	if err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return AppConfig{}, fmt.Errorf("config: %v", errs)
	}

	return AppConfig{
		ListenAddr: envOr("SERVICE_ADDR", ":8080"),
		DBPath:     dbPath,
		RP: s1.RPConfig{
			Name:   envOr("PASSKEY_RP_NAME", "Passkey Demo"),
			ID:     envOr("PASSKEY_RP_ID", "localhost"),
			Origin: envOr("PASSKEY_RP_ORIGIN", "http://localhost"),
		},
		ChallengeTTL: ttl,
		JWT: rf.JWTConfig{
			AccessTTL:  accessTTL,
			RefreshTTL: refreshTTL,
			Issuer:     envOr("PASSKEY_JWT_ISSUER", "passkey-demo"),
		},
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
