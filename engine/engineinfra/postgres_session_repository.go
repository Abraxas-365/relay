package engineinfra

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Abraxas-365/craftable/errx"
	"github.com/Abraxas-365/craftable/storex"
	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/jmoiron/sqlx"
)

type PostgresSessionRepository struct {
	db *sqlx.DB
}

var _ engine.SessionRepository = (*PostgresSessionRepository)(nil)

func NewPostgresSessionRepository(db *sqlx.DB) *PostgresSessionRepository {
	return &PostgresSessionRepository{db: db}
}

// dbSession is an intermediate struct for database operations
type dbSession struct {
	ID             string          `db:"id"`
	TenantID       string          `db:"tenant_id"`
	ChannelID      string          `db:"channel_id"`
	SenderID       string          `db:"sender_id"`
	Context        json.RawMessage `db:"context"`
	History        json.RawMessage `db:"history"`
	CurrentState   string          `db:"current_state"`
	Status         string          `db:"status"`
	ExpiresAt      time.Time       `db:"expires_at"`
	CreatedAt      time.Time       `db:"created_at"`
	LastActivityAt time.Time       `db:"last_activity_at"`
	ClosedAt       *time.Time      `db:"closed_at"`
}

// toDBSession converts domain Session to dbSession
func toDBSession(session engine.Session) (*dbSession, error) {
	contextJSON := []byte("{}")
	if session.Context != nil && len(session.Context) > 0 {
		var err error
		contextJSON, err = json.Marshal(session.Context)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal context: %w", err)
		}
	}

	historyJSON := []byte("[]")
	if session.History != nil && len(session.History) > 0 {
		var err error
		historyJSON, err = json.Marshal(session.History)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal history: %w", err)
		}
	}

	return &dbSession{
		ID:             session.ID.String(),
		TenantID:       session.TenantID.String(),
		ChannelID:      session.ChannelID.String(),
		SenderID:       session.SenderID,
		Context:        contextJSON,
		History:        historyJSON,
		CurrentState:   session.CurrentState,
		Status:         string(session.Status),
		ExpiresAt:      session.ExpiresAt,
		CreatedAt:      session.CreatedAt,
		LastActivityAt: session.LastActivityAt,
		ClosedAt:       session.ClosedAt,
	}, nil
}

// toDomainSession converts dbSession to domain Session
func toDomainSession(dbSess *dbSession) (*engine.Session, error) {
	var context map[string]any
	if len(dbSess.Context) > 0 && string(dbSess.Context) != "null" {
		if err := json.Unmarshal(dbSess.Context, &context); err != nil {
			return nil, fmt.Errorf("failed to unmarshal context: %w", err)
		}
	}

	var history []engine.MessageRef
	if len(dbSess.History) > 0 && string(dbSess.History) != "null" {
		if err := json.Unmarshal(dbSess.History, &history); err != nil {
			return nil, fmt.Errorf("failed to unmarshal history: %w", err)
		}
	}

	return &engine.Session{
		ID:             kernel.SessionID(dbSess.ID),
		TenantID:       kernel.TenantID(dbSess.TenantID),
		ChannelID:      kernel.ChannelID(dbSess.ChannelID),
		SenderID:       dbSess.SenderID,
		Context:        context,
		History:        history,
		CurrentState:   dbSess.CurrentState,
		Status:         engine.SessionStatus(dbSess.Status),
		ExpiresAt:      dbSess.ExpiresAt,
		CreatedAt:      dbSess.CreatedAt,
		LastActivityAt: dbSess.LastActivityAt,
		ClosedAt:       dbSess.ClosedAt,
	}, nil
}

func (r *PostgresSessionRepository) Save(ctx context.Context, session engine.Session) error {
	exists, err := r.sessionExists(ctx, session.ID.String())
	if err != nil {
		return errx.Wrap(err, "failed to check session existence", errx.TypeInternal)
	}

	if exists {
		return r.update(ctx, session)
	}
	return r.create(ctx, session)
}

func (r *PostgresSessionRepository) create(ctx context.Context, session engine.Session) error {
	dbSess, err := toDBSession(session)
	if err != nil {
		return errx.Wrap(err, "failed to convert session", errx.TypeInternal).
			WithDetail("session_id", session.ID)
	}

	query := `
		INSERT INTO sessions (
			id, tenant_id, channel_id, sender_id, context,
			history, current_state, status, expires_at, created_at, last_activity_at, closed_at
		) VALUES (
			:id, :tenant_id, :channel_id, :sender_id, :context,
			:history, :current_state, :status, :expires_at, :created_at, :last_activity_at, :closed_at
		)`

	_, err = r.db.NamedExecContext(ctx, query, dbSess)
	if err != nil {
		log.Printf("Error inserting session: %v", err) // Debug log
		return errx.Wrap(err, "failed to create session", errx.TypeInternal).
			WithDetail("session_id", session.ID)
	}

	return nil
}

func (r *PostgresSessionRepository) update(ctx context.Context, session engine.Session) error {
	dbSess, err := toDBSession(session)
	if err != nil {
		return errx.Wrap(err, "failed to convert session", errx.TypeInternal).
			WithDetail("session_id", session.ID)
	}

	query := `
		UPDATE sessions SET
			context = :context,
			history = :history,
			current_state = :current_state,
			status = :status,
			expires_at = :expires_at,
			last_activity_at = :last_activity_at,
			closed_at = :closed_at
		WHERE id = :id`

	result, err := r.db.NamedExecContext(ctx, query, dbSess)
	if err != nil {
		return errx.Wrap(err, "failed to update session", errx.TypeInternal).
			WithDetail("session_id", session.ID)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errx.Wrap(err, "failed to get rows affected", errx.TypeInternal)
	}

	if rowsAffected == 0 {
		return engine.ErrSessionNotFound().WithDetail("session_id", session.ID)
	}

	return nil
}

func (r *PostgresSessionRepository) FindByID(ctx context.Context, id kernel.SessionID) (*engine.Session, error) {
	query := `
		SELECT 
			id, tenant_id, channel_id, sender_id, context,
			history, current_state, status, expires_at, created_at, last_activity_at, closed_at
		FROM sessions
		WHERE id = $1`

	var dbSess dbSession
	err := r.db.GetContext(ctx, &dbSess, query, string(id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, engine.ErrSessionNotFound().WithDetail("session_id", string(id))
		}
		return nil, errx.Wrap(err, "failed to find session by id", errx.TypeInternal).
			WithDetail("session_id", string(id))
	}

	return toDomainSession(&dbSess)
}

func (r *PostgresSessionRepository) Delete(ctx context.Context, id kernel.SessionID) error {
	query := `DELETE FROM sessions WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, string(id))
	if err != nil {
		return errx.Wrap(err, "failed to delete session", errx.TypeInternal).
			WithDetail("session_id", string(id))
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errx.Wrap(err, "failed to get rows affected", errx.TypeInternal)
	}

	if rowsAffected == 0 {
		return engine.ErrSessionNotFound().WithDetail("session_id", string(id))
	}

	return nil
}

func (r *PostgresSessionRepository) FindByChannelAndSender(ctx context.Context, channelID kernel.ChannelID, senderID string) (*engine.Session, error) {
	query := `
		SELECT 
			id, tenant_id, channel_id, sender_id, context,
			history, current_state, status, expires_at, created_at, last_activity_at, closed_at
		FROM sessions
		WHERE channel_id = $1 AND sender_id = $2
		ORDER BY created_at DESC
		LIMIT 1`

	var dbSess dbSession
	err := r.db.GetContext(ctx, &dbSess, query, channelID.String(), senderID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, engine.ErrSessionNotFound().
				WithDetail("channel_id", channelID.String()).
				WithDetail("sender_id", senderID)
		}
		return nil, errx.Wrap(err, "failed to find session by channel and sender", errx.TypeInternal).
			WithDetail("channel_id", channelID.String()).
			WithDetail("sender_id", senderID)
	}

	return toDomainSession(&dbSess)
}

func (r *PostgresSessionRepository) FindActiveByChannelAndSender(ctx context.Context, channelID kernel.ChannelID, senderID string) (*engine.Session, error) {
	query := `
		SELECT 
			id, tenant_id, channel_id, sender_id, context,
			history, current_state, status, expires_at, created_at, last_activity_at, closed_at
		FROM sessions
		WHERE channel_id = $1 AND sender_id = $2 AND status = 'ACTIVE'
		LIMIT 1`

	var dbSess dbSession
	err := r.db.GetContext(ctx, &dbSess, query, channelID.String(), senderID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, engine.ErrSessionNotFound().
				WithDetail("channel_id", channelID.String()).
				WithDetail("sender_id", senderID)
		}
		return nil, errx.Wrap(err, "failed to find active session by channel and sender", errx.TypeInternal).
			WithDetail("channel_id", channelID.String()).
			WithDetail("sender_id", senderID)
	}

	return toDomainSession(&dbSess)
}

func (r *PostgresSessionRepository) FindByChannel(ctx context.Context, channelID kernel.ChannelID) ([]*engine.Session, error) {
	query := `
		SELECT 
			id, tenant_id, channel_id, sender_id, context,
			history, current_state, status, expires_at, created_at, last_activity_at, closed_at
		FROM sessions
		WHERE channel_id = $1
		ORDER BY last_activity_at DESC`

	var dbSessions []dbSession
	err := r.db.SelectContext(ctx, &dbSessions, query, channelID.String())
	if err != nil {
		return nil, errx.Wrap(err, "failed to find sessions by channel", errx.TypeInternal).
			WithDetail("channel_id", channelID.String())
	}

	result := make([]*engine.Session, 0, len(dbSessions))
	for i := range dbSessions {
		session, err := toDomainSession(&dbSessions[i])
		if err != nil {
			return nil, errx.Wrap(err, "failed to convert session", errx.TypeInternal)
		}
		result = append(result, session)
	}

	return result, nil
}

func (r *PostgresSessionRepository) FindActive(ctx context.Context, tenantID kernel.TenantID) ([]*engine.Session, error) {
	query := `
		SELECT 
			id, tenant_id, channel_id, sender_id, context,
			history, current_state, status, expires_at, created_at, last_activity_at, closed_at
		FROM sessions
		WHERE tenant_id = $1 AND status = 'ACTIVE'
		ORDER BY last_activity_at DESC`

	var dbSessions []dbSession
	err := r.db.SelectContext(ctx, &dbSessions, query, tenantID.String())
	if err != nil {
		return nil, errx.Wrap(err, "failed to find active sessions", errx.TypeInternal).
			WithDetail("tenant_id", tenantID.String())
	}

	result := make([]*engine.Session, 0, len(dbSessions))
	for i := range dbSessions {
		session, err := toDomainSession(&dbSessions[i])
		if err != nil {
			return nil, errx.Wrap(err, "failed to convert session", errx.TypeInternal)
		}
		result = append(result, session)
	}

	return result, nil
}

func (r *PostgresSessionRepository) FindExpired(ctx context.Context) ([]*engine.Session, error) {
	query := `
		SELECT 
			id, tenant_id, channel_id, sender_id, context,
			history, current_state, status, expires_at, created_at, last_activity_at, closed_at
		FROM sessions
		WHERE status = 'ACTIVE' AND expires_at <= NOW()
		ORDER BY expires_at ASC`

	var dbSessions []dbSession
	err := r.db.SelectContext(ctx, &dbSessions, query)
	if err != nil {
		return nil, errx.Wrap(err, "failed to find expired sessions", errx.TypeInternal)
	}

	result := make([]*engine.Session, 0, len(dbSessions))
	for i := range dbSessions {
		session, err := toDomainSession(&dbSessions[i])
		if err != nil {
			return nil, errx.Wrap(err, "failed to convert session", errx.TypeInternal)
		}
		result = append(result, session)
	}

	return result, nil
}

func (r *PostgresSessionRepository) List(ctx context.Context, req engine.SessionListRequest) (engine.SessionListResponse, error) {
	var conditions []string
	var args []any
	argPos := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argPos))
	args = append(args, req.TenantID.String())
	argPos++

	if req.ChannelID != nil {
		conditions = append(conditions, fmt.Sprintf("channel_id = $%d", argPos))
		args = append(args, req.ChannelID.String())
		argPos++
	}

	if req.SenderID != nil {
		conditions = append(conditions, fmt.Sprintf("sender_id = $%d", argPos))
		args = append(args, *req.SenderID)
		argPos++
	}

	if req.IsActive != nil && *req.IsActive {
		conditions = append(conditions, "status = 'ACTIVE'")
	} else if req.IsActive != nil && !*req.IsActive {
		conditions = append(conditions, "status IN ('CLOSED', 'EXPIRED')")
	}

	if req.CurrentState != nil {
		conditions = append(conditions, fmt.Sprintf("current_state = $%d", argPos))
		args = append(args, *req.CurrentState)
		argPos++
	}

	if req.From != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argPos))
		args = append(args, *req.From)
		argPos++
	}

	if req.To != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argPos))
		args = append(args, *req.To)
		argPos++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM sessions WHERE %s", whereClause)
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return engine.SessionListResponse{}, errx.Wrap(err, "failed to count sessions", errx.TypeInternal)
	}

	// Data query
	dataQuery := fmt.Sprintf(`
		SELECT 
			id, tenant_id, channel_id, sender_id, context,
			history, current_state, status, expires_at, created_at, last_activity_at, closed_at
		FROM sessions
		WHERE %s
		ORDER BY last_activity_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, argPos, argPos+1)

	args = append(args, req.PageSize, req.GetOffset())

	var dbSessions []dbSession
	err = r.db.SelectContext(ctx, &dbSessions, dataQuery, args...)
	if err != nil {
		return engine.SessionListResponse{}, errx.Wrap(err, "failed to list sessions", errx.TypeInternal)
	}

	sessions := make([]engine.Session, 0, len(dbSessions))
	for i := range dbSessions {
		session, err := toDomainSession(&dbSessions[i])
		if err != nil {
			return engine.SessionListResponse{}, errx.Wrap(err, "failed to convert session", errx.TypeInternal)
		}
		sessions = append(sessions, *session)
	}

	return storex.NewPaginated(sessions, total, req.Page, req.PageSize), nil
}

func (r *PostgresSessionRepository) CleanExpired(ctx context.Context) error {
	query := `
		UPDATE sessions 
		SET status = 'EXPIRED', closed_at = NOW() 
		WHERE status = 'ACTIVE' AND expires_at <= NOW()`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return errx.Wrap(err, "failed to mark expired sessions", errx.TypeInternal)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errx.Wrap(err, "failed to get rows affected", errx.TypeInternal)
	}

	// Log how many sessions were marked as expired (optional)
	_ = rowsAffected

	return nil
}

func (r *PostgresSessionRepository) ExtendExpiration(ctx context.Context, id kernel.SessionID, duration int64) error {
	query := `
		UPDATE sessions 
		SET expires_at = NOW() + ($2 || ' seconds')::INTERVAL,
		    last_activity_at = NOW()
		WHERE id = $1 AND status = 'ACTIVE'`

	result, err := r.db.ExecContext(ctx, query, string(id), duration)
	if err != nil {
		return errx.Wrap(err, "failed to extend session expiration", errx.TypeInternal).
			WithDetail("session_id", string(id))
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errx.Wrap(err, "failed to get rows affected", errx.TypeInternal)
	}

	if rowsAffected == 0 {
		return engine.ErrSessionNotFound().WithDetail("session_id", string(id))
	}

	return nil
}

func (r *PostgresSessionRepository) CountActive(ctx context.Context, tenantID kernel.TenantID) (int, error) {
	query := `SELECT COUNT(*) FROM sessions WHERE tenant_id = $1 AND status = 'ACTIVE'`

	var count int
	err := r.db.GetContext(ctx, &count, query, tenantID.String())
	if err != nil {
		return 0, errx.Wrap(err, "failed to count active sessions", errx.TypeInternal).
			WithDetail("tenant_id", tenantID.String())
	}

	return count, nil
}

func (r *PostgresSessionRepository) Close(ctx context.Context, id kernel.SessionID) error {
	query := `
		UPDATE sessions 
		SET status = 'CLOSED', closed_at = NOW() 
		WHERE id = $1 AND status = 'ACTIVE'`

	result, err := r.db.ExecContext(ctx, query, string(id))
	if err != nil {
		return errx.Wrap(err, "failed to close session", errx.TypeInternal).
			WithDetail("session_id", string(id))
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errx.Wrap(err, "failed to get rows affected", errx.TypeInternal)
	}

	if rowsAffected == 0 {
		return engine.ErrSessionNotFound().WithDetail("session_id", string(id))
	}

	return nil
}

func (r *PostgresSessionRepository) MarkExpired(ctx context.Context, id kernel.SessionID) error {
	query := `
		UPDATE sessions 
		SET status = 'EXPIRED', closed_at = NOW() 
		WHERE id = $1 AND status = 'ACTIVE'`

	result, err := r.db.ExecContext(ctx, query, string(id))
	if err != nil {
		return errx.Wrap(err, "failed to mark session as expired", errx.TypeInternal).
			WithDetail("session_id", string(id))
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errx.Wrap(err, "failed to get rows affected", errx.TypeInternal)
	}

	if rowsAffected == 0 {
		return engine.ErrSessionNotFound().WithDetail("session_id", string(id))
	}

	return nil
}

func (r *PostgresSessionRepository) sessionExists(ctx context.Context, id string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM sessions WHERE id = $1)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, id)
	if err != nil {
		return false, errx.Wrap(err, "failed to check session existence", errx.TypeInternal)
	}

	return exists, nil
}
