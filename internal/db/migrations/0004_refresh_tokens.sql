-- +goose Up
CREATE TABLE refresh_tokens (
    token_hash  TEXT    PRIMARY KEY,        -- hex(sha256(plaintext))
    user_id     TEXT    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at  INTEGER NOT NULL,
    revoked_at  INTEGER NULL               -- заполняется в S5 при logout
);
CREATE INDEX idx_refresh_tokens_user_id    ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);

-- +goose Down
DROP INDEX IF EXISTS idx_refresh_tokens_expires_at;
DROP INDEX IF EXISTS idx_refresh_tokens_user_id;
DROP TABLE IF EXISTS refresh_tokens;
