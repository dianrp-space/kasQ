-- +goose Up
ALTER TABLE integrations
    ADD COLUMN IF NOT EXISTS tele_use_system_bot BOOLEAN NOT NULL DEFAULT FALSE;

CREATE UNIQUE INDEX IF NOT EXISTS idx_integrations_system_tele_chat
    ON integrations (tele_allowed_chat_id)
    WHERE tele_enabled = TRUE
      AND tele_use_system_bot = TRUE
      AND tele_allowed_chat_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_integrations_system_tele_chat;
ALTER TABLE integrations DROP COLUMN IF EXISTS tele_use_system_bot;
