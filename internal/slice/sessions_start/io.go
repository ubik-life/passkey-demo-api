package sessions_start

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/mattn/go-sqlite3"
	s1 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_start"
	s2 "github.com/ubik-life/passkey-demo-api/internal/slice/registrations_finish"
)

// Store — автономный I/O-объект слайса 3, инкапсулирующий *sql.DB.
// Головной модуль обращается к БД исключительно через методы.
type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) Store { return Store{db: db} }

// LoadUserCredentials загружает пользователя и его credentials из БД.
// Если нет user или credentials пуст — ErrUserNotFound.
// Порядок credentials: ORDER BY created_at ASC (детерминирован для тестов).
func (s Store) LoadUserCredentials(h s1.Handle) (UserWithCredentials, error) {
	var (
		rowUserID    string
		rowHandle    string
		rowCreatedAt int64
	)
	err := s.db.QueryRowContext(
		context.Background(),
		`SELECT id, handle, created_at FROM users WHERE handle = ?`,
		h.Value(),
	).Scan(&rowUserID, &rowHandle, &rowCreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return UserWithCredentials{}, ErrUserNotFound
	}
	if err != nil {
		return UserWithCredentials{}, mapSQLiteErrRead(err)
	}

	user, err := s2.UserFromRow(rowUserID, rowHandle, rowCreatedAt)
	if err != nil {
		return UserWithCredentials{}, fmt.Errorf("load user: %w", err)
	}

	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT credential_id, user_id, public_key, sign_count, transports, created_at
		 FROM credentials WHERE user_id = ? ORDER BY created_at ASC`,
		rowUserID,
	)
	if err != nil {
		return UserWithCredentials{}, mapSQLiteErrRead(err)
	}
	defer rows.Close()

	var creds []s2.Credential
	for rows.Next() {
		var (
			credentialID []byte
			userID       string
			publicKey    []byte
			signCount    uint32
			transports   string
			createdAt    int64
		)
		if err := rows.Scan(&credentialID, &userID, &publicKey, &signCount, &transports, &createdAt); err != nil {
			return UserWithCredentials{}, fmt.Errorf("scan credential: %w", err)
		}
		cred, err := s2.CredentialFromRow(credentialID, userID, publicKey, signCount, transports, createdAt)
		if err != nil {
			return UserWithCredentials{}, fmt.Errorf("credential from row: %w", err)
		}
		creds = append(creds, cred)
	}
	if err := rows.Err(); err != nil {
		return UserWithCredentials{}, mapSQLiteErrRead(err)
	}

	if len(creds) == 0 {
		return UserWithCredentials{}, ErrUserNotFound
	}

	return UserWithCredentials{user: user, credentials: creds}, nil
}

// PersistLoginSession записывает новую сессию входа в БД.
func (s Store) PersistLoginSession(ls LoginSession) error {
	challenge := ls.Challenge().Bytes()
	_, err := s.db.ExecContext(
		context.Background(),
		`INSERT INTO login_sessions (id, user_id, challenge, expires_at) VALUES (?, ?, ?, ?)`,
		ls.ID().String(),
		ls.UserID().String(),
		challenge[:],
		ls.ExpiresAt().Unix(),
	)
	if err != nil {
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
			return fmt.Errorf("persist: %w", s1.ErrDBLocked)
		case sqlite3.ErrFull:
			return fmt.Errorf("persist: %w", s1.ErrDiskFull)
		}
	}
	return fmt.Errorf("persist: %w", err)
}
