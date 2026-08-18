package telegram

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kasq/backend/internal/bot"
	"github.com/kasq/backend/internal/models"
	"github.com/kasq/backend/internal/repository"
	"github.com/kasq/backend/internal/service"
	tele "gopkg.in/telebot.v3"
)

type Manager struct {
	repo        *repository.Repository
	svc         *service.Service
	appURL      string
	systemToken string
	system      *botInstance
	bots        map[uuid.UUID]*botInstance
	albums      map[string]*teleAlbum
	mu          sync.RWMutex
	albumMu     sync.Mutex
}

type BotProfile struct {
	Name      string
	Username  string
	HasAvatar bool
}

type botInstance struct {
	bot           *tele.Bot
	cancel        context.CancelFunc
	teamID        uuid.UUID
	displayName   string
	username      string
	avatarFileID  string
	profileLoaded bool
}

func NewManager(repo *repository.Repository, svc *service.Service, systemToken, appURL string) *Manager {
	return &Manager{
		repo:        repo,
		svc:         svc,
		appURL:      strings.TrimRight(strings.TrimSpace(appURL), "/"),
		systemToken: strings.TrimSpace(systemToken),
		bots:        make(map[uuid.UUID]*botInstance),
		albums:      make(map[string]*teleAlbum),
	}
}

func (m *Manager) SystemBotAvailable() bool {
	return m.systemToken != ""
}

func (m *Manager) IsSystemToken(token string) bool {
	return m.systemToken != "" && strings.TrimSpace(token) == m.systemToken
}

func (m *Manager) SystemBotProfile() BotProfile {
	m.mu.RLock()
	inst := m.system
	m.mu.RUnlock()
	if inst == nil {
		return BotProfile{}
	}
	if !inst.profileLoaded {
		m.refreshInstanceProfile(inst, "system")
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.system == nil {
		return BotProfile{}
	}
	return BotProfile{
		Name:      m.system.displayName,
		Username:  m.system.username,
		HasAvatar: m.system.avatarFileID != "",
	}
}

func (m *Manager) StartAll(ctx context.Context) {
	if m.systemToken != "" {
		if err := m.startSystemBot(); err != nil {
			log.Printf("tele: start system bot: %v", err)
		}
	}
	integrations, err := m.repo.ListEnabledTeleIntegrations(ctx)
	if err != nil {
		log.Printf("tele: list integrations: %v", err)
		return
	}
	for _, i := range integrations {
		if i.TeleUseSystemBot || i.TeleBotToken == nil {
			continue
		}
		if err := m.StartTeam(i.TeamID, *i.TeleBotToken); err != nil {
			log.Printf("tele: start team %s: %v", i.TeamID, err)
		}
	}
}

func (m *Manager) StartTeam(teamID uuid.UUID, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("bot token kosong")
	}
	if m.IsSystemToken(token) {
		return fmt.Errorf("token bot sistem tidak bisa dipakai sebagai bot sendiri — pilih opsi Bot KasQ")
	}
	m.StopTeam(teamID)

	pref := tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	inst := &botInstance{bot: b, cancel: cancel, teamID: teamID}
	inst.displayName = strings.TrimSpace(strings.TrimSpace(b.Me.FirstName + " " + b.Me.LastName))
	inst.username = strings.TrimSpace(b.Me.Username)

	b.Handle("/saldo", func(c tele.Context) error {
		return m.handleSaldo(ctx, teamID, c)
	})
	b.Handle("/link", func(c tele.Context) error {
		return m.handleLink(ctx, teamID, c)
	})
	b.Handle("/start", func(c tele.Context) error {
		return c.Send("KasQ bot aktif.\n\nCek saldo: /saldo atau !saldo\nLink laporan: /link atau !link\nInput transaksi:\nout#Senin#150826#Deskripsi#12000#Keterangan\nout#150826#Deskripsi#12000\n\nHari dan keterangan opsional.")
	})
	b.Handle(tele.OnText, func(c tele.Context) error {
		return m.handleText(ctx, teamID, c)
	})
	b.Handle(tele.OnPhoto, func(c tele.Context) error {
		return m.handlePhoto(ctx, teamID, c)
	})

	m.mu.Lock()
	m.bots[teamID] = inst
	m.mu.Unlock()

	go b.Start()
	go m.refreshBotProfile(teamID)
	log.Printf("tele: bot started for team %s (@%s)", teamID, b.Me.Username)
	return nil
}

func (m *Manager) GetBotProfile(teamID uuid.UUID) BotProfile {
	if m.teamUsesSystemBot(teamID) {
		return m.SystemBotProfile()
	}
	m.mu.RLock()
	inst, ok := m.bots[teamID]
	m.mu.RUnlock()
	if !ok {
		return BotProfile{}
	}
	if !inst.profileLoaded {
		m.refreshBotProfile(teamID)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if inst, ok := m.bots[teamID]; ok {
		return BotProfile{
			Name:      inst.displayName,
			Username:  inst.username,
			HasAvatar: inst.avatarFileID != "",
		}
	}
	return BotProfile{}
}

func (m *Manager) OpenBotAvatar(teamID uuid.UUID) (io.ReadCloser, string, error) {
	if m.teamUsesSystemBot(teamID) {
		return m.openInstanceAvatar(m.systemBot(), "system")
	}
	m.mu.RLock()
	inst, ok := m.bots[teamID]
	m.mu.RUnlock()
	if !ok {
		return nil, "", fmt.Errorf("avatar not available")
	}
	return m.openInstanceAvatar(inst, teamID.String())
}

func (m *Manager) refreshBotProfile(teamID uuid.UUID) {
	m.mu.RLock()
	inst, ok := m.bots[teamID]
	m.mu.RUnlock()
	if !ok {
		return
	}
	m.refreshInstanceProfile(inst, teamID.String())
}

func (m *Manager) teamUsesSystemBot(teamID uuid.UUID) bool {
	integ, err := m.repo.GetIntegration(context.Background(), teamID)
	return err == nil && integ.TeleUseSystemBot
}

func (m *Manager) systemBot() *botInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.system
}

func (m *Manager) openInstanceAvatar(inst *botInstance, label string) (io.ReadCloser, string, error) {
	if inst == nil || inst.bot == nil {
		return nil, "", fmt.Errorf("avatar not available")
	}
	if !inst.profileLoaded {
		m.refreshInstanceProfile(inst, label)
	}
	if inst.avatarFileID == "" {
		return nil, "", fmt.Errorf("avatar not available")
	}
	file := tele.File{FileID: inst.avatarFileID}
	reader, err := inst.bot.File(&file)
	if err != nil {
		return nil, "", err
	}
	contentType := "image/jpeg"
	if strings.HasSuffix(strings.ToLower(file.FilePath), ".png") {
		contentType = "image/png"
	}
	return reader, contentType, nil
}

func (m *Manager) refreshInstanceProfile(inst *botInstance, label string) {
	if inst == nil || inst.bot == nil || inst.bot.Me == nil {
		return
	}

	name := strings.TrimSpace(strings.TrimSpace(inst.bot.Me.FirstName + " " + inst.bot.Me.LastName))
	username := strings.TrimSpace(inst.bot.Me.Username)
	avatarFileID := ""

	photos, err := inst.bot.ProfilePhotosOf(inst.bot.Me)
	if err != nil {
		log.Printf("tele: profile photos %s: %v", label, err)
	} else if len(photos) > 0 && photos[0].FileID != "" {
		avatarFileID = photos[0].FileID
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if name != "" {
		inst.displayName = name
	}
	if username != "" {
		inst.username = username
	}
	inst.avatarFileID = avatarFileID
	inst.profileLoaded = true
}

func (m *Manager) StopTeam(teamID uuid.UUID) {
	m.mu.Lock()
	inst, ok := m.bots[teamID]
	if ok {
		delete(m.bots, teamID)
	}
	m.mu.Unlock()
	if ok {
		inst.cancel()
		inst.bot.Stop()
		log.Printf("tele: bot stopped for team %s", teamID)
	}
}

func (m *Manager) chatAllowed(integ *models.Integration, chatID int64) bool {
	if integ.TeleAllowedChatID == nil {
		return true
	}
	return chatID == *integ.TeleAllowedChatID
}

func (m *Manager) rejectChat(c tele.Context, integ *models.Integration) error {
	if integ.TeleAllowedChatID == nil {
		return nil
	}
	return c.Send(fmt.Sprintf(
		"❌ Chat ini belum terdaftar untuk tim/kas ini.\nChat ID kamu: %d\nChat ID terdaftar: %d\n\nPerbarui Chat ID di halaman Integrasi KasQ.",
		c.Chat().ID, *integ.TeleAllowedChatID,
	))
}

func (m *Manager) handleSaldo(ctx context.Context, teamID uuid.UUID, c tele.Context) error {
	integ, err := m.repo.GetIntegration(ctx, teamID)
	if err != nil {
		return c.Send("❌ Integrasi belum siap")
	}
	if !m.chatAllowed(integ, c.Chat().ID) {
		return m.rejectChat(c, integ)
	}

	team, err := m.repo.GetTeam(ctx, teamID)
	if err != nil {
		return c.Send("❌ Tim/Kas tidak ditemukan")
	}
	balance, err := m.repo.GetBalance(ctx, teamID, nil, nil)
	if err != nil {
		return c.Send("❌ Gagal cek saldo")
	}
	return c.Send(bot.FormatSaldoReply(team.Name, balance.CurrentBalance))
}

func (m *Manager) handleLink(ctx context.Context, teamID uuid.UUID, c tele.Context) error {
	integ, err := m.repo.GetIntegration(ctx, teamID)
	if err != nil {
		return c.Send("❌ Integrasi belum siap")
	}
	if !m.chatAllowed(integ, c.Chat().ID) {
		return m.rejectChat(c, integ)
	}

	team, err := m.repo.GetTeam(ctx, teamID)
	if err != nil {
		return c.Send("❌ Tim/Kas tidak ditemukan")
	}
	rt, err := m.repo.GetReportToken(ctx, teamID)
	if err != nil || !rt.IsActive {
		return c.Send("❌ Link laporan publik belum tersedia")
	}
	return c.Send(bot.FormatLinkReply(team.Name, bot.PublicReportURL(m.appURL, rt.Token)))
}

func (m *Manager) handleText(ctx context.Context, teamID uuid.UUID, c tele.Context) error {
	text := strings.TrimSpace(c.Text())
	if text == "" {
		return nil
	}
	// Skip if already handled as /command
	if strings.HasPrefix(text, "/") {
		return nil
	}

	integ, err := m.repo.GetIntegration(ctx, teamID)
	if err != nil {
		return nil
	}
	if !m.chatAllowed(integ, c.Chat().ID) {
		if integ.TeleAllowedChatID != nil {
			log.Printf("tele: ignore chat %d team %s (want %d)", c.Chat().ID, teamID, *integ.TeleAllowedChatID)
		}
		return m.rejectChat(c, integ)
	}

	team, err := m.repo.GetTeam(ctx, teamID)
	if err != nil {
		return c.Send("❌ Tim/Kas tidak ditemukan")
	}

	if isSaldoCommand(text) {
		return m.handleSaldo(ctx, teamID, c)
	}
	if isLinkCommand(text) {
		return m.handleLink(ctx, teamID, c)
	}

	parsed, err := bot.ParseMessage(text, models.SourceTele)
	if err != nil {
		if err == bot.ErrSaldoCommand {
			return m.handleSaldo(ctx, teamID, c)
		}
		if err == bot.ErrLinkCommand {
			return m.handleLink(ctx, teamID, c)
		}
		return c.Send(bot.FormatErrorReply(err))
	}

	tx, balance, err := m.svc.CreateTransactionFromBot(ctx, teamID, *parsed, nil)
	if err != nil {
		return c.Send("❌ Gagal simpan: " + err.Error())
	}
	return c.Send(bot.FormatSuccessReply(tx, balance.CurrentBalance, team.Name, 0))
}

func (m *Manager) handlePhoto(ctx context.Context, teamID uuid.UUID, c tele.Context) error {
	integ, err := m.repo.GetIntegration(ctx, teamID)
	if err != nil {
		return nil
	}
	if !m.chatAllowed(integ, c.Chat().ID) {
		return m.rejectChat(c, integ)
	}

	file := c.Message().Photo
	if file == nil {
		return c.Send("❌ Foto tidak valid")
	}
	item := teleAlbumItem{
		file:    file.File,
		caption: strings.TrimSpace(c.Message().Caption),
	}
	albumID := strings.TrimSpace(c.Message().AlbumID)
	if albumID == "" {
		return m.commitTelePhotos(ctx, teamID, c, []teleAlbumItem{item})
	}
	m.enqueueTeleAlbum(ctx, teamID, c, albumID, item)
	return nil
}

func (m *Manager) botForTeam(teamID uuid.UUID) *tele.Bot {
	m.mu.RLock()
	if inst, ok := m.bots[teamID]; ok {
		b := inst.bot
		m.mu.RUnlock()
		return b
	}
	sys := m.system
	m.mu.RUnlock()
	if sys == nil || sys.bot == nil || !m.teamUsesSystemBot(teamID) {
		return nil
	}
	return sys.bot
}

func isSaldoCommand(text string) bool {
	return bot.IsSaldoCommand(text)
}

func isLinkCommand(text string) bool {
	return bot.IsLinkCommand(text)
}

func imageMetaFromBytes(data []byte) (filename, contentType string) {
	filename = "tele-nota.jpg"
	contentType = "image/jpeg"
	if len(data) >= 8 && data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "tele-nota.png", "image/png"
	}
	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "tele-nota.webp", "image/webp"
	}
	return filename, contentType
}
