package bot_test

import (
	"testing"
	"time"

	"github.com/kasq/backend/internal/bot"
	"github.com/kasq/backend/internal/models"
)

func TestParseMessageOut(t *testing.T) {
	parsed, err := bot.ParseMessage("out#Senin#100826#Beli air minum#12000", models.SourceWA)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Jenis != models.JenisOut {
		t.Errorf("expected out, got %s", parsed.Jenis)
	}
	if parsed.Hari != "Senin" {
		t.Errorf("expected Senin, got %s", parsed.Hari)
	}
	if parsed.Total != 12000 {
		t.Errorf("expected 12000, got %d", parsed.Total)
	}
	expected := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	if !parsed.Tanggal.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, parsed.Tanggal)
	}
}

func TestParseMessageIn(t *testing.T) {
	parsed, err := bot.ParseMessage("in#Sabtu#010826#Refill kas Batam#2000000", models.SourceTele)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Jenis != models.JenisIn {
		t.Errorf("expected in, got %s", parsed.Jenis)
	}
	if parsed.Total != 2000000 {
		t.Errorf("expected 2000000, got %d", parsed.Total)
	}
}

func TestParseMessageWithKeterangan(t *testing.T) {
	parsed, err := bot.ParseMessage("out#Senin#100826#Beli air minum#12000#Dibayar tunai", models.SourceWA)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Keterangan == nil || *parsed.Keterangan != "Dibayar tunai" {
		t.Errorf("expected keterangan, got %v", parsed.Keterangan)
	}
}

func TestParseMessageWithoutKeterangan(t *testing.T) {
	parsed, err := bot.ParseMessage("out#Senin#100826#Beli air minum#12000", models.SourceWA)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Keterangan != nil {
		t.Errorf("expected nil keterangan, got %v", parsed.Keterangan)
	}
}

func TestSaldoCommand(t *testing.T) {
	_, err := bot.ParseMessage("!saldo", models.SourceWA)
	if err != bot.ErrSaldoCommand {
		t.Errorf("expected ErrSaldoCommand, got %v", err)
	}
}

func TestInvalidFormat(t *testing.T) {
	_, err := bot.ParseMessage("invalid", models.SourceWA)
	if err == nil {
		t.Error("expected error for invalid format")
	}
}
