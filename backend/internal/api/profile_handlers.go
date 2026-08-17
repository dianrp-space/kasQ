package api

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kasq/backend/internal/models"
)

func (h *Handler) UpdateMe(c *gin.Context) {
	user, err := h.getCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	name := strings.TrimSpace(c.PostForm("name"))
	removeAvatar := c.PostForm("remove_avatar") == "true"
	avatarFile := user.AvatarFile
	ctx := c.Request.Context()

	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nama wajib diisi"})
		return
	}

	if removeAvatar {
		if avatarFile != nil {
			_ = h.svc.DeleteAsset(ctx, *avatarFile)
		}
		avatarFile = nil
	}

	if file, header, err := c.Request.FormFile("avatar"); err == nil {
		defer file.Close()
		data, err := io.ReadAll(io.LimitReader(file, maxAvatarSize+1))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "gagal baca foto profil"})
			return
		}
		if len(data) > maxAvatarSize {
			c.JSON(http.StatusBadRequest, gin.H{"error": "foto profil maksimal 2MB"})
			return
		}
		contentType := header.Header.Get("Content-Type")
		if !isImageContent(contentType, header.Filename) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "foto harus berupa gambar (PNG, JPG, WEBP)"})
			return
		}
		if avatarFile != nil {
			_ = h.svc.DeleteAsset(ctx, *avatarFile)
		}
		key, err := h.svc.UploadAvatar(ctx, user.ID, data, contentType, header.Filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		avatarFile = &key
	}

	if err := h.repo.UpdateUserProfile(ctx, user.ID, name, avatarFile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	updated, err := h.repo.GetUserByID(ctx, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.sanitizeUserWithAvatar(c, updated))
}

func (h *Handler) ChangePassword(c *gin.Context) {
	user, err := h.getCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.auth.ChangePassword(c.Request.Context(), user.ID, req.CurrentPassword, req.NewPassword); err != nil {
		if err.Error() == "password lama salah" || err.Error() == "password minimal 6 karakter" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password berhasil diubah"})
}

func (h *Handler) GetMyAvatar(c *gin.Context) {
	user, err := h.getCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if !userHasAvatar(user) {
		c.JSON(http.StatusNotFound, gin.H{"error": "avatar not available"})
		return
	}
	reader, contentType, err := h.svc.OpenAsset(c.Request.Context(), *user.AvatarFile)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "avatar not available"})
		return
	}
	defer reader.Close()
	c.Header("Cache-Control", "private, max-age=3600")
	c.DataFromReader(http.StatusOK, -1, contentType, reader, nil)
}

func userHasAvatar(u *models.User) bool {
	return u.AvatarFile != nil && strings.TrimSpace(*u.AvatarFile) != ""
}
