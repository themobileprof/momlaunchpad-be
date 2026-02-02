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

### Voice (Twilio Webhooks)

#### POST /api/voice/incoming
Twilio webhook for incoming voice calls (premium feature).

**Description:** Handles incoming phone calls from premium users. Identifies user by phone number, plays greeting in their preferred language, and begins conversation.

**Request:** Form data from Twilio
- `CallSid`: Unique call identifier
- `From`: Caller's phone number
- `To`: Called number (your Twilio number)
- `CallStatus`: Current call status

**Response:** TwiML XML
```xml
<?xml version="1.0" encoding="UTF-8"?>
<Response>
  <Say voice="Polly.Joanna" language="en-US">Welcome to MomLaunchpad...</Say>
  <Gather action="/api/voice/gather" input="speech" language="en-US" timeout="5">
    <Say>How can I help you today?</Say>
  </Gather>
</Response>
```

#### POST /api/voice/gather
Twilio webhook for speech recognition results.

**Description:** Receives transcribed user speech, processes through chat engine, and returns AI response as TwiML.

**Request:** Form data from Twilio
- `CallSid`: Unique call identifier
- `SpeechResult`: Transcribed user speech
- `Confidence`: Transcription confidence score

**Response:** TwiML with AI response
```xml
<?xml version="1.0" encoding="UTF-8"?>
<Response>
  <Say voice="Polly.Joanna" language="en-US">Your baby will start kicking around week 18...</Say>
  <Gather action="/api/voice/gather" input="speech" language="en-US" timeout="5">
    <Say>Do you have another question?</Say>
  </Gather>
  <Say>Thank you for calling MomLaunchpad. Take care!</Say>
  <Hangup/>
</Response>
```

#### POST /api/voice/status
Twilio webhook for call status updates.

**Description:** Receives call status updates and cleans up sessions when calls end.

**Request:** Form data from Twilio
- `CallSid`: Unique call identifier
- `CallStatus`: New status (completed, failed, etc.)

**Response:** Plain text "OK"

**Voice Feature Notes:**
- Available only to premium users
- Automatically uses user's preferred language
- Supports AWS Polly voices (Joanna, Lupe, Celine, Vitoria, Vicki)
- Session management with automatic cleanup
- Integrates with same chat engine as WebSocket
- See [VOICE.md](VOICE.md) for detailed setup instructions

---

### Conversations

#### GET /api/conversations
List conversations (protected).

**Headers:**
```
Authorization: Bearer <token>
```

**Query Parameters:**
- `limit`: items per page (default: 20)
- `offset`: pagination offset (default: 0)

**Response:**
```json
[
  {
    "id": "uuid",
    "user_id": "uuid",
    "title": "Conversation Title",
    "is_starred": false,
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-15T10:00:00Z"
  }
]
```

#### POST /api/conversations
Create a new conversation (protected).

**Headers:**
```
Authorization: Bearer <token>
```

**Request:**
```json
{
  "title": "My New Chat"
}
```

**Response:**
```json
{
  "id": "uuid",
  "user_id": "uuid",
  "title": "My New Chat",
  "is_starred": false,
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:00Z"
}
```

#### GET /api/conversations/:id
Get a specific conversation (protected).

**Headers:**
```
Authorization: Bearer <token>
```

**Response:**
```json
{
  "id": "uuid",
  "user_id": "uuid",
  "title": "Conversation Title",
  "is_starred": false,
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:00Z"
}
```

#### PATCH /api/conversations/:id
Update conversation details (protected).

**Headers:**
```
Authorization: Bearer <token>
```

**Request:**
```json
{
  "title": "Updated Title",
  "is_starred": true
}
```

**Response:**
```json
{
  "id": "uuid",
  "user_id": "uuid",
  "title": "Updated Title",
  "is_starred": true,
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:05:00Z"
}
```

#### DELETE /api/conversations/:id
Delete a conversation (protected).

**Headers:**
```
Authorization: Bearer <token>
```

**Response:**
```json
{
  "message": "Conversation deleted"
}
```

#### GET /api/conversations/:id/messages
List messages in a conversation (protected).

**Headers:**
```
Authorization: Bearer <token>
```

**Query Parameters:**
- `limit`: items per page (default: 50)
- `offset`: pagination offset (default: 0)

**Response:**
```json
[
  {
    "id": "uuid",
    "user_id": "uuid",
    "conversation_id": "uuid",
    "role": "user",
    "content": "Hello",
    "created_at": "2024-01-15T10:00:00Z"
  }
]
```

---

### Admin (Protected + Admin Role)

All admin endpoints require:
1. Valid JWT token
2. User's `is_admin` field set to `true`

Non-admin users receive `403 Forbidden`.

---

#### Plan Management

##### GET /api/admin/plans
List all subscription plans.

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Response:**
```json
{
  "plans": [
    {
      "id": 1,
      "code": "free",
      "name": "Free",
      "description": "Default free plan with limited features",
      "active": true,
      "created_at": "2026-01-06T00:00:00Z"
    },
    {
      "id": 2,
      "code": "premium",
      "name": "Premium",
      "description": "Full access to all features and higher quotas",
      "active": true,
      "created_at": "2026-01-06T00:00:00Z"
    }
  ]
}
```

##### POST /api/admin/plans
Create a new subscription plan.

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Request:**
```json
{
  "code": "enterprise",
  "name": "Enterprise",
  "description": "Custom enterprise plan with dedicated support"
}
```

**Response (201 Created):**
```json
{
  "plan": {
    "id": 3,
    "code": "enterprise",
    "name": "Enterprise",
    "description": "Custom enterprise plan with dedicated support",
    "active": true,
    "created_at": "2026-01-06T00:00:00Z"
  }
}
```

##### PUT /api/admin/plans/:planId
Update an existing plan.

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Request:**
```json
{
  "name": "Enterprise Plus",
  "description": "Updated enterprise plan",
  "active": false
}
```

**Response:**
```json
{
  "message": "plan updated successfully"
}
```

##### DELETE /api/admin/plans/:planId
Deactivate a plan (soft delete).

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Response:**
```json
{
  "message": "plan deactivated successfully"
}
```

##### GET /api/admin/plans/:planId/features
Get all features assigned to a plan.

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Response:**
```json
{
  "features": [
    {
      "feature_id": 1,
      "feature_key": "chat",
      "feature_name": "Chat Access",
      "quota_limit": 100,
      "quota_period": "monthly"
    },
    {
      "feature_id": 2,
      "feature_key": "calendar",
      "feature_name": "Calendar",
      "quota_limit": null,
      "quota_period": "unlimited"
    }
  ]
}
```

##### POST /api/admin/plans/:planId/features/:featureId
Assign a feature to a plan with quota settings.

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Request:**
```json
{
  "quota_limit": 50,
  "quota_period": "daily"
}
```

**Note:** `quota_limit` as `null` means unlimited. Valid periods: `daily`, `weekly`, `monthly`, `unlimited`.

**Response:**
```json
{
  "message": "feature assigned to plan successfully"
}
```

##### DELETE /api/admin/plans/:planId/features/:featureId
Remove a feature from a plan.

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Response:**
```json
{
  "message": "feature removed from plan successfully"
}
```

---

#### Feature Management

##### GET /api/admin/features
List all available features.

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Response:**
```json
{
  "features": [
    {
      "id": 1,
      "feature_key": "chat",
      "name": "Chat Access",
      "description": "AI chat support",
      "created_at": "2026-01-06T00:00:00Z"
    },
    {
      "id": 2,
      "feature_key": "calendar",
      "name": "Calendar",
      "description": "Reminders and scheduling",
      "created_at": "2026-01-06T00:00:00Z"
    },
    {
      "id": 3,
      "feature_key": "voice_calls",
      "name": "Voice Calls",
      "description": "Call in via phone for AI assistance",
      "created_at": "2026-01-06T00:00:00Z"
    }
  ]
}
```

##### POST /api/admin/features
Create a new feature.

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Request:**
```json
{
  "feature_key": "video_consultation",
  "name": "Video Consultation",
  "description": "Video calls with healthcare providers"
}
```

**Response (201 Created):**
```json
{
  "feature": {
    "id": 7,
    "feature_key": "video_consultation",
    "name": "Video Consultation",
    "description": "Video calls with healthcare providers",
    "created_at": "2026-01-06T00:00:00Z"
  }
}
```

##### PUT /api/admin/features/:featureId
Update an existing feature.

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Request:**
```json
{
  "name": "Video Consultation Pro",
  "description": "Premium video calls with specialists"
}
```

**Response:**
```json
{
  "message": "feature updated successfully"
}
```

##### DELETE /api/admin/features/:featureId
Delete a feature.

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Response:**
```json
{
  "message": "feature deleted successfully"
}
```

---

#### Language Management

##### GET /api/admin/languages
List all languages.

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Response:**
```json
{
  "languages": [
    {
      "code": "en",
      "name": "English",
      "native_name": "English",
      "is_enabled": true,
      "is_experimental": false,
      "created_at": "2026-01-06T00:00:00Z"
    },
    {
      "code": "es",
      "name": "Spanish",
      "native_name": "Español",
      "is_enabled": true,
      "is_experimental": false,
      "created_at": "2026-01-06T00:00:00Z"
    },
    {
      "code": "yo",
      "name": "Yoruba",
      "native_name": "Yorùbá",
      "is_enabled": true,
      "is_experimental": true,
      "created_at": "2026-01-06T00:00:00Z"
    }
  ]
}
```

##### POST /api/admin/languages
Create a new language.

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Request:**
```json
{
  "code": "pt",
  "name": "Portuguese",
  "native_name": "Português",
  "is_enabled": true,
  "is_experimental": false
}
```

**Response (201 Created):**
```json
{
  "language": {
    "code": "pt",
    "name": "Portuguese",
    "native_name": "Português",
    "is_enabled": true,
    "is_experimental": false,
    "created_at": "2026-01-06T00:00:00Z"
  }
}
```

##### PUT /api/admin/languages/:code
Update an existing language.

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Request:**
```json
{
  "name": "Brazilian Portuguese",
  "is_enabled": false,
  "is_experimental": true
}
```

**Response:**
```json
{
  "message": "language updated successfully",
  "language": {
    "code": "pt",
    "name": "Brazilian Portuguese",
    "native_name": "Português",
    "is_enabled": false,
    "is_experimental": true,
    "created_at": "2026-01-06T00:00:00Z"
  }
}
```

##### DELETE /api/admin/languages/:code
Delete a language. Cannot delete English (default).

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Response:**
```json
{
  "message": "language deleted successfully"
}
```

**Error (403 Forbidden):**
```json
{
  "error": "cannot delete default language (English)"
}
```

---

#### Analytics

##### GET /api/admin/analytics/topics
Analyze what users are asking about (topic analysis).

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Query Parameters:**
- `days` (optional): Number of days to analyze (default: 7, max: 90)
- `limit` (optional): Max topics to return (default: 20, max: 100)

**Response:**
```json
{
  "period_days": 7,
  "analytics": [
    {
      "intent": "nausea_morning_sickness",
      "count": 245,
      "percentage": 28.5,
      "sample_query": "I've been feeling nauseous every morning"
    },
    {
      "intent": "baby_movement",
      "count": 189,
      "percentage": 22.0,
      "sample_query": "When will my baby start kicking?"
    },
    {
      "intent": "diet_nutrition",
      "count": 156,
      "percentage": 18.1,
      "sample_query": "What foods should I avoid during pregnancy?"
    },
    {
      "intent": "pain_cramps",
      "count": 98,
      "percentage": 11.4,
      "sample_query": "Is cramping normal in the first trimester?"
    },
    {
      "intent": "pregnancy_timeline",
      "count": 87,
      "percentage": 10.1,
      "sample_query": "What happens during week 20?"
    },
    {
      "intent": "general_questions",
      "count": 85,
      "percentage": 9.9,
      "sample_query": "How are you today?"
    }
  ]
}
```

##### GET /api/admin/analytics/users
Get user statistics.

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Response:**
```json
{
  "stats": {
    "total_users": 1250,
    "active_users_7_days": 456,
    "active_users_30_days": 892,
    "users_by_plan": {
      "free": 980,
      "premium": 270
    },
    "users_by_language": {
      "en": 850,
      "es": 280,
      "fr": 120
    }
  }
}
```

##### GET /api/admin/analytics/calls
Get voice call history.

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Query Parameters:**
- `days` (optional): Number of days to retrieve (default: 7, max: 90)
- `limit` (optional): Max calls to return (default: 50, max: 200)

**Response:**
```json
{
  "period_days": 7,
  "calls": [
    {
      "call_sid": "CA...",
      "user_id": "uuid",
      "user_email": "user@example.com",
      "phone_number": "+1234567890",
      "duration_seconds": 245,
      "status": "completed",
      "created_at": "2026-01-05T14:30:00Z"
    }
  ]
}
```

---

#### User Management

##### GET /api/admin/users/:userId/subscription
Get a user's subscription details.

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Response:**
```json
{
  "user_id": "uuid",
  "subscription": {
    "id": 1,
    "user_id": "uuid",
    "plan_id": 1,
    "plan_code": "free",
    "status": "active",
    "starts_at": "2026-01-06T00:00:00Z",
    "ends_at": null
  }
}
```

##### PUT /api/admin/users/:userId/plan
Update a user's subscription plan.

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Request:**
```json
{
  "plan_code": "premium"
}
```

**Response:**
```json
{
  "message": "plan updated successfully",
  "user_id": "uuid",
  "plan": "premium"
}
```

##### GET /api/admin/users/:userId/quota/:feature
Get quota usage details for a specific user and feature.

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Response:**
```json
{
  "user_id": "uuid",
  "feature": "chat",
  "quota": {
    "quota_limit": 100,
    "quota_period": "monthly",
    "usage_count": 45,
    "period_start": "2026-01-01T00:00:00Z",
    "period_end": "2026-01-31T23:59:59Z"
  }
}
```

##### POST /api/admin/users/:userId/quota/:feature/reset
Reset quota usage for a user's feature.

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Response:**
```json
{
  "message": "quota reset successfully",
  "user_id": "uuid",
  "feature": "chat"
}
```

##### GET /api/admin/quota/stats
Get system-wide quota statistics.

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Query Parameters:**
- `feature` (optional): Filter by feature code (e.g., `chat`)
- `plan` (optional): Filter by plan code (e.g., `free`, `premium`)
- `period` (optional): Time period (default: `today`, options: `daily`, `weekly`, `monthly`)

**Response:**
```json
{
  "period": "today",
  "stats": {
    "total_users": 150,
    "active_users": 45,
    "total_usage": 1250,
    "by_feature": {
      "chat": 980,
      "calendar": 270
    },
    "by_plan": {
      "free": 800,
      "premium": 450
    }
  }
}
```

##### POST /api/admin/users/:userId/features
Grant a specific feature to a user (bypass plan restrictions).

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Request:**
```json
{
  "feature_key": "voice_calls",
  "expires_at": 1735689600
}
```

**Note:** `expires_at` is optional (Unix timestamp). If omitted, grant is permanent until revoked.

**Response:**
```json
{
  "message": "feature granted successfully",
  "user_id": "uuid",
  "feature": "voice_calls"
}
```

---

### Symptom Tracking

Symptom tracking automatically extracts and stores symptom information from chat conversations. Users and doctors can query symptom history for better care management.

#### GET /api/symptoms/history
Get full symptom history with optional filters (protected).

**Headers:**
```
Authorization: Bearer <token>
```

**Query Parameters:**
- `limit` (optional): Maximum number of results (default: 50, max: 200)
- `type` (optional): Filter by symptom type (e.g., "swelling", "nausea", "headache")

**Example:**
```
GET /api/symptoms/history?limit=20&type=headache
```

**Response:**
```json
{
  "symptoms": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "symptom_type": "swelling",
      "description": "My feet are really swollen and puffy",
      "severity": "moderate",
      "frequency": "daily",
      "onset_time": "3 days ago",
      "associated_symptoms": ["back_pain"],
      "is_resolved": false,
      "reported_at": "2026-01-08T10:30:00Z",
      "resolved_at": null
    },
    {
      "id": "660e8400-e29b-41d4-a716-446655440001",
      "symptom_type": "nausea",
      "description": "Feeling nauseous in the mornings",
      "severity": "mild",
      "frequency": "daily",
      "onset_time": "yesterday",
      "associated_symptoms": [],
      "is_resolved": true,
      "reported_at": "2026-01-07T08:15:00Z",
      "resolved_at": "2026-01-08T09:00:00Z"
    }
  ],
  "count": 2
}
```

**Symptom Types:**
- `swelling` - Swollen feet, ankles, hands, face
- `nausea` - Morning sickness, nausea
- `headache` - Headaches, migraines
- `back_pain` - Back pain, lower back pain
- `cramping` - Cramps, cramping
- `vision_changes` - Blurry vision, vision changes
- `dizziness` - Dizzy, lightheaded
- `fatigue` - Tired, exhausted
- `insomnia` - Can't sleep, insomnia
- `heartburn` - Heartburn, acid reflux
- `vomiting` - Vomiting, throwing up
- `constipation` - Constipation
- `bleeding` - Bleeding, spotting
- `contractions` - Contractions, tightening
- `breast_changes` - Breast tenderness, nipple changes
- `mood_changes` - Mood swings, emotional changes
- `shortness_breath` - Shortness of breath
- `frequent_urination` - Frequent urination

**Severity Levels:**
- `mild` - Minor discomfort
- `moderate` - Noticeable discomfort (default)
- `severe` - Significant pain/concern

**Frequency Options:**
- `once` - Single occurrence
- `occasional` - Sometimes (default)
- `frequent` - Often, multiple times
- `daily` - Every day
- `constant` - All the time, continuous

---

#### GET /api/symptoms/recent
Get recent symptoms for dashboard overview (protected).

**Headers:**
```
Authorization: Bearer <token>
```

**Query Parameters:**
- `limit` (optional): Number of recent symptoms (default: 10, max: 50)

**Example:**
```
GET /api/symptoms/recent?limit=5
```

**Response:**
```json
{
  "symptoms": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "symptom_type": "swelling",
      "description": "My feet are really swollen",
      "severity": "moderate",
      "frequency": "daily",
      "onset_time": "3 days ago",
      "associated_symptoms": ["back_pain"],
      "is_resolved": false,
      "reported_at": "2026-01-08T10:30:00Z",
      "resolved_at": null
    }
  ]
}
```

---

#### GET /api/symptoms/stats
Get symptom statistics and summary (protected).

**Headers:**
```
Authorization: Bearer <token>
```

**Response:**
```json
{
  "total_symptoms": 25,
  "ongoing": 8,
  "resolved": 17,
  "by_type": {
    "swelling": 5,
    "nausea": 8,
    "headache": 3,
    "back_pain": 9
  },
  "by_severity": {
    "mild": 10,
    "moderate": 12,
    "severe": 3
  }
}
```

**Use Cases:**
- Dashboard summary widget
- Doctor consultation preparation
- Health trend analysis

---

#### PUT /api/symptoms/:id/resolve
Mark a symptom as resolved (protected).

**Headers:**
```
Authorization: Bearer <token>
```

**URL Parameters:**
- `id` (required): Symptom UUID

**Example:**
```
PUT /api/symptoms/550e8400-e29b-41d4-a716-446655440000/resolve
```

**Response:**
```json
{
  "message": "Symptom marked as resolved"
}
```

**Notes:**
- Sets `is_resolved` to `true`
- Sets `resolved_at` to current timestamp
- Cannot resolve symptoms belonging to other users

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
- Language codes: `en` (English), `es` (Spanish), `fr` (French), `pt` (Portuguese), `de` (German)
- Priority levels: `low`, `medium`, `high`, `urgent`
- Small talk messages don't trigger AI or memory storage
- All chat messages are persisted to database
- Facts are extracted with confidence scores (0.0-1.0)
- Calendar suggestions require explicit user confirmation to create reminders
- Voice calls are a premium feature requiring Twilio configuration
- Voice responses use AWS Polly voices for natural speech
- Subscription quotas are tracked per feature per period (daily/weekly/monthly)
- Symptoms are automatically extracted from chat conversations when users report health concerns
- Symptom extraction uses pattern matching for 18+ common pregnancy symptoms
- Recent symptoms (last 5) are included in AI prompts for context-aware responses
- AI checks for dangerous symptom patterns and recommends urgent care when needed
- Symptom data is highly sensitive - never shared without explicit user consent
