package registrations_start

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
)

// RPConfig — конфигурация Relying Party для WebAuthn-options.
type RPConfig struct {
	Name   string
	ID     string
	Origin string // ожидаемый origin для clientDataJSON (S2+)
}

// RegistrationStartRequest — невалидированный вход из HTTP-адаптера.
type RegistrationStartRequest struct {
	Handle string `json:"handle"`
}

// CreationOptions — подмножество PublicKeyCredentialCreationOptions.
type CreationOptions struct {
	RP               RPInfo            `json:"rp"`
	User             UserInfo          `json:"user"`
	Challenge        string            `json:"challenge"`
	PubKeyCredParams []PubKeyCredParam `json:"pubKeyCredParams"`
	Timeout          int               `json:"timeout,omitempty"`
	Attestation      string            `json:"attestation"`
}

type RPInfo struct {
	Name string `json:"name"`
	ID   string `json:"id,omitempty"`
}

type UserInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

type PubKeyCredParam struct {
	Type string `json:"type"`
	Alg  int    `json:"alg"`
}

// RegistrationStartResponse — ответ POST /v1/registrations 201.
type RegistrationStartResponse struct {
	ID      string          `json:"id"`
	Options CreationOptions `json:"options"`
}

// RegistrationStartView — value-агрегатор для buildResponse.
type RegistrationStartView struct {
	Session RegistrationSession
	Options CreationOptions
}

func generateChallenge() (Challenge, error) {
	var c Challenge
	if _, err := rand.Read(c.bytes[:]); err != nil {
		return Challenge{}, fmt.Errorf("generate challenge: %w", err)
	}
	return c, nil
}

func generateRegistrationID() RegistrationID {
	return RegistrationID{value: uuid.New()}
}

func buildCreationOptions(s RegistrationSession, rp RPConfig) CreationOptions {
	return CreationOptions{
		RP:   RPInfo{Name: rp.Name, ID: rp.ID},
		User: UserInfo{
			ID:          base64.RawURLEncoding.EncodeToString(s.ID().Bytes()),
			Name:        s.Handle().Value(),
			DisplayName: s.Handle().Value(),
		},
		Challenge: s.Challenge().Base64URL(),
		PubKeyCredParams: []PubKeyCredParam{
			{Type: "public-key", Alg: -7},
			{Type: "public-key", Alg: -8},
		},
		Attestation: "none",
	}
}

func buildResponse(view RegistrationStartView) RegistrationStartResponse {
	return RegistrationStartResponse{
		ID:      view.Session.ID().String(),
		Options: view.Options,
	}
}
