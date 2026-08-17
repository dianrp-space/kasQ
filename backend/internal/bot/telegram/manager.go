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
	repo *repository.Repository
	svc  *service.Service
	bots map[uuid.UUID]*botInstance
	mu   sync.RWMutex
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

func NewManager(repo *repository.Repository, svc *service.Service) *Manager {
	return &Manager{
		repo: repo,
		svc:  svc,
		bots: make(map[uuid.UUID]*botInstance),
	}
}

func (m *Manager) StartAll(ctx context.Context) {
	integrations, err := m.repo.ListEnabledTeleIntegrations(ctx)
	if err != nil {
		log.Printf("tele: list integrations: %v", err)
		return
	}
	for _, i := range integrations {
		if i.TeleBotToken != nil {
			if err := m.StartTeam(i.TeamID, *i.TeleBotToken); err != nil {
				log.Printf("tele: start team %s: %v", i.TeamID, err)
			}
		}
	}
}

func (m *Manager) StartTeam(teamID uuid.UUID, token string) error {
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
	b.Handle("/start", func(c tele.Context) error {
		return c.Send("KasQ bot aktif.\n\nCek saldo: /saldo atau !saldo\nInput transaksi:\nout#Senin#150826#Deskripsi#12000#Keterangan\n\n(Keterangan opsional)")
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
	m.mu.RLock()
	inst, ok := m.bots[teamID]
	m.mu.RUnlock()
	if !ok || inst.avatarFileID == "" {
		return nil, "", fmt.Errorf("avatar not available")
	}
	if !inst.profileLoaded {
		m.refreshBotProfile(teamID)
	}
	m.mu.RLock()
	inst, ok = m.bots[teamID]
	m.mu.RUnlock()
	if !ok || inst.avatarFileID == "" {
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

func (m *Manager) refreshBotProfile(teamID uuid.UUID) {
	m.mu.RLock()
	inst, ok := m.bots[teamID]
	m.mu.RUnlock()
	if !ok || inst.bot == nil || inst.bot.Me == nil {
		return
	}

	name := strings.TrimSpace(strings.TrimSpace(inst.bot.Me.FirstName + " " + inst.bot.Me.LastName))
	username := strings.TrimSpace(inst.bot.Me.Username)
	avatarFileID := ""

	photos, err := inst.bot.ProfilePhotosOf(inst.bot.Me)
	if err != nil {
		log.Printf("tele: profile photos team %s: %v", teamID, err)
	} else if len(photos) > 0 && photos[0].FileID != "" {
		avatarFileID = photos[0].FileID
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if inst, ok := m.bots[teamID]; ok {
		if name != "" {
			inst.displayName = name
		}
		if username != "" {
			inst.username = username
		}
		inst.avatarFileID = avatarFileID
		inst.profileLoaded = true
	}
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

	parsed, err := bot.ParseMessage(text, models.SourceTele)
	if err != nil {
		if err == bot.ErrSaldoCommand {
			return m.handleSaldo(ctx, teamID, c)
		}
		return c.Send(bot.FormatErrorReply(err))
	}

	tx, balance, err := m.svc.CreateTransactionFromBot(ctx, teamID, *parsed, nil)
	if err != nil {
		return c.Send("❌ Gagal simpan: " + err.Error())
	}
	return c.Send(bot.FormatSuccessReply(tx, balance.CurrentBalance, team.Name, false))
}

func (m *Manager) handlePhoto(ctx context.Context, teamID uuid.UUID, c tele.Context) error {
	integ, err := m.repo.GetIntegration(ctx, teamID)
	if err != nil {
		return nil
	}
	if !m.chatAllowed(integ, c.Chat().ID) {
		return m.rejectChat(c, integ)
	}

	team, err := m.repo.GetTeam(ctx, teamID)
	if err != nil {
		return c.Send("❌ Tim/Kas tidak ditemukan")
	}

	caption := c.Message().Caption
	if caption == "" {
		return c.Send("❌ Caption wajib berisi format transaksi")
	}

	file := c.Message().Photo
	if file == nil {
		return c.Send("❌ Foto tidak valid")
	}

	b := m.muBot(teamID)
	if b == nil {
		return c.Send("❌ Bot tidak aktif")
	}
	reader, err := b.File(&file.File)
	if err != nil {
		return c.Send("❌ Gagal unduh foto")
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return c.Send("❌ Gagal baca foto")
	}
	if len(data) == 0 {
		return c.Send("❌ Foto kosong")
	}

	filename, contentType := imageMetaFromBytes(data)
	var notaKey *string
	key, err := m.svc.UploadNota(ctx, teamID, filename, data, contentType)
	if err != nil {
		log.Printf("tele: upload nota team %s: %v", teamID, err)
	} else {
		notaKey = &key
	}

	parsed, err := bot.ParseMessage(caption, models.SourceTele)
	if err != nil {
		return c.Send(bot.FormatErrorReply(err))
	}

	tx, balance, err := m.svc.CreateTransactionFromBot(ctx, teamID, *parsed, notaKey)
	if err != nil {
		return c.Send("❌ Gagal simpan: " + err.Error())
	}
	hasNota := notaKey != nil
	return c.Send(bot.FormatSuccessReply(tx, balance.CurrentBalance, team.Name, hasNota))
}

func (m *Manager) muBot(teamID uuid.UUID) *tele.Bot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if inst, ok := m.bots[teamID]; ok {
		return inst.bot
	}
	return nil
}

func isSaldoCommand(text string) bool {
	return strings.EqualFold(strings.TrimSpace(text), "!saldo")
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
