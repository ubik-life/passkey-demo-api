-- +goose Up
CREATE TABLE login_sessions (
    id          TEXT    PRIMARY KEY,
    user_id     TEXT    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    challenge   BLOB    NOT NULL,
    expires_at  INTEGER NOT NULL
);
CREATE INDEX idx_login_sessions_user_id    ON login_sessions(user_id);
CREATE INDEX idx_login_sessions_expires_at ON login_sessions(expires_at);

-- +goose Down
DROP INDEX IF EXISTS idx_login_sessions_expires_at;
DROP INDEX IF EXISTS idx_login_sessions_user_id;
DROP TABLE IF EXISTS login_sessions;
