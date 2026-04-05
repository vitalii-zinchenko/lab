-- +goose Up
CREATE TABLE IF NOT EXISTS ch_events (
    id         UUID DEFAULT generateUUIDv4(),
    level      LowCardinality(String) NOT NULL,
    event_type String                 NOT NULL,
    details    Nullable(String),
    created_at DateTime               DEFAULT now()
) ENGINE = MergeTree()
ORDER BY (created_at, id);

-- +goose Down
DROP TABLE IF EXISTS ch_events;
