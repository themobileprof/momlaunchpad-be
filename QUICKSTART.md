# Quick Start Guide

This guide will help you get the MomLaunchpad backend running quickly for testing.

## Prerequisites

- Go 1.24.3 or later
- PostgreSQL 12+ running
- Redis (optional)
- DeepSeek API key

## Step 1: Clone and Setup

```bash
cd /path/to/momlaunchpad-be

# Copy environment template
cp .env.example .env
```

## Step 2: Configure Environment

Edit `.env` with your credentials:

```env
# Database
DATABASE_URL=postgresql://username:password@localhost:5432/momlaunchpad?sslmode=disable

# DeepSeek API
DEEPSEEK_API_KEY=your-deepseek-api-key-here

# JWT Secret (generate a secure random string)
JWT_SECRET=your-secure-jwt-secret-here

# Server
PORT=8080

# Redis (optional)
REDIS_URL=redis://localhost:6379/0

# Twilio Voice (optional - for premium voice calls)
TWILIO_ACCOUNT_SID=your-twilio-account-sid
TWILIO_AUTH_TOKEN=your-twilio-auth-token
TWILIO_PHONE_NUMBER=+1234567890
```

**Note:** Twilio credentials are optional. If not provided, voice call endpoints will not be registered.

## Step 3: Run Database Migrations

```bash
# Using Makefile
make migrate-up

# Or manually with psql
psql $DATABASE_URL -f migrations/001_init_schema.up.sql
```

## Step 4: Run Tests

```bash
# Run all tests
make test

# Or with coverage
make test-coverage
```

## Step 5: Start the Server

```bash
# Production mode
make run

# Development mode with hot reload (requires air)
make dev

# Or run directly
go run cmd/server/main.go
```

Server will start on `http://localhost:8080`

## Step 6: Test the API

### Register a User

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123",
    "name": "Test User",
    "language": "en"
  }'
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "uuid-here",
    "email": "test@example.com",
    "name": "Test User",
    "language": "en"
  }
}
```

Save the token for subsequent requests.

### Test OAuth (Google Sign-In)

**Web Flow (Browser):**
```bash
# Open in browser
open http://localhost:8080/api/auth/google

# Or with curl (will redirect)
curl -L http://localhost:8080/api/auth/google
```

**Mobile Flow (ID Token Verification):**
```bash
# Simulate mobile app sending ID token
curl -X POST http://localhost:8080/api/auth/google/token \
  -H "Content-Type: application/json" \
  -d '{
    "id_token": "eyJhbGciOiJSUzI1NiIsImtpZCI6IjE2M..."
  }'
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "uuid-here",
    "email": "user@gmail.com",
    "username": "user"
  }
}
```

**Testing Email-Based Account Linking:**
```bash
# 1. Register with email/password
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@gmail.com",
    "password": "password123",
    "name": "Test User"
  }'

# 2. Later, sign in with Google using same email
# Backend will recognize the email and link accounts
# Both auth methods will access the same user account
```

### Create a Reminder

```bash
export TOKEN="your-jwt-token-here"

curl -X POST http://localhost:8080/api/reminders \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Prenatal checkup",
    "description": "Monthly checkup with Dr. Smith",
    "scheduled_time": "2024-02-01T14:00:00Z",
    "priority": "high"
  }'
```

### Get Reminders

```bash
curl http://localhost:8080/api/reminders \
  -H "Authorization: Bearer $TOKEN"
```

### Test WebSocket Chat

Install `websocat` for WebSocket testing:

```bash
# Install websocat
brew install websocat  # macOS
# or
cargo install websocat  # via Rust

# Connect to chat
websocat "ws://localhost:8080/ws/chat?token=$TOKEN"

# Send messages (type and press Enter)
{"content": "Hello! I'm 14 weeks pregnant"}
{"content": "I'm feeling nauseous, is this normal?"}
{"content": "When will my baby start kicking?"}
```

## Testing with curl (WebSocket upgrade)

```bash
# Test WebSocket upgrade
curl -i -N \
  -H "Connection: Upgrade" \
  -H "Upgrade: websocket" \
  -H "Sec-WebSocket-Version: 13" \
  -H "Sec-WebSocket-Key: SGVsbG8sIHdvcmxkIQ==" \
  "http://localhost:8080/ws/chat?token=$TOKEN"
```

## Test Small Talk (No AI)

Small talk messages won't trigger AI or memory:

```json
{"content": "hello"}
{"content": "hi there"}
{"content": "thanks"}
```

Response will be instant canned responses like:
- "I'm here with you. How can I help today?"

## Test Pregnancy Questions (AI-Powered)

These trigger the full AI pipeline:

```json
{"content": "When will my baby start kicking?"}
{"content": "I'm experiencing morning sickness, what can I do?"}
{"content": "Is it safe to exercise during pregnancy?"}
```

Response flow:
1. Intent classification (pregnancy_question)
2. Load memory (recent messages + facts)
3. Build super-prompt
4. Stream DeepSeek response
5. Extract facts (e.g., pregnancy week)
6. Suggest calendar reminders if applicable

## Test Symptom Reports

```json
{"content": "I have severe headache and dizziness"}
{"content": "I'm bleeding, should I be worried?"}
```

These will:
- Get urgent priority
- Suggest immediate calendar reminders
- Store symptom facts

## Troubleshooting

### Database connection error
```
Error: failed to connect to database
```

**Solution:** Check your `DATABASE_URL` in `.env` and ensure PostgreSQL is running:
```bash
psql $DATABASE_URL -c "SELECT 1;"
```

### Migration errors
```
Error: migration already applied
```

**Solution:** Check migration status or rollback:
```bash
make migrate-down  # Rollback
make migrate-up    # Apply again
```

### DeepSeek API error
```
Error: failed to call DeepSeek API
```

**Solution:** Verify your `DEEPSEEK_API_KEY` is valid:
```bash
curl -X POST https://api.deepseek.com/v1/chat/completions \
  -H "Authorization: Bearer $DEEPSEEK_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-chat",
    "messages": [{"role": "user", "content": "test"}]
  }'
```

### JWT errors
```
Error: invalid token
```

**Solution:** 
- Ensure you're including the full token (no truncation)
- Check token hasn't expired (7 day expiry)
- Register/login to get a fresh token

## Next Steps

- Read [API.md](API.md) for complete API documentation
- See [VOICE.md](VOICE.md) for Twilio voice call setup
- Review [BACKEND_SPEC.md](BACKEND_SPEC.md) for architecture details
- Check [WEBSOCKET_GUIDE.md](WEBSOCKET_GUIDE.md) for WebSocket details
- Read [.github/copilot-instructions.md](.github/copilot-instructions.md) for development guidelines

## Testing Premium Features

### Voice Calls (Premium Only)

**Prerequisites:**
1. Twilio account with purchased phone number
2. Configure webhooks in Twilio Console:
   - Incoming: `https://your-domain.com/api/voice/incoming`
   - Status: `https://your-domain.com/api/voice/status`
3. User must have premium subscription

**Upgrade User to Premium:**
```sql
-- Connect to database
psql $DATABASE_URL

-- Upgrade user to premium
UPDATE subscriptions 
SET plan_id = (SELECT id FROM plans WHERE code = 'premium')
WHERE user_id = 'user-uuid-here';
```

**Test Voice Call:**
1. Start server with Twilio credentials in `.env`
2. Call your Twilio number from registered phone
3. System will greet you and process your questions
4. View logs to see speech-to-text transcription

**Local Testing with ngrok:**
```bash
# Start ngrok tunnel
ngrok http 8080

# Copy HTTPS URL (e.g., https://abc123.ngrok.io)
# Configure Twilio webhooks to use this URL
# Now you can test voice calls locally
```

See [VOICE.md](VOICE.md) for comprehensive voice setup documentation.

## Common Development Commands

```bash
# Run tests
make test

# Build binary
make build

# Run with hot reload
make dev

# Apply migrations
make migrate-up

# Rollback migrations
make migrate-down

# Generate test coverage report
make test-coverage

# Initialize project (first time setup)
make init
```

## Production Deployment

1. Set environment variables on your VM
2. Build the binary: `make build`
3. Run with systemd or supervisor: `./bin/server`
4. Use nginx as reverse proxy for HTTPS
5. Setup PostgreSQL with proper credentials
6. Enable Redis for caching (optional)

Example systemd service file:

```ini
[Unit]
Description=MomLaunchpad Backend
After=network.target postgresql.service

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/momlaunchpad-be
Environment="DATABASE_URL=postgresql://..."
Environment="DEEPSEEK_API_KEY=..."
Environment="JWT_SECRET=..."
ExecStart=/opt/momlaunchpad-be/bin/server
Restart=always

[Install]
WantedBy=multi-user.target
```
