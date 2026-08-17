-- +goose Up
-- Set laporan publik ke slug tim/kas untuk token acak (UUID) yang masih dipakai.
UPDATE report_tokens rt
SET token = t.slug
FROM teams t
WHERE rt.team_id = t.id
  AND rt.token ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$';

-- +goose Down
-- Tidak bisa restore token UUID lama.
