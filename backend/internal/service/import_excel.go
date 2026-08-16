package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kasq/backend/internal/models"
	"github.com/xuri/excelize/v2"
)

var (
	rupiahDigitsRe = regexp.MustCompile(`[^\d]`)
	indonesianDays = []string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}
	hyperlinkURLRe = regexp.MustCompile(`(?i)(?:_xlfn\.)?HYPERLINK\s*\(\s*"([^"]+)"`)
	importNotaHTTPClient = &http.Client{
		Timeout: 25 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        32,
			MaxIdleConnsPerHost: 16,
			IdleConnTimeout:     90 * time.Second,
		},
	}
)

type ImportRowError struct {
	Row     int    `json:"row"`
	Sheet   string `json:"sheet,omitempty"`
	Message string `json:"message"`
}

type ImportResult struct {
	Imported   int              `json:"imported"`
	Failed     int              `json:"failed"`
	Skipped    int              `json:"skipped"`
	Duplicates int              `json:"duplicates"`
	SheetsUsed int              `json:"sheets_used"`
	Errors     []ImportRowError `json:"errors"`
	Balance    *models.Balance  `json:"balance"`
}

type ImportProgressEvent struct {
	Type       string        `json:"type"`
	Phase      string        `json:"phase,omitempty"`
	Message    string        `json:"message,omitempty"`
	Sheet      string        `json:"sheet,omitempty"`
	Row        int           `json:"row,omitempty"`
	Current    int           `json:"current,omitempty"`
	Total      int           `json:"total,omitempty"`
	Imported   int           `json:"imported,omitempty"`
	Failed     int           `json:"failed,omitempty"`
	Skipped    int           `json:"skipped,omitempty"`
	Duplicates int           `json:"duplicates,omitempty"`
	Result     *ImportResult `json:"result,omitempty"`
}

type ImportProgressEmit func(ImportProgressEvent)

func emitImportProgress(emit ImportProgressEmit, ev ImportProgressEvent) {
	if emit != nil {
		emit(ev)
	}
}

type parsedImportRow struct {
	Row       int
	Hari      string
	Tanggal   time.Time
	Jenis     models.TxJenis
	Deskripsi string
	Total     int64
	NotaURL   string
}

func (s *Service) ImportTransactionsFromExcel(ctx context.Context, teamID, userID uuid.UUID, data []byte, fetchNota bool, emit ImportProgressEmit) (*ImportResult, error) {
	emitImportProgress(emit, ImportProgressEvent{Type: "progress", Phase: "prepare", Message: "Membaca file Excel..."})

	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("file excel tidak valid")
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("file excel tidak punya sheet")
	}

	totalRows := countImportDataRows(f, sheets)
	emitImportProgress(emit, ImportProgressEvent{
		Type: "progress", Phase: "prepare", Message: fmt.Sprintf("Ditemukan ~%d baris data", totalRows), Total: totalRows,
	})

	result := &ImportResult{Errors: []ImportRowError{}}
	var lastBalance *models.Balance

	emitImportProgress(emit, ImportProgressEvent{
		Type: "progress", Phase: "prepare", Message: "Memuat indeks duplikat...", Total: totalRows,
	})

	txKeys, err := s.repo.ListImportTxKeys(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("gagal cek duplikat: %w", err)
	}
	existingKeys := make(map[string]struct{}, len(txKeys))
	for _, k := range txKeys {
		existingKeys[importTxFingerprint(k.Tanggal, k.Jenis, k.Deskripsi, k.Total)] = struct{}{}
	}
	seenInFile := make(map[string]struct{})
	current := 0

	for _, sheet := range sheets {
		rows, err := f.GetRows(sheet)
		if err != nil || len(rows) == 0 {
			continue
		}
		headerRow, colMap := findImportHeader(rows)
		if headerRow < 0 {
			continue
		}
		result.SheetsUsed++

		emitImportProgress(emit, ImportProgressEvent{
			Type: "progress", Phase: "sheet", Sheet: sheet,
			Message: fmt.Sprintf("Memproses sheet %q...", sheet),
			Current: current, Total: totalRows,
			Imported: result.Imported, Failed: result.Failed, Skipped: result.Skipped, Duplicates: result.Duplicates,
		})

		for i := headerRow + 1; i < len(rows); i++ {
			rowNum := i + 1
			row := rows[i]
			if isImportSkipRow(row) {
				result.Skipped++
				continue
			}
			current++

			parsed, perr := parseImportRow(row, colMap, rowNum)
			if perr != nil {
				if perr.Error() == "empty" {
					result.Skipped++
					emitImportProgress(emit, importRowProgress(current, totalRows, sheet, rowNum, result, "Baris kosong, dilewati"))
					continue
				}
				result.Failed++
				result.Errors = append(result.Errors, ImportRowError{Sheet: sheet, Row: rowNum, Message: perr.Error()})
				emitImportProgress(emit, importRowProgress(current, totalRows, sheet, rowNum, result, perr.Error()))
				continue
			}

			if parsed.NotaURL == "" && colMap["link_nota"] >= 0 {
				col, _ := excelize.ColumnNumberToName(colMap["link_nota"] + 1)
				cell := fmt.Sprintf("%s%d", col, rowNum)
				parsed.NotaURL = resolveImportNotaURL(f, sheet, cell, cellAt(row, colMap["link_nota"]))
			}

			fp := importTxFingerprint(parsed.Tanggal, parsed.Jenis, parsed.Deskripsi, parsed.Total)
			if _, ok := existingKeys[fp]; ok {
				result.Duplicates++
				result.Skipped++
				emitImportProgress(emit, importRowProgress(current, totalRows, sheet, rowNum, result, "Duplikat, dilewati"))
				continue
			}
			if _, ok := seenInFile[fp]; ok {
				result.Duplicates++
				result.Skipped++
				emitImportProgress(emit, importRowProgress(current, totalRows, sheet, rowNum, result, "Duplikat dalam file, dilewati"))
				continue
			}
			seenInFile[fp] = struct{}{}

			var notaKey *string
			if fetchNota && parsed.NotaURL != "" {
				rowNum := rowNum
				emitImportProgress(emit, ImportProgressEvent{
					Type: "progress", Phase: "nota", Sheet: sheet, Row: rowNum,
					Message: fmt.Sprintf("Mengunduh nota baris %d...", rowNum),
					Current: current, Total: totalRows,
					Imported: result.Imported, Failed: result.Failed, Skipped: result.Skipped, Duplicates: result.Duplicates,
				})
				key, err := s.fetchAndUploadNotaURLWithHeartbeat(ctx, teamID, parsed.NotaURL, func() {
					emitImportProgress(emit, ImportProgressEvent{
						Type: "progress", Phase: "nota", Sheet: sheet, Row: rowNum,
						Message: fmt.Sprintf("Masih mengunduh nota baris %d...", rowNum),
						Current: current, Total: totalRows,
						Imported: result.Imported, Failed: result.Failed, Skipped: result.Skipped, Duplicates: result.Duplicates,
					})
				})
				if err != nil {
					result.Errors = append(result.Errors, ImportRowError{
						Sheet:   sheet,
						Row:     rowNum,
						Message: fmt.Sprintf("nota (transaksi tetap diimport): %v", err),
					})
				} else {
					notaKey = &key
				}
			}

			hari := parsed.Hari
			if hari == "" {
				hari = dayNameID(parsed.Tanggal)
			}

			tx, balance, err := s.CreateTransactionFromWeb(ctx, teamID, userID, CreateWebTxInput{
				Hari:      hari,
				Tanggal:   parsed.Tanggal,
				Jenis:     parsed.Jenis,
				Deskripsi: parsed.Deskripsi,
				Total:     parsed.Total,
				NotaKey:   notaKey,
			})
			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, ImportRowError{Sheet: sheet, Row: rowNum, Message: err.Error()})
				emitImportProgress(emit, importRowProgress(current, totalRows, sheet, rowNum, result, err.Error()))
				continue
			}
			_ = tx
			lastBalance = balance
			existingKeys[fp] = struct{}{}
			result.Imported++
			emitImportProgress(emit, importRowProgress(current, totalRows, sheet, rowNum, result, truncateImportMsg(parsed.Deskripsi, 48)))
		}
	}

	if result.SheetsUsed == 0 {
		return nil, fmt.Errorf("tidak ada sheet dengan header transaksi (Hari, Tanggal, Jenis, Deskripsi, Total)")
	}

	if lastBalance == nil {
		balance, err := s.repo.GetBalance(ctx, teamID, nil, nil)
		if err != nil {
			return result, err
		}
		lastBalance = balance
	}
	result.Balance = lastBalance
	emitImportProgress(emit, ImportProgressEvent{
		Type: "progress", Phase: "finish", Message: "Import selesai",
		Current: totalRows, Total: totalRows,
		Imported: result.Imported, Failed: result.Failed, Skipped: result.Skipped, Duplicates: result.Duplicates,
	})
	return result, nil
}

func importRowProgress(current, total int, sheet string, rowNum int, result *ImportResult, message string) ImportProgressEvent {
	return ImportProgressEvent{
		Type:       "progress",
		Phase:      "row",
		Sheet:      sheet,
		Row:        rowNum,
		Current:    current,
		Total:      total,
		Imported:   result.Imported,
		Failed:     result.Failed,
		Skipped:    result.Skipped,
		Duplicates: result.Duplicates,
		Message:    message,
	}
}

func truncateImportMsg(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func countImportDataRows(f *excelize.File, sheets []string) int {
	total := 0
	for _, sheet := range sheets {
		rows, err := f.GetRows(sheet)
		if err != nil || len(rows) == 0 {
			continue
		}
		headerRow, _ := findImportHeader(rows)
		if headerRow < 0 {
			continue
		}
		for i := headerRow + 1; i < len(rows); i++ {
			if !isImportSkipRow(rows[i]) {
				total++
			}
		}
	}
	return total
}

func (s *Service) fetchAndUploadNotaURLWithHeartbeat(ctx context.Context, teamID uuid.UUID, rawURL string, heartbeat func()) (string, error) {
	type fetchResult struct {
		key string
		err error
	}
	ch := make(chan fetchResult, 1)
	go func() {
		key, err := s.fetchAndUploadNotaURL(ctx, teamID, rawURL)
		ch <- fetchResult{key, err}
	}()

	ticker := time.NewTicker(8 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case r := <-ch:
			return r.key, r.err
		case <-ticker.C:
			if heartbeat != nil {
				heartbeat()
			}
		}
	}
}

func (s *Service) fetchAndUploadNotaURL(ctx context.Context, teamID uuid.UUID, rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("url kosong")
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("url tidak valid")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("skema url harus http/https")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "KasQ-Import/1.0")
	req.Header.Set("Accept", "image/*,*/*")
	resp, err := importNotaHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gagal unduh: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	const maxNota = 10 << 20 // 10MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxNota+1))
	if err != nil {
		return "", err
	}
	if len(body) > maxNota {
		return "", fmt.Errorf("file nota terlalu besar (max 10MB)")
	}

	filename := path.Base(u.Path)
	if filename == "" || filename == "." || filename == "/" {
		filename = "import-nota.jpg"
	}
	contentType := resp.Header.Get("Content-Type")
	return s.UploadNota(ctx, teamID, filename, body, contentType)
}

func resolveImportNotaURL(f *excelize.File, sheet, cell, cellText string) string {
	cellText = strings.TrimSpace(cellText)
	if isHTTPURL(cellText) {
		return cellText
	}
	if has, link, err := f.GetCellHyperLink(sheet, cell); err == nil && has {
		link = strings.TrimSpace(link)
		if isHTTPURL(link) {
			return link
		}
	}
	if formula, err := f.GetCellFormula(sheet, cell); err == nil && formula != "" {
		if u := parseHyperlinkFormula(formula); u != "" {
			return u
		}
	}
	if value, err := f.GetCellValue(sheet, cell); err == nil {
		value = strings.TrimSpace(value)
		if isHTTPURL(value) {
			return value
		}
	}
	return ""
}

func isHTTPURL(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	lower := strings.ToLower(s)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

func parseHyperlinkFormula(formula string) string {
	formula = strings.TrimSpace(formula)
	if formula == "" {
		return ""
	}
	if m := hyperlinkURLRe.FindStringSubmatch(formula); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	// Single-quoted URL: HYPERLINK('https://...','label')
	re := regexp.MustCompile(`(?i)(?:_xlfn\.)?HYPERLINK\s*\(\s*'([^']+)'`)
	if m := re.FindStringSubmatch(formula); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func normalizeImportDeskripsi(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(s)), " "))
}

func importTxFingerprint(tanggal time.Time, jenis models.TxJenis, deskripsi string, total int64) string {
	return fmt.Sprintf("%s|%s|%s|%d",
		tanggal.Format("2006-01-02"),
		jenis,
		normalizeImportDeskripsi(deskripsi),
		total,
	)
}

func findImportHeader(rows [][]string) (int, map[string]int) {
	colMap := map[string]int{
		"hari": -1, "tanggal": -1, "jenis": -1, "deskripsi": -1, "total": -1, "link_nota": -1,
	}
	aliases := map[string][]string{
		"hari":      {"hari", "day"},
		"tanggal":   {"tanggal", "date", "tgl"},
		"jenis":     {"jenis", "type", "tipe"},
		"deskripsi": {"deskripsi", "description", "keterangan"},
		"total":     {"total", "jumlah", "nominal", "amount"},
		"link_nota": {"link nota", "link_nota", "nota", "link", "lampiran"},
	}

	for i, row := range rows {
		if i > 15 {
			break
		}
		found := map[string]int{
			"hari": -1, "tanggal": -1, "jenis": -1, "deskripsi": -1, "total": -1, "link_nota": -1,
		}
		for j, cell := range row {
			norm := strings.ToLower(strings.TrimSpace(cell))
			for key, names := range aliases {
				for _, name := range names {
					if norm == name || strings.HasPrefix(norm, name) {
						found[key] = j
					}
				}
			}
		}
		if found["tanggal"] >= 0 && found["jenis"] >= 0 && found["deskripsi"] >= 0 && found["total"] >= 0 {
			return i, found
		}
	}
	return -1, colMap
}

func cellAt(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func isImportSkipRow(row []string) bool {
	line := strings.ToLower(strings.Join(row, " "))
	if line == "" {
		return true
	}
	for _, skip := range []string{"saldo awal", "saldo saat ini", "saldo akhir", "total pemasukan", "total pengeluaran"} {
		if strings.Contains(line, skip) {
			return true
		}
	}
	return false
}

func parseImportRow(row []string, colMap map[string]int, rowNum int) (*parsedImportRow, error) {
	deskripsi := cellAt(row, colMap["deskripsi"])
	totalStr := cellAt(row, colMap["total"])
	jenisStr := cellAt(row, colMap["jenis"])
	tanggalStr := cellAt(row, colMap["tanggal"])

	if deskripsi == "" && totalStr == "" && jenisStr == "" && tanggalStr == "" {
		return nil, fmt.Errorf("empty")
	}
	if deskripsi == "" {
		return nil, fmt.Errorf("deskripsi kosong")
	}
	if jenisStr == "" {
		return nil, fmt.Errorf("jenis kosong")
	}
	if tanggalStr == "" {
		return nil, fmt.Errorf("tanggal kosong")
	}
	if totalStr == "" {
		return nil, fmt.Errorf("total kosong")
	}

	jenis, err := parseImportJenis(jenisStr)
	if err != nil {
		return nil, err
	}
	tanggal, err := parseImportDate(tanggalStr)
	if err != nil {
		return nil, fmt.Errorf("tanggal: %w", err)
	}
	total, err := parseImportTotal(totalStr)
	if err != nil {
		return nil, fmt.Errorf("total: %w", err)
	}

	notaURL := cellAt(row, colMap["link_nota"])
	if notaURL != "" && !strings.HasPrefix(strings.ToLower(notaURL), "http") {
		// Display text like "Lihat Gambar" — hyperlink resolved later
		notaURL = ""
	}

	return &parsedImportRow{
		Row:       rowNum,
		Hari:      cellAt(row, colMap["hari"]),
		Tanggal:   tanggal,
		Jenis:     jenis,
		Deskripsi: deskripsi,
		Total:     total,
		NotaURL:   notaURL,
	}, nil
}

func parseImportJenis(s string) (models.TxJenis, error) {
	n := strings.ToLower(strings.TrimSpace(s))
	switch n {
	case "in", "pemasukan", "masuk", "income", "credit":
		return models.JenisIn, nil
	case "out", "pengeluaran", "keluar", "expense", "debit":
		return models.JenisOut, nil
	default:
		return "", fmt.Errorf("jenis tidak dikenali: %s", s)
	}
}

func parseImportTotal(s string) (int64, error) {
	digits := rupiahDigitsRe.ReplaceAllString(strings.TrimSpace(s), "")
	if digits == "" {
		return 0, fmt.Errorf("format tidak valid")
	}
	n, err := strconv.ParseInt(digits, 10, 64)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("harus angka positif")
	}
	return n, nil
}

func parseImportDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("kosong")
	}

	// Excel serial number
	if f, err := strconv.ParseFloat(s, 64); err == nil && f > 20000 && f < 100000 {
		if t, err := excelize.ExcelDateToTime(f, false); err == nil {
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
		}
	}

	formats := []string{
		"02/January/2006",
		"2/January/2006",
		"02/01/2006",
		"2/1/2006",
		"2006-01-02",
		"02-01-2006",
		"01/02/2006",
		"02.01.2006",
	}
	for _, layout := range formats {
		if t, err := time.Parse(layout, s); err == nil {
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
		}
	}

	// DD/Month/YYYY with English month (case insensitive via title)
	parts := strings.Split(s, "/")
	if len(parts) == 3 {
		day, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		monthStr := strings.TrimSpace(parts[1])
		year, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
		if monthNum := englishMonthNum(monthStr); monthNum > 0 && day > 0 && year > 0 {
			if year < 100 {
				year += 2000
			}
			t := time.Date(year, time.Month(monthNum), day, 0, 0, 0, 0, time.UTC)
			if t.Day() == day {
				return t, nil
			}
		}
	}

	return time.Time{}, fmt.Errorf("format tidak dikenali (%s)", s)
}

func englishMonthNum(name string) int {
	months := map[string]int{
		"january": 1, "february": 2, "march": 3, "april": 4, "may": 5, "june": 6,
		"july": 7, "august": 8, "september": 9, "october": 10, "november": 11, "december": 12,
		"januari": 1, "februari": 2, "maret": 3, "mei": 5, "juni": 6, "juli": 7,
		"agustus": 8, "oktober": 10, "desember": 12,
	}
	return months[strings.ToLower(strings.TrimSpace(name))]
}

func dayNameID(t time.Time) string {
	// Go: Sunday=0 — our list starts Minggu at index 0
	return indonesianDays[int(t.Weekday())]
}

func BuildImportTemplate() ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	sheet := f.GetSheetName(0)
	headers := []string{"Hari", "Tanggal", "Jenis", "Deskripsi", "Total", "Link Nota"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	examples := [][]any{
		{"Jumat", "06/February/2026", "Pengeluaran", "Beli air minum galon", "-Rp12,000", "https://example.com/nota.jpg"},
		{"Senin", "23/February/2026", "Pemasukan", "Saldo Transfer", "Rp800,000", ""},
	}
	for r, ex := range examples {
		for c, v := range ex {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			_ = f.SetCellValue(sheet, cell, v)
		}
	}
	_ = f.SetColWidth(sheet, "A", "F", 18)
	_ = f.SetColWidth(sheet, "D", "D", 32)
	_ = f.SetColWidth(sheet, "F", "F", 40)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
