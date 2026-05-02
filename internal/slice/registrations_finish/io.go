package registrations_finish

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/mattn/go-sqlite3"
	s1 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_start"
)

// Store — автономный модуль работы с БД. Инкапсулирует *sql.DB;
// головной модуль знает только методы, не зависимости.
type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) Store { return Store{db: db} }

func (s Store) LoadSession(id s1.RegistrationID) (s1.RegistrationSession, error) {
	var (
		rowID        string
		rowHandle    string
		rowChallenge []byte
		rowExpiresAt int64
	)
	err := s.db.QueryRowContext(
		context.Background(),
		`SELECT id, handle, challenge, expires_at FROM registration_sessions WHERE id = ?`,
		id.String(),
	).Scan(&rowID, &rowHandle, &rowChallenge, &rowExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return s1.RegistrationSession{}, ErrSessionNotFound
	}
	if err != nil {
		return s1.RegistrationSession{}, mapSQLiteErrRead(err)
	}
	session, err := s1.RegistrationSessionFromRow(rowID, rowHandle, rowChallenge, rowExpiresAt)
	if err != nil {
		return s1.RegistrationSession{}, fmt.Errorf("load session: %w", err)
	}
	return session, nil
}

func (s Store) Finish(input FinishRegistrationInput) error {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return mapSQLiteErrWrite(err)
	}
	defer tx.Rollback() //nolint:errcheck

	_, err = tx.ExecContext(context.Background(),
		`INSERT INTO users (id, handle, created_at) VALUES (?, ?, ?)`,
		input.User.ID().String(),
		input.User.Handle().Value(),
		input.User.CreatedAt().Unix(),
	)
	if err != nil {
		return mapSQLiteErrWrite(err)
	}

	_, err = tx.ExecContext(context.Background(),
		`INSERT INTO credentials (credential_id, user_id, public_key, sign_count, transports, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		input.Credential.CredentialID(),
		input.Credential.UserID().String(),
		input.Credential.PublicKey(),
		int64(input.Credential.SignCount()),
		strings.Join(input.Credential.Transports(), ","),
		input.User.CreatedAt().Unix(),
	)
	if err != nil {
		return mapSQLiteErrWrite(err)
	}

	_, err = tx.ExecContext(context.Background(),
		`INSERT INTO refresh_tokens (token_hash, user_id, expires_at) VALUES (?, ?, ?)`,
		input.RefreshTokenHash,
		input.User.ID().String(),
		input.RefreshExpiresAt.Unix(),
	)
	if err != nil {
		return mapSQLiteErrWrite(err)
	}

	_, err = tx.ExecContext(context.Background(),
		`DELETE FROM registration_sessions WHERE id = ?`,
		input.RegistrationID.String(),
	)
	if err != nil {
		return mapSQLiteErrWrite(err)
	}

	if err := tx.Commit(); err != nil {
		return mapSQLiteErrWrite(err)
	}
	return nil
}

func mapSQLiteErrRead(err error) error {
	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) {
		if sqliteErr.Code == sqlite3.ErrBusy {
			return fmt.Errorf("load: %w", s1.ErrDBLocked)
		}
	}
	return fmt.Errorf("load: %w", err)
}

func mapSQLiteErrWrite(err error) error {
	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) {
		switch sqliteErr.Code {
		case sqlite3.ErrBusy:
			return fmt.Errorf("finish: %w", s1.ErrDBLocked)
		case sqlite3.ErrFull:
			return fmt.Errorf("finish: %w", s1.ErrDiskFull)
		case sqlite3.ErrConstraint:
			if sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
				return fmt.Errorf("finish: %w", ErrHandleTaken)
			}
		}
	}
	return fmt.Errorf("finish: %w", err)
}
