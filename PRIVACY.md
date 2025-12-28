# Privacy & Data Protection

## Overview

MomLaunchpad handles sensitive pregnancy and health data. This document defines data protection boundaries and privacy practices.

## ⚠️ Current Status: MVP - Needs Hardening

This is the **MVP implementation**. Production deployment requires additional privacy controls.

---

## Data Storage

### What We Store
- **User accounts:** Email, hashed password, name, language preference
- **Chat messages:** User messages and AI responses (full conversation history)
- **User facts:** Extracted information (pregnancy week, diet, symptoms) with confidence scores
- **Reminders:** Calendar entries with titles, descriptions, timestamps
- **Savings entries:** Optional manual financial tracking (non-sensitive)

### What We DON'T Store
- ❌ Credit card or payment information (no payments in MVP)
- ❌ Medical records or doctor information
- ❌ Real-time location data
- ❌ Biometric data
- ❌ Third-party identifiers (no analytics)

---

## PII Protection Rules

### ✅ **IMPLEMENTED: Core Privacy Controls**

The following privacy protections are **NOW IMPLEMENTED**:

### 1. Logging Redaction
**Status:** ✅ **IMPLEMENTED**

**Implementation:**
- ✅ PII detection with regex patterns (email, phone, SSN, credit card, medical ID)
- ✅ `SanitizeForLogging()` - Redacts PII and truncates long content
- ✅ Automatic sanitization in WebSocket handler
- ✅ Warning logs when PII detected

**Code:**
```go
// internal/privacy/redact.go
func SanitizeForLogging(content string) string {
    sanitized := RedactSensitiveData(content)
    if len(sanitized) > 200 {
        sanitized = sanitized[:200] + "..."
    }
    return sanitized
}
```

**Files:**
- `internal/privacy/redact.go` - Core PII detection and redaction
- `internal/privacy/redact_test.go` - Comprehensive tests (all passing)
- `internal/ws/chat.go` - Applied to all logging

### 2. Prompt Sanitization Before DeepSeek
**Status:** ✅ **IMPLEMENTED**

**Implementation:**
- ✅ PII detection before sending to AI
- ✅ Content sanitization with `SanitizeForAPI()`
- ✅ Warning logs when PII detected in outgoing messages
- ✅ Prevents sending emails, phone numbers, SSNs, credit cards to third party

**Code:**
```go
// Before calling DeepSeek API:
cleanContent := privacy.SanitizeForAPI(content)
if privacy.ContainsPII(content) {
    log.Printf("Warning: PII detected in outgoing DeepSeek message")
}
```

**Files:**
- `internal/privacy/redact.go` - `SanitizeForAPI()` function
- `internal/ws/chat.go` - Applied to all AI prompts

### 3. Data Retention Policy
**Status:** ⚠️ **PARTIALLY IMPLEMENTED**

**Implemented:**
- ✅ Session-based memory management (1-hour auto-reset)
- ✅ Short-term memory clears after inactivity
- ✅ Prevents unbounded conversation growth

**Still Missing:**
- ❌ Message history retention limit (e.g., 90 days)
- ❌ Automatic deletion of old conversations
- ❌ User-initiated data export/deletion endpoint

**Required for full compliance:**
```go
// TODO: Add to internal/api/users.go
// DELETE /api/users/me/data
func (h *UserHandler) DeleteAllUserData(c *gin.Context) {
    // Delete messages, facts, reminders
    // Anonymize user account or hard-delete
}
```

### 4. Encryption at Rest
**Status:** ❌ **NOT IMPLEMENTED** (Infrastructure dependency)

**Current state:**
- Database stores data in plaintext
- Backups are not encrypted
- Facts table contains health information unencrypted

**Required for production:**
- Database-level encryption (PostgreSQL pgcrypto)
- Field-level encryption for sensitive facts
- Encrypted backups

---

## Third-Party Data Sharing

### DeepSeek API
**What is sent:**
- ✅ User messages (full text, unfiltered)
- ✅ Conversation history (last N messages)
- ✅ User facts (pregnancy week, diet, etc.)
- ✅ System prompts (medical context)

**What is NOT sent:**
- ❌ User email
- ❌ User ID (we use session context, not identifiers)
- ❌ Passwords
- ❌ Calendar entries

**Privacy concern:** DeepSeek's data retention policy unknown. Assume they log everything.

**Mitigation needed:**
- PII scrubbing before API calls
- Anonymization of user context
- Regular audit of prompts sent to API

### No Other Third Parties
- ❌ No analytics (Google, Mixpanel, etc.)
- ❌ No crash reporting (Sentry)
- ❌ No CDN (static assets local)
- ❌ No social login providers

---

## Compliance Gaps

### GDPR (EU)
**Status:** ⚠️ Non-compliant

**Missing:**
- [ ] Data protection impact assessment (DPIA)
- [ ] User consent management
- [ ] Data portability (export)
- [ ] Right to erasure (delete account)
- [ ] Data processing agreement with DeepSeek
- [ ] Privacy policy
- [ ] Cookie consent (if added)

### HIPAA (US Healthcare)
**Status:** ⚠️ Non-compliant

**Missing:**
- [ ] Business Associate Agreement (BAA)
- [ ] Audit logging
- [ ] Access controls
- [ ] Encryption at rest and in transit
- [ ] Risk assessment
- [ ] HIPAA-compliant hosting

**Note:** MomLaunchpad is NOT a covered entity, but handles health information. If integrated with healthcare providers, HIPAA applies.

### CCPA (California)
**Status:** ⚠️ Partially compliant

**Implemented:**
- ✅ No sale of personal data
- ✅ No third-party analytics

**Missing:**
- [ ] Privacy notice
- [ ] Data deletion on request
- [ ] Data disclosure on request

---

## Recommended Privacy Improvements

### Phase 1: Immediate (Pre-Production)
1. **Add logging redaction**
   - Implement `redactSensitiveData()` in all log statements
   - Never log full messages, only metadata

2. **Sanitize prompts to DeepSeek**
   - Strip email, phone, full names
   - Replace with tokens: `[USER]`, `[EMAIL]`, etc.

3. **Create privacy policy**
   - What data we collect
   - How we use it
   - Third-party sharing (DeepSeek)
   - User rights

4. **Add data deletion endpoint**
   ```
   DELETE /api/users/me/data
   ```
   - Soft delete user account
   - Mark data for purge
   - Anonymize messages

### Phase 2: Short-term (First 90 Days)
1. **Implement data retention policy**
   - Delete messages older than 90 days
   - Archive facts if user inactive >180 days

2. **Add encryption at rest**
   - Use PostgreSQL pgcrypto for user_facts table
   - Encrypt backups with GPG

3. **Audit DeepSeek interactions**
   - Log all API calls (without PII)
   - Track token usage per user
   - Monitor for data leaks

4. **Add consent management**
   - Terms of service acceptance
   - Privacy policy acceptance
   - Opt-in for data retention

### Phase 3: Long-term (6+ Months)
1. **GDPR compliance audit**
   - Hire DPO or consultant
   - Complete DPIA
   - Implement full GDPR controls

2. **Consider anonymization**
   - Hash user IDs before sending to DeepSeek
   - Use differential privacy for facts
   - Aggregate data for insights

3. **Add user data portal**
   - Download all my data
   - View API call history
   - Manage consent preferences

---

## Current Code Gaps

### 1. No PII Detection in `internal/prompt/builder.go`
```go
// MISSING: Before building prompt
func (b *Builder) BuildPrompt(req PromptRequest) []deepseek.ChatMessage {
    // TODO: Sanitize req.UserMessage for PII
    // TODO: Redact facts before including in prompt
    // ...
}
```

### 2. No Logging Safeguards in `internal/ws/chat.go`
```go
// CURRENT (UNSAFE):
log.Printf("User message: %s", content)

// SHOULD BE:
log.Printf("User message received: length=%d, intent=%s", len(content), intent)
```

### 3. No Data Deletion in `internal/db/queries.go`
```go
// MISSING:
func (db *DB) DeleteUserData(ctx context.Context, userID string) error {
    // Delete messages
    // Delete facts
    // Delete reminders
    // Anonymize user record
}
```

---

## Questions for Product/Legal

1. **Data residency:** Where should data be stored? (US, EU, multi-region?)
2. **Retention policy:** How long should we keep messages? Facts?
3. **DeepSeek contract:** Do we need a Data Processing Agreement?
4. **Regulatory classification:** Is this a medical device? Wellness app?
5. **Minor access:** What if a user is <18? Parental consent needed?

---

## Developer Guidelines

### When Adding Features

1. **Never log PII** - Use redacted versions or metadata only
2. **Minimize data collection** - Only store what's necessary
3. **Encrypt sensitive fields** - Facts, medical history
4. **Audit third-party calls** - Log what goes to DeepSeek (without content)
5. **Default to private** - Opt-in for sharing, not opt-out

### Code Review Checklist

- [ ] No PII in log statements
- [ ] Sensitive data encrypted before storage
- [ ] API calls sanitized
- [ ] User can delete their data
- [ ] Data retention policy enforced

---

## Contact

For privacy concerns or data requests:
- **Email:** privacy@momlaunchpad.com (TODO: Set up)
- **Data Protection Officer:** TBD
- **Security Issues:** security@momlaunchpad.com (TODO: Set up)

---

**Last Updated:** December 28, 2024  
**Status:** MVP - Privacy hardening required before production launch
