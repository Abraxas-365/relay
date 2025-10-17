package agentinfra

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Abraxas-365/craftable/errx"
	"github.com/Abraxas-365/craftable/logx"
	"github.com/Abraxas-365/relay/pkg/agent"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PostgresAgentChatRepository struct {
	db *sqlx.DB
}

var _ agent.AgentChatRepository = (*PostgresAgentChatRepository)(nil)

func NewPostgresAgentChatRepository(db *sqlx.DB) *PostgresAgentChatRepository {
	return &PostgresAgentChatRepository{db: db}
}

// dbAgentMessage is an intermediate struct for database operations
type dbAgentMessage struct {
	ID               string          `db:"id"`
	TenantID         string          `db:"tenant_id"` // ✅ ADDED
	SessionID        string          `db:"session_id"`
	Role             string          `db:"role"`
	Content          *string         `db:"content"`
	Name             *string         `db:"name"`
	FunctionCall     json.RawMessage `db:"function_call"`
	ToolCalls        json.RawMessage `db:"tool_calls"`
	ToolCallID       *string         `db:"tool_call_id"`
	Metadata         json.RawMessage `db:"metadata"`
	MessageType      string          `db:"message_type"`
	ProcessingTimeMs *int            `db:"processing_time_ms"`
	ModelUsed        *string         `db:"model_used"`
	TokensUsed       *int            `db:"tokens_used"`
	CreatedAt        time.Time       `db:"created_at"`
	UpdatedAt        time.Time       `db:"updated_at"`
}

// toDBAgentMessage converts domain AgentMessage to dbAgentMessage
func toDBAgentMessage(m *agent.AgentMessage) (*dbAgentMessage, error) {
	dbMsg := &dbAgentMessage{
		ID:               m.ID,
		TenantID:         m.TenantID.String(), // ✅ ADDED
		SessionID:        m.SessionID.String(),
		Role:             m.Role,
		Content:          m.Content,
		Name:             m.Name,
		ToolCallID:       m.ToolCallID,
		MessageType:      m.MessageType,
		ProcessingTimeMs: m.ProcessingTimeMs,
		ModelUsed:        m.ModelUsed,
		TokensUsed:       m.TokensUsed,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}

	// Convert FunctionCall - set to null if nil
	if m.FunctionCall != nil {
		fcBytes, err := json.Marshal(m.FunctionCall)
		if err != nil {
			return nil, errx.Wrap(err, "failed to marshal function_call", errx.TypeInternal)
		}
		dbMsg.FunctionCall = fcBytes
	} else {
		dbMsg.FunctionCall = json.RawMessage("null")
	}

	// Convert ToolCalls - set to null if nil
	if m.ToolCalls != nil && len(m.ToolCalls) > 0 {
		tcBytes, err := json.Marshal(m.ToolCalls)
		if err != nil {
			return nil, errx.Wrap(err, "failed to marshal tool_calls", errx.TypeInternal)
		}
		dbMsg.ToolCalls = tcBytes
	} else {
		dbMsg.ToolCalls = json.RawMessage("null")
	}

	// Convert Metadata - set to empty object if nil
	if m.Metadata != nil && len(m.Metadata) > 0 {
		mdBytes, err := json.Marshal(m.Metadata)
		if err != nil {
			return nil, errx.Wrap(err, "failed to marshal metadata", errx.TypeInternal)
		}
		dbMsg.Metadata = mdBytes
	} else {
		dbMsg.Metadata = json.RawMessage("{}")
	}

	return dbMsg, nil
}

// toDomainAgentMessage converts dbAgentMessage to domain AgentMessage
func toDomainAgentMessage(db *dbAgentMessage) (*agent.AgentMessage, error) {
	msg := &agent.AgentMessage{
		ID:               db.ID,
		TenantID:         kernel.TenantID(db.TenantID), // ✅ ADDED
		SessionID:        kernel.SessionID(db.SessionID),
		Role:             db.Role,
		Content:          db.Content,
		Name:             db.Name,
		ToolCallID:       db.ToolCallID,
		MessageType:      db.MessageType,
		ProcessingTimeMs: db.ProcessingTimeMs,
		ModelUsed:        db.ModelUsed,
		TokensUsed:       db.TokensUsed,
		CreatedAt:        db.CreatedAt,
		UpdatedAt:        db.UpdatedAt,
	}

	// Convert FunctionCall
	if len(db.FunctionCall) > 0 && string(db.FunctionCall) != "null" {
		var fc map[string]any
		if err := json.Unmarshal(db.FunctionCall, &fc); err != nil {
			return nil, errx.Wrap(err, "failed to unmarshal function_call", errx.TypeInternal)
		}
		msg.FunctionCall = fc
	}

	// Convert ToolCalls
	if len(db.ToolCalls) > 0 && string(db.ToolCalls) != "null" {
		var tc []map[string]any
		if err := json.Unmarshal(db.ToolCalls, &tc); err != nil {
			return nil, errx.Wrap(err, "failed to unmarshal tool_calls", errx.TypeInternal)
		}
		msg.ToolCalls = tc
	}

	// Convert Metadata
	if len(db.Metadata) > 0 && string(db.Metadata) != "null" {
		var md map[string]any
		if err := json.Unmarshal(db.Metadata, &md); err != nil {
			return nil, errx.Wrap(err, "failed to unmarshal metadata", errx.TypeInternal)
		}
		msg.Metadata = md
	}

	return msg, nil
}

// GetAllMessagesBySession retrieves all messages for a session ordered by creation time
func (r *PostgresAgentChatRepository) GetAllMessagesBySession(ctx context.Context, sessionID kernel.SessionID) ([]agent.AgentMessage, error) {
	query := `
		SELECT 
			id, tenant_id, session_id, role, content, name, function_call, tool_calls, 
			tool_call_id, metadata, message_type, processing_time_ms, 
			model_used, tokens_used, created_at, updated_at
		FROM agent_messages
		WHERE session_id = $1
		ORDER BY created_at ASC, id ASC
	` // ✅ ADDED tenant_id to SELECT

	var dbMessages []dbAgentMessage
	err := r.db.SelectContext(ctx, &dbMessages, query, sessionID.String())
	if err != nil {
		return nil, errx.Wrap(err, "failed to get messages by session", errx.TypeInternal).
			WithDetail("session_id", sessionID.String())
	}

	// Convert to domain messages
	messages := make([]agent.AgentMessage, 0, len(dbMessages))
	for _, dbMsg := range dbMessages {
		domainMsg, err := toDomainAgentMessage(&dbMsg)
		if err != nil {
			return nil, err
		}
		messages = append(messages, *domainMsg)
	}

	return messages, nil
}

// CreateMessage creates a new message in the database
func (r *PostgresAgentChatRepository) CreateMessage(ctx context.Context, req agent.CreateMessageRequest) (*agent.AgentMessage, error) {
	// ✅ Validate TenantID is present
	if req.TenantID.IsEmpty() {
		return nil, errx.New("tenant_id is required", errx.TypeValidation)
	}

	// Generate ID and timestamps
	now := time.Now()
	msg := &agent.AgentMessage{
		ID:               uuid.New().String(),
		TenantID:         req.TenantID, // ✅ ADDED
		SessionID:        req.SessionID,
		Role:             req.Role,
		Content:          req.Content,
		Name:             req.Name,
		FunctionCall:     req.FunctionCall,
		ToolCalls:        req.ToolCalls,
		ToolCallID:       req.ToolCallID,
		Metadata:         req.Metadata,
		MessageType:      agent.MessageTypeText,
		ProcessingTimeMs: req.ProcessingTimeMs,
		ModelUsed:        req.ModelUsed,
		TokensUsed:       req.TokensUsed,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Override message type if provided
	if req.MessageType != nil {
		msg.MessageType = *req.MessageType
	}

	// Convert to DB struct
	dbMsg, err := toDBAgentMessage(msg)
	if err != nil {
		logx.Error("Error converting to DB agent message: %v", err)
		return nil, err
	}

	// ✅ Insert query - ADDED tenant_id
	query := `
		INSERT INTO agent_messages (
			id, tenant_id, session_id, role, content, name, function_call, tool_calls,
			tool_call_id, metadata, message_type, processing_time_ms,
			model_used, tokens_used, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :session_id, :role, :content, :name, :function_call, :tool_calls,
			:tool_call_id, :metadata, :message_type, :processing_time_ms,
			:model_used, :tokens_used, :created_at, :updated_at
		)
	`

	_, err = r.db.NamedExecContext(ctx, query, dbMsg)
	if err != nil {
		logx.Error("Error inserting agent message: %v", err)
		return nil, errx.Wrap(err, "failed to create message", errx.TypeInternal).
			WithDetail("session_id", req.SessionID.String()).
			WithDetail("tenant_id", req.TenantID.String()) // ✅ ADDED
	}

	return msg, nil
}

// ClearSessionMessages deletes all messages for a session, optionally keeping system prompts
func (r *PostgresAgentChatRepository) ClearSessionMessages(ctx context.Context, sessionID kernel.SessionID, keepSystemPrompt bool) error {
	var query string

	if keepSystemPrompt {
		query = `
			DELETE FROM agent_messages
			WHERE session_id = $1 AND role != 'system'
		`
	} else {
		query = `
			DELETE FROM agent_messages
			WHERE session_id = $1
		`
	}

	result, err := r.db.ExecContext(ctx, query, sessionID.String())
	if err != nil {
		return errx.Wrap(err, "failed to clear session messages", errx.TypeInternal).
			WithDetail("session_id", sessionID.String()).
			WithDetail("keep_system_prompt", keepSystemPrompt)
	}

	rowsAffected, _ := result.RowsAffected()
	_ = rowsAffected

	return nil
}
