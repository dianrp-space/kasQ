package api

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kasq/backend/internal/models"
	"github.com/kasq/backend/internal/storage"
)

const maxLogoSize = 2 << 20       // 2MB
const maxFaviconSize = 512 << 10  // 512KB
const maxAvatarSize = 2 << 20     // 2MB

func (h *Handler) appSettingsResponse(c *gin.Context, s *models.AppSettings) gin.H {
	v := s.UpdatedAt.Unix()
	resp := gin.H{
		"app_name":    s.AppName,
		"app_tagline": s.AppTagline,
	}
	if s.LogoFile != nil && strings.TrimSpace(*s.LogoFile) != "" {
		resp["logo_url"] = "/api/public/branding/logo?v=" + strconv.FormatInt(v, 10)
	}
	if s.FaviconFile != nil && strings.TrimSpace(*s.FaviconFile) != "" {
		resp["favicon_url"] = "/api/public/branding/favicon?v=" + strconv.FormatInt(v, 10)
	}
	return resp
}

func (h *Handler) GetAppSettings(c *gin.Context) {
	s, err := h.repo.GetAppSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.appSettingsResponse(c, s))
}

func (h *Handler) UpdateAppSettings(c *gin.Context) {
	current, err := h.repo.GetAppSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	appName := strings.TrimSpace(c.PostForm("app_name"))
	appTagline := strings.TrimSpace(c.PostForm("app_tagline"))
	if appName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nama aplikasi wajib diisi"})
		return
	}
	if appTagline == "" {
		appTagline = current.AppTagline
	}

	logoFile := current.LogoFile
	faviconFile := current.FaviconFile
	ctx := c.Request.Context()

	if c.PostForm("remove_logo") == "true" {
		if logoFile != nil {
			_ = h.svc.DeleteAsset(ctx, *logoFile)
		}
		logoFile = nil
	}
	if c.PostForm("remove_favicon") == "true" {
		if faviconFile != nil {
			_ = h.svc.DeleteAsset(ctx, *faviconFile)
		}
		faviconFile = nil
	}

	if file, header, err := c.Request.FormFile("logo"); err == nil {
		defer file.Close()
		data, err := io.ReadAll(io.LimitReader(file, maxLogoSize+1))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "gagal baca logo"})
			return
		}
		if len(data) > maxLogoSize {
			c.JSON(http.StatusBadRequest, gin.H{"error": "logo maksimal 2MB"})
			return
		}
		contentType := header.Header.Get("Content-Type")
		if !isImageContent(contentType, header.Filename) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "logo harus berupa gambar (PNG, JPG, SVG, WEBP)"})
			return
		}
		if logoFile != nil {
			_ = h.svc.DeleteAsset(ctx, *logoFile)
		}
		key, err := h.svc.UploadBranding(ctx, "logo", data, contentType, header.Filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		logoFile = &key
	}

	if file, header, err := c.Request.FormFile("favicon"); err == nil {
		defer file.Close()
		data, err := io.ReadAll(io.LimitReader(file, maxFaviconSize+1))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "gagal baca favicon"})
			return
		}
		if len(data) > maxFaviconSize {
			c.JSON(http.StatusBadRequest, gin.H{"error": "favicon maksimal 512KB"})
			return
		}
		contentType := header.Header.Get("Content-Type")
		if !isFaviconContent(contentType, header.Filename) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "favicon harus berupa gambar (ICO, PNG, JPG, SVG)"})
			return
		}
		if faviconFile != nil {
			_ = h.svc.DeleteAsset(ctx, *faviconFile)
		}
		key, err := h.svc.UploadBranding(ctx, "favicon", data, contentType, header.Filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		faviconFile = &key
	}

	updated, err := h.repo.UpdateAppSettings(c.Request.Context(), appName, appTagline, logoFile, faviconFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.appSettingsResponse(c, updated))
}

func (h *Handler) ServeBrandingLogo(c *gin.Context) {
	h.serveBrandingFile(c, "logo")
}

func (h *Handler) ServeBrandingFavicon(c *gin.Context) {
	h.serveBrandingFile(c, "favicon")
}

func (h *Handler) serveBrandingFile(c *gin.Context, kind string) {
	s, err := h.repo.GetAppSettings(c.Request.Context())
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	var objectKey *string
	if kind == "logo" {
		objectKey = s.LogoFile
	} else {
		objectKey = s.FaviconFile
	}
	if objectKey == nil || strings.TrimSpace(*objectKey) == "" {
		c.Status(http.StatusNotFound)
		return
	}
	url, err := h.svc.GetAssetViewURL(c.Request.Context(), *objectKey)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	c.Redirect(http.StatusFound, url)
}

func isImageContent(contentType, filename string) bool {
	contentType = strings.ToLower(contentType)
	ext := strings.ToLower(storage.ObjectExt(filename, contentType))
	switch {
	case strings.HasPrefix(contentType, "image/png"),
		strings.HasPrefix(contentType, "image/jpeg"),
		strings.HasPrefix(contentType, "image/jpg"),
		strings.HasPrefix(contentType, "image/svg+xml"),
		strings.HasPrefix(contentType, "image/webp"):
		return true
	case ext == ".png", ext == ".jpg", ext == ".jpeg", ext == ".svg", ext == ".webp":
		return true
	default:
		return false
	}
}

func isFaviconContent(contentType, filename string) bool {
	contentType = strings.ToLower(contentType)
	ext := strings.ToLower(storage.ObjectExt(filename, contentType))
	switch {
	case strings.HasPrefix(contentType, "image/png"),
		strings.HasPrefix(contentType, "image/jpeg"),
		strings.HasPrefix(contentType, "image/jpg"),
		strings.HasPrefix(contentType, "image/svg+xml"),
		strings.HasPrefix(contentType, "image/x-icon"),
		strings.HasPrefix(contentType, "image/vnd.microsoft.icon"):
		return true
	case ext == ".png", ext == ".jpg", ext == ".jpeg", ext == ".svg", ext == ".ico":
		return true
	default:
		return false
	}
}

func extFromFilename(filename, contentType string) string {
	return storage.ObjectExt(filename, contentType)
}
