package service

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/kasq/backend/internal/storage"
)

func (s *Service) UploadBranding(ctx context.Context, kind string, data []byte, contentType, filename string) (string, error) {
	ext := storage.ObjectExt(filename, contentType)
	key := fmt.Sprintf("%s%s%s", storage.PrefixBranding, kind, ext)
	return key, s.storage.Upload(ctx, key, data, contentType)
}

func (s *Service) UploadAvatar(ctx context.Context, userID uuid.UUID, data []byte, contentType, filename string) (string, error) {
	ext := storage.ObjectExt(filename, contentType)
	key := fmt.Sprintf("%s%s%s", storage.PrefixAvatars, userID.String(), ext)
	return key, s.storage.Upload(ctx, key, data, contentType)
}

func (s *Service) DeleteAsset(ctx context.Context, storedKey string) error {
	if storedKey == "" {
		return nil
	}
	key := storage.ResolveObjectKey(storedKey)
	if err := s.storage.Delete(ctx, key); err != nil {
		return err
	}
	if key != storedKey {
		_ = s.storage.Delete(ctx, storedKey)
	}
	return nil
}

func (s *Service) AssetExists(ctx context.Context, storedKey string) (bool, error) {
	if storedKey == "" {
		return false, nil
	}
	key := storage.ResolveObjectKey(storedKey)
	exists, err := s.storage.Exists(ctx, key)
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}
	if key != storedKey {
		return s.storage.Exists(ctx, storedKey)
	}
	return false, nil
}

func (s *Service) OpenAsset(ctx context.Context, storedKey string) (io.ReadCloser, string, error) {
	key := storage.ResolveObjectKey(storedKey)
	reader, contentType, err := s.storage.GetObject(ctx, key)
	if err == nil {
		return reader, contentType, nil
	}
	if key != storedKey {
		return s.storage.GetObject(ctx, storedKey)
	}
	return nil, "", err
}

func (s *Service) GetAssetViewURL(ctx context.Context, storedKey string) (string, error) {
	if storedKey == "" {
		return "", fmt.Errorf("empty object key")
	}
	key := storage.ResolveObjectKey(storedKey)
	url, err := s.storage.PresignedViewURL(ctx, key)
	if err == nil {
		return url, nil
	}
	if key != storedKey {
		return s.storage.PresignedViewURL(ctx, storedKey)
	}
	return "", err
}
