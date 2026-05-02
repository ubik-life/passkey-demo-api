package sessions_finish

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/go-webauthn/webauthn/protocol"
	s1 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_start"
)

// ParsedAssertion — обёртка над *protocol.ParsedCredentialAssertionData.
type ParsedAssertion struct {
	parsed *protocol.ParsedCredentialAssertionData
}

// CredentialID возвращает raw credential ID из распарсенного assertion.
func (p ParsedAssertion) CredentialID() []byte {
	return p.parsed.RawID
}

// parseAssertion парсит JSON-тело AssertionRequest.
// Не верифицирует — только синтаксис и базовая структура.
func parseAssertion(raw []byte) (ParsedAssertion, error) {
	parsed, err := protocol.ParseCredentialRequestResponseBody(strings.NewReader(string(raw)))
	if err != nil {
		return ParsedAssertion{}, fmt.Errorf("%w: %v", ErrAssertionParse, err)
	}
	return ParsedAssertion{parsed: parsed}, nil
}

// verifyAssertion верифицирует assertion против challenge свежей сессии и публичного ключа.
func verifyAssertion(input AssertionVerification, rp s1.RPConfig) (VerifiedAssertion, error) {
	challengeBytes := input.Fresh.Challenge().Bytes()
	challengeB64 := base64.RawURLEncoding.EncodeToString(challengeBytes[:])

	err := input.Parsed.parsed.Verify(
		challengeB64,
		rp.ID,
		"",
		[]string{rp.Origin},
		nil,
		protocol.TopOriginAutoVerificationMode,
		false,
		false,
		true,
		input.Target.Credential().PublicKey(),
	)
	if err != nil {
		return VerifiedAssertion{}, fmt.Errorf("%w: %v", ErrAssertionInvalid, err)
	}

	newSignCount := input.Parsed.parsed.Response.AuthenticatorData.Counter

	return VerifiedAssertion{newSignCount: newSignCount}, nil
}
