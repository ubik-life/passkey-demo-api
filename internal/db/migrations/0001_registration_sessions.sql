-- +goose Up
CREATE TABLE registration_sessions (
    id         TEXT    PRIMARY KEY,
    handle     TEXT    NOT NULL,
    challenge  BLOB    NOT NULL,
    expires_at INTEGER NOT NULL
);

CREATE INDEX idx_registration_sessions_expires_at ON registration_sessions(expires_at);

-- +goose Down
DROP INDEX IF EXISTS idx_registration_sessions_expires_at;
DROP TABLE IF EXISTS registration_sessions;
