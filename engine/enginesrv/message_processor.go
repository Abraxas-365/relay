package enginesrv

import (
	"context"
	"github.com/Abraxas-365/relay/channels"
	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/parser"
)

type MessageProcessor struct {
	messageRepo    engine.MessageRepository
	workflowRepo   engine.WorkflowRepository
	sessionManager engine.SessionManager
	workflowExec   engine.WorkflowExecutor
	channelManager channels.ChannelManager
	parserSelector parser.ParserSelector
}

func NewMessageProcessor(
	messageRepo engine.MessageRepository,
	workflowRepo engine.WorkflowRepository,
	sessionManager engine.SessionManager,
	workflowExec engine.WorkflowExecutor,
	channelManager channels.ChannelManager,
	parserSelector parser.ParserSelector,
) *MessageProcessor {
	return &MessageProcessor{
		messageRepo,
		workflowRepo,
		sessionManager,
		workflowExec,
		channelManager,
		parserSelector,
	}
}

// ProcessMessage es el entry point principal
func (mp *MessageProcessor) ProcessMessage(ctx context.Context, msg engine.Message) error {
	// 1. Guardar mensaje
	if err := mp.messageRepo.Save(ctx, msg); err != nil {
		return err
	}

	// 2. Obtener o crear sesión
	session, err := mp.sessionManager.GetOrCreate(ctx, msg.ChannelID, msg.SenderID)
	if err != nil {
		return err
	}

	// 3. Buscar workflow apropiado
	workflows, err := mp.workflowRepo.FindActiveByTrigger(ctx, engine.WorkflowTrigger{
		Type:       "message_received",
		ChannelIDs: []string{msg.ChannelID.String()},
	})
	if err != nil || len(workflows) == 0 {
		return mp.handleNoWorkflow(ctx, msg)
	}

	// 4. Ejecutar workflow
	result, err := mp.workflowExec.Execute(ctx, *workflows[0], msg)
	if err != nil {
		return err
	}

	// 5. Actualizar sesión
	session.Context = result.Context
	session.CurrentState = result.NextState
	mp.sessionManager.Update(ctx, *session)

	// 6. Enviar respuesta si es necesario
	if result.ShouldRespond {
		return mp.channelManager.SendMessage(ctx, msg.ChannelID, channels.OutgoingMessage{
			RecipientID: msg.SenderID,
			Content: channels.MessageContent{
				Type: "text",
				Text: result.Response,
			},
		})
	}

	return nil
}

func (mp *MessageProcessor) handleNoWorkflow(ctx context.Context, msg engine.Message) error {
	// Lógica por defecto cuando no hay workflow
	// Por ejemplo, usar un parser default
	return nil
}
