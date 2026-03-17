-- +goose Up
CREATE TYPE event_level AS ENUM ('error', 'warn', 'info');

CREATE TABLE IF NOT EXISTS event_history (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    level       event_level NOT NULL,
    event_type  TEXT        NOT NULL,
    details     TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS event_history;
DROP TYPE IF EXISTS event_level;
