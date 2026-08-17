package email

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/mail"
	"net/smtp"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kasq/backend/internal/config"
)

type Sender struct {
	cfg          config.SMTPConfig
	envelopeFrom string
	headerFrom   string
}

var emailAddrRe = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)

func NewSender(cfg config.SMTPConfig) *Sender {
	headerFrom := cfg.From
	if headerFrom == "" {
		headerFrom = cfg.User
	}
	envelopeFrom := extractEmail(headerFrom)
	if envelopeFrom == "" {
		envelopeFrom = cfg.User
	}
	if cfg.Enabled {
		log.Printf("smtp: enabled host=%s port=%d user=%s from=%s", cfg.Host, cfg.Port, cfg.User, envelopeFrom)
	} else {
		log.Printf("smtp: disabled (set SMTP_HOST + SMTP_USER + SMTP_PASS)")
	}
	return &Sender{cfg: cfg, envelopeFrom: envelopeFrom, headerFrom: headerFrom}
}

func extractEmail(s string) string {
	s = strings.Trim(s, `"' `)
	if addr, err := mail.ParseAddress(s); err == nil {
		return addr.Address
	}
	if m := emailAddrRe.FindString(s); m != "" {
		return m
	}
	return s
}

func encodeSubject(subject string) string {
	return fmt.Sprintf("=?UTF-8?B?%s?=", base64.StdEncoding.EncodeToString([]byte(subject)))
}

func (s *Sender) buildMessage(to, subject, bodyHTML, bodyPlain string) []byte {
	boundary := uuid.New().String()
	headers := []string{
		"From: " + s.headerFrom,
		"To: " + to,
		"Subject: " + encodeSubject(subject),
		"Date: " + time.Now().Format(time.RFC1123Z),
		"Message-ID: <" + uuid.New().String() + "@" + s.cfg.Host + ">",
		"Reply-To: " + s.envelopeFrom,
		"MIME-Version: 1.0",
		"Content-Type: multipart/alternative; boundary=" + boundary,
		"",
		"--" + boundary,
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		bodyPlain,
		"--" + boundary,
		"Content-Type: text/html; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		bodyHTML,
		"--" + boundary + "--",
	}
	return []byte(strings.Join(headers, "\r\n"))
}

func (s *Sender) Send(to, subject, bodyHTML, bodyPlain string) error {
	if !s.cfg.Enabled {
		return fmt.Errorf("SMTP belum dikonfigurasi (set SMTP_HOST, SMTP_USER, SMTP_PASS di .env)")
	}

	msg := s.buildMessage(to, subject, bodyHTML, bodyPlain)
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	authHost := s.authHost()
	auth := smtp.PlainAuth("", s.cfg.User, s.cfg.Password, authHost)

	var err error
	switch {
	case s.cfg.Port == 465:
		err = sendSMTPS(addr, s.tlsConfig(), auth, s.envelopeFrom, to, msg)
	default:
		err = sendSTARTTLS(addr, s.tlsConfig(), auth, s.envelopeFrom, to, msg)
	}
	if err != nil {
		log.Printf("smtp: send to %s failed: %v", to, err)
		return fmt.Errorf("gagal kirim email: %w", err)
	}
	log.Printf("smtp: accepted by server %s -> %s subject=%q", s.cfg.Host, to, subject)
	return nil
}

func (s *Sender) authHost() string {
	if s.cfg.TLSServerName != "" {
		return s.cfg.TLSServerName
	}
	return s.cfg.Host
}

func (s *Sender) tlsConfig() *tls.Config {
	serverName := s.cfg.TLSServerName
	if serverName == "" {
		serverName = s.cfg.Host
	}
	return &tls.Config{
		ServerName:         serverName,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: s.cfg.SkipTLSVerify,
	}
}

func sendSTARTTLS(addr string, tlsCfg *tls.Config, auth smtp.Auth, from, to string, msg []byte) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, tlsCfg.ServerName)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err = client.StartTLS(tlsCfg); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}

	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err = client.Auth(auth); err != nil {
				return fmt.Errorf("auth: %w", err)
			}
		}
	}

	if err = client.Mail(from); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("rcpt to: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err = w.Write(msg); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("close data: %w", err)
	}
	return client.Quit()
}

func sendSMTPS(addr string, tlsCfg *tls.Config, auth smtp.Auth, from, to string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	client, err := smtp.NewClient(conn, tlsCfg.ServerName)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("auth: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("rcpt to: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func VerificationEmail(name, verifyURL string) (subject, html, plain string) {
	subject = "Konfirmasi Email — KasQ"
	html = fmt.Sprintf(`<p>Halo <strong>%s</strong>,</p>
<p>Terima kasih telah mendaftar di KasQ. Klik tombol di bawah untuk mengaktifkan akun:</p>
<p><a href="%s" style="display:inline-block;padding:12px 24px;background:#059669;color:#fff;text-decoration:none;border-radius:8px;">Konfirmasi Email</a></p>
<p>Atau salin link: %s</p>
<p>Link berlaku 24 jam.</p>`, name, verifyURL, verifyURL)
	plain = fmt.Sprintf("Halo %s,\n\nKonfirmasi email KasQ:\n%s\n\nLink berlaku 24 jam.", name, verifyURL)
	return subject, html, plain
}

func ResetPasswordEmail(name, resetURL string) (subject, html, plain string) {
	subject = "Reset Password — KasQ"
	html = fmt.Sprintf(`<p>Halo <strong>%s</strong>,</p>
<p>Klik link berikut untuk reset password:</p>
<p><a href="%s" style="display:inline-block;padding:12px 24px;background:#059669;color:#fff;text-decoration:none;border-radius:8px;">Reset Password</a></p>
<p>Atau salin link: %s</p>
<p>Link berlaku 1 jam.</p>`, name, resetURL, resetURL)
	plain = fmt.Sprintf("Halo %s,\n\nReset password KasQ:\n%s\n\nLink berlaku 1 jam.", name, resetURL)
	return subject, html, plain
}
