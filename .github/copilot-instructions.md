# MomLaunchpad Backend - AI Agent Instructions

## Project Overview
This is a **Go-based conversational backend** for a pregnancy support chatbot (MomLaunchpad MVP). The backend uses **rule-based intent classification** followed by AI-assisted responses via DeepSeek API. See [BACKEND_SPEC.md](../BACKEND_SPEC.md) for full architectural authority.

**Core Architecture:**
- HTTP API (auth, calendar, admin) + WebSocket (chat streaming only)
- PostgreSQL for persistent data, optional Redis cache
- DeepSeek API as the AI provider (not OpenAI)
- Single VM deployment (no microservices)

## Critical Design Principles

1. **Backend is the brain, AI is a dependency** - All logic lives in backend, LLM is just an API call
2. **Determinism before intelligence** - Rule-based classifier runs first, AI only when needed
3. **MVP discipline** - Excluded features: payments, analytics, vector search, tiny LLMs, microservices
4. **TDD-first** - Every domain must have table-driven tests before implementation

## Intent Classification Flow (CRITICAL)

**All chat messages must pass through the rule-based classifier first:**

```go
// Classifier is deterministic, NO LLM involved
type Intent string
const (
    IntentSmallTalk       Intent = "small_talk"
    IntentPregnancyQ      Intent = "pregnancy_question"
    IntentSymptom         Intent = "symptom_report"
    IntentScheduling      Intent = "scheduling_related"
    IntentUnclear         Intent = "unclear"
)

type ClassifierResult struct {
    Intent     Intent  `json:"intent"`
    Confidence float64 `json:"confidence"`
}
```

**Small talk NEVER triggers:**
- Memory loading
- Super-prompt construction
- Database writes
- AI calls

Use canned responses: `"I'm here with you. How can I help today?"`

## Super-Prompt Path (Pregnancy/Symptom Questions Only)

**Pipeline:**
1. Incoming message → Intent classifier
2. Load short-term memory (last 5-10 messages)
3. Load long-term facts (pregnancy_week, diet, etc.)
4. Build language-aware super-prompt
5. Call DeepSeek API
6. Stream response via WebSocket
7. Extract new facts (AI-assisted, backend-approved)
8. Suggest calendar action if applicable

**Super-prompt MUST include:**
- Pregnancy stage (if known)
- Relevant long-term facts
- Recent messages (only if relevant)
- Language-specific system instructions

**Super-prompt MUST NEVER include:**
- Small talk messages
- UX fillers
- Irrelevant history
- Raw admin configuration

## Transport Rules

**WebSocket** (`/ws/chat`):
- JWT validated on connection
- Stateless protocol, session-aware application logic
- For streaming AI responses and user messages only

**HTTP**:
- Auth endpoints
- Calendar CRUD operations
- Admin APIs (language management)

## Memory Model

**Short-term:** Last 5-10 messages (conversation continuity, not permanent)

**Long-term:** Extracted facts with confidence scores
```go
// Example structure
type UserFact struct {
    Key        string  `json:"key"`        // "pregnancy_week"
    Value      string  `json:"value"`      // "14"
    Confidence float64 `json:"confidence"` // 0.9
}
```

Memory extraction is AI-assisted but backend validates before storage.

## Database Schema (PostgreSQL)

Core tables: `users`, `messages`, `user_facts`, `reminders`, `languages`, `savings_entries` (optional)

**No ORMs required** - use `database/sql` or `pgx` directly.

## Calendar Intelligence

Backend **suggests** reminders, never auto-creates:
```json
{
  "type": "calendar",
  "message": "Would you like to set a reminder to monitor this symptom?"
}
```

Reminder creation requires explicit user confirmation via HTTP API.

## Multilingual Support

- Backend **never guesses** language - trust validated language codes only
- Unsupported languages fallback to English
- Admin can enable/disable languages or mark as experimental

## Security & Cost Controls

- JWT middleware on all authenticated routes
- Rate limiting on chat endpoints
- Cache repeated queries cautiously
- **Never cache sensitive medical responses long-term**
- Never expose LLM API keys

## Testing Requirements (MANDATORY)

Every domain needs:
- **Pure functions** (easy to test)
- **Table-driven tests** (Go idiom)
- **Mocked DeepSeek client**

Test categories:
- Intent classification (must be 100% deterministic)
- Prompt construction
- Memory extraction
- Language fallback logic
- Calendar suggestion logic

Example test structure:
```go
func TestIntentClassifier(t *testing.T) {
    tests := []struct {
        name       string
        input      string
        lang       string
        wantIntent Intent
        wantConf   float64
    }{
        {"hello greeting", "hello", "en", IntentSmallTalk, 0.9},
        {"pregnancy query", "when will baby kick?", "en", IntentPregnancyQ, 0.85},
    }
    // ... table-driven test loop
}
```

## Project Structure

```
momlaunchpad-be/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── internal/
│   ├── classifier/              # Rule-based intent classification
│   │   ├── classifier.go
│   │   └── classifier_test.go
│   ├── memory/                  # Short-term & long-term memory
│   │   ├── manager.go
│   │   └── manager_test.go
│   ├── prompt/                  # Super-prompt builder
│   │   ├── builder.go
│   │   └── builder_test.go
│   ├── calendar/                # Calendar suggestion engine
│   │   ├── suggester.go
│   │   └── suggester_test.go
│   ├── language/                # Multilingual manager
│   │   ├── manager.go
│   │   └── manager_test.go
│   ├── api/                     # HTTP handlers (Gin)
│   │   ├── auth.go
│   │   ├── calendar.go
│   │   ├── admin.go
│   │   └── middleware/
│   ├── ws/                      # WebSocket handler
│   │   └── chat.go
│   └── db/                      # Database layer
│       ├── postgres.go
│       └── queries.go
├── pkg/
│   └── deepseek/                # DeepSeek client
│       ├── client.go
│       ├── client_test.go
│       └── mock.go              # Mock for testing
├── migrations/                  # SQL migrations
│   ├── 001_init_schema.sql
│   └── ...
├── .env.example
├── go.mod
├── go.sum
├── Makefile
└── BACKEND_SPEC.md
```

## Framework: Gin

Use **Gin** for HTTP/WebSocket routing. Example setup:

```go
// cmd/server/main.go
router := gin.Default()
router.Use(middleware.CORS())

// HTTP routes
auth := router.Group("/api/auth")
calendar := router.Group("/api/calendar").Use(middleware.JWT())
admin := router.Group("/api/admin").Use(middleware.JWT(), middleware.AdminOnly())

// WebSocket
router.GET("/ws/chat", middleware.JWTWebSocket(), ws.HandleChat)
```

## DeepSeek Integration

**Client Setup:**
```go
// pkg/deepseek/client.go
type Client struct {
    apiKey  string
    baseURL string // https://api.deepseek.com/v1
    model   string // deepseek-chat
    httpClient *http.Client
}

func (c *Client) StreamChatCompletion(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error) {
    // Returns channel for streaming responses
}
```

**Model Selection:**
- Use `deepseek-chat` for conversational AI (primary model)
- Never use `deepseek-coder` (wrong use case)

**Streaming Implementation:**
```go
// Stream to WebSocket
chunks, err := deepseekClient.StreamChatCompletion(ctx, req)
for chunk := range chunks {
    wsConn.WriteJSON(map[string]interface{}{
        "type": "message",
        "content": chunk.Delta.Content,
    })
}
```

**Prompt Format:**
```go
type ChatMessage struct {
    Role    string `json:"role"`    // "system" | "user" | "assistant"
    Content string `json:"content"`
}

// System prompt MUST set context:
// - Pregnancy stage
// - Language instructions
// - Relevant facts
// User messages: normalized input
// Assistant messages: previous AI responses (if in memory window)
```

**Cost Control:**
- Max tokens: 1000 per response (adjust based on language)
- Temperature: 0.7 (deterministic enough, creative enough)
- No streaming for memory extraction (use regular completion)

## TDD Workflow (MANDATORY)

**Red-Green-Refactor:**
1. Write test first (table-driven)
2. Run test (should fail)
3. Implement minimal code to pass
4. Refactor
5. Commit

**Test Structure Example:**
```go
// internal/classifier/classifier_test.go
func TestClassifier_Classify(t *testing.T) {
    tests := []struct {
        name       string
        input      string
        lang       string
        wantIntent Intent
        minConf    float64
    }{
        {
            name:       "greeting in English",
            input:      "hello there",
            lang:       "en",
            wantIntent: IntentSmallTalk,
            minConf:    0.8,
        },
        {
            name:       "pregnancy question",
            input:      "when will my baby start kicking?",
            lang:       "en",
            wantIntent: IntentPregnancyQ,
            minConf:    0.7,
        },
        {
            name:       "symptom report",
            input:      "I'm experiencing nausea",
            lang:       "en",
            wantIntent: IntentSymptom,
            minConf:    0.7,
        },
    }
    
    classifier := NewClassifier()
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := classifier.Classify(tt.input, tt.lang)
            if result.Intent != tt.wantIntent {
                t.Errorf("got %v, want %v", result.Intent, tt.wantIntent)
            }
            if result.Confidence < tt.minConf {
                t.Errorf("confidence %v below threshold %v", result.Confidence, tt.minConf)
            }
        })
    }
}
```

**Mock DeepSeek Client:**
```go
// pkg/deepseek/mock.go
type MockClient struct {
    StreamFunc func(context.Context, ChatRequest) (<-chan ChatChunk, error)
}

func (m *MockClient) StreamChatCompletion(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error) {
    if m.StreamFunc != nil {
        return m.StreamFunc(ctx, req)
    }
    // Default mock behavior
    ch := make(chan ChatChunk, 1)
    ch <- ChatChunk{Delta: Delta{Content: "Mock response"}}
    close(ch)
    return ch, nil
}
```

**Run Tests:**
```bash
# All tests with race detection
make test

# Single package
go test -v ./internal/classifier/

# With coverage
make test-coverage
```

## What NOT to Build

- Payment integrations
- Automated savings/deductions
- Product recommendations
- Doctor dashboards
- Analytics pipelines
- Vector search (Qdrant)
- Tiny LLMs
- Microservice architecture

## Development Commands

```bash
# Initialize project
make init          # Install dependencies, run migrations

# Testing (TDD workflow)
make test          # Run all tests with race detection
make test-coverage # Generate coverage report
make test-watch    # Watch mode for TDD

# Development
make dev           # Run with hot reload (air)
make migrate-up    # Apply migrations
make migrate-down  # Rollback migrations

# Build & Deploy
make build         # Build binary
make docker-build  # Build Docker image
make run           # Run compiled binary
```

## Environment Variables

See [.env.example](.env.example) for required configuration:
- `DATABASE_URL` - PostgreSQL connection string
- `REDIS_URL` - Redis connection (optional)
- `DEEPSEEK_API_KEY` - DeepSeek API key
- `JWT_SECRET` - JWT signing secret
- `PORT` - Server port (default: 8080)

## Getting Started (TDD First)

1. **Write tests first** - Start with `internal/classifier/classifier_test.go`
2. **Implement classifier** - Rule-based, no LLM
3. **Mock DeepSeek** - Create `pkg/deepseek/mock.go` before real client
4. **Build super-prompt** - Test prompt builder with mocked AI responses
5. **Wire up handlers** - HTTP/WebSocket with JWT middleware
