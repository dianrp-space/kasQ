-- +goose Up
ALTER TABLE integrations ADD COLUMN IF NOT EXISTS wa_allowed_phones TEXT;

-- +goose Down
ALTER TABLE integrations DROP COLUMN IF EXISTS wa_allowed_phones;
