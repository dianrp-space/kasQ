package service

import (
	"testing"
	"time"

	"github.com/kasq/backend/internal/models"
)

func TestParseHyperlinkFormula(t *testing.T) {
	tests := []struct {
		formula string
		want    string
	}{
		{
			`=HYPERLINK("https://bon.dianrp.com/statics/media/1778203994-9956c9fe-9e8b-47a6-96b0-5711399f2c2b.jpeg", "Lihat Gambar")`,
			"https://bon.dianrp.com/statics/media/1778203994-9956c9fe-9e8b-47a6-96b0-5711399f2c2b.jpeg",
		},
		{
			`=HYPERLINK("https://example.com/nota.jpg";"Lihat")`,
			"https://example.com/nota.jpg",
		},
		{
			`=_xlfn.HYPERLINK("https://example.com/a.png","Gambar")`,
			"https://example.com/a.png",
		},
		{
			`=HYPERLINK('https://example.com/b.jpg','Gambar')`,
			"https://example.com/b.jpg",
		},
		{"=SUM(A1:A2)", ""},
	}
	for _, tc := range tests {
		got := parseHyperlinkFormula(tc.formula)
		if got != tc.want {
			t.Fatalf("parseHyperlinkFormula(%q) = %q, want %q", tc.formula, got, tc.want)
		}
	}
}

func TestImportTxFingerprint(t *testing.T) {
	d := time.Date(2026, 2, 6, 0, 0, 0, 0, time.UTC)
	a := importTxFingerprint(d, models.JenisOut, "Beli air minum", 12000)
	b := importTxFingerprint(d, models.JenisOut, "  beli   air   minum  ", 12000)
	if a != b {
		t.Fatalf("fingerprint should normalize deskripsi: %q vs %q", a, b)
	}
	c := importTxFingerprint(d, models.JenisIn, "Beli air minum", 12000)
	if a == c {
		t.Fatal("jenis should affect fingerprint")
	}
}

func TestIsHTTPURL(t *testing.T) {
	if !isHTTPURL("https://bon.dianrp.com/x.jpeg") {
		t.Fatal("expected valid https url")
	}
	if isHTTPURL("Lihat Gambar") {
		t.Fatal("display text should not be url")
	}
}
