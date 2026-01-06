# Twilio Voice Integration

## Overview

Premium users can now call a Twilio phone number to access the MomLaunchpad AI assistant via voice. The system automatically:
- Transcribes user speech to text
- Processes the message through the same chat engine as WebSocket
- Converts AI response to speech
- Manages conversation continuity within the call

## Setup

### 1. Twilio Account

1. Sign up at [https://www.twilio.com/](https://www.twilio.com/)
2. Purchase a phone number with Voice capabilities
3. Get your credentials from [https://console.twilio.com](https://console.twilio.com):
   - Account SID
   - Auth Token
   - Phone Number

### 2. Environment Variables

Add to your `.env`:

```bash
TWILIO_ACCOUNT_SID=ACxxxxxxxxxxxxxxxxxxxxxxxxxxxx
TWILIO_AUTH_TOKEN=your_auth_token_here
TWILIO_PHONE_NUMBER=+1234567890
```

### 3. Webhook Configuration

In Twilio Console, configure your phone number webhooks:

**Voice Configuration:**
- When a call comes in: `https://your-domain.com/api/voice/incoming` (HTTP POST)
- Status callback URL: `https://your-domain.com/api/voice/status` (HTTP POST)

**Important:** Use HTTPS in production. Twilio requires secure endpoints.

### 4. Database Migration

The `voice_calls` feature is automatically added to the `features` table during migration. Make sure your migration has been applied:

```bash
make migrate-up
```

### 5. Assign Premium Plan

Voice calls are a premium feature. Users need to be on the premium plan:

```sql
-- Assign premium plan to user
UPDATE subscriptions 
SET plan_id = (SELECT id FROM plans WHERE code = 'premium')
WHERE user_id = 'user-uuid-here';
```

## How It Works

### Call Flow

1. **User calls** → Twilio receives call → Webhook to `/api/voice/incoming`
2. **User lookup** → System identifies user by phone number (matches display_name or email)
3. **Premium check** → System verifies user has `voice_calls` feature access
4. **Greeting** → TwiML responds with welcome message
5. **Speech gathering** → Twilio listens for user speech
6. **Processing** → Speech transcribed → Sent to chat engine → AI generates response
7. **Response** → AI response converted to speech and played to user
8. **Continuation** → User can ask follow-up questions
9. **Hang up** → Session cleaned up automatically

### User Identification

Currently, user lookup is done by matching phone number to `display_name` or `email` fields. For production, consider:

1. **Add phone_number column:**
```sql
ALTER TABLE users ADD COLUMN phone_number VARCHAR(20);
CREATE INDEX idx_users_phone ON users(phone_number);
```

2. **Update voice.go:**
```go
// In getUserByPhone method
query := `SELECT ... FROM users WHERE phone_number = $1`
```

3. **User registration:** Collect phone number during signup

### Session Management

- Each call creates a `VoiceSession` stored in memory
- Sessions are automatically cleaned up when call ends
- Conversation history is maintained during the call
- Short-term memory from previous text chats is NOT included (voice is isolated)

## API Endpoints

### POST /api/voice/incoming

Twilio webhook for incoming calls.

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

### POST /api/voice/gather

Twilio webhook for speech recognition results.

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

### POST /api/voice/status

Twilio webhook for call status updates (optional).

**Request:** Form data from Twilio
- `CallSid`: Unique call identifier
- `CallStatus`: New status (completed, failed, etc.)

**Response:** Plain text "OK"

## Multilingual Support

The system automatically detects user's preferred language from their profile and:
- Uses appropriate Polly voice (Joanna for English, Lupe for Spanish, etc.)
- Sets correct Twilio language code for speech recognition
- Maintains language consistency throughout the call

### Supported Languages

| Language | Voice | Twilio Code |
|----------|-------|-------------|
| English  | Polly.Joanna | en-US |
| Spanish  | Polly.Lupe | es-ES |
| French   | Polly.Celine | fr-FR |
| Portuguese | Polly.Vitoria | pt-BR |
| German   | Polly.Vicki | de-DE |

Add more in `pkg/twilio/voice.go`:
```go
func GetVoiceForLanguage(language string) string {
    voices := map[string]string{
        "en": "Polly.Joanna",
        "es": "Polly.Lupe",
        "it": "Polly.Carla", // Add Italian
        // ...
    }
    // ...
}
```

## Security

### Webhook Validation

Twilio signs all webhook requests. While validation is implemented (`ValidateRequest` method), it's currently not enforced in production handlers. To enable:

```go
// In voice.go HandleIncoming
signature := c.GetHeader("X-Twilio-Signature")
url := c.Request.URL.String()

// Parse form to map
params := make(map[string]string)
for k, v := range c.Request.Form {
    params[k] = v[0]
}

if !h.twilioClient.ValidateRequest(url, params, signature) {
    c.String(http.StatusForbidden, "Invalid signature")
    return
}
```

### Rate Limiting

Voice webhooks are public (no JWT), but:
- User lookup enforces registration
- Premium feature gate blocks free users
- Twilio rate limits incoming calls naturally
- Consider IP-based rate limiting for webhook endpoints

## Cost Management

### Twilio Pricing (US, approximate)
- Incoming calls: $0.0085/min
- Speech Recognition: $0.02/min
- Text-to-Speech (Polly): $0.004/character

### Example Costs
- 5-minute conversation: ~$0.08
- 1000 calls/month (5 min avg): ~$80/month

### Optimization Tips

1. **Limit call duration:** Add max duration in TwiML
2. **Concise responses:** Keep AI responses brief for voice
3. **Smart timeouts:** 5-second timeout prevents long silences
4. **Session cleanup:** Auto-cleanup prevents memory leaks

## Testing

### Local Testing with ngrok

Twilio requires HTTPS webhooks. Use ngrok for local development:

```bash
# Install ngrok
brew install ngrok  # macOS
# or download from https://ngrok.com/

# Start your server
make dev

# In another terminal, start ngrok
ngrok http 8080

# Copy the HTTPS URL (e.g., https://abc123.ngrok.io)
# Configure Twilio webhook: https://abc123.ngrok.io/api/voice/incoming
```

### Test Call Flow

1. Start server with Twilio credentials
2. Configure Twilio webhooks to your ngrok URL
3. Call your Twilio number
4. Verify:
   - Greeting plays
   - Speech recognition works
   - AI response is spoken
   - Follow-up questions work
   - Hangup cleans up session

### Mock Twilio for Unit Tests

```go
// In voice_test.go
func TestVoiceHandler(t *testing.T) {
    // Mock Twilio client
    mockTwilio := &MockTwilioClient{}
    
    // Mock chat engine
    mockEngine := &MockChatEngine{}
    
    handler := api.NewVoiceHandler(mockTwilio, mockEngine, mockDB)
    
    // Test incoming call
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    c.Request = httptest.NewRequest("POST", "/api/voice/incoming", nil)
    
    handler.HandleIncoming(c)
    
    assert.Equal(t, 200, w.Code)
    assert.Contains(t, w.Body.String(), "<Response>")
}
```

## Troubleshooting

### User Not Found

**Error:** "User not registered"

**Fix:** Ensure user's phone number matches `display_name` or `email` in database, or add `phone_number` column.

### No Speech Detected

**Error:** Twilio keeps saying "I didn't catch that"

**Possible causes:**
- Poor connection
- Background noise
- Incorrect language code
- User not speaking clearly

**Fix:** Increase timeout, improve prompts, test with different devices

### Circuit Breaker Open

If DeepSeek API is down, circuit breaker opens and fallback responses are used:

```go
// In chat/engine.go
if e.circuitBreaker.State() == circuitbreaker.StateOpen {
    fbResp := fallback.GetCircuitOpenResponse(req.Language)
    // Returns canned response instead of hanging up
}
```

### Session Not Found

**Error:** "Session expired. Please call again."

**Cause:** Session was cleaned up (call ended) but Gather callback arrived late

**Fix:** Increase session timeout or handle gracefully with retry prompt

## Production Checklist

- [ ] Add `phone_number` column to users table
- [ ] Enable webhook signature validation
- [ ] Use production Twilio account (not trial)
- [ ] Configure HTTPS endpoints with valid certificate
- [ ] Set up monitoring for failed calls
- [ ] Implement IP rate limiting on webhook endpoints
- [ ] Add call duration limits
- [ ] Monitor Twilio costs
- [ ] Test all supported languages
- [ ] Configure proper error handling
- [ ] Set up logging and alerting

## Future Enhancements

- [ ] SMS notifications for call transcripts
- [ ] Call recording (with user consent)
- [ ] Voicemail support
- [ ] Conference calls with healthcare providers
- [ ] Multi-party calls (partner support)
- [ ] Interactive voice menu (IVR)
- [ ] Call analytics and insights
- [ ] Automatic call-back system

## Architecture

```
┌─────────────┐
│   User      │
│   Phone     │
└──────┬──────┘
       │ Call
       ▼
┌─────────────────┐
│   Twilio        │ ◄── Webhook: /api/voice/incoming
│   Phone Number  │ ◄── Webhook: /api/voice/gather
└────────┬────────┘ ◄── Webhook: /api/voice/status
         │
         ▼
┌──────────────────────────────────┐
│  MomLaunchpad Backend            │
│                                  │
│  ┌────────────────────────────┐ │
│  │  Voice Handler             │ │
│  │  - User lookup             │ │
│  │  - Session management      │ │
│  │  - TwiML generation        │ │
│  └──────────┬─────────────────┘ │
│             │                    │
│             ▼                    │
│  ┌────────────────────────────┐ │
│  │  Chat Engine               │ │
│  │  - Intent classification   │ │
│  │  - Memory management       │ │
│  │  - AI processing           │ │
│  └──────────┬─────────────────┘ │
│             │                    │
│             ▼                    │
│  ┌────────────────────────────┐ │
│  │  DeepSeek API              │ │
│  └────────────────────────────┘ │
└──────────────────────────────────┘
```

## Resources

- [Twilio Voice Docs](https://www.twilio.com/docs/voice)
- [TwiML Reference](https://www.twilio.com/docs/voice/twiml)
- [Twilio Go SDK](https://github.com/twilio/twilio-go)
- [AWS Polly Voices](https://docs.aws.amazon.com/polly/latest/dg/voicelist.html)
