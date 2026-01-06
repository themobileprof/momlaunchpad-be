# Quota & Subscription System - Complete Implementation

## Overview
The quota and subscription system provides:
- **Feature gating** - Control access to features by subscription plan
- **Quota tracking** - Limit usage per feature with daily/weekly/monthly periods
- **Usage monitoring** - Track consumption and enforce limits
- **Admin management** - Manage plans, quotas, and user subscriptions

## Components

### 1. Database Schema
**Tables:**
- `plans` - Subscription plans (free, premium, etc.)
- `features` - Available features (chat, calendar, savings)
- `plan_features` - Features included in each plan with quota limits
- `subscriptions` - User subscriptions to plans
- `feature_usage` - Tracks usage consumption per user/feature/period

**Key Columns:**
- `plan_features.quota_limit` - Max usage (NULL = unlimited)
- `plan_features.quota_period` - Period type (daily/weekly/monthly/unlimited)
- `feature_usage.usage_count` - Current usage in period
- `feature_usage.period_end` - When period resets

### 2. Subscription Manager (`internal/subscription/manager.go`)

**Core Methods:**
```go
// Feature access check
HasFeature(ctx, userID, featureKey) (bool, error)

// Quota checking
CheckQuota(ctx, userID, featureCode) (bool, error)
IncrementUsage(ctx, userID, featureCode) error
GetQuotaInfo(ctx, userID, featureCode) (*QuotaInfo, error)

// User queries
GetUserFeatures(ctx, userID) ([]UserFeature, error)
GetActiveSubscription(ctx, userID) (*Subscription, error)

// Admin operations
ListPlans(ctx) ([]Plan, error)
UpdateUserPlan(ctx, userID, planCode) error
ResetQuota(ctx, userID, featureCode) error
```

**Period Calculations:**
- Daily: Midnight to midnight
- Weekly: Monday to Sunday
- Monthly: 1st to last day of month
- Unlimited: No enforcement

### 3. Middleware (`internal/api/middleware/feature_gate.go`)

**RequireFeature** - Blocks access if feature not in user's plan
```go
router.Use(middleware.RequireFeature(subMgr, "calendar"))
```

**CheckQuota** - Blocks access if quota exceeded
```go
router.Use(middleware.CheckQuota(subMgr, "chat"))
```

**Response Codes:**
- `200 OK` - Access granted
- `401 Unauthorized` - No user ID in context
- `403 Forbidden` - Feature not available in plan
- `429 Too Many Requests` - Quota exceeded
- `500 Internal Server Error` - System error

### 4. API Endpoints (`internal/api/subscription.go`)

#### User Endpoints (Protected)

**GET /api/subscription/me**
Returns active subscription details
```json
{
  "subscription": {
    "id": 1,
    "plan_code": "free",
    "plan_name": "Free",
    "status": "active",
    "starts_at": "2026-01-01T00:00:00Z",
    "ends_at": null
  }
}
```

**GET /api/subscription/features**
Lists all features available to user
```json
{
  "features": [
    {
      "feature_key": "chat",
      "name": "Chat Access",
      "description": "AI chat support",
      "quota_limit": 100,
      "quota_period": "monthly"
    }
  ]
}
```

**GET /api/subscription/quota/:feature**
Get quota status for a feature
```json
{
  "feature": "chat",
  "has_access": true,
  "within_quota": true,
  "quota_limit": 100,
  "quota_used": 42,
  "quota_period": "monthly",
  "period_end": "2026-02-01T00:00:00Z"
}
```

#### Admin Endpoints (Protected + Admin Only)

**GET /api/admin/plans**
List all subscription plans

**GET /api/admin/users/:userId/subscription**
Get user's subscription details

**PUT /api/admin/users/:userId/plan**
Change user's plan
```json
{
  "plan_code": "premium"
}
```

**GET /api/admin/users/:userId/quota/:feature**
Get user's quota usage for a feature

**POST /api/admin/users/:userId/quota/:feature/reset**
Reset user's quota (e.g., for support)

**GET /api/admin/quota/stats**
System-wide quota statistics (coming soon)

**POST /api/admin/users/:userId/features**
Grant temporary feature access (coming soon)

### 5. WebSocket Integration (`internal/ws/chat.go`)

**Flow:**
1. User connects to `/ws/chat`
2. JWT validated
3. For each message:
   - Check quota: `CheckQuota(userID, "chat")`
   - If exceeded: Send error, don't process
   - Process message via engine
   - Increment usage: `IncrementUsage(userID, "chat")`

**Error Messages:**
- Quota exceeded: `"You've reached your message quota for this period..."`
- System error: Logged, generic error to user

## Configuration

### Default Plans (from migration)

**Free Plan:**
- Chat: 100 messages/month
- Calendar: Enabled (unlimited)
- Savings: Disabled

**To Add More Plans:**
```sql
-- Insert premium plan
INSERT INTO plans (code, name, description, active)
VALUES ('premium', 'Premium', 'Full access', TRUE);

-- Add features
INSERT INTO plan_features (plan_id, feature_id, quota_limit, quota_period)
SELECT 
  (SELECT id FROM plans WHERE code = 'premium'),
  id,
  NULL,  -- Unlimited
  'unlimited'
FROM features
WHERE feature_key IN ('chat', 'calendar', 'savings');
```

## Integration Guide

### 1. Add Feature Gate to Route
```go
// In cmd/server/main.go
router.Use(middleware.RequireFeature(subMgr, "my_feature"))
```

### 2. Add Quota Check (if needed)
```go
// For rate-limited features
router.Use(middleware.CheckQuota(subMgr, "my_feature"))
```

### 3. Track Usage
```go
// After successful operation
if err := subMgr.IncrementUsage(ctx, userID, "my_feature"); err != nil {
    log.Printf("Failed to increment usage: %v", err)
}
```

### 4. Add Feature to Database
```sql
-- Add feature
INSERT INTO features (feature_key, name, description)
VALUES ('my_feature', 'My Feature', 'Description');

-- Add to plans with quotas
INSERT INTO plan_features (plan_id, feature_id, quota_limit, quota_period)
SELECT 
  p.id, 
  f.id,
  1000,  -- quota limit
  'monthly'  -- period
FROM plans p, features f
WHERE p.code = 'free' AND f.feature_key = 'my_feature';
```

## Testing

### Unit Tests
```bash
# Subscription manager
go test -v ./internal/subscription/

# Middleware
go test -v ./internal/api/middleware/
```

### Manual Testing

**Check quota status:**
```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/subscription/quota/chat
```

**Test quota enforcement:**
```bash
# Send 101 messages to exceed free quota
for i in {1..101}; do
  # WebSocket message
done
```

**Admin: Change user plan:**
```bash
curl -X PUT \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"plan_code":"premium"}' \
  http://localhost:8080/api/admin/users/$USER_ID/plan
```

## Monitoring

### Key Metrics to Track
- Quota exhaustion rate per plan
- Average usage per user/feature
- Users hitting limits
- Feature adoption by plan

### Database Queries

**Users near quota limit:**
```sql
SELECT 
  fu.user_id,
  f.feature_key,
  pf.quota_limit,
  fu.usage_count,
  (fu.usage_count::float / pf.quota_limit * 100) as usage_percent
FROM feature_usage fu
JOIN features f ON f.feature_key = fu.feature_key
JOIN subscriptions s ON s.user_id = fu.user_id AND s.status = 'active'
JOIN plan_features pf ON pf.plan_id = s.plan_id
WHERE fu.period_end > NOW()
  AND pf.quota_limit IS NOT NULL
  AND fu.usage_count > (pf.quota_limit * 0.8);
```

**Usage by plan:**
```sql
SELECT 
  p.name,
  f.feature_key,
  COUNT(DISTINCT fu.user_id) as active_users,
  SUM(fu.usage_count) as total_usage,
  AVG(fu.usage_count) as avg_usage
FROM feature_usage fu
JOIN features f ON f.feature_key = fu.feature_key
JOIN subscriptions s ON s.user_id = fu.user_id AND s.status = 'active'
JOIN plans p ON p.id = s.plan_id
WHERE fu.period_end > NOW()
GROUP BY p.name, f.feature_key;
```

## Future Enhancements

### Phase 2 (Not Implemented)
- [ ] Temporary feature grants
- [ ] Plan upgrades/downgrades with prorating
- [ ] Grace period after quota exhaustion
- [ ] Webhook notifications for quota events
- [ ] Usage analytics dashboard

### Phase 3 (Future)
- [ ] Payment integration
- [ ] Self-service plan changes
- [ ] Usage-based billing
- [ ] Custom enterprise plans

## Troubleshooting

**Issue: User can't access feature despite having subscription**
```bash
# Check user subscription
psql -c "SELECT * FROM subscriptions WHERE user_id = '$USER_ID';"

# Check plan features
psql -c "SELECT * FROM plan_features WHERE plan_id = (
  SELECT plan_id FROM subscriptions WHERE user_id = '$USER_ID' AND status = 'active'
);"
```

**Issue: Quota not resetting**
```bash
# Check feature_usage table
psql -c "SELECT * FROM feature_usage WHERE user_id = '$USER_ID';"

# Manually reset
curl -X POST \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  http://localhost:8080/api/admin/users/$USER_ID/quota/chat/reset
```

**Issue: Usage not incrementing**
- Check logs for `IncrementUsage` errors
- Verify period calculation logic
- Check database constraints

## Security Considerations

1. **No client-side quota checks** - Always validate server-side
2. **Atomic operations** - Use upsert to avoid race conditions
3. **Rate limiting independent** - Quota enforcement separate from rate limiting
4. **Admin endpoints** - Require admin role check (TODO: implement)
5. **Audit logging** - Log plan changes and quota resets

## Files Changed/Created

- `migrations/005_add_feature_quotas.{up,down}.sql` - Schema
- `internal/subscription/manager.go` - Core logic (expanded)
- `internal/subscription/manager_test.go` - Tests (expanded)
- `internal/api/middleware/feature_gate.go` - Middleware
- `internal/api/middleware/feature_gate_test.go` - Tests
- `internal/api/subscription.go` - API handlers (new)
- `internal/ws/chat.go` - WebSocket integration
- `cmd/server/main.go` - Route wiring
- `HANDOFF_quota_tracking.md` - TDD handoff doc
- `TDD_WORKFLOW.md` - Process guide
- `QUOTA_SYSTEM.md` - This document
