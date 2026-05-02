package sessions_finish

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	s1 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_start"
)

type httpHandler struct {
	deps Deps
}

func newHTTPHandler(deps Deps) *httpHandler {
	return &httpHandler{deps: deps}
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "cannot read body")
		return
	}

	req := SessionFinishRequest{
		LoginSessionIDRaw: id,
		AssertionBody:     body,
	}

	resp, err := ProcessSessionFinish(req, h.deps)
	if err != nil {
		h.mapError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *httpHandler) mapError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidLoginSessionID),
		errors.Is(err, ErrAssertionParse):
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, ErrLoginSessionNotFound),
		errors.Is(err, ErrLoginSessionExpired),
		errors.Is(err, ErrCredentialNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, ErrAssertionInvalid):
		writeError(w, http.StatusUnprocessableEntity, "ASSERTION_INVALID", err.Error())
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
