package sessions_logout

import "errors"

var ErrMissingBearer = errors.New("authorization: missing or empty Bearer token")
