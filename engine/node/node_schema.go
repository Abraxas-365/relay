package node

import "github.com/Abraxas-365/craftable/ptrx"

// ============================================================================
// Schema Types
// ============================================================================

type NodeConfigSchema struct {
	NodeType    string        `json:"node_type"`
	DisplayName string        `json:"display_name"`
	Description string        `json:"description"`
	Icon        string        `json:"icon,omitempty"`
	Category    string        `json:"category"`
	Fields      []FieldSchema `json:"fields"`
}

type FieldSchema struct {
	Name         string        `json:"name"`
	Label        string        `json:"label"`
	Type         FieldType     `json:"type"`
	Required     bool          `json:"required"`
	DefaultValue any           `json:"default_value,omitempty"`
	Description  string        `json:"description"`
	Placeholder  string        `json:"placeholder,omitempty"`
	Options      []FieldOption `json:"options,omitempty"` // For select/radio
	Validation   *Validation   `json:"validation,omitempty"`
	DependsOn    *Dependency   `json:"depends_on,omitempty"` // Conditional fields
}

type FieldType string

const (
	FieldTypeString   FieldType = "string"
	FieldTypeNumber   FieldType = "number"
	FieldTypeBoolean  FieldType = "boolean"
	FieldTypeSelect   FieldType = "select"
	FieldTypeTextarea FieldType = "textarea"
	FieldTypeJSON     FieldType = "json"
	FieldTypeURL      FieldType = "url"
	FieldTypeEmail    FieldType = "email"
	FieldTypePhone    FieldType = "phone"
	FieldTypeArray    FieldType = "array"
	FieldTypeKeyValue FieldType = "key_value" // For maps like headers
)

type FieldOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

type Validation struct {
	Min     *float32 `json:"min,omitempty"`
	Max     *float32 `json:"max,omitempty"`
	Pattern string   `json:"pattern,omitempty"` // Regex
	Message string   `json:"message,omitempty"`
}

type Dependency struct {
	Field string `json:"field"`
	Value any    `json:"value"`
}

// ============================================================================
// All Node Schemas
// ============================================================================

func GetAllNodeSchemas() map[string]NodeConfigSchema {
	return map[string]NodeConfigSchema{
		"AI_AGENT":     GetAIAgentSchema(),
		"HTTP":         GetHTTPSchema(),
		"SEND_MESSAGE": GetSendMessageSchema(),
		"TRANSFORM":    GetTransformSchema(),
		"CONDITION":    GetConditionSchema(),
		"SWITCH":       GetSwitchSchema(),
		"LOOP":         GetLoopSchema(),
		"VALIDATE":     GetValidateSchema(),
		"DELAY":        GetDelaySchema(),
		"ACTION":       GetActionSchema(),
	}
}

// ============================================================================
// 1. AI_AGENT Schema
// ============================================================================

func GetAIAgentSchema() NodeConfigSchema {
	return NodeConfigSchema{
		NodeType:    "AI_AGENT",
		DisplayName: "AI Agent",
		Description: "Execute AI tasks with LLM models (OpenAI, Claude, etc.)",
		Icon:        "🤖",
		Category:    "AI",
		Fields: []FieldSchema{
			{
				Name:         "provider",
				Label:        "Provider",
				Type:         FieldTypeSelect,
				Required:     true,
				Description:  "AI provider",
				DefaultValue: "openai",
				Options: []FieldOption{
					{Value: "openai", Label: "OpenAI", Description: "GPT models"},
					{Value: "anthropic", Label: "Anthropic", Description: "Claude models"},
					{Value: "google", Label: "Google", Description: "Gemini models"},
				},
			},
			{
				Name:         "model",
				Label:        "Model",
				Type:         FieldTypeSelect,
				Required:     true,
				Description:  "AI model to use",
				DefaultValue: "gpt-4",
				Options: []FieldOption{
					{Value: "gpt-4", Label: "GPT-4"},
					{Value: "gpt-4-turbo", Label: "GPT-4 Turbo"},
					{Value: "gpt-3.5-turbo", Label: "GPT-3.5 Turbo"},
					{Value: "claude-3-opus", Label: "Claude 3 Opus"},
					{Value: "claude-3-sonnet", Label: "Claude 3 Sonnet"},
				},
			},
			{
				Name:        "system_prompt",
				Label:       "System Prompt",
				Type:        FieldTypeTextarea,
				Required:    true,
				Description: "Instructions for the AI assistant",
				Placeholder: "You are a helpful assistant that...",
			},
			{
				Name:        "prompt",
				Label:       "User Prompt",
				Type:        FieldTypeTextarea,
				Required:    false,
				Description: "User message (if not from trigger). Supports {{variable}} syntax",
				Placeholder: "Generate a summary of {{trigger.body.text}}",
			},
			{
				Name:         "temperature",
				Label:        "Temperature",
				Type:         FieldTypeNumber,
				Required:     false,
				DefaultValue: 0.7,
				Description:  "Creativity level (0-2). Higher = more creative",
				Validation: &Validation{
					Min:     ptrx.Float32(0),
					Max:     ptrx.Float32(2),
					Message: "Temperature must be between 0 and 2",
				},
			},
			{
				Name:         "max_tokens",
				Label:        "Max Tokens",
				Type:         FieldTypeNumber,
				Required:     false,
				DefaultValue: 1000,
				Description:  "Maximum response length",
				Validation: &Validation{
					Min:     ptrx.Float32(1),
					Max:     ptrx.Float32(8000),
					Message: "Max tokens must be between 1 and 8000",
				},
			},
			{
				Name:         "use_memory",
				Label:        "Use Conversation Memory",
				Type:         FieldTypeBoolean,
				Required:     false,
				DefaultValue: false,
				Description:  "Enable persistent conversation memory",
			},
			{
				Name:         "max_auto_iterations",
				Label:        "Max Auto Iterations",
				Type:         FieldTypeNumber,
				Required:     false,
				DefaultValue: 3,
				Description:  "Max auto-iterations for agent",
				DependsOn: &Dependency{
					Field: "use_memory",
					Value: true,
				},
			},
		},
	}
}

// ============================================================================
// 2. HTTP Schema
// ============================================================================

func GetHTTPSchema() NodeConfigSchema {
	return NodeConfigSchema{
		NodeType:    "HTTP",
		DisplayName: "HTTP Request",
		Description: "Make HTTP/REST API calls",
		Icon:        "🌐",
		Category:    "Integration",
		Fields: []FieldSchema{
			{
				Name:         "method",
				Label:        "Method",
				Type:         FieldTypeSelect,
				Required:     true,
				DefaultValue: "GET",
				Description:  "HTTP method",
				Options: []FieldOption{
					{Value: "GET", Label: "GET"},
					{Value: "POST", Label: "POST"},
					{Value: "PUT", Label: "PUT"},
					{Value: "PATCH", Label: "PATCH"},
					{Value: "DELETE", Label: "DELETE"},
				},
			},
			{
				Name:        "url",
				Label:       "URL",
				Type:        FieldTypeURL,
				Required:    true,
				Description: "Request URL (supports {{variables}})",
				Placeholder: "https://api.example.com/users/{{trigger.body.user_id}}",
			},
			{
				Name:        "headers",
				Label:       "Headers",
				Type:        FieldTypeKeyValue,
				Required:    false,
				Description: "HTTP headers",
				Placeholder: "Authorization: Bearer {{token}}",
			},
			{
				Name:        "body",
				Label:       "Request Body",
				Type:        FieldTypeJSON,
				Required:    false,
				Description: "Request body (JSON)",
				Placeholder: `{"user_id": "{{trigger.body.user_id}}"}`,
			},
			{
				Name:         "timeout",
				Label:        "Timeout (seconds)",
				Type:         FieldTypeNumber,
				Required:     false,
				DefaultValue: 30,
				Description:  "Request timeout",
			},
			{
				Name:         "retry_on_failure",
				Label:        "Retry on Failure",
				Type:         FieldTypeBoolean,
				Required:     false,
				DefaultValue: false,
				Description:  "Automatically retry failed requests",
			},
			{
				Name:         "max_retries",
				Label:        "Max Retries",
				Type:         FieldTypeNumber,
				Required:     false,
				DefaultValue: 3,
				Description:  "Maximum retry attempts",
				DependsOn: &Dependency{
					Field: "retry_on_failure",
					Value: true,
				},
			},
		},
	}
}

// ============================================================================
// 3. SEND_MESSAGE Schema
// ============================================================================

func GetSendMessageSchema() NodeConfigSchema {
	return NodeConfigSchema{
		NodeType:    "SEND_MESSAGE",
		DisplayName: "Send Message",
		Description: "Send messages via channels (WhatsApp, SMS, etc.)",
		Icon:        "💬",
		Category:    "Communication",
		Fields: []FieldSchema{
			{
				Name:        "channel_id",
				Label:       "Channel ID",
				Type:        FieldTypeString,
				Required:    true,
				Description: "Channel to send through (or use {{trigger.body.channel_id}})",
				Placeholder: "{{trigger.body.channel_id}}",
			},
			{
				Name:        "recipient_id",
				Label:       "Recipient",
				Type:        FieldTypeString,
				Required:    true,
				Description: "Phone number or user ID",
				Placeholder: "+51987654321 or {{trigger.body.sender_id}}",
			},
			{
				Name:        "text",
				Label:       "Message Text",
				Type:        FieldTypeTextarea,
				Required:    true,
				Description: "Message content (supports {{variables}})",
				Placeholder: "Hello {{trigger.body.user_name}}, your order is ready!",
			},
			{
				Name:         "message_type",
				Label:        "Message Type",
				Type:         FieldTypeSelect,
				Required:     false,
				DefaultValue: "text",
				Description:  "Type of message",
				Options: []FieldOption{
					{Value: "text", Label: "Text"},
					{Value: "image", Label: "Image"},
					{Value: "document", Label: "Document"},
					{Value: "audio", Label: "Audio"},
					{Value: "video", Label: "Video"},
				},
			},
			{
				Name:        "attachments",
				Label:       "Attachments",
				Type:        FieldTypeArray,
				Required:    false,
				Description: "Media attachments (URLs or file paths)",
				Placeholder: "[{\"type\": \"image\", \"url\": \"https://...\"}]",
			},
		},
	}
}

// ============================================================================
// 4. TRANSFORM Schema
// ============================================================================

func GetTransformSchema() NodeConfigSchema {
	return NodeConfigSchema{
		NodeType:    "TRANSFORM",
		DisplayName: "Transform Data",
		Description: "Map and transform data fields",
		Icon:        "🔄",
		Category:    "Data",
		Fields: []FieldSchema{
			{
				Name:        "mappings",
				Label:       "Field Mappings",
				Type:        FieldTypeKeyValue,
				Required:    true,
				Description: "Map source fields to target fields",
				Placeholder: "user_name: {{trigger.body.name}}\nuser_email: {{trigger.body.email}}",
			},
		},
	}
}

// ============================================================================
// 5. CONDITION Schema
// ============================================================================

func GetConditionSchema() NodeConfigSchema {
	return NodeConfigSchema{
		NodeType:    "CONDITION",
		DisplayName: "Condition",
		Description: "Branch workflow based on conditions",
		Icon:        "🔀",
		Category:    "Logic",
		Fields: []FieldSchema{
			{
				Name:        "condition_type",
				Label:       "Condition Type",
				Type:        FieldTypeSelect,
				Required:    true,
				Description: "Type of condition to check",
				Options: []FieldOption{
					{Value: "equals", Label: "Equals", Description: "Check if values are equal"},
					{Value: "contains", Label: "Contains", Description: "Check if text contains substring"},
					{Value: "exists", Label: "Exists", Description: "Check if field exists"},
					{Value: "regex", Label: "Regex", Description: "Match regular expression"},
				},
			},
			{
				Name:        "field",
				Label:       "Field to Check",
				Type:        FieldTypeString,
				Required:    true,
				Description: "Field path (e.g., trigger.body.status)",
				Placeholder: "trigger.body.status",
			},
			{
				Name:        "value",
				Label:       "Expected Value",
				Type:        FieldTypeString,
				Required:    false,
				Description: "Value to compare against",
				Placeholder: "active",
			},
			{
				Name:         "case_insensitive",
				Label:        "Case Insensitive",
				Type:         FieldTypeBoolean,
				Required:     false,
				DefaultValue: false,
				Description:  "Ignore case when comparing",
			},
		},
	}
}

// ============================================================================
// 6. SWITCH Schema
// ============================================================================

func GetSwitchSchema() NodeConfigSchema {
	return NodeConfigSchema{
		NodeType:    "SWITCH",
		DisplayName: "Switch",
		Description: "Route to different nodes based on value",
		Icon:        "🎛️",
		Category:    "Logic",
		Fields: []FieldSchema{
			{
				Name:        "field",
				Label:       "Field to Evaluate",
				Type:        FieldTypeString,
				Required:    true,
				Description: "Field path to evaluate",
				Placeholder: "trigger.body.event_type",
			},
			{
				Name:        "cases",
				Label:       "Cases",
				Type:        FieldTypeKeyValue,
				Required:    true,
				Description: "Map values to node IDs (case_value: node_id)",
				Placeholder: "user.created: send_welcome\nuser.deleted: send_goodbye\ndefault: log_event",
			},
		},
	}
}

// ============================================================================
// 7. LOOP Schema
// ============================================================================

func GetLoopSchema() NodeConfigSchema {
	return NodeConfigSchema{
		NodeType:    "LOOP",
		DisplayName: "Loop",
		Description: "Iterate over arrays or collections",
		Icon:        "🔁",
		Category:    "Logic",
		Fields: []FieldSchema{
			{
				Name:        "iterate_over",
				Label:       "Collection",
				Type:        FieldTypeString,
				Required:    true,
				Description: "Array or collection to iterate",
				Placeholder: "trigger.body.items",
			},
			{
				Name:         "item_var",
				Label:        "Item Variable Name",
				Type:         FieldTypeString,
				Required:     false,
				DefaultValue: "item",
				Description:  "Variable name for current item",
			},
			{
				Name:         "index_var",
				Label:        "Index Variable Name",
				Type:         FieldTypeString,
				Required:     false,
				DefaultValue: "index",
				Description:  "Variable name for current index",
			},
			{
				Name:        "body_node",
				Label:       "Body Node ID",
				Type:        FieldTypeString,
				Required:    true,
				Description: "Node to execute for each item",
				Placeholder: "process_item",
			},
			{
				Name:         "max_iterations",
				Label:        "Max Iterations",
				Type:         FieldTypeNumber,
				Required:     false,
				DefaultValue: 1000,
				Description:  "Maximum number of iterations",
				Validation: &Validation{
					Min:     ptrx.Float32(1),
					Max:     ptrx.Float32(10000),
					Message: "Max iterations must be between 1 and 10000",
				},
			},
		},
	}
}

// ============================================================================
// 8. VALIDATE Schema
// ============================================================================

func GetValidateSchema() NodeConfigSchema {
	return NodeConfigSchema{
		NodeType:    "VALIDATE",
		DisplayName: "Validate Data",
		Description: "Validate input data against rules",
		Icon:        "✅",
		Category:    "Data",
		Fields: []FieldSchema{
			{
				Name:        "schema",
				Label:       "Validation Rules",
				Type:        FieldTypeKeyValue,
				Required:    true,
				Description: "Field validation rules (field: rule)",
				Placeholder: "email: required,email\nage: number,min:18\nname: required,string",
			},
			{
				Name:         "fail_on_error",
				Label:        "Fail on Validation Error",
				Type:         FieldTypeBoolean,
				Required:     false,
				DefaultValue: true,
				Description:  "Stop workflow if validation fails",
			},
		},
	}
}

// ============================================================================
// 9. DELAY Schema
// ============================================================================

func GetDelaySchema() NodeConfigSchema {
	return NodeConfigSchema{
		NodeType:    "DELAY",
		DisplayName: "Delay",
		Description: "Pause workflow execution",
		Icon:        "⏱️",
		Category:    "Control",
		Fields: []FieldSchema{
			{
				Name:        "duration",
				Label:       "Duration",
				Type:        FieldTypeString,
				Required:    false,
				Description: "Delay duration (e.g., 5s, 10m, 1h)",
				Placeholder: "5m",
			},
			{
				Name:        "duration_ms",
				Label:       "Duration (milliseconds)",
				Type:        FieldTypeNumber,
				Required:    false,
				Description: "Delay in milliseconds",
				Placeholder: "5000",
			},
			{
				Name:        "duration_seconds",
				Label:       "Duration (seconds)",
				Type:        FieldTypeNumber,
				Required:    false,
				Description: "Delay in seconds",
				Placeholder: "300",
			},
		},
	}
}

// ============================================================================
// 10. ACTION Schema
// ============================================================================

func GetActionSchema() NodeConfigSchema {
	return NodeConfigSchema{
		NodeType:    "ACTION",
		DisplayName: "Action",
		Description: "Execute custom actions",
		Icon:        "⚡",
		Category:    "Utility",
		Fields: []FieldSchema{
			{
				Name:        "action_type",
				Label:       "Action Type",
				Type:        FieldTypeSelect,
				Required:    true,
				Description: "Type of action to perform",
				Options: []FieldOption{
					{Value: "console_log", Label: "Console Log", Description: "Log to console"},
					{Value: "set_context", Label: "Set Context", Description: "Set workflow variables"},
				},
			},
			{
				Name:        "message",
				Label:       "Message",
				Type:        FieldTypeTextarea,
				Required:    false,
				Description: "Message to log (for console_log)",
				Placeholder: "Processing user: {{trigger.body.user_id}}",
				DependsOn: &Dependency{
					Field: "action_type",
					Value: "console_log",
				},
			},
			{
				Name:        "context",
				Label:       "Context Data",
				Type:        FieldTypeJSON,
				Required:    false,
				Description: "Variables to set (for set_context)",
				Placeholder: `{"user_id": "{{trigger.body.user_id}}"}`,
				DependsOn: &Dependency{
					Field: "action_type",
					Value: "set_context",
				},
			},
			{
				Name:         "print_input",
				Label:        "Print Input Data",
				Type:         FieldTypeBoolean,
				Required:     false,
				DefaultValue: false,
				Description:  "Also log input data",
				DependsOn: &Dependency{
					Field: "action_type",
					Value: "console_log",
				},
			},
		},
	}
}
