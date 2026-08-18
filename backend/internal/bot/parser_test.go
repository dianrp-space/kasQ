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

func TestLinkCommand(t *testing.T) {
	_, err := bot.ParseMessage("!link", models.SourceWA)
	if err != bot.ErrLinkCommand {
		t.Errorf("expected ErrLinkCommand, got %v", err)
	}
}

func TestPublicReportURL(t *testing.T) {
	got := bot.PublicReportURL("http://localhost:3008", "my-kas")
	if got != "http://localhost:3008/report/my-kas" {
		t.Fatalf("got %q", got)
	}
}

func TestParseMessageWithoutHari(t *testing.T) {
	parsed, err := bot.ParseMessage("out#100826#Beli air minum#12000", models.SourceWA)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Hari != "Senin" {
		t.Errorf("expected Senin from 10/08/26, got %s", parsed.Hari)
	}
	if parsed.Deskripsi != "Beli air minum" || parsed.Total != 12000 {
		t.Errorf("fields mismatch: %+v", parsed)
	}
}

func TestParseMessageWithoutHariWithKeterangan(t *testing.T) {
	parsed, err := bot.ParseMessage("out#100826#Beli air minum#12000#Tunai", models.SourceTele)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Hari != "Senin" {
		t.Errorf("expected Senin, got %s", parsed.Hari)
	}
	if parsed.Keterangan == nil || *parsed.Keterangan != "Tunai" {
		t.Errorf("expected keterangan Tunai, got %v", parsed.Keterangan)
	}
}

func TestParseMessageEmptyHariSlot(t *testing.T) {
	parsed, err := bot.ParseMessage("in##010826#Refill kas#2000000", models.SourceWA)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Hari != "Sabtu" {
		t.Errorf("expected Sabtu from 01/08/26, got %s", parsed.Hari)
	}
}

func TestInvalidFormat(t *testing.T) {
	_, err := bot.ParseMessage("invalid", models.SourceWA)
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestParseAlbumCaptionLastWins(t *testing.T) {
	parsed, err := bot.ParseAlbumCaption([]string{"", "bukan format", "out#Senin#100826#Beli galon#12000"}, models.SourceWA)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Deskripsi != "Beli galon" {
		t.Fatalf("got %q", parsed.Deskripsi)
	}
}

func TestParseAlbumCaptionAnyPhoto(t *testing.T) {
	parsed, err := bot.ParseAlbumCaption([]string{"in#Sabtu#010826#Refill#2000000", "", ""}, models.SourceTele)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Jenis != models.JenisIn {
		t.Fatalf("got %s", parsed.Jenis)
	}
}
