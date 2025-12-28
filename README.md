# MomLaunchpad Backend

A Go-based conversational backend for pregnancy support chatbot (MVP).

## ✅ Completed Components

### 1. Intent Classifier (TDD ✓)
- **Location:** `internal/classifier/`
- **Coverage:** 93.9%
- **Status:** Fully implemented with passing tests
- **Features:**
  - Rule-based classification (no LLM)
  - 5 intent types: small_talk, pregnancy_question, symptom_report, scheduling_related, unclear
  - Multilingual support (EN/ES)
  - 31 test cases passing

### 2. DeepSeek Client (TDD ✓)
- **Location:** `pkg/deepseek/`
- **Coverage:** 29.9% (mock: 100%, http client tested via integration)
- **Status:** Fully implemented with mock client
- **Features:**
  - Streaming chat completion (SSE)
  - Non-streaming chat completion
  - Mock client for testing
  - All tests passing

### 3. Memory Manager (TDD ✓)
- **Location:** `internal/memory/`
- **Coverage:** 85.5%
- **Status:** Fully implemented with passing tests
- **Features:**
  - Short-term memory (last N messages with size limit)
  - Long-term memory (facts with confidence scores)
  - Thread-safe concurrent access
  - Multi-user support
  - 10 test cases passing

### 4. Prompt Builder (TDD ✓)
- **Location:** `internal/prompt/`
- **Coverage:** 89.1%
- **Status:** Fully implemented with passing tests
- **Features:**
  - Super-prompt construction with user context
  - Pregnancy stage awareness
  - Multilingual system prompts
  - Small talk detection and filtering
  - Fact integration
  - 3 test cases passing

### 5. Calendar Suggester (TDD ✓)
- **Location:** `internal/calendar/`
- **Coverage:** 92.3%
- **Status:** Fully implemented with passing tests
- **Features:**
  - Intent-based reminder suggestions
  - Priority classification (urgent/high/medium/low)
  - Urgent keyword detection for symptoms
  - Automated suggestion building
  - 7 test cases passing

### 6. Database Layer
- **Location:** `internal/db/` + `migrations/`
- **Status:** Schema applied, queries implemented
- **Features:**
  - PostgreSQL schema with 6 tables (users, messages, user_facts, reminders, languages, savings_entries)
  - Connection pooling and lifecycle management
  - Models for all entities
  - CRUD queries implemented
  - Migrations applied successfully

### 7. Language Manager (TDD ✓)
- **Location:** `internal/language/`
- **Coverage:** 91.2%
- **Status:** Fully implemented with passing tests
- **Features:**
  - Language validation with fallback to English
  - Enable/disable language support
  - Thread-safe concurrent access
  - Default language protection (English cannot be disabled)
  - 5 test functions with 18 test cases

### 8. API Handlers
- **Location:** `internal/api/`
- **Status:** Core handlers implemented
- **Features:**
  - **Auth Handler**: Registration, login, JWT token generation
  - **Calendar Handler**: Reminder CRUD operations with ownership validation
  - **Middleware**: JWT authentication, CORS, admin-only access
  - Password hashing with bcrypt
  - Gin web framework integration
  - Database integration complete

## 🚧 Next Steps

1. WebSocket chat handler (`internal/ws/`)
2. Main server (`cmd/server/main.go`) with routing

## 📊 Test Coverage

```
✅ internal/calendar: 92.3% coverage (7 tests)
✅ internal/classifier: 93.9% coverage (31 tests)
✅ internal/language: 91.2% coverage (18 tests)
✅ internal/memory: 85.5% coverage (10 tests)
✅ internal/prompt: 89.1% coverage (3 tests)
✅ pkg/deepseek: 29.9% coverage (4 tests - mock focused)
```

**Total: 73 test cases, ALL PASS**
**Average coverage: 80.3%**

## Quick Start

```bash
# Run tests
make test

# Run specific package tests
go test -v ./internal/classifier/
go test -v ./pkg/deepseek/

# Setup environment
cp .env.example .env
# Edit .env with your configuration
```

## Project Structure

```
momlaunchpad-be/
├── cmd/server/           # 🚧 Entry point
├── internal/
│   ├── classifier/       # ✅ Intent classification (TDD) - 93.9%
│   ├── memory/           # ✅ Memory management (TDD) - 85.5%
│   ├── prompt/           # ✅ Prompt builder (TDD) - 89.1%
│   ├── calendar/         # ✅ Calendar suggestions (TDD) - 92.3%
│   ├── language/         # ✅ Language manager (TDD) - 91.2%
│   ├── api/              # ✅ HTTP handlers (auth, calendar, middleware)
│   ├── ws/               # 🚧 WebSocket
│   └── db/               # ✅ Database layer
├── pkg/
│   └── deepseek/         # ✅ DeepSeek client (TDD) - 29.9%
└── migrations/           # ✅ SQL migrations (applied)
```

## Testing

Following TDD methodology:
1. Write tests first
2. Implement to pass tests
3. Refactor
4. Commit

All components must have table-driven tests before implementation.

## Architecture

See [BACKEND_SPEC.md](BACKEND_SPEC.md) for complete architecture documentation.

**Key Principles:**
- Backend is the brain, AI is a dependency
- Determinism before intelligence
- Rule-based classifier runs first
- MVP discipline (no feature creep)

## Documentation

- [BACKEND_SPEC.md](BACKEND_SPEC.md) - Complete technical specification
- [.github/copilot-instructions.md](.github/copilot-instructions.md) - AI agent guidelines
- [.env.example](.env.example) - Environment configuration template

## License

TBD
