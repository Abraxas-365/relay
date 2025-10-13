-- ============================================================================
-- AGENT MESSAGES Migration
-- ============================================================================
-- Stores AI agent conversation messages linked to sessions
-- ============================================================================

-- Agent messages table
CREATE TABLE agent_messages (
    id TEXT PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL CHECK (role IN ('system', 'user', 'assistant', 'tool', 'function')),
    content TEXT,
    name VARCHAR(255),
    function_call JSONB,
    tool_calls JSONB,
    tool_call_id VARCHAR(255),
    metadata JSONB DEFAULT '{}',
    message_type VARCHAR(50) NOT NULL DEFAULT 'text' CHECK (message_type IN ('text', 'image', 'document', 'audio', 'video', 'template')),
    processing_time_ms INTEGER CHECK (processing_time_ms >= 0),
    model_used VARCHAR(100),
    tokens_used INTEGER CHECK (tokens_used >= 0),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_agent_messages_session_id ON agent_messages(session_id);
CREATE INDEX idx_agent_messages_role ON agent_messages(role);
CREATE INDEX idx_agent_messages_message_type ON agent_messages(message_type);
CREATE INDEX idx_agent_messages_created_at ON agent_messages(created_at);
CREATE INDEX idx_agent_messages_session_created ON agent_messages(session_id, created_at);
CREATE INDEX idx_agent_messages_metadata ON agent_messages USING gin(metadata);
CREATE INDEX idx_agent_messages_tool_call_id ON agent_messages(tool_call_id) WHERE tool_call_id IS NOT NULL;

-- Trigger for updated_at
CREATE TRIGGER update_agent_messages_updated_at 
    BEFORE UPDATE ON agent_messages 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Comments
COMMENT ON TABLE agent_messages IS 'AI agent conversation messages with tool/function call support';
COMMENT ON COLUMN agent_messages.role IS 'Message role: system, user, assistant, tool, or function';
COMMENT ON COLUMN agent_messages.function_call IS 'Legacy function call data (deprecated, use tool_calls)';
COMMENT ON COLUMN agent_messages.tool_calls IS 'Array of tool calls made by the assistant';
COMMENT ON COLUMN agent_messages.tool_call_id IS 'ID of the tool call this message responds to (for tool role)';
COMMENT ON COLUMN agent_messages.metadata IS 'Additional metadata for the message';
COMMENT ON COLUMN agent_messages.message_type IS 'Type of message content (text, image, document, etc.)';

-- Success message
DO $$
BEGIN
    RAISE NOTICE '========================================';
    RAISE NOTICE '✅ AGENT MESSAGES TABLE CREATED!';
    RAISE NOTICE '========================================';
    RAISE NOTICE '';
    RAISE NOTICE '📊 Table: agent_messages';
    RAISE NOTICE '🔗 Indexes: 7';
    RAISE NOTICE '🔔 Triggers: 1 (updated_at)';
    RAISE NOTICE '';
    RAISE NOTICE '✨ Ready for AI agent conversations!';
    RAISE NOTICE '========================================';
END $$;

-- Add status column to sessions table
ALTER TABLE sessions 
ADD COLUMN status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE' 
CHECK (status IN ('ACTIVE', 'CLOSED', 'EXPIRED'));

ALTER TABLE sessions 
ADD COLUMN closed_at TIMESTAMP WITH TIME ZONE;


-- Drop the old unique constraint
ALTER TABLE sessions 
DROP CONSTRAINT sessions_channel_id_sender_id_key;

-- Create new unique constraint only for ACTIVE sessions
CREATE UNIQUE INDEX sessions_channel_sender_active_idx 
ON sessions(channel_id, sender_id) 
WHERE status = 'ACTIVE';
