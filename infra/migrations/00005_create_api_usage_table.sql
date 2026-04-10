-- +goose Up
CREATE TABLE IF NOT EXISTS api_usage (
    id        BIGSERIAL   PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id   BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    operation TEXT        NOT NULL
);

CREATE INDEX idx_api_usage_user_timestamp ON api_usage (user_id, timestamp DESC);

-- +goose Down
DROP TABLE IF EXISTS api_usage;
