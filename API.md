# MomLaunchpad API Documentation

Base URL: `http://localhost:8080`

## Authentication

All protected endpoints require a JWT token in the Authorization header:
```
Authorization: Bearer <token>
```

## Endpoints

### Health Check

#### GET /health
Check server status.

**Response:**
```json
{
  "status": "healthy",
  "time": 1735471200
}
```

---

### Authentication

#### POST /api/auth/register
Register a new user.

**Request:**
```json
{
  "email": "user@example.com",
  "password": "securepassword",
  "name": "Jane Doe",
  "language": "en"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "name": "Jane Doe",
    "language": "en",
    "is_admin": false,
    "created_at": "2024-01-15T10:00:00Z"
  }
}
```

#### POST /api/auth/login
Authenticate and get JWT token.

**Request:**
```json
{
  "email": "user@example.com",
  "password": "securepassword"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "name": "Jane Doe",
    "language": "en",
    "is_admin": false
  }
}
```

#### GET /api/auth/me
Get current user information (protected).

**Headers:**
```
Authorization: Bearer <token>
```

**Response:**
```json
{
  "id": "uuid",
  "email": "user@example.com",
  "name": "Jane Doe",
  "language": "en",
  "is_admin": false,
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:00Z"
}
```

---

### Calendar / Reminders

#### GET /api/reminders
Get all reminders for the authenticated user (protected).

**Headers:**
```
Authorization: Bearer <token>
```

**Response:**
```json
[
  {
    "id": "uuid",
    "user_id": "uuid",
    "title": "Doctor appointment",
    "description": "Prenatal checkup",
    "scheduled_time": "2024-01-20T14:00:00Z",
    "priority": "high",
    "is_completed": false,
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-15T10:00:00Z"
  }
]
```

#### POST /api/reminders
Create a new reminder (protected).

**Headers:**
```
Authorization: Bearer <token>
```

**Request:**
```json
{
  "title": "Doctor appointment",
  "description": "Prenatal checkup",
  "scheduled_time": "2024-01-20T14:00:00Z",
  "priority": "high"
}
```

**Response:**
```json
{
  "id": "uuid",
  "user_id": "uuid",
  "title": "Doctor appointment",
  "description": "Prenatal checkup",
  "scheduled_time": "2024-01-20T14:00:00Z",
  "priority": "high",
  "is_completed": false,
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:00Z"
}
```

#### PUT /api/reminders/:id
Update an existing reminder (protected, owner only).

**Headers:**
```
Authorization: Bearer <token>
```

**Request:**
```json
{
  "title": "Doctor appointment - Updated",
  "description": "Prenatal checkup with ultrasound",
  "scheduled_time": "2024-01-20T15:00:00Z",
  "priority": "urgent",
  "is_completed": true
}
```

**Response:**
```json
{
  "message": "Reminder updated successfully"
}
```

#### DELETE /api/reminders/:id
Delete a reminder (protected, owner only).

**Headers:**
```
Authorization: Bearer <token>
```

**Response:**
```json
{
  "message": "Reminder deleted successfully"
}
```

---

### Savings Tracker

#### GET /api/savings/summary
Get savings summary with EDD, goal, and progress (protected).

**Headers:**
```
Authorization: Bearer <token>
```

**Response:**
```json
{
  "expected_delivery_date": "2026-09-15T00:00:00Z",
  "savings_goal": 5000.00,
  "total_saved": 400.50,
  "progress_percentage": 8.01,
  "days_until_delivery": 254
}
```

#### GET /api/savings/entries
Get all savings entries for the current user (protected).

**Headers:**
```
Authorization: Bearer <token>
```

**Response:**
```json
[
  {
    "id": "uuid",
    "amount": 250.50,
    "description": "Baby clothes shopping",
    "entry_date": "2026-01-03T00:00:00Z",
    "created_at": "2026-01-03T12:36:23.820698Z"
  },
  {
    "id": "uuid",
    "amount": 150.00,
    "description": "Weekly savings",
    "entry_date": "2026-01-03T00:00:00Z",
    "created_at": "2026-01-03T12:36:27.939641Z"
  }
]
```

#### POST /api/savings/entries
Create a new savings entry (protected).

**Headers:**
```
Authorization: Bearer <token>
```

**Request:**
```json
{
  "amount": 250.50,
  "description": "Baby clothes shopping",
  "entry_date": "2026-01-03T00:00:00Z"
}
```

**Note:** `entry_date` is optional. If not provided, current date is used.

**Response:**
```json
{
  "id": "uuid",
  "amount": 250.50,
  "description": "Baby clothes shopping",
  "entry_date": "2026-01-03T00:00:00Z",
  "created_at": "2026-01-03T12:36:23.820698Z"
}
```

#### PUT /api/savings/edd
Update expected delivery date (protected).

**Headers:**
```
Authorization: Bearer <token>
```

**Request:**
```json
{
  "expected_delivery_date": "2026-09-15T00:00:00Z"
}
```

**Note:** Set to `null` to clear the EDD.

**Response:**
```json
{
  "message": "Expected delivery date updated successfully"
}
```

#### PUT /api/savings/goal
Update savings goal (protected).

**Headers:**
```
Authorization: Bearer <token>
```

**Request:**
```json
{
  "savings_goal": 5000.00
}
```

**Response:**
```json
{
  "message": "Savings goal updated successfully"
}
```

---

### WebSocket Chat

#### WS /ws/chat
Real-time chat with AI streaming (protected).

**Connection:**
Connect with JWT token in query parameter or header:
```
ws://localhost:8080/ws/chat?token=<jwt_token>
```

Or with header:
```
Authorization: Bearer <token>
```

**Send Message:**
```json
{
  "content": "I'm feeling nauseous today"
}
```

**Receive Streaming Response:**

Messages are sent as JSON objects with different types:

1. **Message chunks** (AI response streaming):
```json
{
  "type": "message",
  "content": "I understand you're experiencing nausea..."
}
```

2. **Calendar suggestion**:
```json
{
  "type": "calendar",
  "data": {
    "should_suggest": true,
    "reason": "Symptom tracking recommended",
    "suggested_title": "Monitor nausea symptom",
    "suggested_time": "2024-01-16T10:00:00Z",
    "priority": "high"
  }
}
```

3. **Error**:
```json
{
  "type": "error",
  "content": "Failed to process message"
}
```

4. **Done** (response complete):
```json
{
  "type": "done"
}
```

**Flow:**
1. User sends message
2. Server classifies intent (rule-based)
3. For small talk → immediate canned response
4. For pregnancy/symptom questions:
   - Load user memory (recent messages + facts)
   - Build super-prompt with context
   - Stream AI response chunks
   - Send calendar suggestion if applicable
   - Save message and extract facts
5. Send "done" signal

---

## Error Responses

All endpoints may return error responses:

**400 Bad Request:**
```json
{
  "error": "Invalid request body"
}
```

**401 Unauthorized:**
```json
{
  "error": "Invalid or missing token"
}
```

**403 Forbidden:**
```json
{
  "error": "You do not have permission to access this resource"
}
```

**404 Not Found:**
```json
{
  "error": "Reminder not found"
}
```

**500 Internal Server Error:**
```json
{
  "error": "Internal server error"
}
```

---

## Notes

- JWT tokens expire after 7 days
- All timestamps are in ISO 8601 format (UTC)
- Language codes: `en` (English), `es` (Spanish)
- Priority levels: `low`, `medium`, `high`, `urgent`
- Small talk messages don't trigger AI or memory storage
- All chat messages are persisted to database
- Facts are extracted with confidence scores (0.0-1.0)
- Calendar suggestions require explicit user confirmation to create reminders
