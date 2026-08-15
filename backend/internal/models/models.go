package models

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	RoleAdmin UserRole = "admin"
	RoleOps   UserRole = "ops"
)

type TxJenis string

const (
	JenisIn  TxJenis = "in"
	JenisOut TxJenis = "out"
)

type TxSource string

const (
	SourceWeb  TxSource = "web"
	SourceWA   TxSource = "wa"
	SourceTele TxSource = "tele"
)

type Team struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	InitialBalance int64     `json:"initial_balance"`
	CreatedAt      time.Time `json:"created_at"`
}

type User struct {
	ID                   uuid.UUID  `json:"id"`
	TeamID               *uuid.UUID `json:"team_id,omitempty"`
	Name                 string     `json:"name"`
	Email                string     `json:"email"`
	PasswordHash         string     `json:"-"`
	Role                 UserRole   `json:"role"`
	EmailVerified        bool       `json:"email_verified"`
	AvatarFile           *string    `json:"-"`
	CreatedAt            time.Time  `json:"created_at"`
}

type Transaction struct {
	ID          uuid.UUID  `json:"id"`
	TeamID      uuid.UUID  `json:"team_id"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	Hari        string     `json:"hari"`
	Tanggal     time.Time  `json:"tanggal"`
	Jenis       TxJenis    `json:"jenis"`
	Deskripsi   string     `json:"deskripsi"`
	Total       int64      `json:"total"`
	NotaKey     *string    `json:"nota_key,omitempty"`
	Keterangan  *string    `json:"keterangan,omitempty"`
	Source      TxSource   `json:"source"`
	CreatedAt   time.Time  `json:"created_at"`
	CreatorName *string    `json:"creator_name,omitempty"`
}

type Integration struct {
	TeamID             uuid.UUID `json:"team_id"`
	WAEnabled          bool      `json:"wa_enabled"`
	WAStatus           string    `json:"wa_status"`
	WAPhone            *string   `json:"wa_phone,omitempty"`
	WAName             *string   `json:"wa_name,omitempty"`
	TeleEnabled        bool      `json:"tele_enabled"`
	TeleBotToken       *string   `json:"tele_bot_token,omitempty"`
	TeleAllowedChatID  *int64    `json:"tele_allowed_chat_id,omitempty"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type ReportToken struct {
	TeamID    uuid.UUID `json:"team_id"`
	Token     string    `json:"token"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type Balance struct {
	InitialBalance int64   `json:"initial_balance"`
	OpeningBalance int64   `json:"opening_balance"`
	TotalIn        int64   `json:"total_in"`
	TotalOut       int64   `json:"total_out"`
	CurrentBalance int64   `json:"current_balance"`
	PeriodFrom     *string `json:"period_from,omitempty"`
	PeriodTo       *string `json:"period_to,omitempty"`
}

type AppSettings struct {
	AppName      string    `json:"app_name"`
	AppTagline   string    `json:"app_tagline"`
	LogoFile     *string   `json:"-"`
	FaviconFile  *string   `json:"-"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateTransactionInput struct {
	Hari       string
	Tanggal    time.Time
	Jenis      TxJenis
	Deskripsi  string
	Total      int64
	NotaKey    *string
	Keterangan *string
	Source     TxSource
	CreatedBy  *uuid.UUID
}

type UpdateTransactionInput struct {
	Hari       string
	Tanggal    time.Time
	Jenis      TxJenis
	Deskripsi  string
	Total      int64
	NotaKey    *string
	Keterangan *string
}
