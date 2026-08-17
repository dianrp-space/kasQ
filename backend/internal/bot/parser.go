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

// ParseMessage parses: {in|out}#{hari}#{DDMMYY}#{deskripsi}#{total}[#{keterangan}]
func ParseMessage(text string, source models.TxSource) (*ParsedMessage, error) {
	text = strings.TrimSpace(text)
	if strings.EqualFold(text, "!saldo") {
		return nil, ErrSaldoCommand
	}

	parts := strings.Split(text, "#")
	if len(parts) != 5 && len(parts) != 6 {
		return nil, fmt.Errorf("format tidak valid. Gunakan: in/out#Hari#DDMMYY#Deskripsi#Total#Keterangan")
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

	hari := strings.TrimSpace(parts[1])
	dateStr := strings.TrimSpace(parts[2])
	deskripsi := strings.TrimSpace(parts[3])
	totalStr := strings.TrimSpace(parts[4])
	var keterangan *string
	if len(parts) == 6 {
		ket := strings.TrimSpace(parts[5])
		if ket != "" {
			keterangan = &ket
		}
	}

	if deskripsi == "" {
		return nil, fmt.Errorf("deskripsi tidak boleh kosong")
	}

	tanggal, err := parseDateDDMMYY(dateStr)
	if err != nil {
		return nil, err
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

func FormatSuccessReply(tx *models.Transaction, balance int64, teamName string, hasNota bool) string {
	jenisLabel := "Pemasukan"
	if tx.Jenis == models.JenisOut {
		jenisLabel = "Pengeluaran"
	}
	notaLabel := "—"
	if hasNota {
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

func FormatErrorReply(err error) string {
	return fmt.Sprintf("❌ Gagal: %s\n\nFormat:\nout#Senin#100826#Deskripsi#12000#Keterangan\nin#Sabtu#010826#Deskripsi#2000000\n\n(Keterangan opsional)", err.Error())
}
