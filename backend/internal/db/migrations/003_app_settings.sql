-- +goose Up
CREATE TABLE app_settings (
    id INT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    app_name TEXT NOT NULL DEFAULT 'KasQ',
    app_tagline TEXT NOT NULL DEFAULT 'Kas Ku — Pencatatan Keuangan Tim',
    logo_file TEXT,
    favicon_file TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO app_settings (id) VALUES (1);

-- +goose Down
DROP TABLE IF EXISTS app_settings;
