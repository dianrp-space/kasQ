package telegram

import (
	"context"
	"io"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/kasq/backend/internal/bot"
	"github.com/kasq/backend/internal/models"
	tele "gopkg.in/telebot.v3"
)

const teleAlbumWait = 1200 * time.Millisecond

type teleAlbumItem struct {
	file    tele.File
	caption string
}

type teleAlbum struct {
	teamID uuid.UUID
	ctx    context.Context
	reply  tele.Context
	items  []teleAlbumItem
	timer  *time.Timer
}

func (m *Manager) enqueueTeleAlbum(ctx context.Context, teamID uuid.UUID, c tele.Context, albumID string, item teleAlbumItem) {
	key := teamID.String() + ":" + albumID
	m.albumMu.Lock()
	defer m.albumMu.Unlock()
	buf, ok := m.albums[key]
	if !ok {
		buf = &teleAlbum{teamID: teamID, ctx: ctx, reply: c}
		m.albums[key] = buf
	}
	if len(buf.items) < models.MaxNotaFiles {
		buf.items = append(buf.items, item)
	}
	buf.reply = c
	if buf.timer != nil {
		buf.timer.Stop()
	}
	buf.timer = time.AfterFunc(teleAlbumWait, func() {
		m.flushTeleAlbum(key)
	})
}

func (m *Manager) flushTeleAlbum(key string) {
	m.albumMu.Lock()
	buf, ok := m.albums[key]
	if ok {
		delete(m.albums, key)
	}
	m.albumMu.Unlock()
	if !ok || len(buf.items) == 0 || buf.reply == nil {
		return
	}
	if err := m.commitTelePhotos(buf.ctx, buf.teamID, buf.reply, buf.items); err != nil {
		log.Printf("tele: album %s: %v", key, err)
	}
}

func (m *Manager) commitTelePhotos(ctx context.Context, teamID uuid.UUID, c tele.Context, items []teleAlbumItem) error {
	team, err := m.repo.GetTeam(ctx, teamID)
	if err != nil {
		return c.Send("❌ Tim/Kas tidak ditemukan")
	}

	captions := make([]string, 0, len(items))
	for _, it := range items {
		captions = append(captions, it.caption)
	}
	parsed, err := bot.ParseAlbumCaption(captions, models.SourceTele)
	if err != nil {
		return c.Send(bot.FormatErrorReply(err))
	}

	b := m.botForTeam(teamID)
	if b == nil {
		return c.Send("❌ Bot tidak aktif")
	}

	var notaKeys []string
	for _, it := range items {
		file := it.file
		reader, err := b.File(&file)
		if err != nil {
			log.Printf("tele: unduh foto team %s: %v", teamID, err)
			continue
		}
		data, err := io.ReadAll(reader)
		reader.Close()
		if err != nil || len(data) == 0 {
			continue
		}
		filename, contentType := imageMetaFromBytes(data)
		key, err := m.svc.UploadNota(ctx, teamID, filename, data, contentType)
		if err != nil {
			log.Printf("tele: upload nota team %s: %v", teamID, err)
			continue
		}
		notaKeys = append(notaKeys, key)
	}

	tx, balance, err := m.svc.CreateTransactionFromBot(ctx, teamID, *parsed, notaKeys)
	if err != nil {
		return c.Send("❌ Gagal simpan: " + err.Error())
	}
	return c.Send(bot.FormatSuccessReply(tx, balance.CurrentBalance, team.Name, len(notaKeys)))
}
