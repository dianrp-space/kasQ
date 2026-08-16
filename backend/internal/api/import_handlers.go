package api

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kasq/backend/internal/middleware"
	"github.com/kasq/backend/internal/service"
)

const maxImportFileSize = 5 << 20 // 5MB

func (h *Handler) ImportTransactions(c *gin.Context) {
	teamID, err := parseTeamID(c)
	if err != nil || !h.canAccessTeam(c, teamID) {
		h.respondTeamForbidden(c)
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file excel wajib (field: file)"})
		return
	}
	defer file.Close()

	if header.Size > maxImportFileSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file terlalu besar (max 5MB)"})
		return
	}

	data, err := io.ReadAll(io.LimitReader(file, maxImportFileSize+1))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal baca file"})
		return
	}
	if len(data) > maxImportFileSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file terlalu besar (max 5MB)"})
		return
	}

	fetchNota := c.DefaultPostForm("fetch_nota", "true") != "false"
	userID := middleware.GetUserID(c)

	result, err := h.svc.ImportTransactionsFromExcel(c.Request.Context(), teamID, userID, data, fetchNota)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) ImportTemplate(c *gin.Context) {
	teamID, err := parseTeamID(c)
	if err != nil || !h.canAccessTeam(c, teamID) {
		h.respondTeamForbidden(c)
		return
	}

	data, err := service.BuildImportTemplate()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Content-Disposition", `attachment; filename="kasq-import-template.xlsx"`)
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}
