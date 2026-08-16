package service

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestTryReadLocalNotaFile(t *testing.T) {
	dir := t.TempDir()
	media := filepath.Join(dir, "statics", "media")
	if err := os.MkdirAll(media, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(media, "sample.jpeg")
	if err := os.WriteFile(file, []byte("fake-image"), 0o644); err != nil {
		t.Fatal(err)
	}

	roots := map[string]string{"bon.dianrp.com": dir}
	u, _ := url.Parse("https://bon.dianrp.com/statics/media/sample.jpeg")

	body, name, _, ok := tryReadLocalNotaFile(u, roots)
	if !ok {
		t.Fatal("expected local file hit")
	}
	if name != "sample.jpeg" || string(body) != "fake-image" {
		t.Fatalf("unexpected read: name=%q body=%q", name, body)
	}

	uTraversal, _ := url.Parse("https://bon.dianrp.com/statics/media/../secret")
	if _, _, _, ok := tryReadLocalNotaFile(uTraversal, roots); ok {
		t.Fatal("path traversal should be blocked")
	}
}
