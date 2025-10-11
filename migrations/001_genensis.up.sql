-- ============================================================================
-- RELAY GENESIS MIGRATION - Complete Database Schema
-- ============================================================================
-- Multi-tenant messaging automation platform with AI-powered workflows
-- Includes: IAM, Auth, Channels, Messages, Sessions, Workflows, Parsers, Tools
-- ============================================================================

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- IAM Tables (Identity and Access Management)
-- ============================================================================

-- Tenants table
CREATE TABLE tenants (
    id TEXT PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_name VARCHAR(255) NOT NULL,
    ruc VARCHAR(11) UNIQUE NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'TRIAL' CHECK (status IN ('ACTIVE', 'SUSPENDED', 'CANCELED', 'TRIAL')),
    subscription_plan VARCHAR(50) NOT NULL DEFAULT 'TRIAL' CHECK (subscription_plan IN ('TRIAL', 'BASIC', 'PROFESSIONAL', 'ENTERPRISE')),
    max_users INTEGER NOT NULL DEFAULT 5,
    current_users INTEGER NOT NULL DEFAULT 0,
    trial_expires_at TIMESTAMP WITH TIME ZONE,
    subscription_expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Users table
CREATE TABLE users (
    id TEXT PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    picture TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('ACTIVE', 'INACTIVE', 'SUSPENDED', 'PENDING')),
    is_admin BOOLEAN NOT NULL DEFAULT false,
    oauth_provider VARCHAR(50) NOT NULL DEFAULT 'GOOGLE' CHECK (oauth_provider IN ('GOOGLE', 'MICROSOFT', 'AUTH0')),
    oauth_provider_id VARCHAR(255) NOT NULL,
    email_verified BOOLEAN NOT NULL DEFAULT false,
    last_login_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(email, tenant_id)
);

-- Roles table
CREATE TABLE roles (
    id TEXT PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(name, tenant_id)
);

-- User roles junction table
CREATE TABLE user_roles (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id)
);

-- Role permissions table
CREATE TABLE role_permissions (
    role_id TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission VARCHAR(255) NOT NULL,
    assigned_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_id, permission)
);

-- Tenant configuration table
CREATE TABLE tenant_config (
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    key VARCHAR(255) NOT NULL,
    value TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, key)
);

-- ============================================================================
-- Auth Tables (Authentication & Session Management)
-- ============================================================================

-- Refresh tokens table
CREATE TABLE refresh_tokens (
    id TEXT PRIMARY KEY DEFAULT uuid_generate_v4(),
    token TEXT UNIQUE NOT NULL,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    is_revoked BOOLEAN NOT NULL DEFAULT false
);

-- User sessions table
CREATE TABLE user_sessions (
    id TEXT PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    session_token TEXT UNIQUE NOT NULL,
    ip_address INET,
    user_agent TEXT,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_activity TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Password reset tokens table
CREATE TABLE password_reset_tokens (
    id TEXT PRIMARY KEY DEFAULT uuid_generate_v4(),
    token TEXT UNIQUE NOT NULL,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    is_used BOOLEAN NOT NULL DEFAULT false
);

-- ============================================================================
-- CHANNELS Tables (Multi-Channel Messaging)
-- ============================================================================

-- Channels table (supports WhatsApp, Instagram, Telegram, Email, SMS, etc.)
CREATE TABLE channels (
    id TEXT PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL CHECK (type IN ('WHATSAPP', 'INSTAGRAM', 'TELEGRAM', 'INFOBIP', 'EMAIL', 'SMS', 'WEBCHAT', 'VOICE')),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    config JSONB NOT NULL, -- Channel-specific configuration (credentials, settings)
    is_active BOOLEAN NOT NULL DEFAULT true,
    webhook_url TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(name, tenant_id)
);

-- Channel message statistics
CREATE TABLE channel_stats (
    id TEXT PRIMARY KEY DEFAULT uuid_generate_v4(),
    channel_id TEXT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    stat_date DATE NOT NULL DEFAULT CURRENT_DATE,
    messages_sent INTEGER NOT NULL DEFAULT 0,
    messages_received INTEGER NOT NULL DEFAULT 0,
    messages_failed INTEGER NOT NULL DEFAULT 0,
    avg_response_time_ms INTEGER,
    success_rate DECIMAL(5,4),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(channel_id, stat_date)
);

-- ============================================================================
-- MESSAGES Tables (Normalized Message Storage)
-- ============================================================================

-- Messages table (normalized messages from all channels)
CREATE TABLE messages (
    id TEXT PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    channel_id TEXT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    sender_id VARCHAR(255) NOT NULL, -- External sender ID (phone, username, email)
    content JSONB NOT NULL, -- {type, text, attachments, metadata}
    context JSONB, -- Additional context data
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'PROCESSING', 'PROCESSED', 'FAILED')),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- SESSIONS Tables (Conversation Context)
-- ============================================================================

-- Sessions table (conversation context tracking)
CREATE TABLE sessions (
    id TEXT PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    channel_id TEXT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    sender_id VARCHAR(255) NOT NULL,
    context JSONB, -- Session context variables
    history JSONB, -- Message history references
    current_state VARCHAR(255),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_activity_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(channel_id, sender_id)
);

-- ============================================================================
-- WORKFLOWS Tables (Automation Engine)
-- ============================================================================

-- Workflows table (automation workflows)
CREATE TABLE workflows (
    id TEXT PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    trigger JSONB NOT NULL, -- {type, channel_ids, schedule, filters}
    steps JSONB NOT NULL, -- Array of workflow steps
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(name, tenant_id)
);

-- Workflow executions (tracking)
CREATE TABLE workflow_executions (
    id TEXT PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_id TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    message_id TEXT REFERENCES messages(id) ON DELETE CASCADE,
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    session_id TEXT REFERENCES sessions(id) ON DELETE SET NULL,
    status VARCHAR(50) NOT NULL CHECK (status IN ('RUNNING', 'SUCCESS', 'FAILED', 'TIMEOUT')),
    response TEXT,
    should_respond BOOLEAN NOT NULL DEFAULT false,
    next_state VARCHAR(255),
    context JSONB,
    error TEXT,
    executed_steps JSONB, -- Array of step results
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms INTEGER
);

-- ============================================================================
-- PARSERS Tables (Message Parsing Engine)
-- ============================================================================

-- Parsers table (message parsers - AI, Regex, Rule-based, Keyword, NLP)
CREATE TABLE parsers (
    id TEXT PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL CHECK (type IN ('REGEX', 'AI', 'RULE', 'KEYWORD', 'NLP')),
    config JSONB NOT NULL, -- Parser-specific configuration
    priority INTEGER NOT NULL DEFAULT 100, -- Lower number = higher priority
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(name, tenant_id)
);

-- Parser executions (tracking and caching)
CREATE TABLE parser_executions (
    id TEXT PRIMARY KEY DEFAULT uuid_generate_v4(),
    parser_id TEXT NOT NULL REFERENCES parsers(id) ON DELETE CASCADE,
    message_id TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    success BOOLEAN NOT NULL,
    response TEXT,
    should_respond BOOLEAN NOT NULL DEFAULT false,
    actions JSONB, -- Array of actions to execute
    context JSONB, -- Context updates
    extracted_data JSONB, -- Extracted entities, regex groups, etc.
    confidence DECIMAL(3,2), -- 0.00 to 1.00
    next_parser_id TEXT REFERENCES parsers(id),
    error TEXT,
    processed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    processing_time_ms INTEGER
);

-- ============================================================================
-- TOOLS Tables (Executable Tools)
-- ============================================================================

-- Tools table (HTTP, Database, Email, Custom code execution)
CREATE TABLE tools (
    id TEXT PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL CHECK (type IN ('HTTP', 'DATABASE', 'EMAIL', 'CUSTOM')),
    config JSONB NOT NULL, -- Tool-specific configuration
    input_schema JSONB, -- JSON Schema for input validation
    output_schema JSONB, -- JSON Schema for output validation
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(name, tenant_id)
);

-- Tool executions (tracking)
CREATE TABLE tool_executions (
    id TEXT PRIMARY KEY DEFAULT uuid_generate_v4(),
    tool_id TEXT NOT NULL REFERENCES tools(id) ON DELETE CASCADE,
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    input JSONB NOT NULL,
    output JSONB,
    status VARCHAR(50) NOT NULL CHECK (status IN ('PENDING', 'RUNNING', 'SUCCESS', 'FAILED')),
    error TEXT,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMP WITH TIME ZONE,
    duration_ms INTEGER
);

-- ============================================================================
-- Indexes for Performance
-- ============================================================================

-- Tenants indexes
CREATE INDEX idx_tenants_ruc ON tenants(ruc);
CREATE INDEX idx_tenants_status ON tenants(status);

-- Users indexes
CREATE INDEX idx_users_tenant_id ON users(tenant_id);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_oauth_provider ON users(oauth_provider, oauth_provider_id);

-- Roles indexes
CREATE INDEX idx_roles_tenant_id ON roles(tenant_id);
CREATE INDEX idx_roles_is_active ON roles(is_active);

-- User roles indexes
CREATE INDEX idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);

-- Auth tables indexes
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX idx_user_sessions_expires_at ON user_sessions(expires_at);

-- Channels indexes
CREATE INDEX idx_channels_tenant_id ON channels(tenant_id);
CREATE INDEX idx_channels_type ON channels(type);
CREATE INDEX idx_channels_is_active ON channels(is_active);
CREATE INDEX idx_channels_webhook_url ON channels(webhook_url);

-- Channel stats indexes
CREATE INDEX idx_channel_stats_channel_id ON channel_stats(channel_id);
CREATE INDEX idx_channel_stats_date ON channel_stats(stat_date);
CREATE INDEX idx_channel_stats_channel_date ON channel_stats(channel_id, stat_date);

-- Messages indexes
CREATE INDEX idx_messages_tenant_id ON messages(tenant_id);
CREATE INDEX idx_messages_channel_id ON messages(channel_id);
CREATE INDEX idx_messages_sender_id ON messages(sender_id);
CREATE INDEX idx_messages_status ON messages(status);
CREATE INDEX idx_messages_created_at ON messages(created_at);
CREATE INDEX idx_messages_channel_sender ON messages(channel_id, sender_id);
CREATE INDEX idx_messages_content ON messages USING gin(content);

-- Sessions indexes
CREATE INDEX idx_sessions_tenant_id ON sessions(tenant_id);
CREATE INDEX idx_sessions_channel_id ON sessions(channel_id);
CREATE INDEX idx_sessions_sender_id ON sessions(sender_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX idx_sessions_last_activity ON sessions(last_activity_at);
CREATE INDEX idx_sessions_current_state ON sessions(current_state);
CREATE INDEX idx_sessions_context ON sessions USING gin(context);

-- Workflows indexes
CREATE INDEX idx_workflows_tenant_id ON workflows(tenant_id);
CREATE INDEX idx_workflows_is_active ON workflows(is_active);
CREATE INDEX idx_workflows_name ON workflows(name);
CREATE INDEX idx_workflows_trigger ON workflows USING gin(trigger);

-- Workflow executions indexes
CREATE INDEX idx_workflow_executions_workflow_id ON workflow_executions(workflow_id);
CREATE INDEX idx_workflow_executions_message_id ON workflow_executions(message_id);
CREATE INDEX idx_workflow_executions_session_id ON workflow_executions(session_id);
CREATE INDEX idx_workflow_executions_tenant_id ON workflow_executions(tenant_id);
CREATE INDEX idx_workflow_executions_status ON workflow_executions(status);
CREATE INDEX idx_workflow_executions_started_at ON workflow_executions(started_at);

-- Parsers indexes
CREATE INDEX idx_parsers_tenant_id ON parsers(tenant_id);
CREATE INDEX idx_parsers_type ON parsers(type);
CREATE INDEX idx_parsers_priority ON parsers(priority);
CREATE INDEX idx_parsers_is_active ON parsers(is_active);
CREATE INDEX idx_parsers_name ON parsers(name);

-- Parser executions indexes
CREATE INDEX idx_parser_executions_parser_id ON parser_executions(parser_id);
CREATE INDEX idx_parser_executions_message_id ON parser_executions(message_id);
CREATE INDEX idx_parser_executions_tenant_id ON parser_executions(tenant_id);
CREATE INDEX idx_parser_executions_success ON parser_executions(success);
CREATE INDEX idx_parser_executions_processed_at ON parser_executions(processed_at);
CREATE INDEX idx_parser_executions_confidence ON parser_executions(confidence);

-- Tools indexes
CREATE INDEX idx_tools_tenant_id ON tools(tenant_id);
CREATE INDEX idx_tools_type ON tools(type);
CREATE INDEX idx_tools_is_active ON tools(is_active);
CREATE INDEX idx_tools_name ON tools(name);

-- Tool executions indexes
CREATE INDEX idx_tool_executions_tool_id ON tool_executions(tool_id);
CREATE INDEX idx_tool_executions_tenant_id ON tool_executions(tenant_id);
CREATE INDEX idx_tool_executions_status ON tool_executions(status);
CREATE INDEX idx_tool_executions_started_at ON tool_executions(started_at);

-- ============================================================================
-- Triggers for updated_at
-- ============================================================================

-- Function to automatically update the updated_at column
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply triggers
CREATE TRIGGER update_tenants_updated_at BEFORE UPDATE ON tenants FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_roles_updated_at BEFORE UPDATE ON roles FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_tenant_config_updated_at BEFORE UPDATE ON tenant_config FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_channels_updated_at BEFORE UPDATE ON channels FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_channel_stats_updated_at BEFORE UPDATE ON channel_stats FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_messages_updated_at BEFORE UPDATE ON messages FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_workflows_updated_at BEFORE UPDATE ON workflows FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_parsers_updated_at BEFORE UPDATE ON parsers FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_tools_updated_at BEFORE UPDATE ON tools FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- Comments
-- ============================================================================

COMMENT ON TABLE tenants IS 'Multi-tenant organizations using the platform';
COMMENT ON TABLE users IS 'Users belonging to tenants with OAuth authentication';
COMMENT ON TABLE roles IS 'Role-based access control';
COMMENT ON TABLE channels IS 'Multi-channel messaging endpoints (WhatsApp, Email, SMS, etc.)';
COMMENT ON TABLE messages IS 'Normalized messages from all channels';
COMMENT ON TABLE sessions IS 'Conversation context and state management';
COMMENT ON TABLE workflows IS 'Automated workflows triggered by messages or events';
COMMENT ON TABLE parsers IS 'Message parsing engines (AI, Regex, Rules, Keywords, NLP)';
COMMENT ON TABLE tools IS 'Executable tools for HTTP calls, DB queries, emails, custom code';

COMMENT ON COLUMN channels.config IS 'JSONB containing channel-specific credentials and settings';
COMMENT ON COLUMN messages.content IS 'JSONB containing message type, text, attachments, and metadata';
COMMENT ON COLUMN sessions.context IS 'JSONB containing session variables and state';
COMMENT ON COLUMN workflows.trigger IS 'JSONB defining when workflow should execute';
COMMENT ON COLUMN workflows.steps IS 'JSONB array of workflow steps to execute';
COMMENT ON COLUMN parsers.config IS 'JSONB containing parser configuration (patterns, prompts, rules, etc.)';
COMMENT ON COLUMN parsers.priority IS 'Lower number = higher priority for parser selection';

-- ============================================================================
-- Completion Message
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE '========================================';
    RAISE NOTICE '✅ RELAY GENESIS MIGRATION COMPLETED!';
    RAISE NOTICE '========================================';
    RAISE NOTICE '';
    RAISE NOTICE '📊 Created IAM Tables: tenants, users, roles, user_roles, role_permissions, tenant_config';
    RAISE NOTICE '🔐 Created Auth Tables: refresh_tokens, user_sessions, password_reset_tokens';
    RAISE NOTICE '📱 Created Channel Tables: channels, channel_stats';
    RAISE NOTICE '💬 Created Message Tables: messages, sessions';
    RAISE NOTICE '⚙️ Created Automation Tables: workflows, workflow_executions, parsers, parser_executions, tools, tool_executions';
    RAISE NOTICE '';
    RAISE NOTICE '📈 Total Tables: 18';
    RAISE NOTICE '📌 Total Indexes: 65+';
    RAISE NOTICE '';
    RAISE NOTICE '🚀 Relay Platform is ready!';
    RAISE NOTICE '========================================';
END $$;
