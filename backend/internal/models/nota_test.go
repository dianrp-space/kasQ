package models

import "testing"

func TestParseJoinNotaKeys(t *testing.T) {
	single := "nota/team/a.jpg"
	if got := ParseNotaKeys(&single); len(got) != 1 || got[0] != single {
		t.Fatalf("single: %#v", got)
	}

	raw := `["nota/a.jpg","nota/b.jpg"]`
	got := ParseNotaKeys(&raw)
	if len(got) != 2 || got[0] != "nota/a.jpg" || got[1] != "nota/b.jpg" {
		t.Fatalf("multi: %#v", got)
	}

	joined := JoinNotaKeys([]string{"nota/a.jpg", "nota/b.jpg"})
	if joined == nil || *joined != raw {
		t.Fatalf("join: %v", joined)
	}
	if JoinNotaKeys(nil) != nil {
		t.Fatal("empty should be nil")
	}
}

func TestHydrateNota(t *testing.T) {
	raw := `["nota/a.jpg","nota/b.jpg"]`
	tx := Transaction{NotaKey: &raw}
	tx.HydrateNota()
	if tx.NotaKey == nil || *tx.NotaKey != "nota/a.jpg" {
		t.Fatalf("first key: %v", tx.NotaKey)
	}
	if len(tx.NotaKeys) != 2 {
		t.Fatalf("keys: %#v", tx.NotaKeys)
	}
	if tx.StoredNota() == nil || *tx.StoredNota() != raw {
		t.Fatalf("stored: %v", tx.StoredNota())
	}
}
