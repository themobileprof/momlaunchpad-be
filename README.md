# MomLaunchpad Backend

A Go-based conversational backend for pregnancy support chatbot (MVP).

## ✅ Project Status: COMPLETE

All core components have been implemented following TDD methodology. The server is ready to run.

## 🎯 Completed Components

### 1. Intent Classifier (TDD ✓)
- **Location:** `internal/classifier/`
- **Coverage:** 93.9%
- **Features:**
  - Rule-based classification (no LLM)
  - 5 intent types: small_talk, pregnancy_question, symptom_report, scheduling_related, unclear
  - Multilingual support (EN/ES)
  - 31 test cases passing

### 2. DeepSeek Client (TDD ✓)
- **Location:** `pkg/deepseek/`
- **Coverage:** 29.9% (mock: 100%, http client tested via integration)
- **Features:**
  - Streaming chat completion (SSE)
  - Non-streaming chat completion
  - Mock client for testing
  - Config-based initialization

### 3. Memory Manager (TDD ✓)
- **Location:** `internal/memory/`
- **Coverage:** 85.5%
- **Features:**
  - Short-term memory (last N messages with size limit)
  - Long-term memory (facts with confidence scores)
  - Thread-safe concurrent access
  - 10 test cases passing

### 4. Prompt Builder (TDD ✓)
- **Location:** `internal/prompt/`
- **Coverage:** 89.1%
- **Features:**
  - Super-prompt construction with user context
  - Pregnancy stage awareness
  - Multilingual system prompts
  - Small talk detection and filtering
  - 3 test cases passing

### 5. Calendar Suggester (TDD ✓)
- **Location:** `internal/calendar/`
- **Coverage:** 92.3%
- **Features:**
  - Intent-based reminder suggestions
  - Priority classification (urgent/high/medium/low)
  - Urgent keyword detection for symptoms
  - 7 test cases passing

### 6. Database Layer ✓
- **Location:** `internal/db/` + `migrations/`
- **Features:**
  - PostgreSQL schema with 6 tables
  - Connection pooling and lifecycle management
  - Complete CRUD operations
  - Migrations applied successfully

### 7. Language Manager (TDD ✓)
- **Location:** `internal/language/`
- **Coverage:** 91.2%
- **Features:**
  - Language validation with fallback to English
  - Enable/disable language support
  - Thread-safe concurrent access
  - 18 test cases passing

### 8. API Handlers ✓
- **Location:** `internal/api/`
- **Features:**
  - **Auth Handler**: Registration, login, JWT token generation (7 day expiry)
  - **OAuth Handler**: Google Sign-In (web + mobile), Apple Sign-In (coming soon)
  - **Calendar Handler**: Reminder CRUD operations with ownership validation
  - **Middleware**: JWT authentication, CORS, admin-only access
  - Password hashing with bcrypt
  - Multi-platform OAuth support (web, Android, iOS)
  - Email-based account linking across providers
  - Gin web framework integration

### 9. WebSocket Chat Handler ✓
- **Location:** `internal/ws/`
- **Features:**
  - JWT-authenticated WebSocket connections
  - Real-time chat streaming
  - Pipeline: Classify → Load memory → Build prompt → Stream DeepSeek → Extract facts → Suggest reminders
  - Small talk handled without AI
  - Calendar suggestion integration
  - Fact extraction and persistence

### 10. Main Server ✓
- **Location:** `cmd/server/main.go`
- **Features:**
  - Dependency injection and initialization
  - Gin router with all endpoints
  - Graceful shutdown support
  - Environment variable configuration
  - Health check endpoint

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
## 🚀 Quick Start

### Local Development

```bash
# 1. Configure environment
cp .env.example .env
# Edit .env with your database credentials and API keys

# 2. Apply migrations
make migrate-up

# 3. Run the server
make run
# or for development with hot reload:
make dev
```

### Docker Development

```bash
# 1. Setup environment
cp .env.docker .env
# Edit .env with your API keys

# 2. Start all services
docker-compose up -d

# 3. View logs
docker-compose logs -f backend
```

See [DOCKER.md](DOCKER.md) for complete Docker documentation (development only).

### Production Deployment

Production uses **GitHub Actions CI/CD pipelines** - no manual deployments.

```bash
# Deploy by pushing a version tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# GitHub Actions automatically:
# - Builds Docker image
# - Runs security scans
# - Deploys to production server
# - Runs database migrations
# - Verifies health checks
```

See [.github/CICD.md](.github/CICD.md) for complete CI/CD documentation.

**Server will start on `http://localhost:8080`**

## 📋 API Endpoints

### Authentication (Public)
- `POST /api/auth/register` - Register new user
- `POST /api/auth/login` - Login and get JWT token
- `GET /api/auth/me` - Get current user (protected)

### Calendar (Protected)
- `GET /api/reminders` - Get user's reminders
- `POST /api/reminders` - Create reminder
- `PUT /api/reminders/:id` - Update reminder
- `DELETE /api/reminders/:id` - Delete reminder

### WebSocket (Protected)
- `WS /ws/chat` - Real-time chat with AI streaming

### Health Check
- `GET /health` - Server health status

## 📁 Project Structure

```
momlaunchpad-be/
├── cmd/
│   └── server/           # ✅ Main server entry point
├── internal/
│   ├── classifier/       # ✅ Intent classification (TDD) - 93.9%
│   ├── memory/           # ✅ Memory manager (TDD) - 85.5%
│   ├── prompt/           # ✅ Prompt builder (TDD) - 89.1%
│   ├── calendar/         # ✅ Calendar suggestions (TDD) - 92.3%
│   ├── language/         # ✅ Language manager (TDD) - 91.2%
│   ├── api/              # ✅ HTTP handlers (auth, calendar, middleware)
│   ├── ws/               # ✅ WebSocket chat handler
│   └── db/               # ✅ Database layer
├── pkg/
│   └── deepseek/         # ✅ DeepSeek client (TDD) - 29.9%
└── migrations/           # ✅ SQL migrations (applied)
```

## 🧪 Testing

Following TDD methodology:
1. Write tests first
2. Implement to pass tests
3. Refactor
4. Commit

All components have table-driven tests.

**Run tests:**
```bash
make test          # Run all tests with race detection
make test-coverage # Generate coverage report
```

**Current Coverage:**
- Intent Classifier: 93.9%
- Calendar Suggester: 92.3%
- Language Manager: 91.2%
- Prompt Builder: 89.1%
- Memory Manager: 85.5%

## 📐 Architecture

See [BACKEND_SPEC.md](BACKEND_SPEC.md) for complete architecture documentation.

**Key Principles:**
- Backend is the brain, AI is a dependency
- Determinism before intelligence
- Rule-based classifier runs first
- MVP discipline (no feature creep)

**Chat Flow:**
1. Incoming message → Intent classifier (rule-based)
2. Small talk → Canned response (no AI)
3. Pregnancy/symptom → Load memory → Build super-prompt → DeepSeek streaming → Extract facts → Suggest reminders

## 📚 Documentation

- [API.md](API.md) - **Complete API documentation with OAuth examples**
- [.github/CICD.md](.github/CICD.md) - CI/CD pipelines & production deployment
- [DOCKER.md](DOCKER.md) - Docker for local development
- [BACKEND_SPEC.md](BACKEND_SPEC.md) - Complete technical specification
- [WEBSOCKET_GUIDE.md](WEBSOCKET_GUIDE.md) - Flutter WebSocket integration
- [PRODUCTION_FEATURES.md](PRODUCTION_FEATURES.md) - Production readiness features
- [QUICKSTART.md](QUICKSTART.md) - Quick start with examples
- [.github/copilot-instructions.md](.github/copilot-instructions.md) - AI agent guidelines
- [.env.example](.env.example) - Environment configuration template

## 🔐 OAuth Implementation

### Supported Providers
- ✅ **Google Sign-In** (Web + Mobile)
- 🚧 **Apple Sign-In** (Coming soon)
- ✅ **Email/Password** (Traditional auth)

### Google OAuth Architecture

**Three separate OAuth clients (same Google Cloud Project):**
1. **Web Client** - Browser redirect flow
2. **Android Client** - Flutter Android apps
3. **iOS Client** - Flutter iOS apps

**All clients validated by backend** via `GOOGLE_ALLOWED_CLIENT_IDS`

### Endpoints

**Web Flow (Browser):**
- `GET /api/auth/google` - Initiate OAuth
- `GET /api/auth/google/callback` - Handle callback

**Mobile Flow (Flutter/React Native):**
- `POST /api/auth/google/token` - Verify ID token

### Email-Based Account Linking

**Users are unified by email across:**
- Different platforms (web, Android, iOS)
- Different providers (Google, Apple, email/password)
- Different devices

**Example:** User signs up on Android with Google (`user@gmail.com`) → Later opens web app → Signs in with Google → Same account recognized ✅

### Configuration

Required environment variables:
```bash
# Google OAuth - Multiple clients
GOOGLE_WEB_CLIENT_ID=your-web-client.apps.googleusercontent.com
GOOGLE_ANDROID_CLIENT_ID=your-android-client.apps.googleusercontent.com
GOOGLE_IOS_CLIENT_ID=your-ios-client.apps.googleusercontent.com
GOOGLE_ALLOWED_CLIENT_IDS=web-client,android-client,ios-client
GOOGLE_CLIENT_SECRET=your-secret

# Apple OAuth (coming soon)
APPLE_CLIENT_ID=your-service-id
APPLE_TEAM_ID=your-team-id
APPLE_IOS_BUNDLE_ID=com.yourapp.bundle
```

See [API.md](API.md) for detailed OAuth integration examples.

## License

TBD
