# Security Implementation Guide

## Overview
This document outlines the security measures implemented in the MomLaunchpad backend.

## Security Layers

### 1. Network Security

#### Binding Configuration
- **Development**: Server binds to `0.0.0.0:8080` (all interfaces)
- **Access Control**: Docker port mapping controls external exposure
- **Recommendation**: Use reverse proxy (nginx/caddy) in production

#### Docker Network Isolation
```yaml
# docker-compose.yml
ports:
  - "8080:8080"  # Development: Allow LAN access
  # - "127.0.0.1:8080:8080"  # Production: Localhost only (use with reverse proxy)
```

### 2. Application Security

#### Security Headers Middleware
Located at: `internal/api/middleware/security.go`

**Implemented Headers:**
- `X-Frame-Options: DENY` - Prevents clickjacking attacks
- `X-Content-Type-Options: nosniff` - Prevents MIME type sniffing
- `X-XSS-Protection: 1; mode=block` - Enables XSS protection
- `X-Powered-By: ""` - Removes server fingerprinting
- `Strict-Transport-Security` - Forces HTTPS (when TLS enabled)
- `Content-Security-Policy: default-src 'self'` - Restricts resource loading
- `Referrer-Policy: strict-origin-when-cross-origin` - Controls referrer information
- `Permissions-Policy` - Disables unnecessary browser features

#### Authentication & Authorization
- **JWT Tokens**: All protected endpoints require valid JWT
- **Admin Middleware**: `AdminOnly()` checks for admin role
- **Token Expiry**: 7 days (configurable)
- **Password Hashing**: bcrypt with default cost

#### Rate Limiting
- **Global Limit**: 100 requests/minute per IP
- **Burst**: 200 requests allowed
- **Location**: `internal/api/middleware/ratelimit.go`

### 3. Container Security

#### Docker Hardening
```yaml
security_opt:
  - no-new-privileges:true  # Prevents privilege escalation
cap_drop:
  - ALL  # Drop all Linux capabilities
cap_add:
  - NET_BIND_SERVICE  # Only allow binding to ports
```

#### Read-Only Filesystem (Optional)
```yaml
# Uncomment for production
# read_only: true
# tmpfs:
#   - /tmp
```

### 4. Database Security

#### SQL Injection Prevention
- ✅ Parameterized queries throughout (`database/sql` with placeholders)
- ✅ No string concatenation in queries
- ✅ Input validation on all user inputs

#### Connection Security
- Connection pooling with timeout limits
- Prepared statements for repeated queries
- Context-aware queries with cancellation

### 5. API Security

#### Protected Endpoints
All endpoints under `/api/*` (except auth) require JWT:
```go
router.Use(middleware.JWTAuth(jwtSecret))
```

#### Admin-Only Endpoints
Admin routes have double protection:
```go
adminGroup.Use(middleware.JWTAuth(jwtSecret))
adminGroup.Use(middleware.AdminOnly())
```

#### Public Endpoints
Only these endpoints are accessible without authentication:
- `GET /health` - Health check
- `POST /api/auth/register` - User registration
- `POST /api/auth/login` - User login
- `GET /api/auth/google` - OAuth redirect
- `GET /api/auth/google/callback` - OAuth callback
- `POST /api/auth/google/token` - Mobile OAuth
- `POST /api/voice/*` - Twilio webhooks (validated by phone number lookup)

### 6. Input Validation

#### Request Validation
Using Gin's binding with validation tags:
```go
type RegisterRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=8"`
    Name     string `json:"name"`
    Language string `json:"language"`
}
```

#### Sanitization
- Email addresses validated with RFC 5322 format
- Passwords minimum 8 characters
- UUIDs validated before database queries
- File paths sanitized (when file operations added)

### 7. CORS Configuration

#### Current Setup
```go
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: POST, OPTIONS, GET, PUT, DELETE, PATCH
Access-Control-Allow-Headers: Content-Type, Authorization, ...
```

#### Production Recommendation
```go
// Update middleware/cors.go for production:
c.Writer.Header().Set("Access-Control-Allow-Origin", "https://yourdomain.com")
c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
```

### 8. Secrets Management

#### Environment Variables
Sensitive data stored in `.env` (not committed to git):
- `JWT_SECRET` - JWT signing key
- `DEEPSEEK_API_KEY` - AI provider key
- `DATABASE_URL` - Database connection string
- `TWILIO_ACCOUNT_SID` - Twilio credentials
- `TWILIO_AUTH_TOKEN` - Twilio auth token

#### Secret Rotation
Recommendation: Rotate secrets every 90 days in production.

## Production Deployment Checklist

### Pre-Deployment
- [ ] Update CORS to specific domains
- [ ] Set `GIN_MODE=release`
- [ ] Enable HTTPS/TLS
- [ ] Configure reverse proxy (nginx/caddy)
- [ ] Bind Docker to localhost only: `127.0.0.1:8080:8080`
- [ ] Set strong `JWT_SECRET` (32+ characters)
- [ ] Enable database SSL/TLS
- [ ] Configure firewall rules

### Monitoring & Logging
- [ ] Set up failed login attempt monitoring
- [ ] Enable API abuse detection
- [ ] Configure alerting for suspicious activity
- [ ] Set up log aggregation (ELK/Loki)
- [ ] Monitor rate limit violations

### Regular Maintenance
- [ ] Review and update dependencies monthly
- [ ] Scan for vulnerabilities (`go list -m -json all | nancy sleuth`)
- [ ] Audit admin actions
- [ ] Review access logs
- [ ] Test disaster recovery procedures

## Security Testing

### Automated Tests
```bash
# Run security audit
go list -m -json all | nancy sleuth

# Check for known vulnerabilities
govulncheck ./...

# Static analysis
golangci-lint run
```

### Manual Testing
- [ ] Test JWT expiration handling
- [ ] Verify admin-only endpoint protection
- [ ] Test rate limiting behavior
- [ ] Verify CORS restrictions
- [ ] Test SQL injection resistance
- [ ] Test XSS prevention

## Incident Response

### Security Breach Protocol
1. Immediately rotate all secrets (JWT_SECRET, API keys)
2. Force logout all users (invalidate tokens)
3. Review access logs for compromised accounts
4. Notify affected users if data exposure occurred
5. Patch vulnerability and deploy fix
6. Document incident and lessons learned

### Contact
Security issues: security@yourdomain.com

## Additional Resources

### OWASP Top 10 Compliance
- ✅ A01:2021 – Broken Access Control (JWT + AdminOnly)
- ✅ A02:2021 – Cryptographic Failures (bcrypt, HTTPS)
- ✅ A03:2021 – Injection (Parameterized queries)
- ✅ A04:2021 – Insecure Design (TDD, security headers)
- ✅ A05:2021 – Security Misconfiguration (Hardened Docker)
- ✅ A06:2021 – Vulnerable Components (Dependency scanning)
- ✅ A07:2021 – Authentication Failures (JWT, rate limiting)
- ⚠️ A08:2021 – Software and Data Integrity (TODO: Code signing)
- ⚠️ A09:2021 – Logging & Monitoring (TODO: Enhanced logging)
- ✅ A10:2021 – SSRF (No external requests from user input)

### References
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [Go Security Best Practices](https://github.com/OWASP/Go-SCP)
- [Docker Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Docker_Security_Cheat_Sheet.html)
