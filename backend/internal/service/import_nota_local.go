package service

import (
	"mime"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const maxImportNotaBytes = 10 << 20 // 10MB

// IMPORT_NOTA_LOCAL_ROOTS maps public host → docroot on the same server.
// Example: bon.dianrp.com:/www/wwwroot/bon.dianrp.com
// Bypasses hairpin NAT when KasQ and the nota site share one VPS.
func loadImportNotaLocalRoots() map[string]string {
	raw := strings.TrimSpace(os.Getenv("IMPORT_NOTA_LOCAL_ROOTS"))
	if raw == "" {
		return nil
	}
	out := make(map[string]string)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		host, root, ok := strings.Cut(part, ":")
		if !ok || strings.TrimSpace(host) == "" || strings.TrimSpace(root) == "" {
			continue
		}
		host = strings.ToLower(strings.TrimSpace(host))
		root = filepath.Clean(strings.TrimSpace(root))
		out[host] = root
	}
	return out
}

func tryReadLocalNotaFile(u *url.URL, roots map[string]string) (body []byte, filename, contentType string, ok bool) {
	if len(roots) == 0 || u == nil {
		return nil, "", "", false
	}
	host := strings.ToLower(u.Hostname())
	root, exists := roots[host]
	if !exists {
		return nil, "", "", false
	}

	rel := strings.TrimPrefix(path.Clean(u.Path), "/")
	if rel == "" || rel == "." || strings.Contains(rel, "..") {
		return nil, "", "", false
	}

	full := filepath.Join(root, filepath.FromSlash(rel))
	rootPrefix := root + string(os.PathSeparator)
	if full != root && !strings.HasPrefix(full, rootPrefix) {
		return nil, "", "", false
	}

	info, err := os.Stat(full)
	if err != nil || info.IsDir() {
		return nil, "", "", false
	}
	if info.Size() > maxImportNotaBytes {
		return nil, "", "", false
	}

	data, err := os.ReadFile(full)
	if err != nil {
		return nil, "", "", false
	}

	filename = filepath.Base(full)
	if filename == "" || filename == "." {
		filename = "import-nota.jpg"
	}
	contentType = mime.TypeByExtension(filepath.Ext(filename))
	return data, filename, contentType, true
}
