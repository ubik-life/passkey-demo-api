-- +goose Up
CREATE TABLE users (
    id          TEXT    PRIMARY KEY,        -- UUID v4 string
    handle      TEXT    NOT NULL UNIQUE,    -- UNIQUE: SQLITE_CONSTRAINT_UNIQUE → ErrHandleTaken
    created_at  INTEGER NOT NULL            -- unix seconds
);
CREATE UNIQUE INDEX idx_users_handle ON users(handle);

-- +goose Down
DROP INDEX IF EXISTS idx_users_handle;
DROP TABLE IF EXISTS users;
