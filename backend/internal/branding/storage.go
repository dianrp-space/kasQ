package branding

import (
	"fmt"
	"os"
	"path/filepath"
)

type Storage struct {
	dir string
}

func NewStorage(dir string) (*Storage, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("branding dir: %w", err)
	}
	return &Storage{dir: dir}, nil
}

func (s *Storage) Save(filename string, data []byte) error {
	path := filepath.Join(s.dir, filename)
	return os.WriteFile(path, data, 0o644)
}

func (s *Storage) Path(filename string) string {
	return filepath.Join(s.dir, filename)
}

func (s *Storage) Exists(filename string) bool {
	_, err := os.Stat(filepath.Join(s.dir, filename))
	return err == nil
}

func (s *Storage) Remove(filename string) error {
	if filename == "" {
		return nil
	}
	err := os.Remove(filepath.Join(s.dir, filename))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
