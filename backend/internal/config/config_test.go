package config

import "testing"

func TestLoadMinIOPublicEndpoint(t *testing.T) {
	t.Setenv("MINIO_ENDPOINT", "127.0.0.1:9000")
	t.Setenv("MINIO_USE_SSL", "false")
	t.Setenv("MINIO_PUBLIC_ENDPOINT", "https://s3.example.com")
	t.Setenv("MINIO_PUBLIC_USE_SSL", "")

	cfg := Load()
	if cfg.MinIO.Endpoint != "127.0.0.1:9000" {
		t.Fatalf("endpoint = %q", cfg.MinIO.Endpoint)
	}
	if cfg.MinIO.UseSSL {
		t.Fatal("expected internal UseSSL false")
	}
	if cfg.MinIO.PublicEndpoint != "s3.example.com" {
		t.Fatalf("public endpoint = %q", cfg.MinIO.PublicEndpoint)
	}
	if !cfg.MinIO.PublicUseSSL {
		t.Fatal("expected public UseSSL true")
	}
}

func TestLoadMinIOPublicEndpointDefaultsToInternal(t *testing.T) {
	t.Setenv("MINIO_ENDPOINT", "https://s3.example.com")
	t.Setenv("MINIO_USE_SSL", "false")
	t.Setenv("MINIO_PUBLIC_ENDPOINT", "")

	cfg := Load()
	if cfg.MinIO.Endpoint != "s3.example.com" {
		t.Fatalf("endpoint = %q", cfg.MinIO.Endpoint)
	}
	if cfg.MinIO.PublicEndpoint != "s3.example.com" {
		t.Fatalf("public endpoint = %q", cfg.MinIO.PublicEndpoint)
	}
	if !cfg.MinIO.UseSSL || !cfg.MinIO.PublicUseSSL {
		t.Fatal("expected SSL true from https endpoint")
	}
}

func TestLoadTelegramBotToken(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", " 123:abc ")
	cfg := Load()
	if cfg.TelegramBotToken != "123:abc" {
		t.Fatalf("TelegramBotToken = %q", cfg.TelegramBotToken)
	}
}
