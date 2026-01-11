# AI Name Feature Implementation

## Overview
The AI assistant now has a configurable name that can be managed through the admin panel. This allows personalizing the AI's identity without code changes.

## Implementation Details

### 1. Database Schema
**Migration:** `003_system_settings.up.sql`
- Created `system_settings` table with key-value storage
- Default AI name: `MomBot`
- Auto-updating timestamp trigger

```sql
CREATE TABLE system_settings (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 2. Database Functions
**File:** `internal/db/queries.go`

Added three new functions:
- `GetSystemSetting(ctx, key)` - Retrieve single setting
- `UpdateSystemSetting(ctx, key, value)` - Update setting value
- `GetAllSystemSettings(ctx)` - List all settings

### 3. Admin API Endpoints
**File:** `internal/api/admin.go`

Added three new admin-only endpoints:
- `GET /api/admin/settings` - List all settings
- `GET /api/admin/settings/:key` - Get specific setting
- `PUT /api/admin/settings/:key` - Update setting value

**Request format:**
```json
{
  "value": "Luna"
}
```

**Response format:**
```json
{
  "setting": {
    "Key": "ai_name",
    "Value": "Luna",
    "Description": "The name of the AI assistant displayed to users",
    "UpdatedAt": "2026-01-11T05:18:23.997001Z"
  }
}
```

### 4. Prompt Builder Integration
**File:** `internal/prompt/builder.go`

- Added `AIName` field to `PromptRequest` struct
- Modified `buildSystemPrompt()` to inject AI name dynamically
- Small talk responses also use custom AI name
- Fallback to default text if name not provided

**Example system prompt:**
```
You are Luna, a knowledgeable and empathetic assistant.
Your role is to provide accurate, helpful, and supportive information about pregnancy...
```

### 5. Chat Engine Integration
**File:** `internal/chat/engine.go`

- Added concurrent goroutine to fetch `ai_name` setting
- Added `GetSystemSetting()` to `DBInterface`
- AI name passed to prompt builder with every message
- Fallback to "MomBot" if setting retrieval fails

**Log output includes AI name:**
```
Building prompt for user=5cbd012e, intent=pregnancy_question, aiName=Luna
```

### 6. Tests
**File:** `internal/prompt/builder_test.go`

Added comprehensive test `TestBuilder_AIName` with three scenarios:
- Custom AI name ("Luna")
- Empty name (defaults to "pregnancy support assistant")
- MomBot name

All tests pass successfully.

## Usage

### Admin Panel Usage

1. **Get current AI name:**
```bash
curl -X GET http://localhost:8080/api/admin/settings/ai_name \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"
```

2. **Update AI name:**
```bash
curl -X PUT http://localhost:8080/api/admin/settings/ai_name \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"value":"Luna"}'
```

3. **List all settings:**
```bash
curl -X GET http://localhost:8080/api/admin/settings \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"
```

### Database Direct Access

```sql
-- View current AI name
SELECT * FROM system_settings WHERE key = 'ai_name';

-- Update AI name directly
UPDATE system_settings SET value = 'Luna' WHERE key = 'ai_name';
```

## Security

- All endpoints protected by JWT middleware
- Requires `is_admin = true` flag
- Setting updates logged with timestamps
- No sensitive data exposed

## Extensibility

The `system_settings` table can be extended for other configuration:
- `max_tokens` - Maximum response length
- `temperature` - AI creativity level
- `greeting_message` - Custom welcome message
- `disclaimer_text` - Legal disclaimer

## Testing Verification

```bash
# Run all tests
go test ./...

# Run prompt builder tests specifically
go test ./internal/prompt -v

# Run AI name test specifically
go test ./internal/prompt -v -run TestBuilder_AIName
```

## Deployment Status

✅ Migration applied to database
✅ Backend rebuilt and deployed
✅ All tests passing (except unrelated symptom tests)
✅ Admin API endpoints functional
✅ AI name successfully updated from "MomBot" to "Luna"

## Next Steps for Flutter Client

The Flutter app should:
1. Add admin settings screen with AI name field
2. Call `GET /api/admin/settings/ai_name` to retrieve current name
3. Call `PUT /api/admin/settings/ai_name` to update name
4. Optionally display AI name in chat UI header
5. Consider caching AI name client-side with periodic refresh

## Example Conversation

**Before (default):**
> User: "When will my baby start kicking?"
> 
> MomBot: "Baby movements typically start between 18-25 weeks..."

**After (updated to "Luna"):**
> User: "When will my baby start kicking?"
> 
> Luna: "Baby movements typically start between 18-25 weeks..."

The AI introduces herself as Luna in the system prompt, influencing how she refers to herself in responses.
