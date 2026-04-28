package steps

import (
	"encoding/json"
	"fmt"

	"github.com/cucumber/godog"
	"github.com/descope/virtualwebauthn"
)

// registerWebAuthnSteps регистрирует степы для виртуального аутентификатора.
//
// Подход: используем github.com/descope/virtualwebauthn — библиотеку для
// тестирования WebAuthn-флоу. Она строит attestation/assertion в правильном
// CBOR-формате и подписывает их сгенерированным ключом, не требуя реального
// браузера или USB-ключа.
//
// Жизненный цикл аутентификатора привязан к сценарию (создаётся в Background,
// сбрасывается в afterScenario вместе со всем World).
func (w *World) registerWebAuthnSteps(ctx *godog.ScenarioContext) {
	ctx.Step(`^у пользователя "([^"]+)" есть виртуальный аутентификатор$`, w.givenVirtualAuthenticator)
	ctx.Step(`^клиент собирает attestation для challenge с id "([^"]+)" и отправляет его$`, w.sendAttestation)
	ctx.Step(`^клиент собирает assertion для challenge с id "([^"]+)" и отправляет его$`, w.sendAssertion)
}

// givenVirtualAuthenticator создаёт пустой виртуальный аутентификатор.
// Credential генерируется при первом attestation-флоу (в sendAttestation).
//
// User-данные (id/name/displayName) приходят с сервера в attestation-options
// и заполняются библиотекой автоматически — отдельного User-объекта на стороне
// раннера держать не нужно.
func (w *World) givenVirtualAuthenticator(handle string) error {
	w.userHandle = handle
	auth := virtualwebauthn.NewAuthenticator()
	w.authenticator = &auth
	w.rp = virtualwebauthn.RelyingParty{
		Name:   "Passkey Demo",
		ID:     "localhost",
		Origin: "http://localhost",
	}
	return nil
}

// sendAttestation: парсит attestation-options из последнего ответа сервера
// (фаза 1 регистрации сохранила challenge), генерирует валидный attestation
// через virtualwebauthn и POST-ит на /v1/registrations/{id}/attestation.
//
// challengeID берётся из аргумента шага (возвращён сервером в фазе 1)
// и подставляется в URL.
func (w *World) sendAttestation(challengeID string) error {
	if w.lastBody == nil {
		return fmt.Errorf("нет options от фазы 1: сначала клиент должен получить challenge")
	}
	if w.authenticator == nil {
		return fmt.Errorf("у пользователя нет виртуального аутентификатора")
	}

	// В ответе фазы 1 ожидаем поле "options" с PublicKeyCredentialCreationOptions.
	var phase1 struct {
		ID      string          `json:"id"`
		Options json.RawMessage `json:"options"`
	}
	if err := json.Unmarshal(w.lastBody, &phase1); err != nil {
		return fmt.Errorf("ответ фазы 1 не парсится: %w", err)
	}

	options, err := virtualwebauthn.ParseAttestationOptions(string(phase1.Options))
	if err != nil {
		return fmt.Errorf("attestation options невалидны: %w", err)
	}

	cred := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)
	w.credential = &cred

	attResponse := virtualwebauthn.CreateAttestationResponse(w.rp, *w.authenticator, cred, *options)

	w.challengeID = challengeID
	path := fmt.Sprintf("/v1/registrations/%s/attestation", challengeID)
	return w.doRequest("POST", path, []byte(attResponse))
}

// sendAssertion: аналогично, но для фазы 2 входа. Использует уже
// существующий credential.
func (w *World) sendAssertion(challengeID string) error {
	if w.lastBody == nil {
		return fmt.Errorf("нет options от фазы 1: сначала клиент должен получить challenge")
	}
	if w.credential == nil {
		return fmt.Errorf("у пользователя нет credential — сначала пройдите регистрацию")
	}

	var phase1 struct {
		ID      string          `json:"id"`
		Options json.RawMessage `json:"options"`
	}
	if err := json.Unmarshal(w.lastBody, &phase1); err != nil {
		return fmt.Errorf("ответ фазы 1 не парсится: %w", err)
	}

	options, err := virtualwebauthn.ParseAssertionOptions(string(phase1.Options))
	if err != nil {
		return fmt.Errorf("assertion options невалидны: %w", err)
	}

	assResponse := virtualwebauthn.CreateAssertionResponse(w.rp, *w.authenticator, *w.credential, *options)

	w.challengeID = challengeID
	path := fmt.Sprintf("/v1/sessions/%s/assertion", challengeID)
	return w.doRequest("POST", path, []byte(assResponse))
}
