package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
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
		NotaKey:    models.JoinNotaKeys(input.NotaKeys),
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

func (s *Service) CreateTransactionFromBot(ctx context.Context, teamID uuid.UUID, parsed bot.ParsedMessage, notaKeys []string) (*models.Transaction, *models.Balance, error) {
	txInput := models.CreateTransactionInput{
		Hari:       parsed.Hari,
		Tanggal:    parsed.Tanggal,
		Jenis:      parsed.Jenis,
		Deskripsi:  parsed.Deskripsi,
		Total:      parsed.Total,
		NotaKey:    models.JoinNotaKeys(notaKeys),
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
	prepared, err := prepareNotaUpload(filename, contentType, data)
	if err != nil {
		return "", err
	}
	if prepared.contentType == "" {
		prepared.contentType = "application/octet-stream"
	}
	ext := notaObjectExt(prepared.filename, prepared.contentType)
	key := fmt.Sprintf("%s%s/%s/%s%s", storage.PrefixNota, teamID.String(), time.Now().Format("2006/01"), uuid.New().String(), ext)
	return key, s.storage.Upload(ctx, key, prepared.data, prepared.contentType)
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

func (s *Service) GetNotaURL(ctx context.Context, stored string, download bool) (string, error) {
	key := strings.TrimSpace(stored)
	if key == "" {
		return "", fmt.Errorf("empty nota key")
	}
	if download {
		return s.storage.PresignedDownloadURL(ctx, key)
	}
	return s.storage.PresignedViewURL(ctx, key)
}

func notaObjectKeys(stored string) []string {
	stored = strings.TrimSpace(stored)
	if stored == "" {
		return nil
	}
	keys := []string{stored}
	if strings.HasPrefix(stored, storage.PrefixNota) {
		if legacy := strings.TrimPrefix(stored, storage.PrefixNota); legacy != stored {
			keys = append(keys, legacy)
		}
	} else {
		keys = append(keys, storage.PrefixNota+stored)
	}
	return keys
}

func (s *Service) OpenNota(ctx context.Context, storedKey string) (io.ReadCloser, string, error) {
	var lastErr error
	for _, key := range notaObjectKeys(storedKey) {
		reader, contentType, err := s.storage.GetObject(ctx, key)
		if err == nil {
			return reader, contentType, nil
		}
		lastErr = err
		if storage.IsAccessDenied(err) {
			log.Printf("nota: get %q: access denied", key)
		}
	}
	if storage.IsAccessDenied(lastErr) {
		return nil, "", fmt.Errorf("minio access denied")
	}
	if lastErr != nil {
		return nil, "", lastErr
	}
	return nil, "", fmt.Errorf("nota not found")
}

func NotaFilename(key string) string {
	return notaFilename(key)
}

func notaFilename(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return "nota.jpg"
	}
	if i := strings.LastIndex(key, "/"); i >= 0 && i+1 < len(key) {
		return key[i+1:]
	}
	return key
}

func (s *Service) UpdateTransaction(ctx context.Context, teamID, txID uuid.UUID, input UpdateWebTxInput) (*models.Transaction, *models.Balance, error) {
	existing, err := s.repo.GetTransaction(ctx, teamID, txID)
	if err != nil {
		return nil, nil, err
	}

	notaKey := existing.StoredNota()
	oldKeys := models.ParseNotaKeys(notaKey)

	if input.RemoveNota {
		notaKey = nil
	}
	if len(input.NotaReplace) > 0 {
		notaKey = models.JoinNotaKeys(input.NotaReplace)
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
	if input.RemoveNota || len(input.NotaReplace) > 0 {
		s.deleteNotaKeys(ctx, oldKeys)
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
	s.deleteNotaKeys(ctx, models.ParseNotaKeys(tx.StoredNota()))
	return s.repo.GetBalance(ctx, teamID, nil, nil)
}

type BatchDeleteResult struct {
	Deleted int             `json:"deleted"`
	Balance *models.Balance `json:"balance"`
}

func (s *Service) BatchDeleteTransactions(ctx context.Context, teamID uuid.UUID, txIDs []uuid.UUID) (*BatchDeleteResult, error) {
	if len(txIDs) == 0 {
		return nil, fmt.Errorf("tidak ada transaksi dipilih")
	}
	deleted := 0
	for _, txID := range txIDs {
		tx, err := s.repo.GetTransaction(ctx, teamID, txID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				continue
			}
			return nil, err
		}
		if err := s.repo.DeleteTransaction(ctx, teamID, txID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				continue
			}
			return nil, err
		}
		s.deleteNotaKeys(ctx, models.ParseNotaKeys(tx.StoredNota()))
		deleted++
	}
	if deleted == 0 {
		return nil, repository.ErrNotFound
	}
	balance, err := s.repo.GetBalance(ctx, teamID, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BatchDeleteResult{Deleted: deleted, Balance: balance}, nil
}

type CreateWebTxInput struct {
	Hari       string
	Tanggal    time.Time
	Jenis      models.TxJenis
	Deskripsi  string
	Total      int64
	NotaKeys   []string
	Keterangan *string
}

type UpdateWebTxInput struct {
	Hari        string
	Tanggal     time.Time
	Jenis       models.TxJenis
	Deskripsi   string
	Total       int64
	Keterangan  *string
	NotaReplace []string
	RemoveNota  bool
}

func (s *Service) deleteNotaKeys(ctx context.Context, keys []string) {
	for _, key := range keys {
		if key != "" {
			_ = s.storage.Delete(ctx, key)
		}
	}
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
