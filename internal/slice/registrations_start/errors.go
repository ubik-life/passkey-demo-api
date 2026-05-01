package registrations_start

import "errors"

var (
	ErrHandleEmpty    = errors.New("handle: empty")
	ErrHandleTooShort = errors.New("handle: too short (min 3)")
	ErrHandleTooLong  = errors.New("handle: too long (max 64)")

	ErrDBLocked = errors.New("db: locked")
	ErrDiskFull = errors.New("db: disk full")
)
