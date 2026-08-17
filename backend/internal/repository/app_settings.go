package repository

import (
	"context"
	"fmt"

	"github.com/kasq/backend/internal/models"
)

func (r *Repository) GetAppSettings(ctx context.Context) (*models.AppSettings, error) {
	var s models.AppSettings
	var logoFile, faviconFile *string
	err := r.pool.QueryRow(ctx, `
		SELECT app_name, app_tagline, logo_file, favicon_file, updated_at
		FROM app_settings WHERE id = 1`,
	).Scan(&s.AppName, &s.AppTagline, &logoFile, &faviconFile, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get app settings: %w", err)
	}
	s.LogoFile = logoFile
	s.FaviconFile = faviconFile
	return &s, nil
}

func (r *Repository) UpdateAppSettings(ctx context.Context, appName, appTagline string, logoFile, faviconFile *string) (*models.AppSettings, error) {
	_, err := r.pool.Exec(ctx, `
		UPDATE app_settings
		SET app_name = $1, app_tagline = $2, logo_file = $3, favicon_file = $4, updated_at = NOW()
		WHERE id = 1`,
		appName, appTagline, logoFile, faviconFile,
	)
	if err != nil {
		return nil, fmt.Errorf("update app settings: %w", err)
	}
	return r.GetAppSettings(ctx)
}
