package telegram

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"

	"github.com/kasq/backend/internal/repository"
)

const systemPrivateOnlyMsg = "Bot KasQ hanya untuk chat pribadi. Buka percakapan langsung dengan bot ini, lalu kirim /start untuk mendapat Chat ID."

func (m *Manager) startSystemBot() error {
	if m.systemToken == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN kosong")
	}

	pref := tele.Settings{
		Token:  m.systemToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}
	b, err := tele.NewBot(pref)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	inst := &botInstance{bot: b, cancel: cancel}
	inst.displayName = strings.TrimSpace(strings.TrimSpace(b.Me.FirstName + " " + b.Me.LastName))
	inst.username = strings.TrimSpace(b.Me.Username)

	b.Handle("/start", func(c tele.Context) error {
		return m.handleSystemStart(ctx, c)
	})
	b.Handle("/saldo", func(c tele.Context) error {
		return m.handleSystemSaldo(ctx, c)
	})
	b.Handle("/link", func(c tele.Context) error {
		return m.handleSystemLink(ctx, c)
	})
	b.Handle(tele.OnText, func(c tele.Context) error {
		return m.handleSystemText(ctx, c)
	})
	b.Handle(tele.OnPhoto, func(c tele.Context) error {
		return m.handleSystemPhoto(ctx, c)
	})

	m.mu.Lock()
	m.system = inst
	m.mu.Unlock()

	go b.Start()
	go m.refreshInstanceProfile(inst, "system")
	log.Printf("tele: system bot started (@%s)", b.Me.Username)
	return nil
}

func isPrivateChat(c tele.Context) bool {
	ch := c.Chat()
	return ch != nil && ch.Type == tele.ChatPrivate
}

func (m *Manager) requirePrivate(c tele.Context) error {
	if !isPrivateChat(c) {
		return c.Send(systemPrivateOnlyMsg)
	}
	return nil
}

func formatSystemChatIDHelp(chatID int64) string {
	return fmt.Sprintf(
		"Chat ID kamu: %d\n\n"+
			"1. Salin angka di atas\n"+
			"2. Buka KasQ → Integrasi → Telegram\n"+
			"3. Pilih Bot KasQ dan tempel Chat ID\n"+
			"4. Klik Aktifkan\n\n"+
			"Bot ini hanya untuk chat pribadi antara kamu dan KasQ.",
		chatID,
	)
}

func formatSystemConnectedHelp(chatID int64, teamName string) string {
	return fmt.Sprintf(
		"KasQ bot aktif untuk %s.\n\n"+
			"Chat ID kamu: %d\n\n"+
			"Cek saldo: /saldo\n"+
			"Link laporan: /link\n"+
			"Input transaksi:\n"+
			"out#Senin#150826#Deskripsi#12000#Keterangan\n"+
			"out#150826#Deskripsi#12000\n\n"+
			"Hari dan keterangan opsional.",
		teamName, chatID,
	)
}

func (m *Manager) handleSystemStart(ctx context.Context, c tele.Context) error {
	if err := m.requirePrivate(c); err != nil {
		return err
	}
	chatID := c.Chat().ID
	integ, err := m.repo.GetEnabledSystemTeleByChatID(ctx, chatID)
	if err != nil {
		if !errors.Is(err, repository.ErrNotFound) {
			log.Printf("tele: system lookup chat %d: %v", chatID, err)
		}
		return c.Send(formatSystemChatIDHelp(chatID))
	}
	teamName := "kas kamu"
	if team, tErr := m.repo.GetTeam(ctx, integ.TeamID); tErr == nil && team != nil {
		teamName = team.Name
	}
	return c.Send(formatSystemConnectedHelp(chatID, teamName))
}

func (m *Manager) handleSystemSaldo(ctx context.Context, c tele.Context) error {
	if err := m.requirePrivate(c); err != nil {
		return err
	}
	chatID := c.Chat().ID
	integ, err := m.repo.GetEnabledSystemTeleByChatID(ctx, chatID)
	if err != nil {
		return c.Send(formatSystemChatIDHelp(chatID))
	}
	return m.handleSaldo(ctx, integ.TeamID, c)
}

func (m *Manager) handleSystemLink(ctx context.Context, c tele.Context) error {
	if err := m.requirePrivate(c); err != nil {
		return err
	}
	chatID := c.Chat().ID
	integ, err := m.repo.GetEnabledSystemTeleByChatID(ctx, chatID)
	if err != nil {
		return c.Send(formatSystemChatIDHelp(chatID))
	}
	return m.handleLink(ctx, integ.TeamID, c)
}

func (m *Manager) handleSystemText(ctx context.Context, c tele.Context) error {
	text := strings.TrimSpace(c.Text())
	if strings.HasPrefix(text, "/") {
		return nil
	}
	if err := m.requirePrivate(c); err != nil {
		return err
	}
	chatID := c.Chat().ID
	integ, err := m.repo.GetEnabledSystemTeleByChatID(ctx, chatID)
	if err != nil {
		return c.Send(formatSystemChatIDHelp(chatID))
	}
	return m.handleText(ctx, integ.TeamID, c)
}

func (m *Manager) handleSystemPhoto(ctx context.Context, c tele.Context) error {
	if err := m.requirePrivate(c); err != nil {
		return err
	}
	chatID := c.Chat().ID
	integ, err := m.repo.GetEnabledSystemTeleByChatID(ctx, chatID)
	if err != nil {
		return c.Send(formatSystemChatIDHelp(chatID))
	}
	return m.handlePhoto(ctx, integ.TeamID, c)
}
