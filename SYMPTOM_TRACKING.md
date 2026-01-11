# Symptom Tracking System - Implementation Guide

## Overview

The MomLaunchpad backend now includes comprehensive symptom tracking functionality that automatically extracts, stores, and provides context about pregnancy-related symptoms reported by users during conversations.

## 🎯 Key Features

### 1. **Automatic Symptom Extraction**
- AI-assisted extraction from natural language during chat
- Detects 18+ common pregnancy symptoms including:
  - Swelling/edema
  - Nausea/morning sickness
  - Headaches/migraines
  - Back pain
  - Vision changes
  - Bleeding/spotting
  - Contractions
  - And more...

### 2. **Rich Context Capture**
Each symptom record includes:
- **Type**: Category (e.g., "swelling", "nausea")
- **Description**: User's exact words
- **Severity**: mild/moderate/severe (auto-detected)
- **Frequency**: once/occasional/daily/constant
- **Onset Time**: When it started ("yesterday", "3 days ago")
- **Associated Symptoms**: Related symptoms mentioned together
- **Resolution Status**: ongoing vs resolved

### 3. **AI Context Integration**
Symptoms are automatically included in AI prompts:
- Last 5 symptoms shown to AI for pattern recognition
- Enables AI to spot dangerous combinations
- RED FLAG detection for urgent care recommendations
- Historical context for better personalized advice

### 4. **User/Doctor Access APIs**
Four REST endpoints for querying symptom history:

## 📡 API Endpoints

### 1. Get Symptom History
```http
GET /api/symptoms/history?limit=50&type=headache
Authorization: Bearer <jwt_token>
```

**Query Parameters:**
- `limit` (optional): Max results, default 50, max 200
- `type` (optional): Filter by symptom type (e.g., "headache", "swelling")

**Response:**
```json
{
  "symptoms": [
    {
      "id": "uuid",
      "symptom_type": "swelling",
      "description": "My feet are really swollen and puffy",
      "severity": "moderate",
      "frequency": "daily",
      "onset_time": "3 days ago",
      "associated_symptoms": ["back_pain"],
      "is_resolved": false,
      "reported_at": "2026-01-08T10:30:00Z",
      "resolved_at": null
    }
  ],
  "count": 1
}
```

### 2. Get Recent Symptoms (Dashboard View)
```http
GET /api/symptoms/recent?limit=10
Authorization: Bearer <jwt_token>
```

**Use Case:** Quick overview for user dashboard or doctor consultation prep

### 3. Get Symptom Statistics
```http
GET /api/symptoms/stats
Authorization: Bearer <jwt_token>
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

**Use Case:** Analytics dashboard, doctor consultation summary

### 4. Mark Symptom as Resolved
```http
PUT /api/symptoms/:id/resolve
Authorization: Bearer <jwt_token>
```

**Use Case:** User or doctor indicates symptom has been resolved

## 🔍 How It Works

### Automatic Extraction During Chat

1. **User sends message**: "My feet have been really swollen for the past 3 days"

2. **Intent classification**: Identified as `symptom_report`

3. **Symptom extraction**: 
   ```go
   {
     Type: "swelling",
     Description: "My feet have been really swollen for the past 3 days",
     Severity: "moderate",  // "really" detected
     Frequency: "daily",     // implied by "past 3 days"
     OnsetTime: "3 days ago",
     AssociatedSymptoms: []
   }
   ```

4. **Database save**: Automatically stored with timestamp

5. **AI context**: Next AI response includes this symptom in context

### AI Safety Features

The AI prompt now includes:
```
RECENT SYMPTOM HISTORY (important for tracking patterns):
- swelling (ongoing): moderate, daily - 3 days ago
- headache (resolved): mild, once - yesterday

IMPORTANT: Check for patterns or worsening symptoms that may require urgent attention.
If you see RED FLAGS (severe/frequent bleeding, severe headaches + vision changes, 
severe abdominal pain), advise immediate medical care.
```

## 🔒 Security & Privacy

- **JWT Protected**: All endpoints require authentication
- **User Isolation**: Users can only access their own symptoms
- **No PII in Logs**: Sensitive data sanitized before logging
- **Doctor Access**: Future feature - controlled sharing with healthcare providers

## 📊 Database Schema

```sql
CREATE TABLE symptoms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    symptom_type VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    severity VARCHAR(20),
    frequency VARCHAR(50),
    onset_time VARCHAR(100),
    associated_symptoms TEXT[],
    is_resolved BOOLEAN DEFAULT FALSE,
    reported_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Optimized indexes for common queries
CREATE INDEX idx_symptoms_user_reported ON symptoms(user_id, reported_at DESC);
CREATE INDEX idx_symptoms_type ON symptoms(symptom_type);
```

## 🚀 Integration Points

### Backend Components

1. **Symptom Tracker** (`internal/symptoms/tracker.go`)
   - Pattern matching for 18+ symptom types
   - Severity/frequency extraction
   - Onset time parsing

2. **Chat Engine** (`internal/chat/engine.go`)
   - Automatic extraction on symptom reports
   - Loads recent symptoms for AI context
   - Concurrent DB queries for performance

3. **Prompt Builder** (`internal/prompt/builder.go`)
   - Formats symptoms for AI context
   - Includes RED FLAG warnings
   - Tracks patterns across time

4. **API Handler** (`internal/api/symptoms.go`)
   - REST endpoints for symptom access
   - Stats calculation
   - Resolution tracking

5. **Database Queries** (`internal/db/queries.go`)
   - `SaveSymptom()` - Insert new symptom
   - `GetRecentSymptoms()` - Last N symptoms
   - `GetSymptomHistory()` - Full history with filters
   - `MarkSymptomResolved()` - Update resolution status

## 📱 Flutter Integration

### Symptom History Screen

```dart
// Fetch symptom history
final response = await dio.get(
  '/api/symptoms/history',
  queryParameters: {'limit': 50},
  options: Options(headers: {'Authorization': 'Bearer $token'}),
);

final symptoms = (response.data['symptoms'] as List)
    .map((s) => Symptom.fromJson(s))
    .toList();
```

### Dashboard Stats Widget

```dart
// Get symptom stats for dashboard
final stats = await dio.get(
  '/api/symptoms/stats',
  options: Options(headers: {'Authorization': 'Bearer $token'}),
);

// Display ongoing vs resolved, severity breakdown, etc.
```

### Mark Resolved Action

```dart
// User taps "Mark as Resolved" button
await dio.put(
  '/api/symptoms/$symptomId/resolve',
  options: Options(headers: {'Authorization': 'Bearer $token'}),
);
```

## 🎨 UI/UX Recommendations

### Symptom Timeline View
- Chronological list with date separators
- Color-coded by severity (green/yellow/red)
- Badge for "ongoing" vs "resolved"
- Tap to see full details

### Doctor Sharing (Future)
- Export symptom history as PDF
- Secure share link for doctors
- Privacy controls (what to share)

### Insights Dashboard
- "Most frequent symptom this month"
- "Symptoms that improved"
- "Patterns detected" (e.g., "headaches often follow nausea")

## 🧪 Testing

### Manual Testing

1. **Chat about symptoms**: "I've had swollen feet for 3 days"
2. **Check logs**: Look for "Saved symptom: swelling (ID: ...)"
3. **Query API**: `curl http://localhost:8080/api/symptoms/recent -H "Authorization: Bearer <token>"`
4. **Verify AI context**: Next chat should mention recent symptoms

### Automated Tests

Located in:
- `internal/symptoms/tracker_test.go` (extraction logic)
- `internal/chat/engine_test.go` (integration tests)
- `internal/api/symptoms_test.go` (API endpoint tests)

## 📈 Future Enhancements

1. **Doctor Dashboard**
   - Healthcare provider view of patient symptoms
   - Trend charts and analytics
   - Flagged urgent cases

2. **Pattern Detection**
   - ML-based pattern recognition
   - "Your headaches often occur with nausea"
   - Predictive alerts for potential complications

3. **Symptom Journaling**
   - Manual symptom entry (not just chat-based)
   - Photo attachments (for visual symptoms)
   - Severity slider, mood tracking

4. **Export & Sharing**
   - PDF export for doctor visits
   - Integration with EHR systems
   - Anonymous data for research (with consent)

## 📝 Logging & Monitoring

Look for these logs:
```
2026/01/08 10:30:15 Extracted 1 symptom(s) from message
2026/01/08 10:30:15 Saved symptom: swelling (ID: abc-123)
2026/01/08 10:30:16 Building prompt for user=xyz, intent=symptom_report
```

## 🔧 Configuration

No special configuration needed. Symptom tracking is:
- ✅ Always enabled
- ✅ Automatic on symptom reports
- ✅ Zero-config setup

## 🚨 Important Notes

1. **Not Medical Diagnosis**: This system tracks symptoms for user convenience and AI context, NOT for medical diagnosis
2. **Privacy Critical**: Symptom data is highly sensitive - never log, cache, or share without explicit consent
3. **Doctor Access**: Future feature requires HIPAA compliance considerations
4. **Data Retention**: Consider implementing data retention policies (e.g., delete after 1 year)

## ✅ Deployment Checklist

- [x] Database migration applied (`002_symptoms_tracking.up.sql`)
- [x] Symptom tracker integrated into chat engine
- [x] AI prompts include symptom history
- [x] REST APIs tested and documented
- [x] JWT protection on all endpoints
- [x] Logs sanitized for privacy
- [x] Docker container rebuilt and deployed

## 🆘 Troubleshooting

**Symptoms not being extracted?**
- Check logs for "Extracted N symptom(s)" message
- Verify intent is `symptom_report` or `pregnancy_question`
- Review symptom keyword patterns in `symptoms/tracker.go`

**API returning empty results?**
- Ensure JWT token is valid
- Check user has reported symptoms in chat
- Verify database migration applied

**AI not mentioning symptoms?**
- Check prompt includes "RECENT SYMPTOM HISTORY" section
- Verify `RecentSymptoms` populated in `PromptRequest`
- Check logs for "Building prompt with conversation state"

---

**Deployment Date**: January 8, 2026
**Status**: ✅ Production Ready
**Migration**: `002_symptoms_tracking.up.sql` applied
