package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/kasq/backend/internal/api"
	"github.com/kasq/backend/internal/bot/telegram"
	"github.com/kasq/backend/internal/bot/whatsapp"
	"github.com/kasq/backend/internal/config"
	"github.com/kasq/backend/internal/db"
	"github.com/kasq/backend/internal/email"
	"github.com/kasq/backend/internal/repository"
	"github.com/kasq/backend/internal/service"
	"github.com/kasq/backend/internal/storage"
)

func main() {
	_ = godotenv.Load() // load backend/.env when running from backend/
	cfg := config.Load()
	ctx := context.Background()

	if err := db.RunMigrations(cfg.DatabaseURL); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer pool.Close()

	repo := repository.New(pool)
	if err := db.Seed(ctx, repo); err != nil {
		log.Printf("seed warning: %v", err)
	}

	store, err := storage.New(cfg.MinIO)
	if err != nil {
		log.Fatalf("minio failed: %v", err)
	}

	svc := service.New(repo, store)
	mailer := email.NewSender(cfg.SMTP)
	authSvc := service.NewAuthService(repo, mailer, cfg.AppURL)

	waDataDir := filepath.Join("data", "wa-sessions")
	if err := os.MkdirAll(waDataDir, 0755); err != nil {
		log.Fatalf("wa dir: %v", err)
	}

	waManager := whatsapp.NewManager(repo, svc, waDataDir, cfg.AppURL)
	teleManager := telegram.NewManager(repo, svc, cfg.TelegramBotToken, cfg.AppURL)

	go waManager.StartAll(ctx)
	go teleManager.StartAll(ctx)

	handler := api.NewHandler(repo, svc, authSvc, cfg.JWTSecret, cfg.AppURL, waManager, teleManager)

	r := gin.Default()
	handler.RegisterRoutes(r)

	log.Printf("KasQ server running on :%s (MinIO bucket: %s)", cfg.Port, cfg.MinIO.Bucket)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
