-- +goose Up
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_file TEXT;

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS avatar_file;
