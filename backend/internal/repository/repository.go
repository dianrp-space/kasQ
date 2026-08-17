package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kasq/backend/internal/models"
)

var ErrNotFound = errors.New("not found")
var ErrTokenTaken = errors.New("report token already taken")

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, team_id, name, email, password_hash, role, email_verified, avatar_file, created_at
		FROM users WHERE email = $1`, email)
	return scanUser(row)
}

func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, team_id, name, email, password_hash, role, email_verified, avatar_file, created_at
		FROM users WHERE id = $1`, id)
	return scanUser(row)
}

func (r *Repository) ListUsers(ctx context.Context) ([]models.User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, team_id, name, email, password_hash, role, email_verified, avatar_file, created_at
		FROM users ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []models.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}

func (r *Repository) CreateUser(ctx context.Context, u *models.User) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO users (team_id, name, email, password_hash, role, email_verified)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`,
		u.TeamID, u.Name, u.Email, u.PasswordHash, u.Role, u.EmailVerified,
	).Scan(&u.ID, &u.CreatedAt)
}

func (r *Repository) UpdateUser(ctx context.Context, u *models.User) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE users SET team_id=$2, name=$3, email=$4, password_hash=$5, role=$6
		WHERE id=$1`,
		u.ID, u.TeamID, u.Name, u.Email, u.PasswordHash, u.Role,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteUser(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) ListTeams(ctx context.Context) ([]models.Team, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, slug, initial_balance, created_at FROM teams ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var teams []models.Team
	for rows.Next() {
		var t models.Team
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.InitialBalance, &t.CreatedAt); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, rows.Err()
}

func (r *Repository) GetTeam(ctx context.Context, id uuid.UUID) (*models.Team, error) {
	var t models.Team
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, slug, initial_balance, created_at FROM teams WHERE id=$1`, id,
	).Scan(&t.ID, &t.Name, &t.Slug, &t.InitialBalance, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &t, err
}

func (r *Repository) CreateTeam(ctx context.Context, t *models.Team) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO teams (name, slug, initial_balance) VALUES ($1, $2, $3)
		RETURNING id, created_at`, t.Name, t.Slug, t.InitialBalance,
	).Scan(&t.ID, &t.CreatedAt)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO integrations (team_id) VALUES ($1)`, t.ID)
	if err != nil {
		return err
	}
	token := t.Slug
	_, err = r.pool.Exec(ctx, `INSERT INTO report_tokens (team_id, token) VALUES ($1, $2)`, t.ID, token)
	return err
}

func (r *Repository) UpdateTeam(ctx context.Context, t *models.Team) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE teams SET name=$2, slug=$3, initial_balance=$4 WHERE id=$1`,
		t.ID, t.Name, t.Slug, t.InitialBalance,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteTeam(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM teams WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) CreateTransaction(ctx context.Context, teamID uuid.UUID, input models.CreateTransactionInput) (*models.Transaction, error) {
	var tx models.Transaction
	err := r.pool.QueryRow(ctx, `
		INSERT INTO transactions (team_id, created_by, hari, tanggal, jenis, deskripsi, total, nota_key, keterangan, source)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, team_id, created_by, hari, tanggal, jenis, deskripsi, total, nota_key, keterangan, source, created_at`,
		teamID, input.CreatedBy, input.Hari, input.Tanggal, input.Jenis, input.Deskripsi,
		input.Total, input.NotaKey, input.Keterangan, input.Source,
	).Scan(&tx.ID, &tx.TeamID, &tx.CreatedBy, &tx.Hari, &tx.Tanggal, &tx.Jenis,
		&tx.Deskripsi, &tx.Total, &tx.NotaKey, &tx.Keterangan, &tx.Source, &tx.CreatedAt)
	return &tx, err
}

type TxFilter struct {
	TeamID    uuid.UUID
	Jenis     *models.TxJenis
	DateFrom  *time.Time
	DateTo    *time.Time
	Limit     int
	Offset    int
}

func txFilterWhere(f TxFilter) (string, []any, int) {
	args := []any{f.TeamID}
	where := []string{"t.team_id = $1"}
	idx := 2
	if f.Jenis != nil {
		where = append(where, fmt.Sprintf("t.jenis = $%d", idx))
		args = append(args, *f.Jenis)
		idx++
	}
	if f.DateFrom != nil {
		where = append(where, fmt.Sprintf("t.tanggal >= $%d", idx))
		args = append(args, *f.DateFrom)
		idx++
	}
	if f.DateTo != nil {
		where = append(where, fmt.Sprintf("t.tanggal <= $%d", idx))
		args = append(args, *f.DateTo)
		idx++
	}
	return strings.Join(where, " AND "), args, idx
}

func (r *Repository) CountTransactions(ctx context.Context, f TxFilter) (int, error) {
	where, args, _ := txFilterWhere(f)
	var count int
	err := r.pool.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM transactions t WHERE %s`, where), args...).Scan(&count)
	return count, err
}

func (r *Repository) ListTransactions(ctx context.Context, f TxFilter) ([]models.Transaction, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}
	where, args, idx := txFilterWhere(f)
	args = append(args, f.Limit, f.Offset)
	query := fmt.Sprintf(`
		SELECT t.id, t.team_id, t.created_by, t.hari, t.tanggal, t.jenis, t.deskripsi,
		       t.total, t.nota_key, t.keterangan, t.source, t.created_at, u.name
		FROM transactions t
		LEFT JOIN users u ON u.id = t.created_by
		WHERE %s
		ORDER BY t.tanggal DESC, t.created_at DESC
		LIMIT $%d OFFSET $%d`, where, idx, idx+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var txs []models.Transaction
	for rows.Next() {
		var tx models.Transaction
		if err := rows.Scan(&tx.ID, &tx.TeamID, &tx.CreatedBy, &tx.Hari, &tx.Tanggal,
			&tx.Jenis, &tx.Deskripsi, &tx.Total, &tx.NotaKey, &tx.Keterangan,
			&tx.Source, &tx.CreatedAt, &tx.CreatorName); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, rows.Err()
}

func (r *Repository) GetTransaction(ctx context.Context, teamID, txID uuid.UUID) (*models.Transaction, error) {
	var tx models.Transaction
	err := r.pool.QueryRow(ctx, `
		SELECT t.id, t.team_id, t.created_by, t.hari, t.tanggal, t.jenis, t.deskripsi,
		       t.total, t.nota_key, t.keterangan, t.source, t.created_at, u.name
		FROM transactions t
		LEFT JOIN users u ON u.id = t.created_by
		WHERE t.id = $1 AND t.team_id = $2`,
		txID, teamID,
	).Scan(&tx.ID, &tx.TeamID, &tx.CreatedBy, &tx.Hari, &tx.Tanggal,
		&tx.Jenis, &tx.Deskripsi, &tx.Total, &tx.NotaKey, &tx.Keterangan,
		&tx.Source, &tx.CreatedAt, &tx.CreatorName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &tx, err
}

func (r *Repository) UpdateTransaction(ctx context.Context, teamID, txID uuid.UUID, input models.UpdateTransactionInput) (*models.Transaction, error) {
	var tx models.Transaction
	err := r.pool.QueryRow(ctx, `
		UPDATE transactions
		SET hari=$1, tanggal=$2, jenis=$3, deskripsi=$4, total=$5, nota_key=$6, keterangan=$7
		WHERE id=$8 AND team_id=$9
		RETURNING id, team_id, created_by, hari, tanggal, jenis, deskripsi, total, nota_key, keterangan, source, created_at`,
		input.Hari, input.Tanggal, input.Jenis, input.Deskripsi, input.Total, input.NotaKey, input.Keterangan,
		txID, teamID,
	).Scan(&tx.ID, &tx.TeamID, &tx.CreatedBy, &tx.Hari, &tx.Tanggal,
		&tx.Jenis, &tx.Deskripsi, &tx.Total, &tx.NotaKey, &tx.Keterangan,
		&tx.Source, &tx.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &tx, err
}

func (r *Repository) DeleteTransaction(ctx context.Context, teamID, txID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM transactions WHERE id=$1 AND team_id=$2`, txID, teamID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type ImportTxKey struct {
	Tanggal   time.Time
	Jenis     models.TxJenis
	Deskripsi string
	Total     int64
}

func (r *Repository) ListImportTxKeys(ctx context.Context, teamID uuid.UUID) ([]ImportTxKey, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT tanggal, jenis, deskripsi, total FROM transactions WHERE team_id = $1`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []ImportTxKey
	for rows.Next() {
		var k ImportTxKey
		if err := rows.Scan(&k.Tanggal, &k.Jenis, &k.Deskripsi, &k.Total); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (r *Repository) GetBalance(ctx context.Context, teamID uuid.UUID, dateFrom, dateTo *time.Time) (*models.Balance, error) {
	var initial int64
	err := r.pool.QueryRow(ctx, `SELECT initial_balance FROM teams WHERE id = $1`, teamID).Scan(&initial)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if dateFrom == nil || dateTo == nil {
		var totalIn, totalOut int64
		err = r.pool.QueryRow(ctx, `
			SELECT
				COALESCE(SUM(CASE WHEN jenis = 'in' THEN total ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN jenis = 'out' THEN total ELSE 0 END), 0)
			FROM transactions WHERE team_id = $1`, teamID,
		).Scan(&totalIn, &totalOut)
		if err != nil {
			return nil, err
		}
		return &models.Balance{
			InitialBalance: initial,
			OpeningBalance: initial,
			TotalIn:        totalIn,
			TotalOut:       totalOut,
			CurrentBalance: initial + totalIn - totalOut,
		}, nil
	}

	var beforeIn, beforeOut int64
	err = r.pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN jenis = 'in' THEN total ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN jenis = 'out' THEN total ELSE 0 END), 0)
		FROM transactions WHERE team_id = $1 AND tanggal < $2`, teamID, *dateFrom,
	).Scan(&beforeIn, &beforeOut)
	if err != nil {
		return nil, err
	}

	var periodIn, periodOut int64
	err = r.pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN jenis = 'in' THEN total ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN jenis = 'out' THEN total ELSE 0 END), 0)
		FROM transactions WHERE team_id = $1 AND tanggal >= $2 AND tanggal <= $3`,
		teamID, *dateFrom, *dateTo,
	).Scan(&periodIn, &periodOut)
	if err != nil {
		return nil, err
	}

	fromStr := dateFrom.Format("2006-01-02")
	toStr := dateTo.Format("2006-01-02")
	opening := initial + beforeIn - beforeOut
	return &models.Balance{
		InitialBalance: initial,
		OpeningBalance: opening,
		TotalIn:        periodIn,
		TotalOut:       periodOut,
		CurrentBalance: opening + periodIn - periodOut,
		PeriodFrom:     &fromStr,
		PeriodTo:       &toStr,
	}, nil
}

func (r *Repository) GetIntegration(ctx context.Context, teamID uuid.UUID) (*models.Integration, error) {
	var i models.Integration
	err := r.pool.QueryRow(ctx, `
		SELECT team_id, wa_enabled, wa_status, wa_phone, wa_name, tele_enabled, tele_bot_token, tele_allowed_chat_id, updated_at
		FROM integrations WHERE team_id = $1`, teamID,
	).Scan(&i.TeamID, &i.WAEnabled, &i.WAStatus, &i.WAPhone, &i.WAName, &i.TeleEnabled, &i.TeleBotToken, &i.TeleAllowedChatID, &i.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &i, err
}

func (r *Repository) UpdateWAIntegration(ctx context.Context, teamID uuid.UUID, enabled bool, status string, sessionData *string, phone *string, name *string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE integrations SET wa_enabled=$2, wa_status=$3, wa_session_data=$4, wa_phone=$5, wa_name=$6, updated_at=NOW()
		WHERE team_id=$1`, teamID, enabled, status, sessionData, phone, name)
	return err
}

func (r *Repository) UpdateTeleIntegration(ctx context.Context, teamID uuid.UUID, enabled bool, token *string, chatID *int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE integrations SET tele_enabled=$2, tele_bot_token=$3, tele_allowed_chat_id=$4, updated_at=NOW()
		WHERE team_id=$1`, teamID, enabled, token, chatID)
	return err
}

func (r *Repository) ListEnabledWAIntegrations(ctx context.Context) ([]models.Integration, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT team_id, wa_enabled, wa_status, wa_phone, wa_name, tele_enabled, tele_bot_token, tele_allowed_chat_id, updated_at
		FROM integrations WHERE wa_enabled = true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIntegrations(rows)
}

func (r *Repository) ListEnabledTeleIntegrations(ctx context.Context) ([]models.Integration, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT team_id, wa_enabled, wa_status, wa_phone, wa_name, tele_enabled, tele_bot_token, tele_allowed_chat_id, updated_at
		FROM integrations WHERE tele_enabled = true AND tele_bot_token IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIntegrations(rows)
}

func (r *Repository) GetWASessionData(ctx context.Context, teamID uuid.UUID) (*string, error) {
	var data *string
	err := r.pool.QueryRow(ctx, `SELECT wa_session_data FROM integrations WHERE team_id=$1`, teamID).Scan(&data)
	return data, err
}

func (r *Repository) SetWASessionData(ctx context.Context, teamID uuid.UUID, data *string, status string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE integrations SET wa_session_data=$2, wa_status=$3, updated_at=NOW() WHERE team_id=$1`,
		teamID, data, status)
	return err
}

func (r *Repository) GetReportToken(ctx context.Context, teamID uuid.UUID) (*models.ReportToken, error) {
	var rt models.ReportToken
	err := r.pool.QueryRow(ctx, `
		SELECT team_id, token, is_active, created_at FROM report_tokens WHERE team_id=$1`, teamID,
	).Scan(&rt.TeamID, &rt.Token, &rt.IsActive, &rt.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &rt, err
}

func (r *Repository) GetReportByToken(ctx context.Context, token string) (*models.Team, *models.ReportToken, error) {
	var t models.Team
	var rt models.ReportToken
	err := r.pool.QueryRow(ctx, `
		SELECT rt.team_id, rt.token, rt.is_active, rt.created_at,
		       tm.id, tm.name, tm.slug, tm.initial_balance, tm.created_at
		FROM report_tokens rt
		JOIN teams tm ON tm.id = rt.team_id
		WHERE rt.token = $1 AND rt.is_active = true`, token,
	).Scan(&rt.TeamID, &rt.Token, &rt.IsActive, &rt.CreatedAt,
		&t.ID, &t.Name, &t.Slug, &t.InitialBalance, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	return &t, &rt, err
}

func (r *Repository) SetReportToken(ctx context.Context, teamID uuid.UUID, token string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE report_tokens SET token=$2, created_at=NOW() WHERE team_id=$1`, teamID, token)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrTokenTaken
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		_, err = r.pool.Exec(ctx, `INSERT INTO report_tokens (team_id, token) VALUES ($1, $2)`, teamID, token)
		if err != nil && isUniqueViolation(err) {
			return ErrTokenTaken
		}
		return err
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (r *Repository) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

type scannable interface {
	Scan(dest ...any) error
}

func (r *Repository) UpdateUserProfile(ctx context.Context, id uuid.UUID, name string, avatarFile *string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE users SET name=$2, avatar_file=$3 WHERE id=$1`,
		id, name, avatarFile,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanUser(row scannable) (*models.User, error) {
	var u models.User
	err := row.Scan(&u.ID, &u.TeamID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.EmailVerified, &u.AvatarFile, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &u, err
}

func scanIntegrations(rows pgx.Rows) ([]models.Integration, error) {
	var list []models.Integration
	for rows.Next() {
		var i models.Integration
		if err := rows.Scan(&i.TeamID, &i.WAEnabled, &i.WAStatus, &i.WAPhone, &i.WAName, &i.TeleEnabled, &i.TeleBotToken, &i.TeleAllowedChatID, &i.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, i)
	}
	return list, rows.Err()
}
