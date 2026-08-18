package models

import "testing"

func TestNormalizeWAPhoneDigits(t *testing.T) {
	if got := NormalizeWAPhoneDigits("+62 812-3456-7890"); got != "6281234567890" {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeWAPhoneDigits("081234567890"); got != "6281234567890" {
		t.Fatalf("leading zero should become 62: %q", got)
	}
}

func TestParseJoinWAAllowedPhones(t *testing.T) {
	raw := `["628111","628222"]`
	got := ParseWAAllowedPhones(&raw)
	if len(got) != 2 || got[0] != "628111" {
		t.Fatalf("parse: %#v", got)
	}
	joined := JoinWAAllowedPhones([]string{"628111", "628111", "628222"})
	if joined == nil || *joined != raw {
		t.Fatalf("join: %v", joined)
	}
}

func TestIntegrationSenderAllowed(t *testing.T) {
	all := Integration{WAAllowedPhones: nil}
	if !all.SenderAllowed("628111") {
		t.Fatal("empty list should allow all")
	}
	locked := Integration{WAAllowedPhones: []string{"628111", "628222"}}
	if !locked.SenderAllowed("628111") || locked.SenderAllowed("628999") {
		t.Fatal("whitelist mismatch")
	}
	if !locked.AnySenderAllowed([]string{"272532854849745", "628111"}) {
		t.Fatal("should allow when one candidate is a phone match")
	}
	if !locked.SenderAllowed("08111") {
		t.Fatal("08 should match 628")
	}
}
