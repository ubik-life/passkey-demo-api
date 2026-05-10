package sessions_logout

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/mattn/go-sqlite3"
	s2 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_finish"
	s1 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_start"
)

// RevokeUserSessionsInput — входные данные для отзыва всех refresh-токенов пользователя.
type RevokeUserSessionsInput struct {
	UserID s2.UserID
	Now    time.Time
}

// Store — автономный I/O-объект слайса 5.
type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// RevokeUserSessions отзывает все активные refresh-токены пользователя.
// Возвращает nil при 0 затронутых строках (идемпотентно).
func (s *Store) RevokeUserSessions(input RevokeUserSessionsInput) error {
	_, err := s.db.ExecContext(
		context.Background(),
		`UPDATE refresh_tokens SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`,
		input.Now.Unix(),
		input.UserID.String(),
	)
	if err != nil {
		return mapSQLiteErr(err)
	}
	return nil
}

func mapSQLiteErr(err error) error {
	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) {
		switch sqliteErr.Code {
		case sqlite3.ErrBusy:
			return fmt.Errorf("revoke: %w", s1.ErrDBLocked)
		case sqlite3.ErrFull:
			return fmt.Errorf("revoke: %w", s1.ErrDiskFull)
		}
	}
	return fmt.Errorf("revoke: %w", err)
}
