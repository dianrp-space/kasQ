package bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kasq/backend/internal/models"
)

type ParsedMessage struct {
	Jenis      models.TxJenis
	Hari       string
	Tanggal    time.Time
	Deskripsi  string
	Total      int64
	Keterangan *string
	Source     models.TxSource
}

// ParseMessage parses:
//
//	{in|out}#{hari}#{DDMMYY}#{deskripsi}#{total}[#{keterangan}]
//	{in|out}#{DDMMYY}#{deskripsi}#{total}[#{keterangan}]  (hari otomatis dari tanggal)
func ParseMessage(text string, source models.TxSource) (*ParsedMessage, error) {
	text = strings.TrimSpace(text)
	if IsLinkCommand(text) {
		return nil, ErrLinkCommand
	}
	if IsSaldoCommand(text) {
		return nil, ErrSaldoCommand
	}

	parts := strings.Split(text, "#")
	if len(parts) < 4 || len(parts) > 6 {
		return nil, fmt.Errorf("format tidak valid. Gunakan: in/out#[Hari]#DDMMYY#Deskripsi#Total#[Keterangan]")
	}

	typeStr := strings.ToLower(strings.TrimSpace(parts[0]))
	var jenis models.TxJenis
	switch typeStr {
	case "in":
		jenis = models.JenisIn
	case "out":
		jenis = models.JenisOut
	default:
		return nil, fmt.Errorf("jenis harus 'in' atau 'out'")
	}

	hari, dateStr, deskripsi, totalStr, keterangan, err := splitTxFields(parts)
	if err != nil {
		return nil, err
	}

	if deskripsi == "" {
		return nil, fmt.Errorf("deskripsi tidak boleh kosong")
	}

	tanggal, err := parseDateDDMMYY(dateStr)
	if err != nil {
		return nil, err
	}
	if hari == "" {
		hari = hariFromDate(tanggal)
	}

	total, err := strconv.ParseInt(totalStr, 10, 64)
	if err != nil || total <= 0 {
		return nil, fmt.Errorf("total harus angka positif")
	}

	return &ParsedMessage{
		Jenis:      jenis,
		Hari:       hari,
		Tanggal:    tanggal,
		Deskripsi:  deskripsi,
		Total:      total,
		Keterangan: keterangan,
		Source:     source,
	}, nil
}

func splitTxFields(parts []string) (hari, dateStr, deskripsi, totalStr string, keterangan *string, err error) {
	idx := 1
	hari = strings.TrimSpace(parts[1])
	if hari == "" || looksLikeDDMMYY(hari) {
		hari = ""
		if strings.TrimSpace(parts[1]) == "" {
			idx = 2
		}
	} else {
		idx = 2
	}
	rest := parts[idx:]
	if len(rest) < 3 || len(rest) > 4 {
		return "", "", "", "", nil, fmt.Errorf("format tidak valid. Gunakan: in/out#[Hari]#DDMMYY#Deskripsi#Total#[Keterangan]")
	}
	dateStr = strings.TrimSpace(rest[0])
	deskripsi = strings.TrimSpace(rest[1])
	totalStr = strings.TrimSpace(rest[2])
	if len(rest) == 4 {
		ket := strings.TrimSpace(rest[3])
		if ket != "" {
			keterangan = &ket
		}
	}
	return hari, dateStr, deskripsi, totalStr, keterangan, nil
}

func looksLikeDDMMYY(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) != 6 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

var hariID = [...]string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}

func hariFromDate(t time.Time) string {
	return hariID[t.Weekday()]
}

func parseDateDDMMYY(s string) (time.Time, error) {
	if len(s) != 6 {
		return time.Time{}, fmt.Errorf("tanggal harus format DDMMYY, contoh: 100826")
	}
	day, err1 := strconv.Atoi(s[0:2])
	month, err2 := strconv.Atoi(s[2:4])
	year, err3 := strconv.Atoi(s[4:6])
	if err1 != nil || err2 != nil || err3 != nil {
		return time.Time{}, fmt.Errorf("tanggal tidak valid: %s", s)
	}
	fullYear := 2000 + year
	t := time.Date(fullYear, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if t.Day() != day || int(t.Month()) != month {
		return time.Time{}, fmt.Errorf("tanggal tidak valid: %s", s)
	}
	return t, nil
}

// ParseAlbumCaption tries captions from last to first so WA (caption di foto terakhir)
// and Telegram (caption di foto mana pun) both work.
func ParseAlbumCaption(captions []string, source models.TxSource) (*ParsedMessage, error) {
	var lastErr error
	hasText := false
	for i := len(captions) - 1; i >= 0; i-- {
		c := strings.TrimSpace(captions[i])
		if c == "" {
			continue
		}
		hasText = true
		parsed, err := ParseMessage(c, source)
		if err == nil {
			return parsed, nil
		}
		if err == ErrSaldoCommand {
			return nil, err
		}
		if err == ErrLinkCommand {
			return nil, err
		}
		lastErr = err
	}
	if !hasText {
		return nil, fmt.Errorf("caption wajib berisi format transaksi (lihat !saldo / !link untuk command)")
	}
	return nil, lastErr
}

func FormatRupiah(n int64) string {
	s := strconv.FormatInt(n, 10)
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, byte(c))
	}
	return "Rp " + string(result)
}

func FormatSuccessReply(tx *models.Transaction, balance int64, teamName string, notaCount int) string {
	jenisLabel := "Pemasukan"
	if tx.Jenis == models.JenisOut {
		jenisLabel = "Pengeluaran"
	}
	notaLabel := "—"
	switch {
	case notaCount > 1:
		notaLabel = fmt.Sprintf("✅ %d foto", notaCount)
	case notaCount == 1:
		notaLabel = "✅ Tersimpan"
	}
	dateStr := tx.Tanggal.Format("02/01/06")
	keteranganLine := ""
	if tx.Keterangan != nil && strings.TrimSpace(*tx.Keterangan) != "" {
		keteranganLine = fmt.Sprintf("\nKeterangan: %s", *tx.Keterangan)
	}
	return fmt.Sprintf(
		"✅ Input berhasil!\nJenis   : %s\nTanggal : %s, %s\nDeskripsi: %s\nTotal   : %s\nNota    : %s%s\n\n💰 Saldo terkini (%s): %s",
		jenisLabel, tx.Hari, dateStr, tx.Deskripsi, FormatRupiah(tx.Total), notaLabel, keteranganLine, teamName, FormatRupiah(balance),
	)
}

func FormatSaldoReply(teamName string, balance int64) string {
	return fmt.Sprintf("💰 Saldo terkini (%s): %s", teamName, FormatRupiah(balance))
}

func FormatLinkReply(teamName, reportURL string) string {
	return fmt.Sprintf("🔗 Laporan publik (%s):\n%s", teamName, reportURL)
}

func IsSaldoCommand(text string) bool {
	return strings.EqualFold(strings.TrimSpace(text), "!saldo")
}

func IsLinkCommand(text string) bool {
	return strings.EqualFold(strings.TrimSpace(text), "!link")
}

func PublicReportURL(appBase, token string) string {
	base := strings.TrimRight(strings.TrimSpace(appBase), "/")
	return base + "/report/" + token
}

func BotHelpText() string {
	return `Format transaksi:
out#Senin#100826#Deskripsi#12000#Keterangan
out#100826#Deskripsi#12000

Hari boleh dikosongkan — terisi otomatis dari tanggal.
(Keterangan opsional)

Command:
!saldo — cek saldo kas
!link — link laporan publik`
}

func FormatErrorReply(err error) string {
	return fmt.Sprintf("❌ Gagal: %s\n\n%s", err.Error(), BotHelpText())
}
