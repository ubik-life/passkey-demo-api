// Package steps содержит godog-степы для компонентных тестов passkey-demo-api.
//
// World — общее состояние сценария. Создаётся заново на каждый Scenario,
// держит HTTP-клиент, последний ответ, фикстуры (handle, токены, challenge id),
// виртуальный аутентификатор и опциональную ссылку на удерживаемый SQLite-лок
// (для сценария db_locked).
package steps

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"time"

	"github.com/cucumber/godog"
	"github.com/descope/virtualwebauthn"
)

type World struct {
	// Конфигурация (читается из env, проброшенного docker-compose).
	serviceBaseURL string
	sqlitePath     string

	httpClient *http.Client

	// Последний HTTP-ответ — для Then-степов.
	lastResponse *http.Response
	lastBody     []byte

	// Фикстуры, populated шагами в Background и When.
	userHandle   string
	challengeID  string
	accessToken  string
	refreshToken string

	// Виртуальный WebAuthn-аутентификатор (создаётся в Background-степе).
	// Используем указатели на value-типы, чтобы по nil различать «есть/нет».
	authenticator *virtualwebauthn.Authenticator
	credential    *virtualwebauthn.Credential
	rp            virtualwebauthn.RelyingParty

	// Удерживаемый коннект для сценария db_locked.
	// Закрывается в afterScenario, освобождая EXCLUSIVE-транзакцию.
	lockDB   *sql.DB
	lockConn *sql.Conn
}

func newWorld() *World {
	return &World{
		serviceBaseURL: getenv("SERVICE_BASE_URL", "http://service:8080"),
		sqlitePath:     getenv("SQLITE_PATH", "/var/lib/passkey/data.db"),
		httpClient:     &http.Client{Timeout: 10 * time.Second},
	}
}

// resetState вызывается перед каждым сценарием.
func (w *World) resetState() {
	w.lastResponse = nil
	w.lastBody = nil
	w.userHandle = ""
	w.challengeID = ""
	w.accessToken = ""
	w.refreshToken = ""
	w.authenticator = nil
	w.credential = nil
}

// releaseLock закрывает коннект, удерживающий EXCLUSIVE-транзакцию,
// если такой был взят сценарием db_locked.
func (w *World) releaseLock(ctx context.Context) {
	if w.lockConn != nil {
		_, _ = w.lockConn.ExecContext(ctx, "ROLLBACK")
		_ = w.lockConn.Close()
		w.lockConn = nil
	}
	if w.lockDB != nil {
		_ = w.lockDB.Close()
		w.lockDB = nil
	}
}

// beforeScenario вызывается перед каждым сценарием через godog hooks.
func (w *World) beforeScenario(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
	w.resetState()
	return ctx, nil
}

// afterScenario вызывается после каждого сценария — освобождает ресурсы.
func (w *World) afterScenario(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
	w.releaseLock(ctx)
	if w.lastResponse != nil {
		_ = w.lastResponse.Body.Close()
	}
	return ctx, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
