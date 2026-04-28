package steps

import (
	"encoding/json"
	"fmt"

	"github.com/cucumber/godog"
)

// registerAuthSteps регистрирует доменные степы аутентификации:
// макрошаг «пользователь зарегистрирован и залогинен», который проходит
// все четыре фазы протокола (challenge регистрации → attestation →
// challenge входа → assertion) и сохраняет access_token в World.
//
// Используется как Background-степ в `.feature`-файлах для эндпоинтов
// под BearerAuth (DELETE /v1/sessions/current, GET /v1/users/me).
func (w *World) registerAuthSteps(ctx *godog.ScenarioContext) {
	ctx.Step(`^пользователь "([^"]+)" зарегистрирован и залогинен$`, w.userIsLoggedIn)
}

func (w *World) userIsLoggedIn(handle string) error {
	// Phase 1: регистрация — получить challenge id.
	regBody := fmt.Sprintf(`{"handle":"%s"}`, handle)
	if err := w.doRequest("POST", "/v1/registrations", []byte(regBody)); err != nil {
		return fmt.Errorf("phase 1 register: %w", err)
	}
	if w.lastResponse.StatusCode != 201 {
		return fmt.Errorf("phase 1 register: ожидали 201, получили %d", w.lastResponse.StatusCode)
	}
	// Виртуальный аутентификатор для пользователя.
	if err := w.givenVirtualAuthenticator(handle); err != nil {
		return fmt.Errorf("setup authenticator: %w", err)
	}

	// Phase 2: attestation — завершить регистрацию, получить токен.
	if err := w.sendAttestation(); err != nil {
		return fmt.Errorf("phase 2 attestation: %w", err)
	}
	if w.lastResponse.StatusCode != 200 {
		return fmt.Errorf("phase 2 attestation: ожидали 200, получили %d", w.lastResponse.StatusCode)
	}

	// Сохраняем токены — последующие шаги пойдут с Authorization-заголовком.
	var tokens struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(w.lastBody, &tokens); err != nil {
		return fmt.Errorf("phase 2 attestation: parse tokens: %w", err)
	}
	w.accessToken = tokens.AccessToken
	w.refreshToken = tokens.RefreshToken
	return nil
}
