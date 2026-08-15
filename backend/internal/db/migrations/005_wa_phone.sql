-- +goose Up
ALTER TABLE integrations ADD COLUMN IF NOT EXISTS wa_phone TEXT;

-- +goose Down
ALTER TABLE integrations DROP COLUMN IF EXISTS wa_phone;
