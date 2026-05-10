package registrations_finish

import (
	"crypto/ed25519"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// VerifyAccessTokenInput — входные данные для верификации access JWT.
type VerifyAccessTokenInput struct {
	AccessTokenRaw string
	PublicKey      ed25519.PublicKey
	ExpectedIssuer string
	Now            time.Time
}

// AuthenticatedUserID — UserID из верифицированного access JWT.
type AuthenticatedUserID struct {
	userID UserID
}

func (a AuthenticatedUserID) UserID() UserID { return a.userID }

// VerifyAccessToken верифицирует access JWT: подпись (Ed25519), срок, issuer, subject.
// Все под-причины отказа маппятся в ErrAccessTokenInvalid.
func VerifyAccessToken(input VerifyAccessTokenInput) (AuthenticatedUserID, error) {
	keyFunc := func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return input.PublicKey, nil
	}

	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(input.AccessTokenRaw, claims, keyFunc,
		jwt.WithIssuedAt(),
		jwt.WithLeeway(0),
	)
	if err != nil || !token.Valid {
		return AuthenticatedUserID{}, fmt.Errorf("%w: %v", ErrAccessTokenInvalid, err)
	}

	if claims.Issuer != input.ExpectedIssuer {
		return AuthenticatedUserID{}, fmt.Errorf("%w: issuer mismatch", ErrAccessTokenInvalid)
	}

	if claims.ExpiresAt == nil || !input.Now.Before(claims.ExpiresAt.Time) {
		return AuthenticatedUserID{}, fmt.Errorf("%w: expired", ErrAccessTokenInvalid)
	}

	userID, err := UserIDFromString(claims.Subject)
	if err != nil {
		return AuthenticatedUserID{}, fmt.Errorf("%w: subject not uuid", ErrAccessTokenInvalid)
	}

	return AuthenticatedUserID{userID: userID}, nil
}
