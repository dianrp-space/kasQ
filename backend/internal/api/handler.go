package api

import (
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kasq/backend/internal/bot/telegram"
	"github.com/kasq/backend/internal/bot/whatsapp"
	"github.com/kasq/backend/internal/middleware"
	"github.com/kasq/backend/internal/models"
	"github.com/kasq/backend/internal/repository"
	"github.com/kasq/backend/internal/service"
	"github.com/kasq/backend/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	repo        *repository.Repository
	svc         *service.Service
	auth        *service.AuthService
	jwtSecret   string
	appURL      string
	waManager   WAStarter
	teleManager TeleStarter
}

type WAStarter interface {
	StartTeam(teamID uuid.UUID) error
	StartQRLogin(teamID uuid.UUID) error
	StartPairLogin(teamID uuid.UUID, phone string) (string, error)
	StopTeam(teamID uuid.UUID)
	GetStatus(teamID uuid.UUID) (whatsapp.ConnectStatus, error)
	GetWAProfile(teamID uuid.UUID) (name, pictureURL string)
	OpenWAAvatar(teamID uuid.UUID) (io.ReadCloser, string, error)
}

type TeleStarter interface {
	StartTeam(teamID uuid.UUID, token string) error
	StopTeam(teamID uuid.UUID)
	GetBotProfile(teamID uuid.UUID) telegram.BotProfile
	OpenBotAvatar(teamID uuid.UUID) (io.ReadCloser, string, error)
	SystemBotAvailable() bool
	SystemBotProfile() telegram.BotProfile
	IsSystemToken(token string) bool
}

func NewHandler(repo *repository.Repository, svc *service.Service, auth *service.AuthService, jwtSecret, appURL string, wa WAStarter, tele TeleStarter) *Handler {
	return &Handler{repo: repo, svc: svc, auth: auth, jwtSecret: jwtSecret, appURL: appURL, waManager: wa, teleManager: tele}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.Use(corsMiddleware())

	api := r.Group("/api")
	{
		api.GET("/health", h.Health)
		api.POST("/auth/login", h.Login)
		api.POST("/auth/logout", h.Logout)
		api.POST("/auth/register", h.Register)
		api.GET("/auth/verify-email", h.VerifyEmail)
		api.POST("/auth/forgot-password", h.ForgotPassword)
		api.POST("/auth/reset-password", h.ResetPassword)
		api.POST("/auth/resend-verification", h.ResendVerification)
		api.GET("/app-settings", h.GetAppSettings)

		public := api.Group("/public")
		{
			public.GET("/report/:token", h.PublicReport)
			public.GET("/report/:token/export", h.ExportPublicReport)
			public.GET("/nota/:token", h.PublicNota)
			public.GET("/nota/:token/file", h.ServePublicNota)
			public.GET("/branding/logo", h.ServeBrandingLogo)
			public.GET("/branding/favicon", h.ServeBrandingFavicon)
		}

		protected := api.Group("")
		protected.Use(middleware.Auth(h.jwtSecret))
		{
			protected.GET("/me", h.Me)
			protected.PUT("/me", h.UpdateMe)
			protected.PUT("/me/password", h.ChangePassword)
			protected.GET("/me/avatar", h.GetMyAvatar)

			protected.GET("/teams", h.ListTeams)
			protected.GET("/teams/:id/balance", h.GetBalance)
			protected.GET("/teams/:id/transactions", h.ListTransactions)
			protected.GET("/teams/:id/transactions/export", h.ExportTransactions)
			protected.POST("/teams/:id/transactions", h.CreateTransaction)
			protected.POST("/teams/:id/transactions/import", h.ImportTransactions)
			protected.GET("/teams/:id/transactions/import/template", h.ImportTemplate)
			protected.PUT("/teams/:id/transactions/reorder", h.ReorderTransactions)
			protected.PUT("/teams/:id/transactions/:txId", h.UpdateTransaction)
			protected.DELETE("/teams/:id/transactions/:txId", h.DeleteTransaction)
			protected.POST("/teams/:id/transactions/batch-delete", h.BatchDeleteTransactions)
			protected.GET("/teams/:id/integrations", h.GetIntegration)
			protected.PUT("/teams/:id/integrations/wa", h.UpdateWA)
			protected.PUT("/teams/:id/integrations/wa/allowed-phones", h.UpdateWAAllowedPhones)
			protected.POST("/teams/:id/integrations/wa/qr/start", h.StartWAQRLogin)
			protected.POST("/teams/:id/integrations/wa/pair", h.StartWAPairLogin)
			protected.GET("/teams/:id/integrations/wa/qr", h.GetWAQR)
			protected.GET("/teams/:id/integrations/wa/avatar", h.GetWAAvatar)
			protected.PUT("/teams/:id/integrations/tele", h.UpdateTele)
			protected.GET("/teams/:id/integrations/tele/avatar", h.GetTeleBotAvatar)
			protected.PUT("/teams/:id/report-token", h.UpdateReportToken)
			protected.POST("/teams/:id/report-token/reset", h.ResetReportToken)
			protected.GET("/teams/:id/report-token", h.GetReportToken)
			protected.GET("/teams/:id/nota-url", h.GetNotaURL)
			protected.GET("/teams/:id/nota", h.ServeTeamNota)

			admin := protected.Group("")
			admin.Use(middleware.RequireAdmin())
			{
				admin.POST("/teams", h.CreateTeam)
				admin.PUT("/teams/:id", h.UpdateTeam)
				admin.DELETE("/teams/:id", h.DeleteTeam)
				admin.GET("/users", h.ListUsers)
				admin.POST("/users", h.CreateUser)
				admin.PUT("/users/:id", h.UpdateUser)
				admin.DELETE("/users/:id", h.DeleteUser)
				admin.PUT("/admin/app-settings", h.UpdateAppSettings)
			}
		}
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", c.GetHeader("Origin"))
		if origin := c.GetHeader("Origin"); origin == "" {
			c.Header("Access-Control-Allow-Origin", "http://localhost:3008")
		}
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

type loginReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *Handler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, err := h.repo.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "email atau password salah"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "email atau password salah"})
		return
	}
	if !user.EmailVerified {
		c.JSON(http.StatusForbidden, gin.H{
			"error":              "email belum diverifikasi",
			"needs_verification": true,
		})
		return
	}
	token, err := middleware.SignToken(user, h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal membuat token"})
		return
	}
	c.SetCookie(middleware.CookieName, token, 86400, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"user": h.sanitizeUserWithAvatar(c, user), "token": token})
}

func (h *Handler) Logout(c *gin.Context) {
	c.SetCookie(middleware.CookieName, "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.repo.GetUserByID(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, h.sanitizeUserWithAvatar(c, user))
}

func sanitizeUser(u *models.User) gin.H {
	resp := gin.H{
		"id":      u.ID,
		"name":    u.Name,
		"email":   u.Email,
		"role":    u.Role,
		"team_id": u.TeamID,
	}
	return resp
}

func (h *Handler) sanitizeUserWithAvatar(c *gin.Context, u *models.User) gin.H {
	resp := sanitizeUser(u)
	if userHasAvatar(u) {
		resp["has_avatar"] = true
	}
	return resp
}

func (h *Handler) getCurrentUser(c *gin.Context) (*models.User, error) {
	if cached, ok := c.Get("_current_user"); ok {
		return cached.(*models.User), nil
	}
	user, err := h.repo.GetUserByID(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		return nil, err
	}
	c.Set("_current_user", user)
	return user, nil
}

func (h *Handler) canAccessTeam(c *gin.Context, teamID uuid.UUID) bool {
	if middleware.GetUserRole(c) != models.RoleOps {
		return false
	}
	user, err := h.getCurrentUser(c)
	if err != nil || user.TeamID == nil {
		return false
	}
	return *user.TeamID == teamID
}

func (h *Handler) respondTeamForbidden(c *gin.Context) {
	if middleware.GetUserRole(c) == models.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "admin tidak memiliki akses transaksi — gunakan akun ops",
		})
		return
	}
	if middleware.GetUserRole(c) == models.RoleOps {
		user, err := h.getCurrentUser(c)
		if err == nil && user.TeamID == nil {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Akun belum ditugaskan ke tim/kas. Minta admin menetapkan tim/kas kamu.",
				"no_team": true,
			})
			return
		}
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
}

func parseTeamID(c *gin.Context) (uuid.UUID, error) {
	return uuid.Parse(c.Param("id"))
}

func parseTxID(c *gin.Context) (uuid.UUID, error) {
	return uuid.Parse(c.Param("txId"))
}

func parseDateRange(c *gin.Context) (from, to *time.Time) {
	if df := c.Query("date_from"); df != "" {
		if t, err := time.Parse("2006-01-02", df); err == nil {
			from = &t
		}
	}
	if dt := c.Query("date_to"); dt != "" {
		if t, err := time.Parse("2006-01-02", dt); err == nil {
			to = &t
		}
	}
	return from, to
}

func (h *Handler) ListTeams(c *gin.Context) {
	teams, err := h.repo.ListTeams(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if middleware.GetUserRole(c) == models.RoleOps {
		user, err := h.getCurrentUser(c)
		if err != nil || user.TeamID == nil {
			c.JSON(http.StatusOK, []models.Team{})
			return
		}
		tid := user.TeamID
		filtered := make([]models.Team, 0)
		for _, t := range teams {
			if t.ID == *tid {
				filtered = append(filtered, t)
			}
		}
		c.JSON(http.StatusOK, filtered)
		return
	}
	c.JSON(http.StatusOK, teams)
}

func (h *Handler) CreateTeam(c *gin.Context) {
	var req struct {
		Name           string `json:"name" binding:"required"`
		Slug           string `json:"slug"`
		InitialBalance int64  `json:"initial_balance"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	slug := req.Slug
	if slug == "" {
		slug = service.Slugify(req.Name)
	}
	team := &models.Team{Name: req.Name, Slug: slug, InitialBalance: req.InitialBalance}
	if err := h.repo.CreateTeam(c.Request.Context(), team); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, team)
}

func (h *Handler) UpdateTeam(c *gin.Context) {
	teamID, err := parseTeamID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}
	var req struct {
		Name           string `json:"name" binding:"required"`
		Slug           string `json:"slug" binding:"required"`
		InitialBalance int64  `json:"initial_balance"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	team := &models.Team{ID: teamID, Name: req.Name, Slug: req.Slug, InitialBalance: req.InitialBalance}
	if err := h.repo.UpdateTeam(c.Request.Context(), team); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, team)
}

func (h *Handler) DeleteTeam(c *gin.Context) {
	teamID, err := parseTeamID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}
	if err := h.repo.DeleteTeam(c.Request.Context(), teamID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) GetBalance(c *gin.Context) {
	teamID, err := parseTeamID(c)
	if err != nil || !h.canAccessTeam(c, teamID) {
		h.respondTeamForbidden(c)
		return
	}
	dateFrom, dateTo := parseDateRange(c)
	balance, err := h.repo.GetBalance(c.Request.Context(), teamID, dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, balance)
}

func (h *Handler) ListTransactions(c *gin.Context) {
	teamID, err := parseTeamID(c)
	if err != nil || !h.canAccessTeam(c, teamID) {
		h.respondTeamForbidden(c)
		return
	}
	limit := 20
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			if n > 5000 {
				n = 5000
			}
			limit = n
		}
	}
	page := 1
	if p := c.Query("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}
	filter := repository.TxFilter{TeamID: teamID, Limit: limit, Offset: (page - 1) * limit}
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
	if q := strings.TrimSpace(c.Query("q")); q != "" {
		filter.Search = q
	}
	total, err := h.repo.CountTransactions(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	txs, err := h.repo.ListTransactions(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if txs == nil {
		txs = []models.Transaction{}
	}
	c.JSON(http.StatusOK, gin.H{
		"items":  txs,
		"total":  total,
		"limit":  limit,
		"page":   page,
		"offset": filter.Offset,
	})
}

func (h *Handler) CreateTransaction(c *gin.Context) {
	teamID, err := parseTeamID(c)
	if err != nil || !h.canAccessTeam(c, teamID) {
		h.respondTeamForbidden(c)
		return
	}

	contentType := c.GetHeader("Content-Type")
	if len(contentType) >= 19 && contentType[:19] == "multipart/form-data" {
		hari := c.PostForm("hari")
		tanggalStr := c.PostForm("tanggal")
		jenis := models.TxJenis(c.PostForm("jenis"))
		deskripsi := c.PostForm("deskripsi")
		totalStr := c.PostForm("total")
		keterangan := c.PostForm("keterangan")

		tanggal, err := time.Parse("2006-01-02", tanggalStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tanggal invalid"})
			return
		}
		total, err := strconv.ParseInt(totalStr, 10, 64)
		if err != nil || total <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "total invalid"})
			return
		}

		notaKeys, upErr := h.uploadNotaFiles(c, teamID)
		if upErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": upErr.Error()})
			return
		}

		var ket *string
		if keterangan != "" {
			ket = &keterangan
		}
		userID := middleware.GetUserID(c)
		tx, balance, err := h.svc.CreateTransactionFromWeb(c.Request.Context(), teamID, userID, service.CreateWebTxInput{
			Hari: hari, Tanggal: tanggal, Jenis: jenis, Deskripsi: deskripsi, Total: total, NotaKeys: notaKeys, Keterangan: ket,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"transaction": tx, "balance": balance})
		return
	}

	var req struct {
		Hari       string         `json:"hari" binding:"required"`
		Tanggal    string         `json:"tanggal" binding:"required"`
		Jenis      models.TxJenis `json:"jenis" binding:"required"`
		Deskripsi  string         `json:"deskripsi" binding:"required"`
		Total      int64          `json:"total" binding:"required"`
		Keterangan *string        `json:"keterangan"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tanggal, err := time.Parse("2006-01-02", req.Tanggal)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tanggal invalid"})
		return
	}
	userID := middleware.GetUserID(c)
	tx, balance, err := h.svc.CreateTransactionFromWeb(c.Request.Context(), teamID, userID, service.CreateWebTxInput{
		Hari: req.Hari, Tanggal: tanggal, Jenis: req.Jenis, Deskripsi: req.Deskripsi, Total: req.Total, Keterangan: req.Keterangan,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"transaction": tx, "balance": balance})
}

func (h *Handler) UpdateTransaction(c *gin.Context) {
	teamID, err := parseTeamID(c)
	if err != nil || !h.canAccessTeam(c, teamID) {
		h.respondTeamForbidden(c)
		return
	}
	txID, err := parseTxID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid transaction id"})
		return
	}

	contentType := c.GetHeader("Content-Type")
	if len(contentType) >= 19 && contentType[:19] == "multipart/form-data" {
		hari := c.PostForm("hari")
		tanggalStr := c.PostForm("tanggal")
		jenis := models.TxJenis(c.PostForm("jenis"))
		deskripsi := c.PostForm("deskripsi")
		totalStr := c.PostForm("total")
		keterangan := c.PostForm("keterangan")
		removeNota := c.PostForm("remove_nota") == "true"

		tanggal, err := time.Parse("2006-01-02", tanggalStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tanggal invalid"})
			return
		}
		total, err := strconv.ParseInt(totalStr, 10, 64)
		if err != nil || total <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "total invalid"})
			return
		}

		var notaReplace []string
		files, err := h.uploadNotaFiles(c, teamID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if len(files) > 0 {
			notaReplace = files
			removeNota = false
		}

		var ket *string
		if keterangan != "" {
			ket = &keterangan
		}
		tx, balance, err := h.svc.UpdateTransaction(c.Request.Context(), teamID, txID, service.UpdateWebTxInput{
			Hari: hari, Tanggal: tanggal, Jenis: jenis, Deskripsi: deskripsi, Total: total,
			Keterangan: ket, NotaReplace: notaReplace, RemoveNota: removeNota,
		})
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "transaksi tidak ditemukan"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"transaction": tx, "balance": balance})
		return
	}

	var req struct {
		Hari       string         `json:"hari" binding:"required"`
		Tanggal    string         `json:"tanggal" binding:"required"`
		Jenis      models.TxJenis `json:"jenis" binding:"required"`
		Deskripsi  string         `json:"deskripsi" binding:"required"`
		Total      int64          `json:"total" binding:"required"`
		Keterangan *string        `json:"keterangan"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tanggal, err := time.Parse("2006-01-02", req.Tanggal)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tanggal invalid"})
		return
	}
	if req.Total <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "total invalid"})
		return
	}
	tx, balance, err := h.svc.UpdateTransaction(c.Request.Context(), teamID, txID, service.UpdateWebTxInput{
		Hari: req.Hari, Tanggal: tanggal, Jenis: req.Jenis, Deskripsi: req.Deskripsi, Total: req.Total,
		Keterangan: req.Keterangan,
	})
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "transaksi tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"transaction": tx, "balance": balance})
}

func (h *Handler) DeleteTransaction(c *gin.Context) {
	teamID, err := parseTeamID(c)
	if err != nil || !h.canAccessTeam(c, teamID) {
		h.respondTeamForbidden(c)
		return
	}
	txID, err := parseTxID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid transaction id"})
		return
	}
	balance, err := h.svc.DeleteTransaction(c.Request.Context(), teamID, txID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "transaksi tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "balance": balance})
}

func (h *Handler) BatchDeleteTransactions(c *gin.Context) {
	teamID, err := parseTeamID(c)
	if err != nil || !h.canAccessTeam(c, teamID) {
		h.respondTeamForbidden(c)
		return
	}
	var body struct {
		IDs []string `json:"ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids wajib (array UUID transaksi)"})
		return
	}
	txIDs := make([]uuid.UUID, 0, len(body.IDs))
	for _, raw := range body.IDs {
		id, err := uuid.Parse(strings.TrimSpace(raw))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id transaksi tidak valid: " + raw})
			return
		}
		txIDs = append(txIDs, id)
	}
	result, err := h.svc.BatchDeleteTransactions(c.Request.Context(), teamID, txIDs)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "transaksi tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "deleted": result.Deleted, "balance": result.Balance})
}

func (h *Handler) ReorderTransactions(c *gin.Context) {
	teamID, err := parseTeamID(c)
	if err != nil || !h.canAccessTeam(c, teamID) {
		h.respondTeamForbidden(c)
		return
	}
	var body struct {
		IDs []string `json:"ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.IDs) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids wajib minimal 2 UUID transaksi"})
		return
	}
	txIDs := make([]uuid.UUID, 0, len(body.IDs))
	for _, raw := range body.IDs {
		id, err := uuid.Parse(strings.TrimSpace(raw))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id transaksi tidak valid: " + raw})
			return
		}
		txIDs = append(txIDs, id)
	}
	if err := h.repo.ReorderTransactions(c.Request.Context(), teamID, txIDs); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "transaksi tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) GetNotaURL(c *gin.Context) {
	teamID, err := parseTeamID(c)
	if err != nil || !h.canAccessTeam(c, teamID) {
		h.respondTeamForbidden(c)
		return
	}
	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key required"})
		return
	}
	if !storage.NotaKeyBelongsToTeam(key, teamID.String()) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	download := c.Query("download") == "true"
	u, err := h.svc.GetNotaURL(c.Request.Context(), key, download)
	if err != nil {
		log.Printf("nota: url team=%s key=%q: %v", teamID, key, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "nota not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": u})
}

func (h *Handler) ServeTeamNota(c *gin.Context) {
	teamID, err := parseTeamID(c)
	if err != nil || !h.canAccessTeam(c, teamID) {
		h.respondTeamForbidden(c)
		return
	}
	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key required"})
		return
	}
	if !storage.NotaKeyBelongsToTeam(key, teamID.String()) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	h.serveNotaObject(c, key, c.Query("download") == "true")
}

func (h *Handler) serveNotaObject(c *gin.Context, key string, download bool) {
	if u, err := h.svc.GetNotaURL(c.Request.Context(), key, download); err == nil {
		c.Redirect(http.StatusTemporaryRedirect, u)
		return
	}
	reader, contentType, err := h.svc.OpenNota(c.Request.Context(), key)
	if err != nil {
		log.Printf("nota: serve key=%q: %v", key, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "nota not found"})
		return
	}
	defer reader.Close()
	if download {
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, service.NotaFilename(key)))
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.DataFromReader(http.StatusOK, -1, contentType, reader, nil)
}

func (h *Handler) ListUsers(c *gin.Context) {
	users, err := h.repo.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	result := make([]gin.H, 0, len(users))
	for _, u := range users {
		result = append(result, sanitizeUser(&u))
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) CreateUser(c *gin.Context) {
	var req struct {
		Name     string          `json:"name" binding:"required"`
		Email    string          `json:"email" binding:"required,email"`
		Password string          `json:"password" binding:"required,min=6"`
		Role     models.UserRole `json:"role" binding:"required"`
		TeamID   *uuid.UUID      `json:"team_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Role == models.RoleOps && req.TeamID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user ops wajib ditugaskan ke tim/kas"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	user := &models.User{Name: req.Name, Email: req.Email, PasswordHash: string(hash), Role: req.Role, TeamID: req.TeamID, EmailVerified: true}
	if err := h.repo.CreateUser(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, sanitizeUser(user))
}

func (h *Handler) UpdateUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Name     string          `json:"name" binding:"required"`
		Email    string          `json:"email" binding:"required,email"`
		Password string          `json:"password"`
		Role     models.UserRole `json:"role" binding:"required"`
		TeamID   *uuid.UUID      `json:"team_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Role == models.RoleOps && req.TeamID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user ops wajib ditugaskan ke tim/kas"})
		return
	}
	existing, err := h.repo.GetUserByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	existing.Name = req.Name
	existing.Email = req.Email
	existing.Role = req.Role
	existing.TeamID = req.TeamID
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		existing.PasswordHash = string(hash)
	}
	if err := h.repo.UpdateUser(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sanitizeUser(existing))
}

func (h *Handler) DeleteUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.repo.DeleteUser(c.Request.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) GetIntegration(c *gin.Context) {
	teamID, err := parseTeamID(c)
	if err != nil || !h.canAccessTeam(c, teamID) {
		h.respondTeamForbidden(c)
		return
	}
	integ, err := h.repo.GetIntegration(c.Request.Context(), teamID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	team, _ := h.repo.GetTeam(c.Request.Context(), teamID)
	rt, _ := h.repo.GetReportToken(c.Request.Context(), teamID)
	waStatus := integ.WAStatus
	if integ.WAEnabled && h.waManager != nil {
		if live, err := h.waManager.GetStatus(teamID); err == nil && live.Status != "" {
			waStatus = live.Status
		}
	}

	resp := gin.H{
		"team_id":                   integ.TeamID,
		"wa_enabled":                integ.WAEnabled,
		"wa_status":                 waStatus,
		"wa_phone":                  integ.WAPhone,
		"wa_name":                   integ.WAName,
		"wa_allowed_phones":         integ.WAAllowedPhones,
		"tele_enabled":              integ.TeleEnabled,
		"tele_use_system_bot":       integ.TeleUseSystemBot,
		"tele_allowed_chat_id":      integ.TeleAllowedChatID,
		"has_tele_token":            !integ.TeleUseSystemBot && integ.TeleBotToken != nil && *integ.TeleBotToken != "",
		"system_tele_bot_available": false,
	}
	if team != nil {
		resp["team_slug"] = team.Slug
		resp["team_name"] = team.Name
	}
	if integ.TeleBotToken != nil && !integ.TeleUseSystemBot {
		resp["tele_bot_token"] = *integ.TeleBotToken
	}
	if rt != nil {
		resp["report_token"] = rt.Token
		resp["report_url"] = h.appURL + "/report/" + rt.Token
	}
	if waStatus == "connected" && h.waManager != nil {
		name, pictureURL := h.waManager.GetWAProfile(teamID)
		if name != "" {
			resp["wa_name"] = name
		}
		if pictureURL != "" {
			resp["wa_picture_url"] = pictureURL
			resp["wa_has_avatar"] = true
		}
	}
	if h.teleManager != nil {
		resp["system_tele_bot_available"] = h.teleManager.SystemBotAvailable()
		if sys := h.teleManager.SystemBotProfile(); sys.Username != "" || sys.Name != "" {
			if sys.Name != "" {
				resp["system_tele_bot_name"] = sys.Name
			}
			if sys.Username != "" {
				resp["system_tele_bot_username"] = sys.Username
			}
			if sys.HasAvatar {
				resp["system_tele_bot_has_avatar"] = true
			}
		}
	}
	if integ.TeleEnabled && h.teleManager != nil {
		profile := h.teleManager.GetBotProfile(teamID)
		if profile.Name != "" {
			resp["tele_bot_name"] = profile.Name
		}
		if profile.Username != "" {
			resp["tele_bot_username"] = profile.Username
		}
		if profile.HasAvatar {
			resp["tele_bot_has_avatar"] = true
		}
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) UpdateWA(c *gin.Context) {
	teamID, err := parseTeamID(c)
	if err != nil || !h.canAccessTeam(c, teamID) {
		h.respondTeamForbidden(c)
		return
	}
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Enabled {
		if err := h.repo.UpdateWAIntegration(c.Request.Context(), teamID, true, "connecting", nil, nil, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if h.waManager != nil {
			if err := h.waManager.StartTeam(teamID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
	} else {
		if h.waManager != nil {
			h.waManager.StopTeam(teamID)
		}
		if err := h.repo.UpdateWAIntegration(c.Request.Context(), teamID, false, "disconnected", nil, nil, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) UpdateWAAllowedPhones(c *gin.Context) {
	teamID, err := parseTeamID(c)
	if err != nil || !h.canAccessTeam(c, teamID) {
		h.respondTeamForbidden(c)
		return
	}
	var req struct {
		Phones []string `json:"phones"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	phones := models.JoinWAAllowedPhones(req.Phones)
	var list []string
	if phones != nil {
		list = models.ParseWAAllowedPhones(phones)
	}
	if err := h.repo.UpdateWAAllowedPhones(c.Request.Context(), teamID, list); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "phones": list})
}

func (h *Handler) StartWAQRLogin(c *gin.Context) {
	teamID, err := parseTeamID(c)
	if err != nil || !h.canAccessTeam(c, teamID) {
		h.respondTeamForbidden(c)
		return
	}
	if h.waManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "WA not available"})
		return
	}
	if err := h.waManager.StartQRLogin(teamID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) StartWAPairLogin(c *gin.Context) {
	teamID, err := parseTeamID(c)
	if err != nil || !h.canAccessTeam(c, teamID) {
		h.respondTeamForbidden(c)
		return
	}
	if h.waManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "WA not available"})
		return
	}
	var req struct {
		Phone string `json:"phone" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	code, err := h.waManager.StartPairLogin(teamID, req.Phone)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"pair_code":       code,
		"expires_seconds": 60,
		"status":          "pair_code",
	})
}

func (h *Handler) GetWAQR(c *gin.Context) {
	teamID, err := parseTeamID(c)
	if err != nil || !h.canAccessTeam(c, teamID) {
		h.respondTeamForbidden(c)
		return
	}
	if h.waManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "WA not available"})
		return
	}
	status, err := h.waManager.GetStatus(teamID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":                    status.Status,
		"qr":                        status.QR,
		"phone":                     status.Phone,
		"wa_name":                   status.DisplayName,
		"wa_picture_url":            status.PictureURL,
		"pair_code":                 status.PairCode,
		"qr_timeout_seconds":        status.QRTimeoutSeconds,
		"pair_code_expires_seconds": status.PairCodeExpiresSec,
		"login_mode":                status.LoginMode,
	})
}

func (h *Handler) UpdateTele(c *gin.Context) {
	teamID, err := parseTeamID(c)
	if err != nil || !h.canAccessTeam(c, teamID) {
		h.respondTeamForbidden(c)
		return
	}
	var req struct {
		Enabled      bool    `json:"enabled"`
		UseSystemBot *bool   `json:"use_system_bot"`
		BotToken     *string `json:"bot_token"`
		ChatID       *int64  `json:"chat_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	integ, err := h.repo.GetIntegration(c.Request.Context(), teamID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	useSystem := integ.TeleUseSystemBot
	if req.UseSystemBot != nil {
		useSystem = *req.UseSystemBot
	}

	token := req.BotToken
	if token == nil || strings.TrimSpace(*token) == "" {
		token = integ.TeleBotToken
	}
	if token != nil {
		trimmed := strings.TrimSpace(*token)
		token = &trimmed
	}
	chatID := req.ChatID
	if chatID == nil {
		chatID = integ.TeleAllowedChatID
	}
	if chatID != nil && *chatID == 0 {
		chatID = nil
	}

	if req.Enabled {
		if useSystem {
			if h.teleManager == nil || !h.teleManager.SystemBotAvailable() {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Bot KasQ belum dikonfigurasi. Hubungi admin atau gunakan bot sendiri."})
				return
			}
			if chatID == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Chat ID wajib untuk Bot KasQ. Kirim /start ke bot untuk mendapat Chat ID."})
				return
			}
			if h.teleManager != nil {
				h.teleManager.StopTeam(teamID)
			}
			if err := h.repo.UpdateTeleIntegration(c.Request.Context(), teamID, true, true, nil, chatID); err != nil {
				if errors.Is(err, repository.ErrChatIDTaken) {
					c.JSON(http.StatusConflict, gin.H{"error": "Chat ID ini sudah dipakai tim/kas lain pada Bot KasQ"})
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		} else {
			if token == nil || *token == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "bot_token required"})
				return
			}
			if h.teleManager != nil && h.teleManager.IsSystemToken(*token) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Token bot sistem tidak bisa dipakai di opsi bot sendiri. Pilih Bot KasQ."})
				return
			}
			if err := h.repo.UpdateTeleIntegration(c.Request.Context(), teamID, true, false, token, chatID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if h.teleManager != nil {
				if err := h.teleManager.StartTeam(teamID, *token); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal start bot: " + err.Error()})
					return
				}
			}
		}
	} else {
		if h.teleManager != nil {
			h.teleManager.StopTeam(teamID)
		}
		saveToken := token
		if useSystem {
			saveToken = nil
		}
		if err := h.repo.UpdateTeleIntegration(c.Request.Context(), teamID, false, useSystem, saveToken, chatID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) GetWAAvatar(c *gin.Context) {
	teamID, err := parseTeamID(c)
	if err != nil || !h.canAccessTeam(c, teamID) {
		h.respondTeamForbidden(c)
		return
	}
	if h.waManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "WhatsApp not available"})
		return
	}
	reader, contentType, err := h.waManager.OpenWAAvatar(teamID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "avatar not available"})
		return
	}
	defer reader.Close()
	c.Header("Cache-Control", "private, max-age=3600")
	c.DataFromReader(http.StatusOK, -1, contentType, reader, nil)
}

func (h *Handler) GetTeleBotAvatar(c *gin.Context) {
	teamID, err := parseTeamID(c)
	if err != nil || !h.canAccessTeam(c, teamID) {
		h.respondTeamForbidden(c)
		return
	}
	if h.teleManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Telegram not available"})
		return
	}
	reader, contentType, err := h.teleManager.OpenBotAvatar(teamID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "avatar not available"})
		return
	}
	defer reader.Close()
	c.Header("Cache-Control", "private, max-age=3600")
	c.DataFromReader(http.StatusOK, -1, contentType, reader, nil)
}

func (h *Handler) UpdateReportToken(c *gin.Context) {
	teamID, err := parseTeamID(c)
	if err != nil || !h.canAccessTeam(c, teamID) {
		h.respondTeamForbidden(c)
		return
	}
	var req struct {
		Slug string `json:"slug" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	slug := service.NormalizeReportSlug(req.Slug)
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "slug tidak valid"})
		return
	}
	if err := service.ValidateReportSlug(slug); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.SetReportToken(c.Request.Context(), teamID, slug); err != nil {
		if errors.Is(err, repository.ErrTokenTaken) {
			c.JSON(http.StatusConflict, gin.H{"error": "slug laporan sudah dipakai tim/kas lain"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": slug, "report_url": h.appURL + "/report/" + slug})
}

func (h *Handler) ResetReportToken(c *gin.Context) {
	teamID, err := parseTeamID(c)
	if err != nil || !h.canAccessTeam(c, teamID) {
		h.respondTeamForbidden(c)
		return
	}
	team, err := h.repo.GetTeam(c.Request.Context(), teamID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "tim/kas tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	slug := team.Slug
	if err := h.repo.SetReportToken(c.Request.Context(), teamID, slug); err != nil {
		if errors.Is(err, repository.ErrTokenTaken) {
			c.JSON(http.StatusConflict, gin.H{"error": "slug default sudah dipakai tim/kas lain — gunakan slug kustom"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": slug, "report_url": h.appURL + "/report/" + slug})
}

func (h *Handler) GetReportToken(c *gin.Context) {
	teamID, err := parseTeamID(c)
	if err != nil || !h.canAccessTeam(c, teamID) {
		h.respondTeamForbidden(c)
		return
	}
	rt, err := h.repo.GetReportToken(c.Request.Context(), teamID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": rt.Token, "report_url": h.appURL + "/report/" + rt.Token})
}

func (h *Handler) PublicReport(c *gin.Context) {
	token := c.Param("token")
	team, _, err := h.repo.GetReportByToken(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "laporan tidak ditemukan"})
		return
	}
	filter := repository.TxFilter{TeamID: team.ID, Limit: 500}
	dateFrom, dateTo := parseDateRange(c)
	if j := c.Query("jenis"); j != "" {
		jn := models.TxJenis(j)
		filter.Jenis = &jn
	}
	if dateFrom != nil {
		filter.DateFrom = dateFrom
	}
	if dateTo != nil {
		filter.DateTo = dateTo
	}
	txs, err := h.repo.ListTransactions(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	balance, err := h.repo.GetBalance(c.Request.Context(), team.ID, dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"team": team, "balance": balance, "transactions": txs})
}

func (h *Handler) PublicNota(c *gin.Context) {
	token := c.Param("token")
	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key required"})
		return
	}
	team, _, err := h.repo.GetReportByToken(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if !storage.NotaKeyBelongsToTeam(key, team.ID.String()) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	download := c.Query("download") == "true"
	u, err := h.svc.GetNotaURL(c.Request.Context(), key, download)
	if err != nil {
		log.Printf("nota: public url token=%s key=%q: %v", token, key, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "nota not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": u})
}

func (h *Handler) ServePublicNota(c *gin.Context) {
	token := c.Param("token")
	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key required"})
		return
	}
	team, _, err := h.repo.GetReportByToken(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if !storage.NotaKeyBelongsToTeam(key, team.ID.String()) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	h.serveNotaObject(c, key, c.Query("download") == "true")
}

const maxNotaBytes = 10 << 20

func (h *Handler) uploadNotaFiles(c *gin.Context, teamID uuid.UUID) ([]string, error) {
	headers := notaFileHeaders(c)
	if len(headers) == 0 {
		return nil, nil
	}
	if len(headers) > models.MaxNotaFiles {
		return nil, fmt.Errorf("maksimal %d foto nota", models.MaxNotaFiles)
	}
	keys := make([]string, 0, len(headers))
	for _, header := range headers {
		file, err := header.Open()
		if err != nil {
			return nil, fmt.Errorf("gagal buka file nota")
		}
		data, err := io.ReadAll(io.LimitReader(file, maxNotaBytes+1))
		file.Close()
		if err != nil {
			return nil, fmt.Errorf("gagal baca file nota")
		}
		if len(data) == 0 {
			continue
		}
		if len(data) > maxNotaBytes {
			return nil, fmt.Errorf("file nota terlalu besar (max 10MB)")
		}
		key, err := h.svc.UploadNota(c.Request.Context(), teamID, header.Filename, data, header.Header.Get("Content-Type"))
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

func notaFileHeaders(c *gin.Context) []*multipart.FileHeader {
	if form, err := c.MultipartForm(); err == nil && form != nil {
		if files := form.File["nota"]; len(files) > 0 {
			return files
		}
	}
	if fh, err := c.FormFile("nota"); err == nil {
		return []*multipart.FileHeader{fh}
	}
	return nil
}
