package sessions_finish

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/mattn/go-sqlite3"
	s1 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_start"
	s2 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_finish"
	s3 "github.com/ubik-life/passkey-demo-api/internal/slice/sessions_start"
)

// Store — автономный I/O-объект слайса 4, инкапсулирующий *sql.DB.
type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// LoadLoginSession читает login-сессию по id.
func (s *Store) LoadLoginSession(id s3.LoginSessionID) (s3.LoginSession, error) {
	var (
		rowID        string
		rowUserID    string
		rowChallenge []byte
		rowExpiresAt int64
	)
	err := s.db.QueryRowContext(
		context.Background(),
		`SELECT id, user_id, challenge, expires_at FROM login_sessions WHERE id = ?`,
		id.String(),
	).Scan(&rowID, &rowUserID, &rowChallenge, &rowExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return s3.LoginSession{}, ErrLoginSessionNotFound
	}
	if err != nil {
		return s3.LoginSession{}, mapSQLiteErrRead(err)
	}
	session, err := s3.LoginSessionFromRow(rowID, rowUserID, rowChallenge, rowExpiresAt)
	if err != nil {
		return s3.LoginSession{}, fmt.Errorf("load login session: %w", err)
	}
	return session, nil
}

// LoadAssertionTarget загружает credential по credential_id и проверяет принадлежность пользователю.
func (s *Store) LoadAssertionTarget(input LoadAssertionTargetInput) (AssertionTarget, error) {
	var (
		rowCredentialID []byte
		rowUserID       string
		rowPublicKey    []byte
		rowSignCount    uint32
		rowTransports   string
		rowCreatedAt    int64
	)
	err := s.db.QueryRowContext(
		context.Background(),
		`SELECT credential_id, user_id, public_key, sign_count, transports, created_at
		 FROM credentials WHERE credential_id = ?`,
		input.CredentialID,
	).Scan(&rowCredentialID, &rowUserID, &rowPublicKey, &rowSignCount, &rowTransports, &rowCreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return AssertionTarget{}, ErrCredentialNotFound
	}
	if err != nil {
		return AssertionTarget{}, mapSQLiteErrRead(err)
	}

	if rowUserID != input.UserID.String() {
		return AssertionTarget{}, ErrCredentialNotFound
	}

	cred, err := s2.CredentialFromRow(rowCredentialID, rowUserID, rowPublicKey, rowSignCount, rowTransports, rowCreatedAt)
	if err != nil {
		return AssertionTarget{}, fmt.Errorf("load assertion target: %w", err)
	}

	var (
		rowUserRowID    string
		rowUserHandle   string
		rowUserCreatedAt int64
	)
	err = s.db.QueryRowContext(
		context.Background(),
		`SELECT id, handle, created_at FROM users WHERE id = ?`,
		rowUserID,
	).Scan(&rowUserRowID, &rowUserHandle, &rowUserCreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return AssertionTarget{}, ErrCredentialNotFound
	}
	if err != nil {
		return AssertionTarget{}, mapSQLiteErrRead(err)
	}

	user, err := s2.UserFromRow(rowUserRowID, rowUserHandle, rowUserCreatedAt)
	if err != nil {
		return AssertionTarget{}, fmt.Errorf("load assertion target: %w", err)
	}

	return AssertionTarget{user: user, credential: cred}, nil
}

// FinishLogin выполняет атомарную транзакцию: UPDATE sign_count + INSERT refresh_token + DELETE login_session.
func (s *Store) FinishLogin(input FinishLoginInput) error {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return mapSQLiteErrWrite(err)
	}
	defer tx.Rollback() //nolint:errcheck

	_, err = tx.ExecContext(context.Background(),
		`UPDATE credentials SET sign_count = ? WHERE credential_id = ?`,
		int64(input.NewSignCount),
		input.Credential.CredentialID(),
	)
	if err != nil {
		return mapSQLiteErrWrite(err)
	}

	_, err = tx.ExecContext(context.Background(),
		`INSERT INTO refresh_tokens (token_hash, user_id, expires_at) VALUES (?, ?, ?)`,
		input.RefreshTokenHash,
		input.Credential.UserID().String(),
		input.RefreshExpiresAt.Unix(),
	)
	if err != nil {
		return mapSQLiteErrWrite(err)
	}

	_, err = tx.ExecContext(context.Background(),
		`DELETE FROM login_sessions WHERE id = ?`,
		input.LoginSessionID.String(),
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
		}
	}
	return fmt.Errorf("finish: %w", err)
}
