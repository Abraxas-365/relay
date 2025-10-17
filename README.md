# 📘 Relay Workflow Engine - Complete Reference Guide

## Table of Contents
1. [Overview](#overview)
2. [Workflow Structure](#workflow-structure)
3. [Available Node Types](#available-node-types)
4. [Node Configuration Reference](#node-configuration-reference)
5. [Creating Workflows via API](#creating-workflows-via-api)
6. [Complete Workflow Examples](#complete-workflow-examples)
7. [Expression System](#expression-system)
8. [Best Practices](#best-practices)

---

## Overview

The Relay Workflow Engine is an n8n-style automation system that executes workflows composed of interconnected nodes. Each node performs a specific task and passes data to the next node.

**Key Features:**
- ✅ Visual workflow execution (node-based)
- ✅ Expression evaluation with CEL (Common Expression Language)
- ✅ Async delay scheduling
- ✅ AI agent integration with memory
- ✅ Multi-channel messaging
- ✅ HTTP/API integrations
- ✅ Data transformation and validation

---

## Workflow Structure

### Basic Workflow JSON Schema

```json
{
  "tenant_id": "uuid",
  "name": "Workflow Name",
  "description": "Optional description",
  "is_active": true,
  "trigger": {
    "type": "WEBHOOK|SCHEDULE|MANUAL|CHANNEL_WEBHOOK",
    "config": {},
    "filters": {}
  },
  "nodes": [
    {
      "id": "node_1",
      "name": "Node Name",
      "type": "ACTION|CONDITION|DELAY|AI_AGENT|SEND_MESSAGE|HTTP|TRANSFORM|SWITCH|LOOP|VALIDATE",
      "config": {},
      "on_success": "node_2",
      "on_failure": "error_node",
      "timeout": 30
    }
  ]
}
```

### Trigger Types

#### 1. **WEBHOOK** - HTTP endpoint trigger
```json
{
  "type": "WEBHOOK",
  "config": {
    "path": "/custom/path",
    "method": "POST"
  }
}
```

#### 2. **CHANNEL_WEBHOOK** - Message from channel
```json
{
  "type": "CHANNEL_WEBHOOK",
  "filters": {
    "channel_ids": ["channel-uuid-1", "channel-uuid-2"]
  }
}
```

#### 3. **SCHEDULE** - Cron-based trigger
```json
{
  "type": "SCHEDULE",
  "config": {
    "cron": "0 9 * * *",
    "timezone": "America/Lima"
  }
}
```

#### 4. **MANUAL** - Triggered manually via API
```json
{
  "type": "MANUAL"
}
```

---

## Available Node Types

| Node Type | Description | Use Case |
|-----------|-------------|----------|
| **ACTION** | Execute simple actions | Console logging, set context |
| **CONDITION** | Conditional branching | If/else logic |
| **DELAY** | Wait for duration | Rate limiting, scheduling |
| **AI_AGENT** | LLM interaction with memory | Chatbots, AI responses |
| **SEND_MESSAGE** | Send message via channel | WhatsApp, notifications |
| **HTTP** | Make HTTP API calls | External integrations |
| **TRANSFORM** | Transform/map data | Data reshaping |
| **SWITCH** | Multi-branch routing | Route by field value |
| **LOOP** | Iterate over collections | Batch processing |
| **VALIDATE** | Data validation | Input verification |

---

## Node Configuration Reference

### 1. ACTION Node

Performs simple actions like logging or setting context variables.

```json
{
  "id": "action_1",
  "name": "Log Message",
  "type": "ACTION",
  "config": {
    "action_type": "console_log",
    "message": "User {{trigger.sender_id}} sent: {{trigger.text}}",
    "print_input": false
  },
  "on_success": "next_node"
}
```

**Action Types:**
- `console_log`: Print to console
- `set_context`: Store variables in context

**Set Context Example:**
```json
{
  "action_type": "set_context",
  "context": {
    "user_id": "{{trigger.sender_id}}",
    "message_text": "{{trigger.text}}",
    "timestamp": "{{trigger.timestamp}}"
  }
}
```

---

### 2. CONDITION Node

Branch workflow based on conditions.

```json
{
  "id": "condition_1",
  "name": "Check User Type",
  "type": "CONDITION",
  "config": {
    "condition_type": "contains",
    "field": "trigger.text",
    "value": "help",
    "case_insensitive": true
  },
  "on_success": "help_node",
  "on_failure": "default_node"
}
```

**Condition Types:**

| Type | Description | Config |
|------|-------------|--------|
| `contains` | Check if string contains value | `field`, `value`, `case_insensitive` |
| `equals` | Check equality | `field`, `value` |
| `exists` | Check if field exists | `field` |
| `regex` | Regex matching | `field`, `pattern` |

**Examples:**

```json
// Equals condition
{
  "condition_type": "equals",
  "field": "trigger.message_type",
  "value": "text"
}

// Exists condition
{
  "condition_type": "exists",
  "field": "trigger.attachments"
}
```

---

### 3. DELAY Node

Wait for a specified duration (sync or async).

```json
{
  "id": "delay_1",
  "name": "Wait 5 Seconds",
  "type": "DELAY",
  "config": {
    "duration_ms": 5000
  },
  "on_success": "next_node"
}
```

**Duration Formats:**

```json
// Milliseconds
{"duration_ms": 5000}

// Duration string
{"duration": "5s"}
{"duration": "2m"}
{"duration": "1h"}

// Seconds
{"duration_seconds": 30}
```

**Behavior:**
- **< 30s**: Synchronous (blocks workflow)
- **> 30s**: Asynchronous (pauses and resumes via Redis scheduler)

---

### 4. AI_AGENT Node

Interact with LLMs (OpenAI, etc.) with optional conversation memory.

```json
{
  "id": "ai_agent_1",
  "name": "ChatGPT Assistant",
  "type": "AI_AGENT",
  "config": {
    "provider": "openai",
    "model": "gpt-4",
    "system_prompt": "You are a helpful customer support assistant. Be concise and friendly.",
    "temperature": 0.7,
    "max_tokens": 500,
    "use_memory": true,
    "max_auto_iterations": 3,
    "max_total_iterations": 10
  },
  "on_success": "send_response"
}
```

**Configuration Options:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `provider` | string | ✅ | `openai`, `anthropic`, `gemini` |
| `model` | string | ✅ | Model ID (e.g., `gpt-4`) |
| `system_prompt` | string | ✅ | System instruction |
| `temperature` | float | ❌ | 0.0-2.0 (default: 0.7) |
| `max_tokens` | int | ❌ | Max response tokens (default: 1000) |
| `use_memory` | bool | ❌ | Enable conversation memory |
| `tools` | array | ❌ | Tool IDs to use |
| `timeout` | int | ❌ | Timeout in seconds (default: 60) |

**With Memory (Persistent Conversation):**
```json
{
  "use_memory": true
}
```
Requires `conversation_id` in trigger data (e.g., `sender_id` for WhatsApp).

**Without Memory (Stateless):**
```json
{
  "use_memory": false
}
```
Each call is independent.

---

### 5. SEND_MESSAGE Node

Send messages through channels (WhatsApp, etc.).

```json
{
  "id": "send_msg_1",
  "name": "Send WhatsApp Reply",
  "type": "SEND_MESSAGE",
  "config": {
    "channel_id": "{{trigger.channel_id}}",
    "recipient_id": "{{trigger.sender_id}}",
    "text": "{{ai_agent_1.output.ai_response}}"
  },
  "on_success": "end"
}
```

**With Attachments:**
```json
{
  "channel_id": "{{trigger.channel_id}}",
  "recipient_id": "{{trigger.sender_id}}",
  "text": "Here's your document",
  "attachments": [
    {
      "type": "document",
      "url": "https://example.com/file.pdf",
      "filename": "invoice.pdf",
      "mime_type": "application/pdf"
    }
  ]
}
```

---

### 6. HTTP Node

Make HTTP requests to external APIs.

```json
{
  "id": "http_1",
  "name": "Fetch User Data",
  "type": "HTTP",
  "config": {
    "method": "GET",
    "url": "https://api.example.com/users/{{trigger.sender_id}}",
    "headers": {
      "Authorization": "Bearer YOUR_API_KEY",
      "Content-Type": "application/json"
    },
    "timeout": 30,
    "success_codes": [200, 201],
    "retry_on_failure": true,
    "max_retries": 3
  },
  "on_success": "process_response",
  "on_failure": "error_handler"
}
```

**POST Request with Body:**
```json
{
  "method": "POST",
  "url": "https://api.example.com/orders",
  "headers": {
    "Authorization": "Bearer TOKEN",
    "Content-Type": "application/json"
  },
  "body": {
    "user_id": "{{trigger.sender_id}}",
    "product": "{{condition_1.output.product}}",
    "quantity": 1
  }
}
```

**HTTP Methods:** GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS

---

### 7. TRANSFORM Node

Transform and map data between nodes.

```json
{
  "id": "transform_1",
  "name": "Map User Data",
  "type": "TRANSFORM",
  "config": {
    "mappings": {
      "user_name": "{{http_1.output.json.name}}",
      "user_email": "{{http_1.output.json.email}}",
      "message_count": "{{http_1.output.json.messages.length}}",
      "full_info": "Name: {{http_1.output.json.name}}, Email: {{http_1.output.json.email}}"
    }
  },
  "on_success": "next_node"
}
```

**Output:**
```json
{
  "user_name": "John Doe",
  "user_email": "john@example.com",
  "message_count": 5,
  "full_info": "Name: John Doe, Email: john@example.com"
}
```

---

### 8. SWITCH Node

Route to different nodes based on field value.

```json
{
  "id": "switch_1",
  "name": "Route by Message Type",
  "type": "SWITCH",
  "config": {
    "field": "trigger.text",
    "cases": {
      "help": "help_node",
      "order": "order_node",
      "cancel": "cancel_node",
      "default": "fallback_node"
    }
  }
}
```

**Field Evaluation:**
The switch extracts the field value and matches it against cases (string comparison).

**Example with Nested Field:**
```json
{
  "field": "http_1.output.json.user.role",
  "cases": {
    "admin": "admin_flow",
    "user": "user_flow",
    "guest": "guest_flow",
    "default": "unknown_flow"
  }
}
```

---

### 9. LOOP Node

Iterate over arrays/collections.

```json
{
  "id": "loop_1",
  "name": "Process Orders",
  "type": "LOOP",
  "config": {
    "iterate_over": "http_1.output.json.orders",
    "item_var": "order",
    "index_var": "index",
    "body_node": "process_order_node",
    "max_iterations": 100
  },
  "on_success": "summary_node"
}
```

**Configuration:**
- `iterate_over`: Path to array (supports expressions)
- `item_var`: Variable name for current item (default: `item`)
- `index_var`: Variable name for index (optional)
- `body_node`: Node to execute for each item
- `max_iterations`: Safety limit (default: 1000, max: 10000)

---

### 10. VALIDATE Node

Validate data against rules.

```json
{
  "id": "validate_1",
  "name": "Validate Order",
  "type": "VALIDATE",
  "config": {
    "schema": {
      "trigger.sender_id": "required",
      "trigger.text": "required",
      "http_1.output.json.email": "email",
      "http_1.output.json.age": "number"
    },
    "fail_on_error": true
  },
  "on_success": "process_node",
  "on_failure": "validation_error_node"
}
```

**Validation Rules:**

| Rule | Description | Example |
|------|-------------|---------|
| `required` | Field must exist and not be empty | `"field": "required"` |
| `email` | Valid email format | `"email_field": "email"` |
| `number` | Must be numeric | `"age": "number"` |
| `string` | Must be string | `"name": "string"` |
| `url` | Valid URL | `"website": "url"` |

**Multiple Rules (Comma-Separated):**
```json
{
  "schema": {
    "email": "required,email",
    "age": "required,number"
  }
}
```

---

## Creating Workflows via API

### Endpoint

```http
POST /api/workflows
Authorization: Bearer {token}
Content-Type: application/json
```

### Request Body Structure

```json
{
  "tenant_id": "your-tenant-uuid",
  "name": "Workflow Name",
  "description": "Workflow description",
  "trigger": {
    "type": "TRIGGER_TYPE",
    "config": {},
    "filters": {}
  },
  "nodes": []
}
```

### cURL Example

```bash
curl -X POST https://your-api.com/api/workflows \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "tenant-123",
    "name": "Simple Echo Bot",
    "description": "Echoes user messages",
    "trigger": {
      "type": "CHANNEL_WEBHOOK",
      "filters": {
        "channel_ids": ["channel-456"]
      }
    },
    "nodes": [
      {
        "id": "log_1",
        "name": "Log Message",
        "type": "ACTION",
        "config": {
          "action_type": "console_log",
          "message": "Received: {{trigger.text}}"
        },
        "on_success": "send_1"
      },
      {
        "id": "send_1",
        "name": "Echo Back",
        "type": "SEND_MESSAGE",
        "config": {
          "channel_id": "{{trigger.channel_id}}",
          "recipient_id": "{{trigger.sender_id}}",
          "text": "You said: {{trigger.text}}"
        }
      }
    ]
  }'
```

---

## Complete Workflow Examples

### Example 1: Simple Echo Bot

**Description:** Receives WhatsApp message and echoes it back.

```json
{
  "tenant_id": "tenant-123",
  "name": "Echo Bot",
  "description": "Simple echo bot for WhatsApp",
  "trigger": {
    "type": "CHANNEL_WEBHOOK",
    "filters": {
      "channel_ids": ["whatsapp-channel-id"]
    }
  },
  "nodes": [
    {
      "id": "send_reply",
      "name": "Send Echo",
      "type": "SEND_MESSAGE",
      "config": {
        "channel_id": "{{trigger.channel_id}}",
        "recipient_id": "{{trigger.sender_id}}",
        "text": "You said: {{trigger.text}}"
      }
    }
  ]
}
```

---

### Example 2: AI Customer Support Bot

**Description:** AI-powered support bot with memory.

```json
{
  "tenant_id": "tenant-123",
  "name": "AI Support Bot",
  "description": "Customer support with GPT-4",
  "trigger": {
    "type": "CHANNEL_WEBHOOK",
    "filters": {
      "channel_ids": ["whatsapp-channel-id"]
    }
  },
  "nodes": [
    {
      "id": "log_message",
      "name": "Log Incoming",
      "type": "ACTION",
      "config": {
        "action_type": "console_log",
        "message": "Customer {{trigger.sender_id}}: {{trigger.text}}"
      },
      "on_success": "ai_agent"
    },
    {
      "id": "ai_agent",
      "name": "GPT-4 Agent",
      "type": "AI_AGENT",
      "config": {
        "provider": "openai",
        "model": "gpt-4",
        "system_prompt": "You are a helpful customer support agent. Provide clear, concise answers. If you don't know something, say so.",
        "temperature": 0.7,
        "max_tokens": 300,
        "use_memory": true
      },
      "on_success": "send_response",
      "on_failure": "error_message"
    },
    {
      "id": "send_response",
      "name": "Send AI Response",
      "type": "SEND_MESSAGE",
      "config": {
        "channel_id": "{{trigger.channel_id}}",
        "recipient_id": "{{trigger.sender_id}}",
        "text": "{{ai_agent.output.ai_response}}"
      }
    },
    {
      "id": "error_message",
      "name": "Send Error",
      "type": "SEND_MESSAGE",
      "config": {
        "channel_id": "{{trigger.channel_id}}",
        "recipient_id": "{{trigger.sender_id}}",
        "text": "Sorry, I'm experiencing technical difficulties. Please try again later."
      }
    }
  ]
}
```

---

### Example 3: Conditional Routing with API Call

**Description:** Route based on keyword, call API, transform data, send response.

```json
{
  "tenant_id": "tenant-123",
  "name": "Order Status Checker",
  "description": "Check order status via API",
  "trigger": {
    "type": "CHANNEL_WEBHOOK",
    "filters": {
      "channel_ids": ["whatsapp-channel-id"]
    }
  },
  "nodes": [
    {
      "id": "check_keyword",
      "name": "Check for 'order' keyword",
      "type": "CONDITION",
      "config": {
        "condition_type": "contains",
        "field": "trigger.text",
        "value": "order",
        "case_insensitive": true
      },
      "on_success": "extract_order_id",
      "on_failure": "default_response"
    },
    {
      "id": "extract_order_id",
      "name": "Extract Order ID",
      "type": "TRANSFORM",
      "config": {
        "mappings": {
          "order_id": "{{trigger.text}}",
          "user_id": "{{trigger.sender_id}}"
        }
      },
      "on_success": "fetch_order"
    },
    {
      "id": "fetch_order",
      "name": "Fetch Order from API",
      "type": "HTTP",
      "config": {
        "method": "GET",
        "url": "https://api.mystore.com/orders/{{extract_order_id.output.order_id}}",
        "headers": {
          "Authorization": "Bearer YOUR_API_KEY"
        },
        "timeout": 10,
        "success_codes": [200]
      },
      "on_success": "validate_response",
      "on_failure": "order_not_found"
    },
    {
      "id": "validate_response",
      "name": "Validate Order Data",
      "type": "VALIDATE",
      "config": {
        "schema": {
          "fetch_order.output.json.status": "required",
          "fetch_order.output.json.order_id": "required"
        },
        "fail_on_error": true
      },
      "on_success": "format_response",
      "on_failure": "order_not_found"
    },
    {
      "id": "format_response",
      "name": "Format Order Info",
      "type": "TRANSFORM",
      "config": {
        "mappings": {
          "message": "Order #{{fetch_order.output.json.order_id}} is {{fetch_order.output.json.status}}. Estimated delivery: {{fetch_order.output.json.delivery_date}}"
        }
      },
      "on_success": "send_order_status"
    },
    {
      "id": "send_order_status",
      "name": "Send Order Status",
      "type": "SEND_MESSAGE",
      "config": {
        "channel_id": "{{trigger.channel_id}}",
        "recipient_id": "{{trigger.sender_id}}",
        "text": "{{format_response.output.message}}"
      }
    },
    {
      "id": "order_not_found",
      "name": "Order Not Found",
      "type": "SEND_MESSAGE",
      "config": {
        "channel_id": "{{trigger.channel_id}}",
        "recipient_id": "{{trigger.sender_id}}",
        "text": "Sorry, I couldn't find that order. Please check the order number and try again."
      }
    },
    {
      "id": "default_response",
      "name": "Default Response",
      "type": "SEND_MESSAGE",
      "config": {
        "channel_id": "{{trigger.channel_id}}",
        "recipient_id": "{{trigger.sender_id}}",
        "text": "Hi! Send 'order [number]' to check your order status."
      }
    }
  ]
}
```

---

### Example 4: Multi-Step Survey with Delays

**Description:** Send survey questions with delays between each.

```json
{
  "tenant_id": "tenant-123",
  "name": "Customer Satisfaction Survey",
  "description": "Multi-step survey with delays",
  "trigger": {
    "type": "MANUAL"
  },
  "nodes": [
    {
      "id": "question_1",
      "name": "Ask Question 1",
      "type": "SEND_MESSAGE",
      "config": {
        "channel_id": "{{trigger.channel_id}}",
        "recipient_id": "{{trigger.user_id}}",
        "text": "How satisfied are you with our service? (1-5)"
      },
      "on_success": "delay_1"
    },
    {
      "id": "delay_1",
      "name": "Wait 5 seconds",
      "type": "DELAY",
      "config": {
        "duration": "5s"
      },
      "on_success": "question_2"
    },
    {
      "id": "question_2",
      "name": "Ask Question 2",
      "type": "SEND_MESSAGE",
      "config": {
        "channel_id": "{{trigger.channel_id}}",
        "recipient_id": "{{trigger.user_id}}",
        "text": "Would you recommend us to others? (yes/no)"
      },
      "on_success": "delay_2"
    },
    {
      "id": "delay_2",
      "name": "Wait 5 seconds",
      "type": "DELAY",
      "config": {
        "duration": "5s"
      },
      "on_success": "thank_you"
    },
    {
      "id": "thank_you",
      "name": "Thank You",
      "type": "SEND_MESSAGE",
      "config": {
        "channel_id": "{{trigger.channel_id}}",
        "recipient_id": "{{trigger.user_id}}",
        "text": "Thank you for your feedback! 🙏"
      }
    }
  ]
}
```

---

### Example 5: Switch-Based Router

**Description:** Route to different flows based on user command.

```json
{
  "tenant_id": "tenant-123",
  "name": "Command Router",
  "description": "Routes based on user command",
  "trigger": {
    "type": "CHANNEL_WEBHOOK",
    "filters": {
      "channel_ids": ["whatsapp-channel-id"]
    }
  },
  "nodes": [
    {
      "id": "router",
      "name": "Route Command",
      "type": "SWITCH",
      "config": {
        "field": "trigger.text",
        "cases": {
          "help": "help_flow",
          "order": "order_flow",
          "support": "support_flow",
          "cancel": "cancel_flow",
          "default": "unknown_command"
        }
      }
    },
    {
      "id": "help_flow",
      "name": "Help Message",
      "type": "SEND_MESSAGE",
      "config": {
        "channel_id": "{{trigger.channel_id}}",
        "recipient_id": "{{trigger.sender_id}}",
        "text": "📋 Available commands:\n- order: Check order status\n- support: Contact support\n- cancel: Cancel order"
      }
    },
    {
      "id": "order_flow",
      "name": "Order Flow",
      "type": "SEND_MESSAGE",
      "config": {
        "channel_id": "{{trigger.channel_id}}",
        "recipient_id": "{{trigger.sender_id}}",
        "text": "📦 Please send your order number."
      }
    },
    {
      "id": "support_flow",
      "name": "Support AI Agent",
      "type": "AI_AGENT",
      "config": {
        "provider": "openai",
        "model": "gpt-4",
        "system_prompt": "You are a customer support agent. Help the user with their issue.",
        "use_memory": true
      },
      "on_success": "send_support_response"
    },
    {
      "id": "send_support_response",
      "name": "Send Support Response",
      "type": "SEND_MESSAGE",
      "config": {
        "channel_id": "{{trigger.channel_id}}",
        "recipient_id": "{{trigger.sender_id}}",
        "text": "{{support_flow.output.ai_response}}"
      }
    },
    {
      "id": "cancel_flow",
      "name": "Cancel Confirmation",
      "type": "SEND_MESSAGE",
      "config": {
        "channel_id": "{{trigger.channel_id}}",
        "recipient_id": "{{trigger.sender_id}}",
        "text": "❌ To cancel your order, please reply with your order number."
      }
    },
    {
      "id": "unknown_command",
      "name": "Unknown Command",
      "type": "SEND_MESSAGE",
      "config": {
        "channel_id": "{{trigger.channel_id}}",
        "recipient_id": "{{trigger.sender_id}}",
        "text": "❓ I didn't understand that. Type 'help' for available commands."
      }
    }
  ]
}
```

---

## Expression System

Relay uses **CEL (Common Expression Language)** for expressions in `{{...}}` syntax.

### Accessing Trigger Data

```json
"{{trigger.text}}"           // Message text
"{{trigger.sender_id}}"      // Sender ID
"{{trigger.channel_id}}"     // Channel ID
"{{trigger.message_id}}"     // Message ID
"{{trigger.attachments}}"    // Attachments array
```

### Accessing Node Output

```json
"{{node_id.output.field}}"                    // Node output field
"{{http_1.output.json.user.name}}"           // Nested JSON field
"{{ai_agent_1.output.ai_response}}"          // AI response
"{{transform_1.output.user_email}}"          // Transformed data
```

### String Operations

```json
"Hello {{trigger.sender_id}}"                 // Concatenation
"{{trigger.text}} - processed"                // Append
"Name: {{http_1.output.json.name}}, Age: {{http_1.output.json.age}}"  // Multiple
```

### Conditional Expressions

```json
"{{trigger.text == 'hello' ? 'Hi there!' : 'Hello!'}}"
"{{http_1.output.status_code == 200 ? 'Success' : 'Failed'}}"
```

### Array/Object Access

```json
"{{http_1.output.json.users[0].name}}"       // First user
"{{trigger.attachments.length}}"             // Array length
"{{http_1.output.json.items.size()}}"        // Array size (CEL)
```

---

## Best Practices

### 1. **Node Naming**
- Use descriptive names: `fetch_user_data` not `http_1`
- Include action in name: `send_welcome_message`, `validate_order`

### 2. **Error Handling**
- Always define `on_failure` for critical nodes
- Create dedicated error handler nodes
- Use validation nodes before API calls

### 3. **Timeout Configuration**
- Set appropriate timeouts for HTTP nodes (10-30s)
- AI agents should have 60s+ timeout
- Delays should use async for > 30s

### 4. **Expression Safety**
- Always check field existence before accessing
- Use `{{field ? field : 'default'}}` for optional fields
- Test expressions with various inputs

### 5. **Performance**
- Minimize HTTP calls in loops
- Use transforms to extract only needed data
- Set reasonable `max_iterations` for loops

### 6. **Memory Usage (AI Agents)**
- Use `use_memory: true` for conversational bots
- Use `use_memory: false` for one-off queries
- Ensure `conversation_id` is available in trigger

### 7. **Workflow Organization**
```
1. Trigger → 2. Log/Validate → 3. Process → 4. Transform → 5. Send/Store
```

### 8. **Testing**
- Test with manual trigger first
- Test all conditional branches
- Verify error handling paths
- Test with missing/invalid data

---

## Common Patterns

### Pattern 1: Log → Process → Respond
```
[Log] → [AI/HTTP/Transform] → [Send Message]
```

### Pattern 2: Validate → Branch → Process
```
[Validate] → [Condition] → [Success Path]
                        → [Failure Path]
```

### Pattern 3: API Call → Transform → Send
```
[HTTP] → [Transform] → [Validate] → [Send Message]
```

### Pattern 4: Multi-Step Conditional
```
[Condition 1] → [Condition 2] → [Action A]
             ↘              ↘ → [Action B]
              → [Default Action]
```

---

## Quick Reference Card

### Minimal Working Workflow
```json
{
  "tenant_id": "your-tenant",
  "name": "Minimal Bot",
  "trigger": {
    "type": "CHANNEL_WEBHOOK",
    "filters": {"channel_ids": ["channel-id"]}
  },
  "nodes": [
    {
      "id": "reply",
      "type": "SEND_MESSAGE",
      "config": {
        "channel_id": "{{trigger.channel_id}}",
        "recipient_id": "{{trigger.sender_id}}",
        "text": "Hello!"
      }
    }
  ]
}
```

### Common Expression Patterns
```
Trigger data:        {{trigger.field}}
Node output:         {{node_id.output.field}}
Nested:              {{node_id.output.json.nested.field}}
Conditional:         {{condition ? true_value : false_value}}
String concat:       "Text {{variable}} more text"
Default value:       {{field ? field : 'default'}}
```

---

## Need Help?

- Check logs for node execution results
- Use ACTION node with `console_log` for debugging
- Test expressions in TRANSFORM nodes
- Start simple, add complexity incrementally

**Happy Workflow Building! 🚀**
