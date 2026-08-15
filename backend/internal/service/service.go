package service

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kasq/backend/internal/bot"
	"github.com/kasq/backend/internal/models"
	"github.com/kasq/backend/internal/repository"
	"github.com/kasq/backend/internal/storage"
)

type Service struct {
	repo    *repository.Repository
	storage *storage.MinIOStorage
}

func New(repo *repository.Repository, store *storage.MinIOStorage) *Service {
	return &Service{repo: repo, storage: store}
}

func (s *Service) CreateTransactionFromWeb(ctx context.Context, teamID uuid.UUID, userID uuid.UUID, input CreateWebTxInput) (*models.Transaction, *models.Balance, error) {
	txInput := models.CreateTransactionInput{
		Hari:       input.Hari,
		Tanggal:    input.Tanggal,
		Jenis:      input.Jenis,
		Deskripsi:  input.Deskripsi,
		Total:      input.Total,
		NotaKey:    input.NotaKey,
		Keterangan: input.Keterangan,
		Source:     models.SourceWeb,
		CreatedBy:  &userID,
	}
	tx, err := s.repo.CreateTransaction(ctx, teamID, txInput)
	if err != nil {
		return nil, nil, err
	}
	balance, err := s.repo.GetBalance(ctx, teamID, nil, nil)
	return tx, balance, err
}

func (s *Service) CreateTransactionFromBot(ctx context.Context, teamID uuid.UUID, parsed bot.ParsedMessage, notaKey *string) (*models.Transaction, *models.Balance, error) {
	txInput := models.CreateTransactionInput{
		Hari:       parsed.Hari,
		Tanggal:    parsed.Tanggal,
		Jenis:      parsed.Jenis,
		Deskripsi:  parsed.Deskripsi,
		Total:      parsed.Total,
		NotaKey:    notaKey,
		Keterangan: parsed.Keterangan,
		Source:     parsed.Source,
	}
	tx, err := s.repo.CreateTransaction(ctx, teamID, txInput)
	if err != nil {
		return nil, nil, err
	}
	balance, err := s.repo.GetBalance(ctx, teamID, nil, nil)
	return tx, balance, err
}

func (s *Service) UploadNota(ctx context.Context, teamID uuid.UUID, filename string, data []byte, contentType string) (string, error) {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	ext := notaObjectExt(filename, contentType)
	key := fmt.Sprintf("%s%s/%s/%s%s", storage.PrefixNota, teamID.String(), time.Now().Format("2006/01"), uuid.New().String(), ext)
	return key, s.storage.Upload(ctx, key, data, contentType)
}

func notaObjectExt(filename, contentType string) string {
	name := strings.ReplaceAll(filename, "..", "")
	name = strings.ReplaceAll(name, "/", "-")
	if ext := filepath.Ext(name); ext != "" {
		return ext
	}
	switch strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0])) {
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".jpg"
	}
}

func (s *Service) GetNotaURL(ctx context.Context, key string, download bool) (string, error) {
	if download {
		return s.storage.PresignedDownloadURL(ctx, key)
	}
	return s.storage.PresignedViewURL(ctx, key)
}

func (s *Service) UpdateTransaction(ctx context.Context, teamID, txID uuid.UUID, input UpdateWebTxInput) (*models.Transaction, *models.Balance, error) {
	existing, err := s.repo.GetTransaction(ctx, teamID, txID)
	if err != nil {
		return nil, nil, err
	}

	notaKey := existing.NotaKey
	var oldNotaKey *string

	if input.RemoveNota && existing.NotaKey != nil {
		oldNotaKey = existing.NotaKey
		notaKey = nil
	}
	if input.NotaReplace != nil {
		if existing.NotaKey != nil {
			oldNotaKey = existing.NotaKey
		}
		notaKey = input.NotaReplace
	}

	tx, err := s.repo.UpdateTransaction(ctx, teamID, txID, models.UpdateTransactionInput{
		Hari:       input.Hari,
		Tanggal:    input.Tanggal,
		Jenis:      input.Jenis,
		Deskripsi:  input.Deskripsi,
		Total:      input.Total,
		NotaKey:    notaKey,
		Keterangan: input.Keterangan,
	})
	if err != nil {
		return nil, nil, err
	}
	if oldNotaKey != nil && *oldNotaKey != "" {
		_ = s.storage.Delete(ctx, *oldNotaKey)
	}
	balance, err := s.repo.GetBalance(ctx, teamID, nil, nil)
	return tx, balance, err
}

func (s *Service) DeleteTransaction(ctx context.Context, teamID, txID uuid.UUID) (*models.Balance, error) {
	tx, err := s.repo.GetTransaction(ctx, teamID, txID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.DeleteTransaction(ctx, teamID, txID); err != nil {
		return nil, err
	}
	if tx.NotaKey != nil && *tx.NotaKey != "" {
		_ = s.storage.Delete(ctx, *tx.NotaKey)
	}
	return s.repo.GetBalance(ctx, teamID, nil, nil)
}

type CreateWebTxInput struct {
	Hari       string
	Tanggal    time.Time
	Jenis      models.TxJenis
	Deskripsi  string
	Total      int64
	NotaKey    *string
	Keterangan *string
}

type UpdateWebTxInput struct {
	Hari        string
	Tanggal     time.Time
	Jenis       models.TxJenis
	Deskripsi   string
	Total       int64
	Keterangan  *string
	NotaReplace *string
	RemoveNota  bool
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)
var reportSlugRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func Slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = slugRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func NormalizeReportSlug(input string) string {
	return Slugify(input)
}

func ValidateReportSlug(slug string) error {
	if len(slug) < 2 || len(slug) > 64 {
		return fmt.Errorf("slug harus 2–64 karakter")
	}
	if !reportSlugRe.MatchString(slug) {
		return fmt.Errorf("slug hanya huruf kecil, angka, dan strip")
	}
	return nil
}
