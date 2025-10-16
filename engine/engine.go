package engine

import (
	"time"

	"github.com/Abraxas-365/relay/pkg/kernel"
	"slices"
)

// ============================================================================
// Message Entity
// ============================================================================

// Message representa un mensaje normalizado
type Message struct {
	ID        kernel.MessageID `db:"id" json:"id"`
	TenantID  kernel.TenantID  `db:"tenant_id" json:"tenant_id"`
	ChannelID kernel.ChannelID `db:"channel_id" json:"channel_id"`
	SenderID  string           `db:"sender_id" json:"sender_id"`
	Content   MessageContent   `db:"content" json:"content"`
	Context   map[string]any   `db:"context" json:"context"`
	Status    MessageStatus    `db:"status" json:"status"`
	CreatedAt time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt time.Time        `db:"updated_at" json:"updated_at"`
}

// MessageType tipo de contenido del mensaje
type MessageType string

const (
	MessageTypeText     MessageType = "text"
	MessageTypeImage    MessageType = "image"
	MessageTypeAudio    MessageType = "audio"
	MessageTypeVideo    MessageType = "video"
	MessageTypeDocument MessageType = "document"
)

// MessageContent contenido del mensaje
type MessageContent struct {
	Type        MessageType    `json:"type"` // text, image, audio, video, document
	Text        string         `json:"text,omitempty"`
	Attachments []string       `json:"attachments,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// MessageStatus estado del mensaje
type MessageStatus string

const (
	MessageStatusPending    MessageStatus = "PENDING"
	MessageStatusProcessing MessageStatus = "PROCESSING"
	MessageStatusProcessed  MessageStatus = "PROCESSED"
	MessageStatusFailed     MessageStatus = "FAILED"
)

// ============================================================================
// Workflow Entity
// ============================================================================

// Workflow representa un flujo de trabajo
type Workflow struct {
	ID          kernel.WorkflowID `db:"id" json:"id"`
	TenantID    kernel.TenantID   `db:"tenant_id" json:"tenant_id"`
	Name        string            `db:"name" json:"name"`
	Description string            `db:"description" json:"description"`
	Trigger     WorkflowTrigger   `db:"trigger" json:"trigger"`
	Node        []WorkflowNode    `db:"nodes" json:"nodes"`
	IsActive    bool              `db:"is_active" json:"is_active"`
	CreatedAt   time.Time         `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time         `db:"updated_at" json:"updated_at"`
}

// WorkflowTrigger define cuándo se ejecuta el workflow
type WorkflowTrigger struct {
	Type       TriggerType    `json:"type"` // message_received, scheduled, webhook, manual
	ChannelIDs []string       `json:"channel_ids,omitempty"`
	Schedule   *string        `json:"schedule,omitempty"` // Cron expression
	Filters    map[string]any `json:"filters,omitempty"`  // Filtros adicionales
}

// TriggerType tipo de trigger
type TriggerType string

const (
	TriggerTypeMessageReceived TriggerType = "MESSAGE_RECEIVED"
	TriggerTypeScheduled       TriggerType = "SCHEDULED"
	TriggerTypeWebhook         TriggerType = "WEBHOOK"
	TriggerTypeManual          TriggerType = "MANUAL"
)

// WorkflowNode paso de un workflow
type WorkflowNode struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Type      NodeType       `json:"type"` // condition, parser, tool, action, delay
	Config    map[string]any `json:"config"`
	OnSuccess string         `json:"on_success,omitempty"` // next node ID
	OnFailure string         `json:"on_failure,omitempty"` // next node ID
	Timeout   *int           `json:"timeout,omitempty"`    // seconds
}

// NodeType tipo de paso
type NodeType string

const (
	NodeTypeCondition NodeType = "CONDITION"
	NodeTypeParser    NodeType = "PARSER"
	NodeTypeAction    NodeType = "ACTION"
	NodeTypeDelay     NodeType = "DELAY"
	NodeTypeResponse  NodeType = "RESPONSE"

	NodeTypeSwitch      NodeType = "SWITCH"
	NodeTypeTransform   NodeType = "TRANSFORM"
	NodeTypeHTTP        NodeType = "HTTP"
	NodeTypeLoop        NodeType = "LOOP"
	NodeTypeValidate    NodeType = "VALIDATE"
	NodeTypeAIAgent     NodeType = "AI_AGENT"
	NodeTypeSendMessage NodeType = "SEND_MESSAGE"
)

// ============================================================================
// Session Entity
// ============================================================================

type SessionStatus string

const (
	SessionStatusActive  SessionStatus = "ACTIVE"
	SessionStatusClosed  SessionStatus = "CLOSED"
	SessionStatusExpired SessionStatus = "EXPIRED"
)

type Session struct {
	ID             kernel.SessionID
	TenantID       kernel.TenantID
	ChannelID      kernel.ChannelID
	SenderID       string
	Context        map[string]any
	History        []MessageRef
	CurrentState   string
	Status         SessionStatus
	ExpiresAt      time.Time
	CreatedAt      time.Time
	LastActivityAt time.Time
	ClosedAt       *time.Time
}

// Close marks the session as closed
func (s *Session) Close() {
	s.Status = SessionStatusClosed
	now := time.Now()
	s.ClosedAt = &now
}

// MarkExpired marks the session as expired
func (s *Session) MarkExpired() {
	s.Status = SessionStatusExpired
	now := time.Now()
	s.ClosedAt = &now
}

// MessageRef referencia a un mensaje en el historial
type MessageRef struct {
	MessageID kernel.MessageID `json:"message_id"`
	Role      string           `json:"role"` // user, assistant, system
	Timestamp time.Time        `json:"timestamp"`
}

// ============================================================================
// Execution Result
// ============================================================================

// ExecutionResult resultado de la ejecución de un workflow
type ExecutionResult struct {
	Success      bool           `json:"success"`
	Response     string         `json:"response,omitempty"`
	NextState    string         `json:"next_state,omitempty"`
	Context      map[string]any `json:"context,omitempty"`
	Error        error          `json:"-"`
	ErrorMessage string         `json:"error,omitempty"`
	Executedodes []NodeResult   `json:"executed_nodes,omitempty"`
}

// NodeResult resultado de un paso
type NodeResult struct {
	NodeID    string         `json:"node_id"`
	NodeName  string         `json:"node_name"`
	Success   bool           `json:"success"`
	Output    map[string]any `json:"output,omitempty"`
	Error     string         `json:"error,omitempty"`
	Duration  int64          `json:"duration_ms"`
	Timestamp time.Time      `json:"timestamp"`
}

// ============================================================================
// Domain Methods - Message
// ============================================================================

// IsValid verifica si el mensaje es válido
func (m *Message) IsValid() bool {
	return !m.ID.IsEmpty() && !m.ChannelID.IsEmpty() && m.SenderID != ""
}

// MarkAsProcessing marca el mensaje como en procesamiento
func (m *Message) MarkAsProcessing() {
	m.Status = MessageStatusProcessing
	m.UpdatedAt = time.Now()
}

// MarkAsProcessed marca el mensaje como procesado
func (m *Message) MarkAsProcessed() {
	m.Status = MessageStatusProcessed
	m.UpdatedAt = time.Now()
}

// MarkAsFailed marca el mensaje como fallido
func (m *Message) MarkAsFailed() {
	m.Status = MessageStatusFailed
	m.UpdatedAt = time.Now()
}

// HasTextContent verifica si el mensaje tiene contenido de texto
func (m *Message) HasTextContent() bool {
	return m.Content.Type == "text" && m.Content.Text != ""
}

// HasAttachments verifica si el mensaje tiene adjuntos
func (m *Message) HasAttachments() bool {
	return len(m.Content.Attachments) > 0
}

// ============================================================================
// Domain Methods - Workflow
// ============================================================================

// IsValid verifica si el workflow es válido
func (w *Workflow) IsValid() bool {
	return w.Name != "" && len(w.Node) > 0 && !w.TenantID.IsEmpty()
}

// Activate activa el workflow
func (w *Workflow) Activate() {
	w.IsActive = true
	w.UpdatedAt = time.Now()
}

// Deactivate desactiva el workflow
func (w *Workflow) Deactivate() {
	w.IsActive = false
	w.UpdatedAt = time.Now()
}

// UpdateDetails actualiza nombre y descripción
func (w *Workflow) UpdateDetails(name, description string) {
	if name != "" {
		w.Name = name
	}
	if description != "" {
		w.Description = description
	}
	w.UpdatedAt = time.Now()
}

// UpdateNode actualiza los pasos del workflow
func (w *Workflow) UpdateNode(nodes []WorkflowNode) {
	w.Node = nodes
	w.UpdatedAt = time.Now()
}

// GetNodeByID obtiene un paso por ID
func (w *Workflow) GetNodeByID(nodeID string) *WorkflowNode {
	for i := range w.Node {
		if w.Node[i].ID == nodeID {
			return &w.Node[i]
		}
	}
	return nil
}

// MatchesTrigger verifica si el workflow coincide con un trigger dado
func (w *Workflow) MatchesTrigger(trigger WorkflowTrigger) bool {
	if w.Trigger.Type != trigger.Type {
		return false
	}

	// Si tiene filtro de canales, verificar coincidencia
	if len(w.Trigger.ChannelIDs) > 0 && len(trigger.ChannelIDs) > 0 {
		for _, wChannelID := range w.Trigger.ChannelIDs {
			if slices.Contains(trigger.ChannelIDs, wChannelID) {
				return true
			}
		}
		return false
	}

	return true
}

// ============================================================================
// Domain Methods - Session
// ============================================================================

// IsValid verifica si la sesión es válida
func (s *Session) IsValid() bool {
	return s.ID != "" && !s.ChannelID.IsEmpty() && s.SenderID != ""
}

// IsExpired verifica si la sesión ha expirado
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// UpdateActivity actualiza la última actividad
func (s *Session) UpdateActivity() {
	s.LastActivityAt = time.Now()
}

// AddMessage añade un mensaje al historial
func (s *Session) AddMessage(messageID kernel.MessageID, role string) {
	s.History = append(s.History, MessageRef{
		MessageID: messageID,
		Role:      role,
		Timestamp: time.Now(),
	})
	s.UpdateActivity()
}

// SetContext establece contexto
func (s *Session) SetContext(key string, value any) {
	if s.Context == nil {
		s.Context = make(map[string]any)
	}
	s.Context[key] = value
	s.UpdateActivity()
}

// GetContext obtiene un valor del contexto
func (s *Session) GetContext(key string) (any, bool) {
	if s.Context == nil {
		return nil, false
	}
	val, ok := s.Context[key]
	return val, ok
}

// UpdateState actualiza el estado actual
func (s *Session) UpdateState(state string) {
	s.CurrentState = state
	s.UpdateActivity()
}

// ExtendExpiration extiende la expiración de la sesión
func (s *Session) ExtendExpiration(duration time.Duration) {
	s.ExpiresAt = time.Now().Add(duration)
	s.UpdateActivity()
}

// GetHistoryCount retorna el número de mensajes en el historial
func (s *Session) GetHistoryCount() int {
	return len(s.History)
}
