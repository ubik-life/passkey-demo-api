package sessions_logout

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	s2 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_finish"
	s1 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_start"
)

type httpHandler struct {
	deps Deps
}

func newHTTPHandler(deps Deps) *httpHandler {
	return &httpHandler{deps: deps}
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing or malformed Authorization header")
		return
	}

	accessTokenRaw := strings.TrimPrefix(authHeader, "Bearer ")

	req := SessionLogoutRequest{AccessTokenRaw: accessTokenRaw}
	if err := ProcessSessionLogout(req, h.deps); err != nil {
		h.mapError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *httpHandler) mapError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrMissingBearer),
		errors.Is(err, s2.ErrAccessTokenInvalid):
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
	case errors.Is(err, s1.ErrDBLocked):
		w.Header().Set("Retry-After", "1")
		writeError(w, http.StatusServiceUnavailable, "db_locked", "database is locked, retry later")
	case errors.Is(err, s1.ErrDiskFull):
		writeError(w, http.StatusInsufficientStorage, "db_disk_full", "insufficient storage")
	default:
		h.deps.Logger.Error("internal error", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Code: code, Message: message})
}
