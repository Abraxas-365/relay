package agent

import (
	"context"

	"github.com/Abraxas-365/relay/pkg/kernel"
)

type AgentChatRepository interface {
	GetAllMessagesBySession(ctx context.Context, sessionID kernel.SessionID) ([]AgentMessage, error)
	CreateMessage(ctx context.Context, req CreateMessageRequest) (*AgentMessage, error)
	ClearSessionMessages(ctx context.Context, sessionID kernel.SessionID, keepSystemPrompt bool) error
}
