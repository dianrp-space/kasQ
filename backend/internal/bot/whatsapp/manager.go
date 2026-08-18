package whatsapp

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/kasq/backend/internal/bot"
	"github.com/kasq/backend/internal/models"
	"github.com/kasq/backend/internal/repository"
	"github.com/kasq/backend/internal/service"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	waCompanionReg "go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	_ "modernc.org/sqlite"
)

const pairCodeLifetime = 60 * time.Second
const profileCacheTTL = 5 * time.Minute

func init() {
	version := store.GetWAVersion()
	store.SetOSInfo("Windows", version)
	store.DeviceProps.PlatformType = waCompanionReg.DeviceProps_CHROME.Enum()
}

type ConnectStatus struct {
	Status             string
	QR                 string
	Phone              string
	DisplayName        string
	PictureURL         string
	PairCode           string
	QRTimeoutSeconds   int
	PairCodeExpiresSec int
	LoginMode          string
}

type Manager struct {
	repo         *repository.Repository
	svc          *service.Service
	appURL       string
	dataDir      string
	sessions     map[uuid.UUID]*session
	seenMessages sync.Map
	albums       map[string]*waAlbum
	mu           sync.RWMutex
	albumMu      sync.Mutex
}

type session struct {
	client           *whatsmeow.Client
	teamID           uuid.UUID
	qrCode           string
	qrTimeout        time.Duration
	qrUpdatedAt      time.Time
	pairCode         string
	pairCodeAt       time.Time
	status           string
	phone            string
	displayName      string
	pictureURL       string
	profileFetchedAt time.Time
	loginMode        string
}

func NewManager(repo *repository.Repository, svc *service.Service, dataDir, appURL string) *Manager {
	return &Manager{
		repo:     repo,
		svc:      svc,
		appURL:   strings.TrimRight(strings.TrimSpace(appURL), "/"),
		dataDir:  dataDir,
		sessions: make(map[uuid.UUID]*session),
		albums:   make(map[string]*waAlbum),
	}
}

func waSQLiteDSN(path string) string {
	return path + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)"
}

func openWAStore(teamDir string, dbLog waLog.Logger) (*sqlstore.Container, error) {
	dbPath := filepath.Join(teamDir, "store.db")
	db, err := sql.Open("sqlite", waSQLiteDSN(dbPath))
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	container := sqlstore.NewWithDB(db, "sqlite", dbLog)
	if err := container.Upgrade(context.Background()); err != nil {
		_ = container.Close()
		return nil, fmt.Errorf("upgrade store: %w", err)
	}
	return container, nil
}

func (m *Manager) StartAll(ctx context.Context) {
	integrations, err := m.repo.ListEnabledWAIntegrations(ctx)
	if err != nil {
		log.Printf("wa: list integrations: %v", err)
		return
	}
	for _, i := range integrations {
		if err := m.StartTeam(i.TeamID); err != nil {
			log.Printf("wa: start team %s: %v", i.TeamID, err)
		}
	}
}

func (m *Manager) StartTeam(teamID uuid.UUID) error {
	m.mu.Lock()
	if _, ok := m.sessions[teamID]; ok {
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()

	teamDir := filepath.Join(m.dataDir, teamID.String())
	if err := os.MkdirAll(teamDir, 0755); err != nil {
		return err
	}

	dbLog := waLog.Stdout("Database", "ERROR", true)
	container, err := openWAStore(teamDir, dbLog)
	if err != nil {
		return fmt.Errorf("sqlstore: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return err
	}

	clientLog := waLog.Stdout("Client", "WARN", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	s := &session{client: client, teamID: teamID, status: "awaiting_login"}

	client.AddEventHandler(func(evt any) {
		m.handleEvent(teamID, evt)
	})

	m.mu.Lock()
	m.sessions[teamID] = s
	m.mu.Unlock()

	if client.Store.ID != nil {
		if err := client.Connect(); err != nil {
			return err
		}
		m.onConnected(teamID)
	}
	return nil
}

func (m *Manager) restartSession(teamID uuid.UUID) error {
	m.StopTeam(teamID)
	return m.StartTeam(teamID)
}

func (m *Manager) StartQRLogin(teamID uuid.UUID) error {
	m.mu.RLock()
	s, ok := m.sessions[teamID]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("session WA tidak aktif — aktifkan WA terlebih dahulu")
	}
	if s.client.Store.ID != nil {
		return fmt.Errorf("WA sudah terhubung")
	}
	if s.loginMode == "pair" {
		if err := m.restartSession(teamID); err != nil {
			return err
		}
		m.mu.RLock()
		s, ok = m.sessions[teamID]
		m.mu.RUnlock()
		if !ok {
			return fmt.Errorf("session WA tidak aktif")
		}
	}
	if s.status == "qr" || s.status == "connecting" {
		return nil
	}
	return m.beginQRLogin(teamID)
}

func (m *Manager) beginQRLogin(teamID uuid.UUID) error {
	m.mu.RLock()
	s, ok := m.sessions[teamID]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("session WA tidak aktif")
	}

	qrChan, err := s.client.GetQRChannel(context.Background())
	if err != nil {
		return err
	}

	m.mu.Lock()
	s.loginMode = "qr"
	s.status = "connecting"
	s.pairCode = ""
	m.mu.Unlock()

	go func() {
		if err := s.client.Connect(); err != nil {
			log.Printf("wa connect %s: %v", teamID, err)
			m.setStatus(teamID, "error")
		}
	}()
	go func() {
		for evt := range qrChan {
			switch evt.Event {
			case whatsmeow.QRChannelEventCode:
				m.setQR(teamID, evt.Code, evt.Timeout)
			case "success":
				m.onConnected(teamID)
			case whatsmeow.QRChannelEventError:
				log.Printf("wa qr error team %s: %v", teamID, evt.Error)
				m.setStatus(teamID, "error")
			case "timeout", "err-client-outdated", "err-scanned-without-multidevice", "err-unexpected-state":
				m.setStatus(teamID, "error")
			}
		}
	}()
	return nil
}

func (m *Manager) StartPairLogin(teamID uuid.UUID, phone string) (string, error) {
	normalized, err := normalizePairPhone(phone)
	if err != nil {
		return "", err
	}

	m.mu.RLock()
	s, ok := m.sessions[teamID]
	m.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("session WA tidak aktif — aktifkan WA terlebih dahulu")
	}
	if s.client.Store.ID != nil {
		return "", fmt.Errorf("WA sudah terhubung")
	}
	if s.loginMode == "qr" {
		if err := m.restartSession(teamID); err != nil {
			return "", err
		}
		m.mu.RLock()
		s, ok = m.sessions[teamID]
		m.mu.RUnlock()
		if !ok {
			return "", fmt.Errorf("session WA tidak aktif")
		}
	}

	if !s.client.IsConnected() {
		m.mu.Lock()
		s.loginMode = "pair"
		s.status = "connecting"
		m.mu.Unlock()
		if err := s.client.Connect(); err != nil {
			return "", fmt.Errorf("gagal connect: %w", err)
		}
		for i := 0; i < 40; i++ {
			if s.client.IsConnected() {
				break
			}
			time.Sleep(250 * time.Millisecond)
		}
		if !s.client.IsConnected() {
			return "", fmt.Errorf("timeout menunggu koneksi WA")
		}
	}

	code, err := s.client.PairPhone(context.Background(), normalized, true, whatsmeow.PairClientChrome, "Chrome (Windows)")
	if err != nil {
		return "", fmt.Errorf("gagal buat kode: %w", err)
	}

	m.mu.Lock()
	if sess, ok := m.sessions[teamID]; ok {
		sess.loginMode = "pair"
		sess.status = "pair_code"
		sess.pairCode = code
		sess.pairCodeAt = time.Now()
		sess.qrCode = ""
	}
	m.mu.Unlock()
	return code, nil
}

func normalizePairPhone(phone string) (string, error) {
	var b strings.Builder
	for _, r := range phone {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	digits := b.String()
	if len(digits) < 8 {
		return "", fmt.Errorf("nomor terlalu pendek")
	}
	if strings.HasPrefix(digits, "0") {
		return "", fmt.Errorf("gunakan format internasional tanpa 0 di depan (contoh: 62812xxxxxxx)")
	}
	return digits, nil
}

func (m *Manager) StopTeam(teamID uuid.UUID) {
	m.mu.Lock()
	s, ok := m.sessions[teamID]
	if ok {
		delete(m.sessions, teamID)
	}
	m.mu.Unlock()
	if ok && s.client != nil {
		s.client.Disconnect()
	}
}

func (m *Manager) isAuthenticated(teamID uuid.UUID) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[teamID]
	return ok && s.client != nil && s.client.Store.ID != nil
}

func (m *Manager) reconcileConnectedStatus(teamID uuid.UUID) {
	m.mu.RLock()
	s, ok := m.sessions[teamID]
	if !ok || s.client == nil || s.client.Store.ID == nil || !s.client.IsConnected() {
		m.mu.RUnlock()
		return
	}
	status := s.status
	m.mu.RUnlock()

	switch status {
	case "connected", "qr", "pair_code":
		return
	default:
		m.onConnected(teamID)
	}
}

func (m *Manager) GetStatus(teamID uuid.UUID) (ConnectStatus, error) {
	m.mu.RLock()
	s, ok := m.sessions[teamID]
	m.mu.RUnlock()
	if !ok {
		return ConnectStatus{Status: "disconnected"}, nil
	}

	m.reconcileConnectedStatus(teamID)

	m.mu.RLock()
	s, ok = m.sessions[teamID]
	m.mu.RUnlock()
	if !ok {
		return ConnectStatus{Status: "disconnected"}, nil
	}

	out := ConnectStatus{
		Status:    s.status,
		Phone:     s.phone,
		PairCode:  s.pairCode,
		LoginMode: s.loginMode,
	}

	if s.qrCode != "" && s.status != "connected" {
		png, err := qrcode.Encode(s.qrCode, qrcode.Medium, 256)
		if err == nil {
			out.QR = "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
		}
		if s.qrTimeout > 0 {
			out.QRTimeoutSeconds = int(s.qrTimeout.Seconds())
		} else {
			out.QRTimeoutSeconds = 20
		}
	}

	if s.pairCode != "" && s.status == "pair_code" {
		remaining := int(pairCodeLifetime.Seconds() - time.Since(s.pairCodeAt).Seconds())
		if remaining < 0 {
			remaining = 0
		}
		out.PairCodeExpiresSec = remaining
	}

	if s.status == "connected" {
		out.DisplayName = s.displayName
		out.PictureURL = s.pictureURL
		if time.Since(s.profileFetchedAt) > profileCacheTTL {
			go m.refreshProfile(teamID)
		}
	}

	return out, nil
}

func (m *Manager) GetWAProfile(teamID uuid.UUID) (name, pictureURL string) {
	m.mu.RLock()
	s, ok := m.sessions[teamID]
	m.mu.RUnlock()
	if !ok {
		return "", ""
	}
	if s.status == "connected" && time.Since(s.profileFetchedAt) > profileCacheTTL {
		m.refreshProfile(teamID)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.sessions[teamID]; ok {
		return s.displayName, s.pictureURL
	}
	return "", ""
}

func (m *Manager) OpenWAAvatar(teamID uuid.UUID) (io.ReadCloser, string, error) {
	m.mu.RLock()
	s, ok := m.sessions[teamID]
	pictureURL := ""
	if ok {
		pictureURL = s.pictureURL
	}
	m.mu.RUnlock()
	if !ok || pictureURL == "" {
		if ok && s.status == "connected" {
			m.refreshProfile(teamID)
		}
		m.mu.RLock()
		s, ok = m.sessions[teamID]
		if ok {
			pictureURL = s.pictureURL
		}
		m.mu.RUnlock()
	}
	if pictureURL == "" {
		return nil, "", fmt.Errorf("avatar not available")
	}
	resp, err := http.Get(pictureURL)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, "", fmt.Errorf("avatar download: %s", resp.Status)
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}
	return resp.Body, contentType, nil
}

func (m *Manager) onConnected(teamID uuid.UUID) {
	m.setStatus(teamID, "connected")
	m.mu.Lock()
	if s, ok := m.sessions[teamID]; ok {
		s.pairCode = ""
	}
	m.mu.Unlock()
	phone := m.persistPhone(teamID)
	_ = m.repo.UpdateWAIntegration(context.Background(), teamID, true, "connected", nil, phone, nil)
	go func() {
		time.Sleep(2 * time.Second)
		m.refreshProfile(teamID)
	}()
}

func (m *Manager) refreshProfile(teamID uuid.UUID) {
	m.mu.RLock()
	s, ok := m.sessions[teamID]
	m.mu.RUnlock()
	if !ok || s.client == nil || s.client.Store.ID == nil {
		return
	}

	ctx := context.Background()
	jid := s.client.Store.ID.ToNonAD()
	name := strings.TrimSpace(s.client.Store.PushName)
	if name == "" {
		name = strings.TrimSpace(s.client.Store.BusinessName)
	}
	pictureURL := ""

	if s.client.IsConnected() {
		users, err := s.client.GetUserInfo(ctx, []types.JID{jid})
		if err == nil {
			if info, ok := users[jid]; ok && info.VerifiedName != nil {
				if vn := info.VerifiedName.Details.GetVerifiedName(); vn != "" {
					name = vn
				}
			}
		}
		picInfo, err := s.client.GetProfilePictureInfo(ctx, jid, &whatsmeow.GetProfilePictureParams{Preview: true})
		if err == nil && picInfo != nil {
			pictureURL = picInfo.URL
		} else if err != nil && !errors.Is(err, whatsmeow.ErrProfilePictureNotSet) && !errors.Is(err, whatsmeow.ErrProfilePictureUnauthorized) {
			log.Printf("wa: profile picture team %s: %v", teamID, err)
		}
	}

	phone := formatWAPhone(jid.User)
	var phonePtr, namePtr *string
	if phone != "" {
		phonePtr = &phone
	}
	if name != "" {
		namePtr = &name
	}

	m.mu.Lock()
	if sess, ok := m.sessions[teamID]; ok {
		sess.phone = phone
		sess.displayName = name
		sess.pictureURL = pictureURL
		sess.profileFetchedAt = time.Now()
	}
	m.mu.Unlock()

	_ = m.repo.UpdateWAIntegration(ctx, teamID, true, "connected", nil, phonePtr, namePtr)
}

func (m *Manager) persistPhone(teamID uuid.UUID) *string {
	m.mu.RLock()
	s, ok := m.sessions[teamID]
	m.mu.RUnlock()
	if !ok || s.client == nil || s.client.Store.ID == nil {
		return nil
	}
	formatted := formatWAPhone(s.client.Store.ID.User)
	if formatted == "" {
		return nil
	}
	m.mu.Lock()
	if sess, ok := m.sessions[teamID]; ok {
		sess.phone = formatted
	}
	m.mu.Unlock()
	return &formatted
}

func formatWAPhone(user string) string {
	user = strings.TrimSpace(user)
	if user == "" {
		return ""
	}
	if strings.HasPrefix(user, "+") {
		return user
	}
	return "+" + user
}

func (m *Manager) setQR(teamID uuid.UUID, code string, timeout time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[teamID]; ok {
		s.qrCode = code
		s.qrTimeout = timeout
		s.qrUpdatedAt = time.Now()
		s.status = "qr"
	}
}

func (m *Manager) setStatus(teamID uuid.UUID, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[teamID]; ok {
		s.status = status
		if status == "connected" {
			s.qrCode = ""
			s.pairCode = ""
		}
	}
}

func (m *Manager) handleEvent(teamID uuid.UUID, evt any) {
	switch v := evt.(type) {
	case *events.Message:
		m.handleMessage(teamID, v)
	case *events.Connected:
		if m.isAuthenticated(teamID) || m.sessionLoginMode(teamID) != "" {
			m.onConnected(teamID)
		}
	case *events.Disconnected:
		if !m.isAuthenticated(teamID) {
			m.setStatus(teamID, "disconnected")
			_ = m.repo.UpdateWAIntegration(context.Background(), teamID, true, "disconnected", nil, nil, nil)
			return
		}
		m.setStatus(teamID, "connecting")
	}
}

func (m *Manager) sessionLoginMode(teamID uuid.UUID) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.sessions[teamID]; ok {
		return s.loginMode
	}
	return ""
}

func (m *Manager) shouldProcessMessage(teamID uuid.UUID, msg *events.Message) bool {
	if msg.Message == nil {
		return false
	}
	if msg.Info.Chat.Server == "broadcast" || msg.Info.Chat.Server == "newsletter" {
		return false
	}
	if msg.Message.ProtocolMessage != nil || msg.Message.SenderKeyDistributionMessage != nil {
		return false
	}
	if time.Since(msg.Info.Timestamp) > 3*time.Minute {
		return false
	}

	m.mu.RLock()
	s := m.sessions[teamID]
	m.mu.RUnlock()
	if s == nil || s.client == nil {
		return false
	}
	// Pesan sendiri di chat orang lain diabaikan. Self-chat (catatan/nomor sendiri) tetap diproses.
	if msg.Info.IsFromMe && !isOwnChat(s.client, msg.Info.Chat) {
		return false
	}

	key := teamID.String() + ":" + msg.Info.ID
	if _, seen := m.seenMessages.Load(key); seen {
		return false
	}
	m.seenMessages.Store(key, time.Now())
	return true
}

func isOwnChat(client *whatsmeow.Client, chat types.JID) bool {
	if client == nil || client.Store == nil || chat.IsEmpty() {
		return false
	}
	if chat.Server == types.GroupServer || chat.Server == "broadcast" || chat.Server == "newsletter" {
		return false
	}
	own := client.Store.GetJID()
	if !own.IsEmpty() && chat.User == own.User {
		return true
	}
	lid := client.Store.GetLID()
	if !lid.IsEmpty() && chat.User == lid.User {
		return true
	}
	return false
}

func messageSenderPhones(msg *events.Message, client *whatsmeow.Client) []string {
	seen := make(map[string]struct{})
	var out []string
	addDigits := func(digits string) {
		digits = models.NormalizeWAPhoneDigits(digits)
		if digits == "" {
			return
		}
		if _, ok := seen[digits]; ok {
			return
		}
		seen[digits] = struct{}{}
		out = append(out, digits)
	}
	addJID := func(jid types.JID) {
		if jid.IsEmpty() {
			return
		}
		if jid.Server == types.DefaultUserServer || jid.Server == types.LegacyUserServer {
			addDigits(jid.User)
		}
	}

	info := msg.Info
	addJID(info.Sender)
	addJID(info.SenderAlt)
	addJID(info.Chat)
	addJID(info.RecipientAlt)

	if client != nil && client.Store != nil && client.Store.LIDs != nil {
		ctx := context.Background()
		for _, jid := range []types.JID{info.Sender, info.Chat, info.SenderAlt} {
			if jid.IsEmpty() || jid.Server != types.HiddenUserServer {
				continue
			}
			pn, err := client.Store.LIDs.GetPNForLID(ctx, jid)
			if err == nil {
				addJID(pn)
			}
		}
	}
	if msg.Info.IsFromMe && client != nil && client.Store != nil {
		addJID(client.Store.GetJID())
	}
	return out
}

func (m *Manager) handleMessage(teamID uuid.UUID, msg *events.Message) {
	if !m.shouldProcessMessage(teamID, msg) {
		return
	}

	ctx := context.Background()
	integ, err := m.repo.GetIntegration(ctx, teamID)
	if err != nil {
		return
	}

	m.mu.RLock()
	s, ok := m.sessions[teamID]
	m.mu.RUnlock()
	if !ok {
		return
	}

	senders := messageSenderPhones(msg, s.client)
	if !integ.AnySenderAllowed(senders) {
		log.Printf("wa: ignored message from %v (chat=%s sender=%s alt=%s) team %s (not whitelisted)",
			senders, msg.Info.Chat.String(), msg.Info.Sender.String(), msg.Info.SenderAlt.String(), teamID)
		return
	}

	team, err := m.repo.GetTeam(ctx, teamID)
	if err != nil {
		return
	}

	text := extractText(msg.Message)
	if text == "" && msg.Message.ImageMessage == nil {
		return
	}

	reply := func(body string) {
		jid := msg.Info.Chat
		_, _ = s.client.SendMessage(ctx, jid, &waProto.Message{
			Conversation: &body,
		})
	}

	if bot.IsLinkCommand(strings.TrimSpace(text)) {
		reply(m.formatLinkReply(ctx, teamID, team.Name))
		return
	}

	if strings.EqualFold(strings.TrimSpace(text), "!saldo") {
		balance, err := m.repo.GetBalance(ctx, teamID, nil, nil)
		if err != nil {
			reply("❌ Gagal cek saldo")
			return
		}
		reply(bot.FormatSaldoReply(team.Name, balance.CurrentBalance))
		return
	}

	if msg.Message.ImageMessage != nil {
		m.enqueueWAImage(ctx, teamID, s, msg, team.Name, reply)
		return
	}

	parsed, err := bot.ParseMessage(text, models.SourceWA)
	if err != nil {
		if err == bot.ErrSaldoCommand {
			balance, err := m.repo.GetBalance(ctx, teamID, nil, nil)
			if err != nil {
				reply("❌ Gagal cek saldo")
				return
			}
			reply(bot.FormatSaldoReply(team.Name, balance.CurrentBalance))
			return
		}
		if err == bot.ErrLinkCommand {
			reply(m.formatLinkReply(ctx, teamID, team.Name))
			return
		}
		reply(bot.FormatErrorReply(err))
		return
	}

	tx, balance, err := m.svc.CreateTransactionFromBot(ctx, teamID, *parsed, nil)
	if err != nil {
		reply("❌ Gagal simpan: " + err.Error())
		return
	}
	reply(bot.FormatSuccessReply(tx, balance.CurrentBalance, team.Name, 0))
}

func extractText(msg *waProto.Message) string {
	if msg.Conversation != nil {
		return *msg.Conversation
	}
	if msg.ExtendedTextMessage != nil && msg.ExtendedTextMessage.Text != nil {
		return *msg.ExtendedTextMessage.Text
	}
	if msg.ImageMessage != nil && msg.ImageMessage.Caption != nil {
		return *msg.ImageMessage.Caption
	}
	return ""
}

func (m *Manager) formatLinkReply(ctx context.Context, teamID uuid.UUID, teamName string) string {
	rt, err := m.repo.GetReportToken(ctx, teamID)
	if err != nil || !rt.IsActive {
		return "❌ Link laporan publik belum tersedia"
	}
	return bot.FormatLinkReply(teamName, bot.PublicReportURL(m.appURL, rt.Token))
}
