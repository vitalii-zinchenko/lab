-- +goose Up
ALTER TABLE api_usage RENAME TO usage;
ALTER INDEX idx_api_usage_user_timestamp RENAME TO idx_usage_user_timestamp;

-- +goose Down
ALTER TABLE usage RENAME TO api_usage;
ALTER INDEX idx_usage_user_timestamp RENAME TO idx_api_usage_user_timestamp;
