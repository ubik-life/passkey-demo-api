-- +goose Up
CREATE TABLE credentials (
    credential_id  BLOB    PRIMARY KEY,
    user_id        TEXT    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    public_key     BLOB    NOT NULL,
    sign_count     INTEGER NOT NULL,
    transports     TEXT    NOT NULL DEFAULT '',
    created_at     INTEGER NOT NULL
);
CREATE INDEX idx_credentials_user_id ON credentials(user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_credentials_user_id;
DROP TABLE IF EXISTS credentials;
