-- +goose Up
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS sort_order INTEGER NOT NULL DEFAULT 0;

UPDATE transactions t
SET sort_order = sub.rn
FROM (
    SELECT id,
           (ROW_NUMBER() OVER (PARTITION BY team_id, tanggal ORDER BY created_at ASC) - 1)::INTEGER AS rn
    FROM transactions
) sub
WHERE t.id = sub.id;

CREATE INDEX IF NOT EXISTS idx_transactions_team_tanggal_sort
    ON transactions (team_id, tanggal DESC, sort_order ASC);

-- +goose Down
DROP INDEX IF EXISTS idx_transactions_team_tanggal_sort;
ALTER TABLE transactions DROP COLUMN IF EXISTS sort_order;
