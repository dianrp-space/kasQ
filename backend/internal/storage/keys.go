package storage

import (
	"path/filepath"
	"strings"
)

const (
	PrefixNota     = "nota/"
	PrefixBranding = "branding/"
	PrefixAvatars  = "avatars/"
)

// NotaKeyBelongsToTeam checks legacy and current MinIO nota key formats.
func NotaKeyBelongsToTeam(key, teamID string) bool {
	key = strings.TrimSpace(key)
	teamID = strings.TrimSpace(teamID)
	if key == "" || teamID == "" {
		return false
	}
	legacyPrefix := teamID + "/"
	notaPrefix := PrefixNota + teamID + "/"
	return strings.HasPrefix(key, notaPrefix) || strings.HasPrefix(key, legacyPrefix)
}

// ResolveObjectKey maps legacy DB keys to MinIO object keys.
func ResolveObjectKey(stored string) string {
	stored = strings.TrimSpace(stored)
	if stored == "" || strings.Contains(stored, "/") {
		return stored
	}
	switch {
	case strings.HasPrefix(stored, "logo"), strings.HasPrefix(stored, "favicon"):
		return PrefixBranding + stored
	default:
		return PrefixAvatars + stored
	}
}

func ObjectExt(filename, contentType string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".svg", ".webp", ".ico", ".gif":
		return ext
	}
	switch strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0])) {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/svg+xml":
		return ".svg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "image/x-icon", "image/vnd.microsoft.icon":
		return ".ico"
	default:
		return ".jpg"
	}
}
