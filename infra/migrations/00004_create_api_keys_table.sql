-- +goose Up
CREATE TABLE IF NOT EXISTS api_keys (
    id                 UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            INTEGER      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client_id          UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    client_secret_hash VARCHAR(255) NOT NULL,
    name               VARCHAR(255),
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    expires_at         TIMESTAMPTZ,
    revoked_at         TIMESTAMPTZ,
    last_used_at       TIMESTAMPTZ
);

CREATE INDEX idx_api_keys_client_id ON api_keys(client_id);
CREATE INDEX idx_api_keys_user_id   ON api_keys(user_id);

-- +goose Down
DROP TABLE IF EXISTS api_keys;
