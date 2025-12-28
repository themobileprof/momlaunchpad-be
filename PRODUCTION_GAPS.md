# Production Readiness Gaps

This document explicitly lists features **NOT YET IMPLEMENTED** that are critical for production deployment.

## 🚨 Blockers Before Production

### 1. Rate Limiting & Abuse Control
**Status:** ✅ **IMPLEMENTED**

**Implementation:**
- ✅ Per-IP rate limiting: 100 req/min (burst 200) - `internal/api/middleware/ratelimit.go`
- ✅ Per-user rate limiting: 500/hour (burst 100) - Applied to calendar routes
- ✅ WebSocket flood protection: 10 msg/min - `internal/ws/chat.go`
- ✅ Automatic cleanup of stale limiters (every 5 minutes)
- ✅ Uses `golang.org/x/time/rate` (token bucket algorithm)

**Configuration:**
```go
// Global IP rate limiting
router.Use(middleware.PerIP(100, 200, time.Minute))

// Per-user rate limiting on protected routes
calendar.Use(middleware.PerUser(500, 100, time.Hour))

// WebSocket rate limiting
wsLimiter := middleware.NewWebSocketLimiter(10, time.Minute)
```

**Files:**
- `internal/api/middleware/ratelimit.go` - Rate limiter implementation
- `cmd/server/main.go` - Applied to routes
- `internal/ws/chat.go` - WebSocket message limiting

---

### 2. LLM Failure Handling
**Status:** ✅ **IMPLEMENTED**

**Implementation:**
- ✅ Context timeout (30 seconds) - `internal/ws/chat.go`
- ✅ Malformed chunk validation - Checks for empty choices array
- ✅ Fallback response system - Intent-based, multilingual (EN/ES)
- ✅ Circuit breaker pattern - 5 failures → 5-minute cooldown
- ✅ Timeout-specific fallback messages
- ✅ Emergency handling for symptom reports

**Configuration:**
```go
// Timeout on AI calls
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()

// Circuit breaker protection
if aiCircuitBreaker.State() == circuitbreaker.StateOpen {
    fallbackMsg := fallback.GetCircuitOpenResponse(classifierResult.Intent, "en")
    // Return fallback instead of calling AI
}

// Chunk validation
if len(chunk.Choices) == 0 {
    log.Println("Warning: Malformed chunk from DeepSeek")
    continue
}
```

**Fallback Response Examples:**
- **Timeout (pregnancy question):** "I'm taking longer than usual. Please try rephrasing your question."
- **Circuit open (symptom):** "I'm having technical difficulties. If this is urgent, please contact your healthcare provider immediately."
- **General error:** Intent-specific responses with retry guidance

**Files:**
- `internal/circuitbreaker/breaker.go` - Circuit breaker implementation
- `internal/fallback/responses.go` - Intent-based fallback responses
- `internal/ws/chat.go` - Integrated all failure handling

---

### 3. Conversation Lifecycle Management
**Status:** ⚠️ Implicit, not explicit

**Current state:**
```go
// Memory manager has size limit
memMgr := memory.NewMemoryManager(10) // Last 10 messages

// But no rules for:
// - When to clear history
// - Time-based expiry
// - Session boundaries
```

**What needs documentation/implementation:**

#### A. Session Definition
```
Q: What defines a "conversation"?
A: Currently undefined

Should be:
- New session after 1 hour of inactivity
- Max 100 messages per session
- Explicit "end conversation" command
- New session after major intent shift
```

#### B. Memory Trimming Rules
```go
// TODO: Document and implement
type MemoryPolicy struct {
    MaxMessages     int           // 10 (current)
    MaxAge          time.Duration // TODO: Add time-based trimming
    TrimOnNewIntent bool          // TODO: Clear on intent shift?
    
    // Example: Should small talk be kept in memory?
    // Current: Yes (but filtered from prompts)
    // Better: No (don't store at all)
}
```

#### C. Fact Lifecycle
```go
// Current: Facts stored forever
// TODO: Add expiration rules

type UserFact struct {
    Key        string
    Value      string
    Confidence float64
    UpdatedAt  time.Time
    ExpiresAt  *time.Time // NEW: Optional expiry
}

// Example expiry rules:
// - "current_symptom" expires after 7 days
// - "pregnancy_week" never expires (until birth)
// - "diet_preference" expires after 90 days
```

#### D. Conversation Boundaries
**Status:** ✅ **IMPLEMENTED**

**Implementation:**
- ✅ Time-based reset after 1 hour inactivity
- ✅ Automatic check on every new message
- ✅ Short-term memory cleared when reset triggered
- ✅ Long-term facts (pregnancy stage, etc.) persist

**Configuration:**
```go
// In internal/memory/manager.go
func (m *MemoryManager) ShouldResetSession(userID string) bool {
    userMem := m.users[userID]
    if len(userMem.ShortTerm) == 0 {
        return false
    }
    
    lastMsg := userMem.ShortTerm[len(userMem.ShortTerm)-1]
    
    // Reset after 1 hour of inactivity
    if time.Since(lastMsg.Timestamp) > time.Hour {
        return true
    }
    
    return false
}
```

**Session Lifecycle:**
- User sends message after 1+ hour inactivity → short-term memory clears
- Pregnancy facts (long-term) remain available
- New conversation context starts fresh
- Prevents super-prompt from growing unbounded

**Files:**
- `internal/memory/manager.go` - Added `ShouldResetSession()` method
- `internal/ws/chat.go` - Checks reset condition before building prompt

---

### 4. Admin Language Workflow
**Status:** ⚠️ Partially documented

**Current state:**
```go
// Languages load from database
languages, err := database.GetEnabledLanguages(ctx)

// Admin can add via direct SQL:
INSERT INTO languages (code, name, native_name, is_enabled, is_experimental)
VALUES ('fr', 'French', 'Français', true, true);
```

**What's missing:**

#### A. Admin API Endpoints
```go
// TODO: Add to internal/api/admin.go

// POST /api/admin/languages
func (h *AdminHandler) AddLanguage(c *gin.Context) {
    var req struct {
        Code           string `json:"code"`
        Name           string `json:"name"`
        NativeName     string `json:"native_name"`
        IsExperimental bool   `json:"is_experimental"`
    }
    // Validate language code (ISO 639-1)
    // Insert into database
    // Reload language manager
}

// PUT /api/admin/languages/:code
func (h *AdminHandler) UpdateLanguage(c *gin.Context) {
    // Enable/disable
    // Mark as experimental or stable
}

// GET /api/admin/languages
func (h *AdminHandler) ListLanguages(c *gin.Context) {
    // Return all languages with stats
    // - User count per language
    // - Message count per language
}
```

#### B. Prompt Translation Strategy
```
Q: Are prompts translated or language-specific?
A: Currently language-specific (hardcoded in builder.go)

Options:
1. Hardcoded per language (current)
   Pros: Full control, culturally appropriate
   Cons: Doesn't scale, requires code changes

2. Database-driven templates
   Pros: Admin can edit without deploy
   Cons: Risk of bad translations

3. AI-assisted translation
   Pros: Fast, scales to many languages
   Cons: Quality concerns, cost

Recommended: Hybrid
- Core system prompts: Hardcoded (reviewed by native speakers)
- User-facing canned responses: Database-driven
- Admin can override per language
```

#### C. Language Validation
```go
// TODO: Add to internal/language/manager.go
func (m *Manager) ValidateLanguageCode(code string) error {
    // Check ISO 639-1 format
    if !regexp.MustCompile(`^[a-z]{2}$`).MatchString(code) {
        return errors.New("invalid language code format")
    }
    
    // Check not duplicate
    if _, exists := m.languages[code]; exists {
        return errors.New("language already exists")
    }
    
    return nil
}
```

#### D. Missing Language Features
- [ ] Language-specific medical term glossaries
- [ ] Language-specific symptom keywords for classifier
- [ ] Language-specific calendar reminder templates
- [ ] User language preference history (track changes)

---

### 5. Calendar Write Authority
**Status:** ⚠️ Implicit, not explicit

**Current behavior:**
```go
// In internal/ws/chat.go
if shouldSuggest := h.calSuggester.ShouldSuggest(result.Intent, content); shouldSuggest.ShouldSuggest {
    suggestion := h.calSuggester.BuildSuggestion(result.Intent, content)
    // Sends suggestion to client
    if err := h.sendCalendarSuggestion(conn, suggestion); err != nil {
        return err
    }
}

// Client must explicitly call POST /api/reminders to create
```

**What's unclear:**

#### A. AI Authority Levels
```
Q: Can AI auto-create calendar entries?
A: NO (by design)

But this should be documented in:
- API documentation ✅ (mentioned)
- WebSocket flow ✅ (shows suggestion only)
- BACKEND_SPEC.md ✅ (states "suggest, never create")
- Frontend integration docs ❌ (missing)
```

#### B. Confirmation Rules
```go
// TODO: Add explicit confirmation tracking

type ReminderSuggestion struct {
    ID               string    // Track suggestion
    SuggestedAt      time.Time
    ExpiresAt        time.Time // Suggestion valid for 5 minutes
    ConfirmedByUser  bool
    CreatedReminder  *string   // Link to created reminder
}

// Workflow:
// 1. AI suggests → Store suggestion with expiry
// 2. User confirms → POST /api/reminders with suggestion_id
// 3. Backend validates suggestion hasn't expired
// 4. Create reminder and link to suggestion
```

#### C. Trust & Safety Considerations
```
Why AI can't auto-create:

1. User autonomy
   - Pregnancy is personal
   - Users must control their calendar

2. Medical liability
   - AI might suggest wrong times
   - Critical appointments need human verification

3. Spam prevention
   - Rogue AI could flood calendar
   - No undo mechanism (yet)

4. Notification fatigue
   - Too many reminders → ignored
   - Quality over quantity
```

#### D. Future Enhancement: Smart Defaults
```go
// Possible future feature (NOT MVP):
type ReminderPreferences struct {
    AutoCreateLowPriority  bool // Auto-create "low" reminders
    RequireConfirmForUrgent bool // Always ask for "urgent"
    DefaultReminderTime    string // e.g., "09:00"
    EnableSmartScheduling  bool // Avoid conflicts
}

// This requires:
// - User preferences table
// - Preference management API
// - Rollback mechanism (undo auto-created)
```

---

## 🟡 Nice-to-Have (Post-MVP)

### 6. Session Management
- Redis-based session store (currently in-memory)
- WebSocket reconnection with state recovery
- Multi-device session synchronization

### 7. Audit Logging
- Track all data access (who, what, when)
- Immutable audit trail
- Required for HIPAA compliance

### 8. Backup & Disaster Recovery
- Automated database backups
- Point-in-time recovery
- Failover testing

### 9. Observability
- Prometheus metrics
- OpenTelemetry tracing
- Error rate monitoring
- User flow analytics (privacy-safe)

### 10. Performance Optimization
- Query optimization (database indexes exist, but not tuned)
- Response caching (Redis)
- Connection pooling tuning
- Load testing results

---

## Implementation Priority

### Pre-Launch (Blockers)
1. **Rate limiting** - Prevents abuse, required before public access
2. **LLM fallback** - User experience during failures
3. **Privacy controls** - See [PRIVACY.md](PRIVACY.md)
4. **Data deletion** - Legal requirement in many jurisdictions

### First 30 Days
1. Conversation lifecycle rules (document + implement)
2. Admin language API (enable growth)
3. Circuit breaker for DeepSeek
4. Basic audit logging

### First 90 Days
1. Session management improvements
2. Backup automation
3. Monitoring & alerting
4. Load testing

### 6+ Months
1. Advanced rate limiting (per-feature)
2. AI-assisted fallbacks (use cheaper model when main fails)
3. Multi-region deployment
4. Full GDPR compliance

---

## Testing Gaps

These scenarios **have no tests yet:**

- [ ] WebSocket flood (send 100 messages in 1 second)
- [ ] DeepSeek timeout (mock slow response)
- [ ] Malformed DeepSeek response (invalid JSON)
- [ ] Session expiry during active conversation
- [ ] Concurrent writes to same user facts
- [ ] Language fallback with missing translations
- [ ] Calendar suggestion rate limiting

---

## Documentation Gaps

Need additional docs:

- [ ] **INTEGRATION.md** - How frontend should integrate
- [ ] **DEPLOYMENT.md** - Production deployment guide
- [ ] **MONITORING.md** - What to monitor and alert on
- [ ] **RUNBOOK.md** - Incident response procedures
- [ ] **CHANGELOG.md** - Version history

---

**Last Updated:** December 28, 2024  
**Next Review:** Before production launch
