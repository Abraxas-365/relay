package engineinfra

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Abraxas-365/craftable/errx"
	"github.com/Abraxas-365/craftable/storex"
	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type PostgresMessageRepository struct {
	db *sqlx.DB
}

var _ engine.MessageRepository = (*PostgresMessageRepository)(nil)

func NewPostgresMessageRepository(db *sqlx.DB) *PostgresMessageRepository {
	return &PostgresMessageRepository{db: db}
}

// dbMessage is an intermediate struct for database operations
type dbMessage struct {
	ID        string          `db:"id"`
	TenantID  string          `db:"tenant_id"`
	ChannelID string          `db:"channel_id"`
	SenderID  string          `db:"sender_id"`
	Content   json.RawMessage `db:"content"`
	Context   json.RawMessage `db:"context"`
	Status    string          `db:"status"`
	CreatedAt string          `db:"created_at"`
	UpdatedAt string          `db:"updated_at"`
}

// toDBMessage converts domain Message to dbMessage
func toDBMessage(msg engine.Message) (*dbMessage, error) {
	contentJSON, err := json.Marshal(msg.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal content: %w", err)
	}

	contextJSON := []byte("{}")
	if msg.Context != nil && len(msg.Context) > 0 {
		contextJSON, err = json.Marshal(msg.Context)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal context: %w", err)
		}
	}

	return &dbMessage{
		ID:        msg.ID.String(),
		TenantID:  msg.TenantID.String(),
		ChannelID: msg.ChannelID.String(),
		SenderID:  msg.SenderID,
		Content:   contentJSON,
		Context:   contextJSON,
		Status:    string(msg.Status),
		CreatedAt: msg.CreatedAt.Format("2006-01-02 15:04:05.999999"),
		UpdatedAt: msg.UpdatedAt.Format("2006-01-02 15:04:05.999999"),
	}, nil
}

// toDomainMessage converts dbMessage to domain Message
func toDomainMessage(dbMsg *dbMessage) (*engine.Message, error) {
	var content engine.MessageContent
	if err := json.Unmarshal(dbMsg.Content, &content); err != nil {
		return nil, fmt.Errorf("failed to unmarshal content: %w", err)
	}

	var context map[string]any
	if len(dbMsg.Context) > 0 && string(dbMsg.Context) != "null" {
		if err := json.Unmarshal(dbMsg.Context, &context); err != nil {
			return nil, fmt.Errorf("failed to unmarshal context: %w", err)
		}
	}

	msg := &engine.Message{
		ID:        kernel.MessageID(dbMsg.ID),
		TenantID:  kernel.TenantID(dbMsg.TenantID),
		ChannelID: kernel.ChannelID(dbMsg.ChannelID),
		SenderID:  dbMsg.SenderID,
		Content:   content,
		Context:   context,
		Status:    engine.MessageStatus(dbMsg.Status),
	}

	return msg, nil
}

func (r *PostgresMessageRepository) Save(ctx context.Context, msg engine.Message) error {
	exists, err := r.messageExists(ctx, msg.ID)
	if err != nil {
		return errx.Wrap(err, "failed to check message existence", errx.TypeInternal)
	}

	if exists {
		return r.update(ctx, msg)
	}
	return r.create(ctx, msg)
}

func (r *PostgresMessageRepository) create(ctx context.Context, msg engine.Message) error {
	dbMsg, err := toDBMessage(msg)
	if err != nil {
		return errx.Wrap(err, "failed to convert message", errx.TypeInternal).
			WithDetail("message_id", msg.ID.String())
	}

	query := `
		INSERT INTO messages (
			id, tenant_id, channel_id, sender_id, content,
			context, status, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :channel_id, :sender_id, :content,
			:context, :status, :created_at, :updated_at
		)`

	_, err = r.db.NamedExecContext(ctx, query, dbMsg)
	if err != nil {
		return errx.Wrap(err, "failed to create message", errx.TypeInternal).
			WithDetail("message_id", msg.ID.String())
	}

	return nil
}

func (r *PostgresMessageRepository) update(ctx context.Context, msg engine.Message) error {
	dbMsg, err := toDBMessage(msg)
	if err != nil {
		return errx.Wrap(err, "failed to convert message", errx.TypeInternal).
			WithDetail("message_id", msg.ID.String())
	}

	query := `
		UPDATE messages SET
			content = :content,
			context = :context,
			status = :status,
			updated_at = :updated_at
		WHERE id = :id`

	result, err := r.db.NamedExecContext(ctx, query, dbMsg)
	if err != nil {
		return errx.Wrap(err, "failed to update message", errx.TypeInternal).
			WithDetail("message_id", msg.ID.String())
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errx.Wrap(err, "failed to get rows affected", errx.TypeInternal)
	}

	if rowsAffected == 0 {
		return engine.ErrMessageNotFound().WithDetail("message_id", msg.ID.String())
	}

	return nil
}

func (r *PostgresMessageRepository) FindByID(ctx context.Context, id kernel.MessageID) (*engine.Message, error) {
	query := `
		SELECT 
			id, tenant_id, channel_id, sender_id, content,
			context, status, created_at, updated_at
		FROM messages
		WHERE id = $1`

	var dbMsg dbMessage
	err := r.db.GetContext(ctx, &dbMsg, query, id.String())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, engine.ErrMessageNotFound().WithDetail("message_id", id.String())
		}
		return nil, errx.Wrap(err, "failed to find message by id", errx.TypeInternal).
			WithDetail("message_id", id.String())
	}

	return toDomainMessage(&dbMsg)
}

func (r *PostgresMessageRepository) Delete(ctx context.Context, id kernel.MessageID) error {
	query := `DELETE FROM messages WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id.String())
	if err != nil {
		return errx.Wrap(err, "failed to delete message", errx.TypeInternal).
			WithDetail("message_id", id.String())
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errx.Wrap(err, "failed to get rows affected", errx.TypeInternal)
	}

	if rowsAffected == 0 {
		return engine.ErrMessageNotFound().WithDetail("message_id", id.String())
	}

	return nil
}

func (r *PostgresMessageRepository) FindByChannel(ctx context.Context, channelID kernel.ChannelID) ([]*engine.Message, error) {
	query := `
		SELECT 
			id, tenant_id, channel_id, sender_id, content,
			context, status, created_at, updated_at
		FROM messages
		WHERE channel_id = $1
		ORDER BY created_at DESC`

	var dbMessages []dbMessage
	err := r.db.SelectContext(ctx, &dbMessages, query, channelID.String())
	if err != nil {
		return nil, errx.Wrap(err, "failed to find messages by channel", errx.TypeInternal).
			WithDetail("channel_id", channelID.String())
	}

	result := make([]*engine.Message, 0, len(dbMessages))
	for i := range dbMessages {
		msg, err := toDomainMessage(&dbMessages[i])
		if err != nil {
			return nil, errx.Wrap(err, "failed to convert message", errx.TypeInternal)
		}
		result = append(result, msg)
	}

	return result, nil
}

func (r *PostgresMessageRepository) FindBySender(ctx context.Context, senderID string, tenantID kernel.TenantID) ([]*engine.Message, error) {
	query := `
		SELECT 
			id, tenant_id, channel_id, sender_id, content,
			context, status, created_at, updated_at
		FROM messages
		WHERE sender_id = $1 AND tenant_id = $2
		ORDER BY created_at DESC`

	var dbMessages []dbMessage
	err := r.db.SelectContext(ctx, &dbMessages, query, senderID, tenantID.String())
	if err != nil {
		return nil, errx.Wrap(err, "failed to find messages by sender", errx.TypeInternal).
			WithDetail("sender_id", senderID)
	}

	result := make([]*engine.Message, 0, len(dbMessages))
	for i := range dbMessages {
		msg, err := toDomainMessage(&dbMessages[i])
		if err != nil {
			return nil, errx.Wrap(err, "failed to convert message", errx.TypeInternal)
		}
		result = append(result, msg)
	}

	return result, nil
}

func (r *PostgresMessageRepository) FindByStatus(ctx context.Context, status engine.MessageStatus, tenantID kernel.TenantID) ([]*engine.Message, error) {
	query := `
		SELECT 
			id, tenant_id, channel_id, sender_id, content,
			context, status, created_at, updated_at
		FROM messages
		WHERE status = $1 AND tenant_id = $2
		ORDER BY created_at ASC`

	var dbMessages []dbMessage
	err := r.db.SelectContext(ctx, &dbMessages, query, status, tenantID.String())
	if err != nil {
		return nil, errx.Wrap(err, "failed to find messages by status", errx.TypeInternal).
			WithDetail("status", string(status))
	}

	result := make([]*engine.Message, 0, len(dbMessages))
	for i := range dbMessages {
		msg, err := toDomainMessage(&dbMessages[i])
		if err != nil {
			return nil, errx.Wrap(err, "failed to convert message", errx.TypeInternal)
		}
		result = append(result, msg)
	}

	return result, nil
}

func (r *PostgresMessageRepository) List(ctx context.Context, req engine.MessageListRequest) (engine.MessageListResponse, error) {
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

	if req.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argPos))
		args = append(args, *req.Status)
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
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM messages WHERE %s", whereClause)
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return engine.MessageListResponse{}, errx.Wrap(err, "failed to count messages", errx.TypeInternal)
	}

	// Data query
	dataQuery := fmt.Sprintf(`
		SELECT 
			id, tenant_id, channel_id, sender_id, content,
			context, status, created_at, updated_at
		FROM messages
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, argPos, argPos+1)

	args = append(args, req.PageSize, req.GetOffset())

	var dbMessages []dbMessage
	err = r.db.SelectContext(ctx, &dbMessages, dataQuery, args...)
	if err != nil {
		return engine.MessageListResponse{}, errx.Wrap(err, "failed to list messages", errx.TypeInternal)
	}

	messages := make([]engine.Message, 0, len(dbMessages))
	for i := range dbMessages {
		msg, err := toDomainMessage(&dbMessages[i])
		if err != nil {
			return engine.MessageListResponse{}, errx.Wrap(err, "failed to convert message", errx.TypeInternal)
		}
		messages = append(messages, *msg)
	}

	return storex.NewPaginated(messages, total, req.Page, req.PageSize), nil
}

func (r *PostgresMessageRepository) BulkUpdateStatus(ctx context.Context, ids []kernel.MessageID, status engine.MessageStatus) error {
	if len(ids) == 0 {
		return nil
	}

	idStrings := make([]string, len(ids))
	for i, id := range ids {
		idStrings[i] = id.String()
	}

	query := `
		UPDATE messages 
		SET status = $1, updated_at = NOW()
		WHERE id = ANY($2)`

	_, err := r.db.ExecContext(ctx, query, status, pq.Array(idStrings))
	if err != nil {
		return errx.Wrap(err, "failed to bulk update message status", errx.TypeInternal)
	}

	return nil
}

func (r *PostgresMessageRepository) CountByStatus(ctx context.Context, status engine.MessageStatus, tenantID kernel.TenantID) (int, error) {
	query := `SELECT COUNT(*) FROM messages WHERE status = $1 AND tenant_id = $2`

	var count int
	err := r.db.GetContext(ctx, &count, query, status, tenantID.String())
	if err != nil {
		return 0, errx.Wrap(err, "failed to count messages by status", errx.TypeInternal)
	}

	return count, nil
}

func (r *PostgresMessageRepository) CountByChannel(ctx context.Context, channelID kernel.ChannelID) (int, error) {
	query := `SELECT COUNT(*) FROM messages WHERE channel_id = $1`

	var count int
	err := r.db.GetContext(ctx, &count, query, channelID.String())
	if err != nil {
		return 0, errx.Wrap(err, "failed to count messages by channel", errx.TypeInternal)
	}

	return count, nil
}

func (r *PostgresMessageRepository) messageExists(ctx context.Context, id kernel.MessageID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM messages WHERE id = $1)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, id.String())
	if err != nil {
		return false, errx.Wrap(err, "failed to check message existence", errx.TypeInternal)
	}

	return exists, nil
}
