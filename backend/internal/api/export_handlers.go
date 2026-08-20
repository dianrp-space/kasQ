package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kasq/backend/internal/models"
	"github.com/kasq/backend/internal/repository"
	"github.com/kasq/backend/internal/service"
)

const maxExportRows = 10000

func (h *Handler) ExportTransactions(c *gin.Context) {
	teamID, err := parseTeamID(c)
	if err != nil || !h.canAccessTeam(c, teamID) {
		h.respondTeamForbidden(c)
		return
	}
	team, err := h.repo.GetTeam(c.Request.Context(), teamID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tim tidak ditemukan"})
		return
	}
	h.serveTxExport(c, team)
}

func (h *Handler) ExportPublicReport(c *gin.Context) {
	token := c.Param("token")
	team, _, err := h.repo.GetReportByToken(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "laporan tidak ditemukan"})
		return
	}
	h.serveTxExport(c, team)
}

func (h *Handler) serveTxExport(c *gin.Context, team *models.Team) {
	format, err := service.ParseExportFormat(c.Query("format"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	filter := txFilterFromQuery(c, team.ID, maxExportRows)
	txs, err := h.repo.ListTransactions(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if txs == nil {
		txs = []models.Transaction{}
	}
	dateFrom, dateTo := parseDateRange(c)
	balance, err := h.repo.GetBalance(c.Request.Context(), team.ID, dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	appName := "KasQ"
	if s, err := h.repo.GetAppSettings(c.Request.Context()); err == nil && s.AppName != "" {
		appName = s.AppName
	}
	period := service.FormatPeriodLabel(dateFrom, dateTo)
	data, mime, err := service.BuildExport(service.ExportReport{
		AppName:  appName,
		TeamName: team.Name,
		TeamSlug: team.Slug,
		Period:   period,
		Balance:  balance,
		Items:    txs,
	}, format)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	filename := service.ExportFilename(team.Slug, period, format)
	c.Header("Content-Type", mime)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Cache-Control", "private, no-store")
	c.Data(http.StatusOK, mime, data)
}

func txFilterFromQuery(c *gin.Context, teamID uuid.UUID, limit int) repository.TxFilter {
	filter := repository.TxFilter{TeamID: teamID, Limit: limit}
	if j := c.Query("jenis"); j != "" {
		jn := models.TxJenis(j)
		filter.Jenis = &jn
	}
	if df := c.Query("date_from"); df != "" {
		if t, err := time.Parse("2006-01-02", df); err == nil {
			filter.DateFrom = &t
		}
	}
	if dt := c.Query("date_to"); dt != "" {
		if t, err := time.Parse("2006-01-02", dt); err == nil {
			filter.DateTo = &t
		}
	}
	if q := c.Query("q"); q != "" {
		filter.Search = q
	}
	return filter
}
