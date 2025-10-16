package engine

import (
	"time"

	"github.com/Abraxas-365/relay/pkg/kernel"
	"slices"
)

// ============================================================================
// Workflow Input (NEW - Primary input for workflows)
// ============================================================================

// WorkflowInput represents generic input data for workflow execution
type WorkflowInput struct {
	TriggerData map[string]any  `json:"trigger_data"` // Data from trigger
	TenantID    kernel.TenantID `json:"tenant_id"`
	Metadata    map[string]any  `json:"metadata,omitempty"`
}

// ============================================================================
// Message Entity (OPTIONAL - Only for channel integrations)
// ============================================================================

// Message represents a channel message (optional, only used by channel triggers)
type Message struct {
	ID        kernel.MessageID `db:"id" json:"id"`
	TenantID  kernel.TenantID  `db:"tenant_id" json:"tenant_id"`
	ChannelID kernel.ChannelID `db:"channel_id" json:"channel_id"`
	SenderID  string           `db:"sender_id" json:"sender_id"`
	Content   MessageContent   `db:"content" json:"content"`
	Context   map[string]any   `db:"context" json:"context"`
	CreatedAt time.Time        `db:"created_at" json:"created_at"`
}

type MessageContent struct {
	Type        string         `json:"type"`
	Text        string         `json:"text,omitempty"`
	Attachments []string       `json:"attachments,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// Helper methods for Message
func (m *Message) IsValid() bool {
	return !m.ID.IsEmpty() && !m.ChannelID.IsEmpty() && m.SenderID != ""
}

func (m *Message) HasTextContent() bool {
	return m.Content.Type == "text" && m.Content.Text != ""
}

// ============================================================================
// Workflow Entity
// ============================================================================

type Workflow struct {
	ID          kernel.WorkflowID `db:"id" json:"id"`
	TenantID    kernel.TenantID   `db:"tenant_id" json:"tenant_id"`
	Name        string            `db:"name" json:"name"`
	Description string            `db:"description" json:"description"`
	Trigger     WorkflowTrigger   `db:"trigger" json:"trigger"`
	Nodes       []WorkflowNode    `db:"nodes" json:"nodes"`
	IsActive    bool              `db:"is_active" json:"is_active"`
	CreatedAt   time.Time         `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time         `db:"updated_at" json:"updated_at"`
}

// WorkflowTrigger defines when workflow executes
type WorkflowTrigger struct {
	Type    TriggerType    `json:"type"`
	Config  map[string]any `json:"config,omitempty"`
	Filters map[string]any `json:"filters,omitempty"`
}

// TriggerType defines trigger types
type TriggerType string

const (
	TriggerTypeWebhook        TriggerType = "WEBHOOK"
	TriggerTypeSchedule       TriggerType = "SCHEDULE"
	TriggerTypeManual         TriggerType = "MANUAL"
	TriggerTypeChannelWebhook TriggerType = "CHANNEL_WEBHOOK" // For channel integrations
)

// WorkflowNode represents a workflow step
type WorkflowNode struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Type      NodeType       `json:"type"`
	Config    map[string]any `json:"config"`
	OnSuccess string         `json:"on_success,omitempty"`
	OnFailure string         `json:"on_failure,omitempty"`
	Timeout   *int           `json:"timeout,omitempty"`
}

// NodeType defines node types
type NodeType string

const (
	NodeTypeCondition   NodeType = "CONDITION"
	NodeTypeAction      NodeType = "ACTION"
	NodeTypeDelay       NodeType = "DELAY"
	NodeTypeSwitch      NodeType = "SWITCH"
	NodeTypeTransform   NodeType = "TRANSFORM"
	NodeTypeHTTP        NodeType = "HTTP"
	NodeTypeLoop        NodeType = "LOOP"
	NodeTypeValidate    NodeType = "VALIDATE"
	NodeTypeAIAgent     NodeType = "AI_AGENT"
	NodeTypeSendMessage NodeType = "SEND_MESSAGE"
)

// ============================================================================
// Execution Result
// ============================================================================

type ExecutionResult struct {
	Success       bool           `json:"success"`
	Output        map[string]any `json:"output,omitempty"`
	Error         error          `json:"-"`
	ErrorMessage  string         `json:"error,omitempty"`
	ExecutedNodes []NodeResult   `json:"executed_nodes,omitempty"`
}

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
// Domain Methods - Workflow
// ============================================================================

func (w *Workflow) IsValid() bool {
	return w.Name != "" && len(w.Nodes) > 0 && !w.TenantID.IsEmpty()
}

func (w *Workflow) Activate() {
	w.IsActive = true
	w.UpdatedAt = time.Now()
}

func (w *Workflow) Deactivate() {
	w.IsActive = false
	w.UpdatedAt = time.Now()
}

func (w *Workflow) UpdateDetails(name, description string) {
	if name != "" {
		w.Name = name
	}
	if description != "" {
		w.Description = description
	}
	w.UpdatedAt = time.Now()
}

func (w *Workflow) UpdateNodes(nodes []WorkflowNode) {
	w.Nodes = nodes
	w.UpdatedAt = time.Now()
}

func (w *Workflow) GetNodeByID(nodeID string) *WorkflowNode {
	for i := range w.Nodes {
		if w.Nodes[i].ID == nodeID {
			return &w.Nodes[i]
		}
	}
	return nil
}

func (w *Workflow) MatchesTrigger(trigger WorkflowTrigger) bool {
	if w.Trigger.Type != trigger.Type {
		return false
	}

	// Match filters if present
	if len(w.Trigger.Filters) > 0 && len(trigger.Filters) > 0 {
		for key, expectedVal := range w.Trigger.Filters {
			if actualVal, ok := trigger.Filters[key]; ok {
				// Handle array matching (e.g., channel_ids)
				if key == "channel_ids" {
					expectedIDs, ok1 := expectedVal.([]string)
					actualIDs, ok2 := actualVal.([]string)
					if ok1 && ok2 {
						for _, eid := range expectedIDs {
							if slices.Contains(actualIDs, eid) {
								return true
							}
						}
						return false
					}
				}
				// Simple equality
				if expectedVal != actualVal {
					return false
				}
			}
		}
	}

	return true
}

