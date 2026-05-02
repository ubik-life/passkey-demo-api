package sessions_start

import (
	"encoding/base64"

	s1 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_start"
	s2 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_finish"
)

// SessionStartRequest — невалидированный вход из HTTP-адаптера.
type SessionStartRequest struct {
	Handle string `json:"handle"`
}

// RequestOptions — подмножество PublicKeyCredentialRequestOptions (WebAuthn Level 2).
type RequestOptions struct {
	Challenge        string                      `json:"challenge"`
	RpID             string                      `json:"rpId,omitempty"`
	AllowCredentials []AllowCredentialDescriptor `json:"allowCredentials,omitempty"`
	UserVerification string                      `json:"userVerification,omitempty"`
	Timeout          int                         `json:"timeout,omitempty"`
}

// AllowCredentialDescriptor — элемент allowCredentials.
type AllowCredentialDescriptor struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// SessionStartResponse — ответ POST /v1/sessions 201.
type SessionStartResponse struct {
	ID      string         `json:"id"`
	Options RequestOptions `json:"options"`
}

// BuildRequestOptionsInput — агрегатор для buildRequestOptions («один data-аргумент»).
type BuildRequestOptionsInput struct {
	Session     LoginSession
	Credentials []s2.Credential
}

// SessionStartView — агрегатор для buildResponse.
type SessionStartView struct {
	Session LoginSession
	Options RequestOptions
}

func buildRequestOptions(input BuildRequestOptionsInput, rp s1.RPConfig) RequestOptions {
	allow := make([]AllowCredentialDescriptor, len(input.Credentials))
	for i, c := range input.Credentials {
		allow[i] = AllowCredentialDescriptor{
			Type: "public-key",
			ID:   base64.RawURLEncoding.EncodeToString(c.CredentialID()),
		}
	}
	challengeBytes := input.Session.Challenge().Bytes()
	return RequestOptions{
		Challenge:        base64.RawURLEncoding.EncodeToString(challengeBytes[:]),
		RpID:             rp.ID,
		AllowCredentials: allow,
		UserVerification: "preferred",
	}
}

func buildResponse(view SessionStartView) SessionStartResponse {
	return SessionStartResponse{
		ID:      view.Session.ID().String(),
		Options: view.Options,
	}
}
