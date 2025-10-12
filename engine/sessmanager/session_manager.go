package sessmanager

import (
	"context"
	"time"

	"github.com/Abraxas-365/craftable/errx"
	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/google/uuid"
)

// SessionManager implements SessionManager interface
type SessionManager struct {
	repo                  engine.SessionRepository
	defaultExpirationTime time.Duration // Default session expiration (e.g., 24 hours)
	maxHistorySize        int           // Maximum number of messages in history
}

// SessionManagerConfig configuration for session manager
type SessionManagerConfig struct {
	DefaultExpirationTime time.Duration // Default: 24 hours
	MaxHistorySize        int           // Default: 100 messages
}

// NewSessionManager creates a new session manager
func NewSessionManager(repo engine.SessionRepository, config *SessionManagerConfig) *SessionManager {
	if config == nil {
		config = &SessionManagerConfig{
			DefaultExpirationTime: 24 * time.Hour,
			MaxHistorySize:        100,
		}
	}

	// Set defaults if not provided
	if config.DefaultExpirationTime == 0 {
		config.DefaultExpirationTime = 24 * time.Hour
	}
	if config.MaxHistorySize == 0 {
		config.MaxHistorySize = 100
	}

	return &SessionManager{
		repo:                  repo,
		defaultExpirationTime: config.DefaultExpirationTime,
		maxHistorySize:        config.MaxHistorySize,
	}
}

// GetOrCreate obtains an existing session or creates a new one
func (m *SessionManager) GetOrCreate(ctx context.Context, channelID kernel.ChannelID, senderID string, tenantID kernel.TenantID) (*engine.Session, error) {
	// Try to find existing session
	session, err := m.repo.FindByChannelAndSender(ctx, channelID, senderID)
	if err == nil {
		// Session found - check if expired
		if session.IsExpired() {
			// Session expired, create a new one
			return m.createNewSession(ctx, channelID, senderID, tenantID)
		}

		// Session is valid, extend its activity
		session.UpdateActivity()
		if err := m.repo.Save(ctx, *session); err != nil {
			return nil, errx.Wrap(err, "failed to update session activity", errx.TypeInternal).
				WithDetail("session_id", session.ID)
		}

		return session, nil
	}

	// Check if error is "not found"
	if errx.IsType(err, errx.TypeNotFound) {
		// Session doesn't exist, create new one
		return m.createNewSession(ctx, channelID, senderID, tenantID)
	}

	// Some other error occurred
	return nil, errx.Wrap(err, "failed to find session", errx.TypeInternal)
}

// createNewSession creates and saves a new session
func (m *SessionManager) createNewSession(ctx context.Context, channelID kernel.ChannelID, senderID string, tenantID kernel.TenantID) (*engine.Session, error) {
	now := time.Now()

	session := &engine.Session{
		ID:             kernel.NewSessionID(uuid.New().String()),
		TenantID:       tenantID,
		ChannelID:      channelID,
		SenderID:       senderID,
		Context:        make(map[string]any),
		History:        []engine.MessageRef{},
		CurrentState:   "initial",
		ExpiresAt:      now.Add(m.defaultExpirationTime),
		CreatedAt:      now,
		LastActivityAt: now,
	}

	if err := m.repo.Save(ctx, *session); err != nil {
		return nil, errx.Wrap(err, "failed to create session", errx.TypeInternal).
			WithDetail("channel_id", channelID.String()).
			WithDetail("sender_id", senderID)
	}

	return session, nil
}

// Update updates the entire session
func (m *SessionManager) Update(ctx context.Context, session engine.Session) error {
	// Validate session
	if !session.IsValid() {
		return errx.New("invalid session", errx.TypeValidation).
			WithDetail("session_id", session.ID)
	}

	// Update activity timestamp
	session.UpdateActivity()

	// Trim history if too large
	if len(session.History) > m.maxHistorySize {
		session.History = session.History[len(session.History)-m.maxHistorySize:]
	}

	if err := m.repo.Save(ctx, session); err != nil {
		return errx.Wrap(err, "failed to update session", errx.TypeInternal).
			WithDetail("session_id", session.ID)
	}

	return nil
}

// UpdateContext updates a specific context key in the session
func (m *SessionManager) UpdateContext(ctx context.Context, sessionID kernel.SessionID, key string, value any) error {
	// Get current session
	session, err := m.repo.FindByID(ctx, sessionID)
	if err != nil {
		return errx.Wrap(err, "failed to find session", errx.TypeInternal).
			WithDetail("session_id", string(sessionID))
	}

	// Check if expired
	if session.IsExpired() {
		return errx.New("session expired", errx.TypeValidation).
			WithDetail("session_id", string(sessionID)).
			WithDetail("expires_at", session.ExpiresAt.String())
	}

	// Update context
	session.SetContext(key, value)

	// Save session
	if err := m.repo.Save(ctx, *session); err != nil {
		return errx.Wrap(err, "failed to update session context", errx.TypeInternal).
			WithDetail("session_id", string(sessionID)).
			WithDetail("context_key", key)
	}

	return nil
}

// UpdateState updates the current state of the session
func (m *SessionManager) UpdateState(ctx context.Context, sessionID kernel.SessionID, state string) error {
	// Get current session
	session, err := m.repo.FindByID(ctx, sessionID)
	if err != nil {
		return errx.Wrap(err, "failed to find session", errx.TypeInternal).
			WithDetail("session_id", string(sessionID))
	}

	// Check if expired
	if session.IsExpired() {
		return errx.New("session expired", errx.TypeValidation).
			WithDetail("session_id", string(sessionID)).
			WithDetail("expires_at", session.ExpiresAt.String())
	}

	// Update state
	session.UpdateState(state)

	// Save session
	if err := m.repo.Save(ctx, *session); err != nil {
		return errx.Wrap(err, "failed to update session state", errx.TypeInternal).
			WithDetail("session_id", string(sessionID)).
			WithDetail("state", state)
	}

	return nil
}

// Delete deletes a session
func (m *SessionManager) Delete(ctx context.Context, sessionID kernel.SessionID) error {
	if err := m.repo.Delete(ctx, sessionID); err != nil {
		return errx.Wrap(err, "failed to delete session", errx.TypeInternal).
			WithDetail("session_id", string(sessionID))
	}

	return nil
}

// Get retrieves a session by ID
func (m *SessionManager) Get(ctx context.Context, sessionID kernel.SessionID) (*engine.Session, error) {
	session, err := m.repo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, errx.Wrap(err, "failed to get session", errx.TypeInternal).
			WithDetail("session_id", string(sessionID))
	}

	return session, nil
}

// ExtendSession extends the expiration time of a session
func (m *SessionManager) ExtendSession(ctx context.Context, sessionID kernel.SessionID) error {
	// Get current session
	session, err := m.repo.FindByID(ctx, sessionID)
	if err != nil {
		return errx.Wrap(err, "failed to find session", errx.TypeInternal).
			WithDetail("session_id", string(sessionID))
	}

	// Extend expiration
	session.ExtendExpiration(m.defaultExpirationTime)

	// Save session
	if err := m.repo.Save(ctx, *session); err != nil {
		return errx.Wrap(err, "failed to extend session", errx.TypeInternal).
			WithDetail("session_id", string(sessionID))
	}

	return nil
}

// CleanExpiredSessions removes all expired sessions
func (m *SessionManager) CleanExpiredSessions(ctx context.Context) error {
	if err := m.repo.CleanExpired(ctx); err != nil {
		return errx.Wrap(err, "failed to clean expired sessions", errx.TypeInternal)
	}

	return nil
}

// AddMessageToHistory adds a message reference to the session history
func (m *SessionManager) AddMessageToHistory(ctx context.Context, sessionID kernel.SessionID, messageID kernel.MessageID, role string) error {
	// Get current session
	session, err := m.repo.FindByID(ctx, sessionID)
	if err != nil {
		return errx.Wrap(err, "failed to find session", errx.TypeInternal).
			WithDetail("session_id", string(sessionID))
	}

	// Check if expired
	if session.IsExpired() {
		return errx.New("session expired", errx.TypeValidation).
			WithDetail("session_id", string(sessionID))
	}

	// Add message to history
	session.AddMessage(messageID, role)

	// Trim history if too large
	if len(session.History) > m.maxHistorySize {
		session.History = session.History[len(session.History)-m.maxHistorySize:]
	}

	// Save session
	if err := m.repo.Save(ctx, *session); err != nil {
		return errx.Wrap(err, "failed to add message to session history", errx.TypeInternal).
			WithDetail("session_id", string(sessionID)).
			WithDetail("message_id", messageID.String())
	}

	return nil
}

// GetContext retrieves a value from the session context
func (m *SessionManager) GetContext(ctx context.Context, sessionID kernel.SessionID, key string) (any, bool, error) {
	session, err := m.repo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, false, errx.Wrap(err, "failed to find session", errx.TypeInternal).
			WithDetail("session_id", string(sessionID))
	}

	value, ok := session.GetContext(key)
	return value, ok, nil
}

// ClearContext clears all context variables from a session
func (m *SessionManager) ClearContext(ctx context.Context, sessionID kernel.SessionID) error {
	session, err := m.repo.FindByID(ctx, sessionID)
	if err != nil {
		return errx.Wrap(err, "failed to find session", errx.TypeInternal).
			WithDetail("session_id", string(sessionID))
	}

	// Clear context
	session.Context = make(map[string]any)
	session.UpdateActivity()

	// Save session
	if err := m.repo.Save(ctx, *session); err != nil {
		return errx.Wrap(err, "failed to clear session context", errx.TypeInternal).
			WithDetail("session_id", string(sessionID))
	}

	return nil
}

// ResetSession resets a session to its initial state while keeping the same ID
func (m *SessionManager) ResetSession(ctx context.Context, sessionID kernel.SessionID) error {
	session, err := m.repo.FindByID(ctx, sessionID)
	if err != nil {
		return errx.Wrap(err, "failed to find session", errx.TypeInternal).
			WithDetail("session_id", string(sessionID))
	}

	// Reset session
	now := time.Now()
	session.Context = make(map[string]any)
	session.History = []engine.MessageRef{}
	session.CurrentState = "initial"
	session.ExpiresAt = now.Add(m.defaultExpirationTime)
	session.LastActivityAt = now

	// Save session
	if err := m.repo.Save(ctx, *session); err != nil {
		return errx.Wrap(err, "failed to reset session", errx.TypeInternal).
			WithDetail("session_id", string(sessionID))
	}

	return nil
}

// GetHistoryCount returns the number of messages in the session history
func (m *SessionManager) GetHistoryCount(ctx context.Context, sessionID kernel.SessionID) (int, error) {
	session, err := m.repo.FindByID(ctx, sessionID)
	if err != nil {
		return 0, errx.Wrap(err, "failed to find session", errx.TypeInternal).
			WithDetail("session_id", string(sessionID))
	}

	return session.GetHistoryCount(), nil
}

// IsSessionActive checks if a session exists and is not expired
func (m *SessionManager) IsSessionActive(ctx context.Context, sessionID kernel.SessionID) (bool, error) {
	session, err := m.repo.FindByID(ctx, sessionID)
	if err != nil {
		if errx.IsType(err, errx.TypeNotFound) {
			return false, nil
		}
		return false, errx.Wrap(err, "failed to check session", errx.TypeInternal).
			WithDetail("session_id", string(sessionID))
	}

	return !session.IsExpired(), nil
}
