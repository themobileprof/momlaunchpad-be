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

### OAuth Authentication

#### GET /api/auth/google
Initiate Google OAuth login (web flow).

**Description:** Redirects user to Google's OAuth consent screen. After user approves, Google redirects back to the callback URL with an authorization code.

**Flow:**
1. User clicks "Login with Google"
2. Backend redirects to Google OAuth
3. User authorizes app
4. Google redirects to `/api/auth/google/callback`
5. Backend exchanges code for user info
6. Backend returns JWT token

**Usage:**
```html
<a href="http://localhost:8080/api/auth/google">Login with Google</a>
```

#### GET /api/auth/google/callback
Handle Google OAuth callback (web flow).

**Description:** Receives authorization code from Google, exchanges it for user info, creates/links user account, and returns JWT token.

**Query Parameters:**
- `code`: Authorization code from Google
- `state`: CSRF protection token

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "uuid",
    "email": "user@gmail.com",
    "username": "user",
    "name": "John Doe"
  }
}
```

**Error Response:**
```json
{
  "error": "Invalid state parameter"
}
```

#### POST /api/auth/google/token
Authenticate with Google ID token (mobile flow).

**Description:** Verifies Google ID token from mobile apps (Flutter, React Native, etc.) and returns JWT. Supports tokens from web, Android, and iOS Google OAuth clients.

**Request:**
```json
{
  "id_token": "eyJhbGciOiJSUzI1NiIsImtpZCI6IjE2M..."
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "uuid",
    "email": "user@gmail.com",
    "username": "user"
  }
}
```

**Error Responses:**
```json
{
  "error": "ID token is required"
}
```
```json
{
  "error": "Invalid ID token"
}
```
```json
{
  "error": "Email not verified with Google"
}
```

**Mobile Integration Example (Flutter):**
```dart
// Using google_sign_in package
final GoogleSignInAccount? googleUser = await GoogleSignIn().signIn();
final GoogleSignInAuthentication googleAuth = await googleUser!.authentication;

// Send ID token to backend
final response = await http.post(
  Uri.parse('http://localhost:8080/api/auth/google/token'),
  headers: {'Content-Type': 'application/json'},
  body: jsonEncode({'id_token': googleAuth.idToken}),
);

final data = jsonDecode(response.body);
final jwtToken = data['token'];
```

#### GET /api/auth/apple
Initiate Apple OAuth login (coming soon).

**Response:**
```json
{
  "error": "Apple Sign-In coming soon"
}
```

#### GET /api/auth/apple/callback
Handle Apple OAuth callback (coming soon).

**Response:**
```json
{
  "error": "Apple Sign-In coming soon"
}
```

---

### OAuth Provider Details

**Google OAuth Configuration:**
- Supports 3 separate OAuth clients (web, Android, iOS)
- All clients belong to the same Google Cloud Project
- Backend validates tokens from any of the configured clients
- Email is the canonical identifier across all platforms

**Client Types:**
1. **Web Client** - For browser redirect flow
   - Authorized JavaScript origins: `http://localhost:8080`
   - Redirect URIs: `http://localhost:8080/api/auth/google/callback`

2. **Android Client** - For Flutter/native Android apps
   - Requires package name and SHA-1 certificate
   - Used with `google_sign_in` package

3. **iOS Client** - For Flutter/native iOS apps
   - Requires bundle ID
   - Used with `google_sign_in` package

**User Account Linking:**
- Users are linked by email across all OAuth providers
- Same email on Google + Apple = same user account
- Same email on web + mobile = same user account
- Allows seamless cross-platform experience

**Example Scenarios:**

*Scenario 1: Cross-platform with Google*
1. User signs up on Android with Google → `user@gmail.com`
2. User opens web app, clicks "Sign in with Google" → Backend recognizes email → Same account ✅

*Scenario 2: Multiple OAuth providers*
1. User signs in with Google → `user@gmail.com`
2. Later, user signs in with Apple using same email → Backend links accounts → Same user ✅

*Scenario 3: OAuth + traditional login*
1. User registers with email/password → `user@example.com`
2. Later, user tries Google OAuth with `user@example.com` → Backend links accounts → Same user ✅

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
