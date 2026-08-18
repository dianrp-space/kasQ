package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/kasq/backend/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOStorage struct {
	client        *minio.Client
	presignClient *minio.Client
	bucket        string
}

func New(cfg config.MinIOConfig) (*MinIOStorage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:       cfg.UseSSL,
		BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}

	presignClient := client
	if cfg.PublicEndpoint != "" &&
		(cfg.PublicEndpoint != cfg.Endpoint || cfg.PublicUseSSL != cfg.UseSSL) {
		presignClient, err = minio.New(cfg.PublicEndpoint, &minio.Options{
			Creds:        credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
			Secure:       cfg.PublicUseSSL,
			BucketLookup: minio.BucketLookupPath,
		})
		if err != nil {
			return nil, fmt.Errorf("minio presign client: %w", err)
		}
		scheme := "http"
		if cfg.PublicUseSSL {
			scheme = "https"
		}
		log.Printf("minio: presigned URLs use public endpoint %s://%s", scheme, cfg.PublicEndpoint)
	}

	s := &MinIOStorage{client: client, presignClient: presignClient, bucket: cfg.Bucket}
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		// Remote MinIO may deny ListBucket — allow dev server to start anyway
		log.Printf("minio warning: cannot verify bucket %q: %v (upload nota may fail until fixed)", cfg.Bucket, err)
		return s, nil
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			log.Printf("minio warning: cannot create bucket %q: %v", cfg.Bucket, err)
		}
	}
	return s, nil
}

func (s *MinIOStorage) Upload(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func (s *MinIOStorage) PresignedViewURL(ctx context.Context, key string) (string, error) {
	u, err := s.presignClient.PresignedGetObject(ctx, s.bucket, key, 15*time.Minute, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (s *MinIOStorage) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

func IsAccessDenied(err error) bool {
	if err == nil {
		return false
	}
	errResp := minio.ToErrorResponse(err)
	return errResp.Code == "AccessDenied" || errResp.StatusCode == 403
}

func contentTypeFromKey(key string) string {
	switch strings.ToLower(filepath.Ext(key)) {
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	default:
		return "application/octet-stream"
	}
}

func (s *MinIOStorage) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err == nil {
		return true, nil
	}
	errResp := minio.ToErrorResponse(err)
	if errResp.Code == "NoSuchKey" || errResp.StatusCode == 404 {
		return false, nil
	}
	return false, err
}

func (s *MinIOStorage) GetObject(ctx context.Context, key string) (io.ReadCloser, string, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", err
	}
	contentType := contentTypeFromKey(key)
	info, statErr := obj.Stat()
	if statErr == nil && info.ContentType != "" {
		contentType = info.ContentType
	} else if statErr != nil && !IsAccessDenied(statErr) {
		_ = obj.Close()
		return nil, "", statErr
	}
	return obj, contentType, nil
}

func (s *MinIOStorage) PresignedDownloadURL(ctx context.Context, key string) (string, error) {
	reqParams := make(map[string][]string)
	filename := key
	if idx := strings.LastIndex(key, "/"); idx >= 0 && idx+1 < len(key) {
		filename = key[idx+1:]
	}
	reqParams["response-content-disposition"] = []string{fmt.Sprintf("attachment; filename=\"%s\"", filename)}
	u, err := s.presignClient.PresignedGetObject(ctx, s.bucket, key, 15*time.Minute, reqParams)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
