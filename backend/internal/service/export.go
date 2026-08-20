package service

import (
	"bytes"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/go-pdf/fpdf"
	"github.com/kasq/backend/internal/models"
	"github.com/xuri/excelize/v2"
)

type ExportFormat string

const (
	ExportXLSX ExportFormat = "xlsx"
	ExportPDF  ExportFormat = "pdf"
)

type ExportReport struct {
	AppName  string
	TeamName string
	TeamSlug string
	Period   string
	Balance  *models.Balance
	Items    []models.Transaction
}

func ParseExportFormat(raw string) (ExportFormat, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "xlsx", "excel", "xls":
		return ExportXLSX, nil
	case "pdf":
		return ExportPDF, nil
	default:
		return "", fmt.Errorf("format tidak didukung (xlsx atau pdf)")
	}
}

func FormatPeriodLabel(from, to *time.Time) string {
	if from == nil || to == nil {
		return "Semua periode"
	}
	months := []string{
		"Januari", "Februari", "Maret", "April", "Mei", "Juni",
		"Juli", "Agustus", "September", "Oktober", "November", "Desember",
	}
	m := int(from.Month())
	if m < 1 || m > 12 {
		return from.Format("2006-01")
	}
	if from.Year() == to.Year() && from.Month() == to.Month() {
		return fmt.Sprintf("%s %d", months[m-1], from.Year())
	}
	return fmt.Sprintf("%s %d – %s %d", months[m-1], from.Year(), months[int(to.Month())-1], to.Year())
}

func ExportFilename(slug, period string, format ExportFormat) string {
	base := "kasq"
	if s := slugify(slug); s != "" {
		base += "-" + s
	}
	if p := slugify(period); p != "" {
		base += "-" + p
	}
	return base + "." + string(format)
}

func BuildExport(rep ExportReport, format ExportFormat) ([]byte, string, error) {
	if rep.AppName == "" {
		rep.AppName = "KasQ"
	}
	switch format {
	case ExportPDF:
		data, err := buildExportPDF(rep)
		return data, "application/pdf", err
	default:
		data, err := buildExportExcel(rep)
		return data, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", err
	}
}

func buildExportExcel(rep ExportReport) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	summary := "Ringkasan"
	_ = f.SetSheetName("Sheet1", summary)
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"059669"}, Pattern: 1},
		Alignment: &excelize.Alignment{Vertical: "center"},
	})
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14, Color: "065F46"},
	})
	moneyStyle, _ := f.NewStyle(&excelize.Style{
		NumFmt: 3, // #,##0
	})
	inStyle, _ := f.NewStyle(&excelize.Style{
		Font:   &excelize.Font{Color: "047857"},
		NumFmt: 3,
	})
	outStyle, _ := f.NewStyle(&excelize.Style{
		Font:   &excelize.Font{Color: "B91C1C"},
		NumFmt: 3,
	})

	_ = f.SetCellValue(summary, "A1", rep.AppName+" — Laporan Kas")
	_ = f.SetCellStyle(summary, "A1", "A1", titleStyle)
	_ = f.SetCellValue(summary, "A2", "Tim/kas")
	_ = f.SetCellValue(summary, "B2", rep.TeamName)
	_ = f.SetCellValue(summary, "A3", "Periode")
	_ = f.SetCellValue(summary, "B3", rep.Period)

	labels := []string{"Saldo awal periode", "Pemasukan periode", "Pengeluaran periode", "Saldo akhir periode"}
	values := []int64{0, 0, 0, 0}
	if rep.Balance != nil {
		values = []int64{
			rep.Balance.OpeningBalance,
			rep.Balance.TotalIn,
			rep.Balance.TotalOut,
			rep.Balance.CurrentBalance,
		}
	}
	_ = f.SetCellValue(summary, "A5", "Ringkasan")
	_ = f.SetCellStyle(summary, "A5", "B5", headerStyle)
	for i, label := range labels {
		row := i + 6
		_ = f.SetCellValue(summary, fmt.Sprintf("A%d", row), label)
		_ = f.SetCellValue(summary, fmt.Sprintf("B%d", row), values[i])
		style := moneyStyle
		if i == 1 {
			style = inStyle
		} else if i == 2 {
			style = outStyle
		}
		_ = f.SetCellStyle(summary, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), style)
	}
	_ = f.SetColWidth(summary, "A", "A", 28)
	_ = f.SetColWidth(summary, "B", "B", 22)

	txSheet := "Transaksi"
	_, _ = f.NewSheet(txSheet)
	headers := []string{"No", "Tanggal", "Hari", "Jenis", "Deskripsi", "Total", "Sumber", "Keterangan"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(txSheet, cell, h)
	}
	_ = f.SetCellStyle(txSheet, "A1", "H1", headerStyle)
	_ = f.SetRowHeight(txSheet, 1, 20)

	for i, tx := range rep.Items {
		row := i + 2
		_ = f.SetCellValue(txSheet, fmt.Sprintf("A%d", row), i+1)
		_ = f.SetCellValue(txSheet, fmt.Sprintf("B%d", row), tx.Tanggal.Format("02/01/2006"))
		_ = f.SetCellValue(txSheet, fmt.Sprintf("C%d", row), tx.Hari)
		_ = f.SetCellValue(txSheet, fmt.Sprintf("D%d", row), jenisLabelID(tx.Jenis))
		_ = f.SetCellValue(txSheet, fmt.Sprintf("E%d", row), tx.Deskripsi)
		_ = f.SetCellValue(txSheet, fmt.Sprintf("F%d", row), tx.Total)
		if tx.Jenis == models.JenisIn {
			_ = f.SetCellStyle(txSheet, fmt.Sprintf("F%d", row), fmt.Sprintf("F%d", row), inStyle)
		} else {
			_ = f.SetCellStyle(txSheet, fmt.Sprintf("F%d", row), fmt.Sprintf("F%d", row), outStyle)
		}
		_ = f.SetCellValue(txSheet, fmt.Sprintf("G%d", row), sourceLabelID(tx.Source))
		if tx.Keterangan != nil {
			_ = f.SetCellValue(txSheet, fmt.Sprintf("H%d", row), *tx.Keterangan)
		}
	}
	_ = f.SetColWidth(txSheet, "A", "A", 6)
	_ = f.SetColWidth(txSheet, "B", "D", 14)
	_ = f.SetColWidth(txSheet, "E", "E", 36)
	_ = f.SetColWidth(txSheet, "F", "F", 16)
	_ = f.SetColWidth(txSheet, "G", "G", 12)
	_ = f.SetColWidth(txSheet, "H", "H", 32)
	_ = f.SetPanes(txSheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func buildExportPDF(rep ExportReport) ([]byte, error) {
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.SetTitle(rep.AppName+" — "+rep.TeamName, false)
	pdf.SetAuthor(rep.AppName, false)
	pdf.SetMargins(12, 14, 12)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()

	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetTextColor(6, 95, 70)
	pdf.CellFormat(0, 8, pdfSafe(rep.AppName+" — Laporan Kas"), "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 11)
	pdf.SetTextColor(51, 65, 85)
	pdf.CellFormat(0, 6, pdfSafe("Tim/kas: "+rep.TeamName), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, pdfSafe("Periode: "+rep.Period), "", 1, "L", false, 0, "")
	pdf.Ln(2)

	opening, totalIn, totalOut, closing := int64(0), int64(0), int64(0), int64(0)
	if rep.Balance != nil {
		opening = rep.Balance.OpeningBalance
		totalIn = rep.Balance.TotalIn
		totalOut = rep.Balance.TotalOut
		closing = rep.Balance.CurrentBalance
	}
	summaries := []struct {
		label string
		value int64
		r, g, b int
	}{
		{"Saldo awal " + rep.Period, opening, 30, 41, 59},
		{"Pemasukan " + rep.Period, totalIn, 4, 120, 87},
		{"Pengeluaran " + rep.Period, totalOut, 185, 28, 28},
		{"Saldo akhir " + rep.Period, closing, 5, 150, 105},
	}
	boxW := 66.75
	x0, y0 := pdf.GetX(), pdf.GetY()
	for i, s := range summaries {
		x := x0 + float64(i)*boxW
		pdf.Rect(x, y0, boxW-2, 16, "D")
		pdf.SetXY(x+2, y0+1.5)
		pdf.SetFont("Helvetica", "", 8)
		pdf.SetTextColor(100, 116, 139)
		pdf.CellFormat(boxW-6, 5, pdfSafe(s.label), "", 1, "L", false, 0, "")
		pdf.SetX(x + 2)
		pdf.SetFont("Helvetica", "B", 11)
		pdf.SetTextColor(s.r, s.g, s.b)
		pdf.CellFormat(boxW-6, 8, pdfSafe(formatIDR(s.value)), "", 0, "L", false, 0, "")
	}
	pdf.SetY(y0 + 20)
	pdf.SetTextColor(15, 23, 42)

	headers := []string{"No", "Tanggal", "Hari", "Jenis", "Deskripsi", "Total", "Sumber", "Keterangan"}
	widths := []float64{10, 24, 22, 18, 78, 32, 22, 67}
	writePDFTableHeader(pdf, headers, widths)

	lineH := 5.2
	pageBottom := 190.0
	for i, tx := range rep.Items {
		ket := ""
		if tx.Keterangan != nil {
			ket = *tx.Keterangan
		}
		vals := []string{
			fmt.Sprintf("%d", i+1),
			tx.Tanggal.Format("02/01/2006"),
			pdfSafe(tx.Hari),
			jenisLabelID(tx.Jenis),
			pdfSafe(tx.Deskripsi),
			formatIDR(tx.Total),
			sourceLabelID(tx.Source),
			pdfSafe(ket),
		}
		aligns := []string{"C", "C", "C", "C", "L", "R", "C", "L"}
		rowH := pdfRowHeight(pdf, vals, widths, lineH)
		if pdf.GetY()+rowH > pageBottom {
			pdf.AddPage()
			writePDFTableHeader(pdf, headers, widths)
		}
		if i%2 == 0 {
			pdf.SetFillColor(248, 250, 252)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		writePDFWrapRow(pdf, vals, widths, aligns, lineH, rowH)
	}

	pdf.SetY(-12)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(148, 163, 184)
	pdf.CellFormat(0, 6, fmt.Sprintf("Dicetak %s  ·  %d transaksi", time.Now().Format("02/01/2006 15:04"), len(rep.Items)), "", 0, "R", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func jenisLabelID(j models.TxJenis) string {
	if j == models.JenisIn {
		return "Masuk"
	}
	return "Keluar"
}

func sourceLabelID(s models.TxSource) string {
	switch s {
	case models.SourceWA:
		return "WhatsApp"
	case models.SourceTele:
		return "Telegram"
	default:
		return "Web"
	}
}

func formatIDR(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := fmt.Sprintf("%d", n)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	out := "Rp " + b.String()
	if neg {
		return "-" + out
	}
	return out
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash && b.Len() > 0 {
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func pdfSafe(s string) string {
	s = strings.ReplaceAll(s, "—", "-")
	s = strings.ReplaceAll(s, "–", "-")
	s = strings.ReplaceAll(s, "“", "\"")
	s = strings.ReplaceAll(s, "”", "\"")
	s = strings.ReplaceAll(s, "’", "'")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	var b strings.Builder
	for _, r := range s {
		if r <= 255 {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func writePDFTableHeader(pdf *fpdf.Fpdf, headers []string, widths []float64) {
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetFillColor(5, 150, 105)
	pdf.SetTextColor(255, 255, 255)
	for i, h := range headers {
		pdf.CellFormat(widths[i], 8, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetTextColor(15, 23, 42)
	pdf.SetFont("Helvetica", "", 8)
}

func pdfRowHeight(pdf *fpdf.Fpdf, vals []string, widths []float64, lineH float64) float64 {
	maxLines := 1
	for i, v := range vals {
		lines := pdf.SplitLines([]byte(v), widths[i]-1.5)
		if n := len(lines); n > maxLines {
			maxLines = n
		}
	}
	h := float64(maxLines) * lineH
	if h < lineH+1 {
		return lineH + 1
	}
	return h + 1
}

func writePDFWrapRow(pdf *fpdf.Fpdf, vals []string, widths []float64, aligns []string, lineH, rowH float64) {
	x, y := pdf.GetX(), pdf.GetY()
	left := x
	for i, v := range vals {
		pdf.Rect(x, y, widths[i], rowH, "FD")
		pdf.SetXY(x+0.7, y+0.5)
		pdf.MultiCell(widths[i]-1.4, lineH, v, "", aligns[i], false)
		x += widths[i]
	}
	pdf.SetXY(left, y+rowH)
}
