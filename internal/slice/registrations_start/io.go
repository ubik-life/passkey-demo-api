package registrations_start

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/mattn/go-sqlite3"
)

// Store — автономный модуль работы с БД. Инкапсулирует *sql.DB;
// головной модуль знает только методы, не зависимости.
type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) Store { return Store{db: db} }

func (s Store) PersistRegistrationSession(session RegistrationSession) error {
	challenge := session.Challenge().Bytes()
	_, err := s.db.ExecContext(
		context.Background(),
		`INSERT INTO registration_sessions (id, handle, challenge, expires_at) VALUES (?, ?, ?, ?)`,
		session.ID().String(),
		session.Handle().Value(),
		challenge[:],
		session.ExpiresAt().Unix(),
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
