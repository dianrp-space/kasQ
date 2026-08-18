package whatsapp

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kasq/backend/internal/bot"
	"github.com/kasq/backend/internal/models"
	"go.mau.fi/whatsmeow/types/events"
)

const waAlbumWait = 2500 * time.Millisecond

type waAlbumItem struct {
	data    []byte
	mime    string
	caption string
}

type waAlbum struct {
	teamID   uuid.UUID
	teamName string
	reply    func(string)
	items    []waAlbumItem
	timer    *time.Timer
}

func (m *Manager) enqueueWAImage(ctx context.Context, teamID uuid.UUID, s *session, msg *events.Message, teamName string, reply func(string)) {
	img := msg.Message.GetImageMessage()
	if img == nil {
		return
	}
	data, err := s.client.Download(ctx, img)
	if err != nil {
		reply("❌ Gagal unduh foto")
		return
	}
	mime := "image/jpeg"
	if img.Mimetype != nil && *img.Mimetype != "" {
		mime = *img.Mimetype
	}
	caption := ""
	if img.Caption != nil {
		caption = strings.TrimSpace(*img.Caption)
	}
	item := waAlbumItem{data: data, mime: mime, caption: caption}
	key := teamID.String() + ":" + msg.Info.Chat.String() + ":" + msg.Info.Sender.String()

	m.albumMu.Lock()
	defer m.albumMu.Unlock()
	buf, ok := m.albums[key]
	if !ok {
		buf = &waAlbum{teamID: teamID, teamName: teamName, reply: reply}
		m.albums[key] = buf
	}
	if len(buf.items) < models.MaxNotaFiles {
		buf.items = append(buf.items, item)
	}
	buf.reply = reply
	buf.teamName = teamName
	if buf.timer != nil {
		buf.timer.Stop()
	}
	buf.timer = time.AfterFunc(waAlbumWait, func() {
		m.flushWAAlbum(key)
	})
}

func (m *Manager) flushWAAlbum(key string) {
	m.albumMu.Lock()
	buf, ok := m.albums[key]
	if ok {
		delete(m.albums, key)
	}
	m.albumMu.Unlock()
	if !ok || len(buf.items) == 0 || buf.reply == nil {
		return
	}

	captions := make([]string, 0, len(buf.items))
	for _, it := range buf.items {
		captions = append(captions, it.caption)
	}
	parsed, err := bot.ParseAlbumCaption(captions, models.SourceWA)
	if err != nil {
		buf.reply(bot.FormatErrorReply(err))
		return
	}

	ctx := context.Background()
	var notaKeys []string
	for i, it := range buf.items {
		filename := "wa-nota.jpg"
		key, err := m.svc.UploadNota(ctx, buf.teamID, filename, it.data, it.mime)
		if err != nil {
			log.Printf("wa: upload nota team %s foto %d: %v", buf.teamID, i+1, err)
			continue
		}
		notaKeys = append(notaKeys, key)
	}

	tx, balance, err := m.svc.CreateTransactionFromBot(ctx, buf.teamID, *parsed, notaKeys)
	if err != nil {
		buf.reply("❌ Gagal simpan: " + err.Error())
		return
	}
	buf.reply(bot.FormatSuccessReply(tx, balance.CurrentBalance, buf.teamName, len(notaKeys)))
}
