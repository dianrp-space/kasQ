package repository

import "testing"

func TestSanitizeTxSearch(t *testing.T) {
	t.Parallel()
	if got := sanitizeTxSearch("  %foo_bar\\  "); got != "foobar" {
		t.Fatalf("sanitizeTxSearch = %q, want foobar", got)
	}
	long := make([]byte, 100)
	for i := range long {
		long[i] = 'a'
	}
	if got := sanitizeTxSearch(string(long)); len(got) != 80 {
		t.Fatalf("sanitizeTxSearch len = %d, want 80", len(got))
	}
}
