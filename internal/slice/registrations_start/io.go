package registrations_start

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/mattn/go-sqlite3"
)

func persistRegistrationSession(db *sql.DB, s RegistrationSession) error {
	challenge := s.Challenge().Bytes()
	_, err := db.ExecContext(
		context.Background(),
		`INSERT INTO registration_sessions (id, handle, challenge, expires_at) VALUES (?, ?, ?, ?)`,
		s.ID().String(),
		s.Handle().Value(),
		challenge[:],
		s.ExpiresAt().Unix(),
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
			return fmt.Errorf("persist: %w", ErrDBLocked)
		case sqlite3.ErrFull:
			return fmt.Errorf("persist: %w", ErrDiskFull)
		}
	}
	return fmt.Errorf("persist: %w", err)
}
