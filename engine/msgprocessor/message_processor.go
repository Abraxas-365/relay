package msgprocessor

import (
	"context"
	"log"
	"time"

	"github.com/Abraxas-365/craftable/errx"
	"github.com/Abraxas-365/relay/channels"
	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

// MessageProcessor procesa mensajes entrantes
type MessageProcessor struct {
	messageRepo    engine.MessageRepository
	workflowRepo   engine.WorkflowRepository
	sessionManager engine.SessionManager
	workflowExec   engine.WorkflowExecutor
	channelManager channels.ChannelManager
}

// NewMessageProcessor crea una nueva instancia del procesador de mensajes
func NewMessageProcessor(
	messageRepo engine.MessageRepository,
	workflowRepo engine.WorkflowRepository,
	sessionManager engine.SessionManager,
	workflowExec engine.WorkflowExecutor,
	channelManager channels.ChannelManager,
) *MessageProcessor {
	return &MessageProcessor{
		messageRepo:    messageRepo,
		workflowRepo:   workflowRepo,
		sessionManager: sessionManager,
		workflowExec:   workflowExec,
		channelManager: channelManager,
	}
}

// ProcessMessage es el entry point principal para procesar mensajes
func (mp *MessageProcessor) ProcessMessage(ctx context.Context, msg engine.Message) error {
	log.Printf("🚀 Processing message ID: %s from Sender: %s on Channel: %s", msg.ID.String(), msg.SenderID, msg.ChannelID.String())
	// 1. Validar mensaje
	if !msg.IsValid() {
		return engine.ErrMessageProcessingFailed().WithDetail("reason", "invalid message")
	}

	// 2. Marcar mensaje como en procesamiento
	msg.MarkAsProcessing()
	if err := mp.messageRepo.Save(ctx, msg); err != nil {
		return errx.Wrap(err, "failed to save message", errx.TypeInternal)
	}

	// 3. Obtener o crear sesión
	session, err := mp.sessionManager.GetOrCreate(ctx, msg.ChannelID, msg.SenderID, msg.TenantID)
	if err != nil {
		msg.MarkAsFailed()
		mp.messageRepo.Save(ctx, msg)
		return errx.Wrap(err, "failed to get or create session", errx.TypeInternal)
	}

	// 4. Añadir mensaje al historial de la sesión
	session.AddMessage(msg.ID, "user")

	// 5. Buscar workflow apropiado
	log.Printf("🔍 Searching for matching workflows for message: %s", msg.ID.String())
	workflows, err := mp.findMatchingWorkflows(ctx, msg)
	if err != nil {
		log.Printf("❌ Error finding workflows: %v", err)
		msg.MarkAsFailed()
		mp.messageRepo.Save(ctx, msg)
		return errx.Wrap(err, "failed to find workflows", errx.TypeInternal)
	}
	for _, wf := range workflows {
		// Log workflow IDs found
		log.Printf("🔍 Found matching workflow: %s for message: %s", wf.ID.String(), msg.ID.String())

	}

	// 6. Si no hay workflows, manejar con lógica por defecto
	if len(workflows) == 0 {
		return mp.handleNoWorkflow(ctx, msg, session)
	}

	// 7. Ejecutar el primer workflow que coincida (por prioridad)
	result, err := mp.executeWorkflowWithTimeout(ctx, workflows[0], msg, session)
	if err != nil {
		msg.MarkAsFailed()
		mp.messageRepo.Save(ctx, msg)
		return errx.Wrap(err, "failed to execute workflow", errx.TypeInternal)
	}

	// // 8. Actualizar sesión con el resultado
	if err := mp.updateSessionFromResult(ctx, session, result); err != nil {
		// Log error pero no fallar
		// logger.Error("Failed to update session", err)
	}

	// // 9. Enviar respuesta si es necesario
	if result.ShouldRespond && result.Response != "" {
		if err := mp.sendResponse(ctx, msg, result.Response); err != nil {
			// Log error pero marcar mensaje como procesado
			// logger.Error("Failed to send response", err)
		}
	}

	// 10. Marcar mensaje como procesado
	msg.MarkAsProcessed()
	return mp.messageRepo.Save(ctx, msg)
}

// ProcessWithWorkflow procesa un mensaje con un workflow específico
func (mp *MessageProcessor) ProcessWithWorkflow(ctx context.Context, msg engine.Message, workflowID kernel.WorkflowID) error {
	// 1. Validar mensaje
	if !msg.IsValid() {
		return engine.ErrMessageProcessingFailed().WithDetail("reason", "invalid message")
	}

	// 2. Buscar workflow específico
	workflow, err := mp.workflowRepo.FindByID(ctx, workflowID)
	if err != nil {
		return engine.ErrWorkflowNotFound().WithDetail("workflow_id", workflowID.String())
	}

	// 3. Verificar que el workflow esté activo
	if !workflow.IsActive {
		return engine.ErrWorkflowInactive().WithDetail("workflow_id", workflowID.String())
	}

	// 4. Verificar que pertenezca al mismo tenant
	if workflow.TenantID != msg.TenantID {
		return errx.New("workflow does not belong to message tenant", errx.TypeBusiness).
			WithDetail("workflow_tenant", workflow.TenantID.String()).
			WithDetail("message_tenant", msg.TenantID.String())
	}

	// 5. Marcar mensaje como en procesamiento
	msg.MarkAsProcessing()
	if err := mp.messageRepo.Save(ctx, msg); err != nil {
		return errx.Wrap(err, "failed to save message", errx.TypeInternal)
	}

	// 6. Obtener sesión
	session, err := mp.sessionManager.GetOrCreate(ctx, msg.ChannelID, msg.SenderID, msg.TenantID)
	if err != nil {
		msg.MarkAsFailed()
		mp.messageRepo.Save(ctx, msg)
		return errx.Wrap(err, "failed to get session", errx.TypeInternal)
	}

	// 7. Añadir mensaje al historial
	session.AddMessage(msg.ID, "user")

	// 8. Ejecutar workflow
	result, err := mp.executeWorkflowWithTimeout(ctx, workflow, msg, session)
	if err != nil {
		msg.MarkAsFailed()
		mp.messageRepo.Save(ctx, msg)
		return errx.Wrap(err, "failed to execute workflow", errx.TypeInternal)
	}

	// 9. Actualizar sesión
	if err := mp.updateSessionFromResult(ctx, session, result); err != nil {
		// Log error pero continuar
	}

	// 10. Enviar respuesta
	if result.ShouldRespond && result.Response != "" {
		if err := mp.sendResponse(ctx, msg, result.Response); err != nil {
			// Log error pero continuar
		}
	}

	// 11. Marcar como procesado
	msg.MarkAsProcessed()
	return mp.messageRepo.Save(ctx, msg)
}

// ProcessResponse procesa una respuesta y la envía
func (mp *MessageProcessor) ProcessResponse(ctx context.Context, msg engine.Message, response string) error {
	if response == "" {
		return nil
	}

	// Enviar respuesta
	return mp.sendResponse(ctx, msg, response)
}

// ============================================================================
// Helper Methods
// ============================================================================

// findMatchingWorkflows encuentra workflows que coincidan con el mensaje
func (mp *MessageProcessor) findMatchingWorkflows(ctx context.Context, msg engine.Message) ([]*engine.Workflow, error) {
	// Buscar workflows activos con trigger de mensaje recibido
	trigger := engine.WorkflowTrigger{
		Type:       engine.TriggerTypeMessageReceived,
		ChannelIDs: []string{msg.ChannelID.String()},
	}

	workflows, err := mp.workflowRepo.FindActiveByTrigger(ctx, trigger, msg.TenantID)
	if err != nil {
		log.Printf("❌ Error retrieving workflows from repository: %v", err)
		return nil, err
	}
	log.Printf("🔍 Retrieved %d workflows from repository for tenant: %s", len(workflows), msg.TenantID.String())

	// Filtrar workflows que realmente coincidan
	var matching []*engine.Workflow
	for _, wf := range workflows {
		log.Printf("🔍 Evaluating workflow: %s for message: %s", wf.ID.String(), msg.ID.String())
		if wf.MatchesTrigger(trigger) {
			matching = append(matching, wf)
		}
	}

	return matching, nil
}

// executeWorkflowWithTimeout ejecuta un workflow con timeout
func (mp *MessageProcessor) executeWorkflowWithTimeout(
	ctx context.Context,
	workflow *engine.Workflow,
	msg engine.Message,
	session *engine.Session,
) (*engine.ExecutionResult, error) {
	// Crear contexto con timeout (default 30 segundos)
	timeout := 30 * time.Second
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Canal para resultado
	resultChan := make(chan *engine.ExecutionResult, 1)
	errorChan := make(chan error, 1)

	// Ejecutar en goroutine
	go func() {
		result, err := mp.workflowExec.Execute(ctxWithTimeout, *workflow, msg, session)
		if err != nil {
			errorChan <- err
		} else {
			resultChan <- result
		}
	}()

	// Esperar resultado o timeout
	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errorChan:
		return nil, err
	case <-ctxWithTimeout.Done():
		return nil, engine.ErrExecutionTimeout().
			WithDetail("workflow_id", workflow.ID.String()).
			WithDetail("timeout", timeout.String())
	}
}

// updateSessionFromResult actualiza la sesión con el resultado de la ejecución
func (mp *MessageProcessor) updateSessionFromResult(
	ctx context.Context,
	session *engine.Session,
	result *engine.ExecutionResult,
) error {
	// Actualizar contexto
	if result.Context != nil {
		for key, value := range result.Context {
			session.SetContext(key, value)
		}
	}

	// Actualizar estado
	if result.NextState != "" {
		session.UpdateState(result.NextState)
	}

	// Extender expiración de la sesión
	session.ExtendExpiration(30 * time.Minute)

	// Guardar sesión
	return mp.sessionManager.Update(ctx, *session)
}

// sendResponse envía una respuesta al canal
func (mp *MessageProcessor) sendResponse(ctx context.Context, msg engine.Message, response string) error {
	outgoingMsg := channels.OutgoingMessage{
		RecipientID: msg.SenderID,
		Content: channels.MessageContent{
			Type: "text",
			Text: response,
		},
		Metadata: map[string]any{
			"in_reply_to": msg.ID.String(),
			"timestamp":   time.Now().Unix(),
		},
	}

	return mp.channelManager.SendMessage(ctx, msg.ChannelID, outgoingMsg)
}

// handleNoWorkflow maneja mensajes cuando no hay workflow disponible
func (mp *MessageProcessor) handleNoWorkflow(ctx context.Context, msg engine.Message, session *engine.Session) error {
	// Intentar usar parser por defecto si está disponible

	// Respuesta por defecto
	defaultResponse := "Gracias por tu mensaje. En este momento no hay workflows configurados para procesarlo."

	if err := mp.sendResponse(ctx, msg, defaultResponse); err != nil {
		// Log error pero no fallar
		// logger.Error("Failed to send default response", err)
	}

	// Marcar mensaje como procesado
	msg.MarkAsProcessed()
	return mp.messageRepo.Save(ctx, msg)
}

// ============================================================================
// Additional Utility Methods
// ============================================================================

// GetMessageStatus obtiene el estado de un mensaje
func (mp *MessageProcessor) GetMessageStatus(ctx context.Context, messageID kernel.MessageID) (engine.MessageStatus, error) {
	msg, err := mp.messageRepo.FindByID(ctx, messageID)
	if err != nil {
		return "", engine.ErrMessageNotFound().WithDetail("message_id", messageID.String())
	}
	return msg.Status, nil
}

// RetryFailedMessage reintenta procesar un mensaje fallido
func (mp *MessageProcessor) RetryFailedMessage(ctx context.Context, messageID kernel.MessageID) error {
	msg, err := mp.messageRepo.FindByID(ctx, messageID)
	if err != nil {
		return engine.ErrMessageNotFound().WithDetail("message_id", messageID.String())
	}

	if msg.Status != engine.MessageStatusFailed {
		return errx.New("message is not in failed state", errx.TypeBusiness).
			WithDetail("current_status", string(msg.Status))
	}

	// Resetear estado y procesar de nuevo
	msg.Status = engine.MessageStatusPending
	return mp.ProcessMessage(ctx, *msg)
}

// GetProcessingStats obtiene estadísticas de procesamiento
func (mp *MessageProcessor) GetProcessingStats(ctx context.Context, tenantID kernel.TenantID) (*ProcessingStats, error) {
	pendingCount, _ := mp.messageRepo.CountByStatus(ctx, engine.MessageStatusPending, tenantID)
	processingCount, _ := mp.messageRepo.CountByStatus(ctx, engine.MessageStatusProcessing, tenantID)
	processedCount, _ := mp.messageRepo.CountByStatus(ctx, engine.MessageStatusProcessed, tenantID)
	failedCount, _ := mp.messageRepo.CountByStatus(ctx, engine.MessageStatusFailed, tenantID)

	return &ProcessingStats{
		TenantID:           tenantID,
		PendingMessages:    pendingCount,
		ProcessingMessages: processingCount,
		ProcessedMessages:  processedCount,
		FailedMessages:     failedCount,
		Timestamp:          time.Now(),
	}, nil
}

// ProcessingStats estadísticas de procesamiento
type ProcessingStats struct {
	TenantID           kernel.TenantID `json:"tenant_id"`
	PendingMessages    int             `json:"pending_messages"`
	ProcessingMessages int             `json:"processing_messages"`
	ProcessedMessages  int             `json:"processed_messages"`
	FailedMessages     int             `json:"failed_messages"`
	ActiveSessions     int             `json:"active_sessions"`
	Timestamp          time.Time       `json:"timestamp"`
}

// CleanupSessions limpia sesiones expiradas
func (mp *MessageProcessor) CleanupSessions(ctx context.Context) error {
	return mp.sessionManager.CleanExpiredSessions(ctx)
}

// BulkRetryFailedMessages reintenta procesar múltiples mensajes fallidos
func (mp *MessageProcessor) BulkRetryFailedMessages(ctx context.Context, tenantID kernel.TenantID, limit int) (*BulkRetryResult, error) {
	// Buscar mensajes fallidos
	failedMessages, err := mp.messageRepo.FindByStatus(ctx, engine.MessageStatusFailed, tenantID)
	if err != nil {
		return nil, err
	}

	// Limitar cantidad
	if limit > 0 && len(failedMessages) > limit {
		failedMessages = failedMessages[:limit]
	}

	result := &BulkRetryResult{
		Total:      len(failedMessages),
		Successful: []kernel.MessageID{},
		Failed:     make(map[kernel.MessageID]string),
	}

	// Reintentar cada mensaje
	for _, msg := range failedMessages {
		if err := mp.RetryFailedMessage(ctx, msg.ID); err != nil {
			result.Failed[msg.ID] = err.Error()
		} else {
			result.Successful = append(result.Successful, msg.ID)
		}
	}

	return result, nil
}

// BulkRetryResult resultado de reintento masivo
type BulkRetryResult struct {
	Total      int                         `json:"total"`
	Successful []kernel.MessageID          `json:"successful"`
	Failed     map[kernel.MessageID]string `json:"failed"`
}
