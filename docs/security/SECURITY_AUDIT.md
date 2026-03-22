# Security Audit Report

**Date:** March 8, 2026  
**Auditor:** Development Team  
**Scope:** Backend API + Frontend Web Application

---

## Executive Summary

| Category | Status | Risk Level |
|----------|--------|------------|
| SQL Injection Prevention | ✅ **PASS** | None |
| XSS Prevention | ✅ **PASS** | Low |
| Request Timeout | ✅ **IMPLEMENTED** | None |
| Rate Limiting | ✅ **IMPLEMENTED** | None |
| Input Validation | ✅ **IMPLEMENTED** | None |
| Audit Logging | ⚠️ **PARTIAL** | Medium |

**Overall Security Posture:** **GOOD** - Critical vulnerabilities addressed, audit logging pending.

---

## 1. SQL Injection Prevention

**Status:** ✅ **PASS**

### Findings

- **109 SQLite queries** audited
- **100% use parameterized queries** (`QueryContext`, `ExecContext`, `QueryRowContext`)
- **Zero raw SQL queries** (`Query`, `Exec` without Context)
- **Zero string concatenation** in SQL statements
- **All queries use `?` placeholders** for parameters

### Example Patterns (Correct)

```go
// ✅ Parameterized query
db.QueryContext(ctx, `SELECT * FROM vms WHERE id = ?`, vmid)

// ✅ Parameterized insert
db.ExecContext(ctx, `INSERT INTO vms (name, cpu) VALUES (?, ?)`, name, cpu)

// ✅ Parameterized with multiple values
db.QueryContext(ctx, `SELECT * FROM metrics WHERE vmid = ? AND timestamp > ?`, vmid, ts)
```

### Files Audited

- `internal/repository/sqlite/*.go` - All 109 queries verified

### Recommendation

**No action required.** Current implementation follows security best practices.

---

## 2. XSS Prevention

**Status:** ✅ **PASS** (Low Risk)

### Findings

- **1 use of `dangerouslySetInnerHTML`** found
- **Usage is SAFE:** Injects CSS for chart theming from controlled config
- **No user content** rendered without escaping
- **React's default escaping** protects against XSS in JSX

### Safe Usage Example

```tsx
// ✅ Safe - CSS from controlled config, not user input
<style
  dangerouslySetInnerHTML={{
    __html: Object.entries(THEMES)
      .map(([theme, prefix]) => `...`)
  }}
/>
```

### Recommendation

**No immediate action required.** Consider adding ESLint rule to flag future `dangerouslySetInnerHTML` usage for review.

---

## 3. Request Timeout Enforcement

**Status:** ✅ **IMPLEMENTED**

### Implementation

- **Default timeout:** 60 seconds
- **Path-specific timeouts:**
  - Backup operations: 5 minutes
  - Snapshot operations: 3 minutes
  - ISO downloads: 30 minutes
  - Health checks: 10 seconds

### Files

- `internal/middleware/timeout.go` - Timeout middleware
- `internal/router/router.go` - Wired into router chain

### Testing

- **9 comprehensive tests** covering timeout behavior
- All tests pass in x86_64 VM

### Recommendation

**Monitor production metrics** for timeout frequency. Adjust timeouts if legitimate operations are being cancelled.

---

## 4. Rate Limiting

**Status:** ✅ **IMPLEMENTED**

### Current Implementation

- **Global rate limiter:** 100 requests/second, burst of 200
- **Auth endpoints:** 5 attempts/second, burst of 10
- **Applied to:** All ConnectRPC handlers

### Files

- `internal/middleware/ratelimit.go` - Rate limiter implementation
- `internal/router/router.go` - Wired into router chain

### Recommendation

**Consider adding:**
- Per-IP rate limiting (currently global)
- Per-user rate limiting for authenticated endpoints
- Rate limit headers (`X-RateLimit-Limit`, `X-RateLimit-Remaining`)

---

## 5. Input Validation

**Status:** ✅ **IMPLEMENTED**

### Coverage

- **Auth handlers:** All inputs validated (Register, Login, MFA, API keys)
- **VM operations:** Name, description, CPU, memory validation
- **UUID validation:** Added `ValidateUUID()` helper
- **Path safety:** Path traversal prevention (`..` blocked)
- **Email validation:** RFC-compliant email format checking
- **Password strength:** Minimum 8 chars, upper/lower/digit/special required

### Files

- `internal/validator/validator.go` - Validation helpers
- All handler files - Input validation before service calls

### Recommendation

**No immediate action required.** Continue adding validation for new features.

---

## 6. Audit Logging

**Status:** ⚠️ **PARTIAL** (Pending Implementation)

### Current State

- **Auth events:** Login, logout, MFA changes logged
- **Task tracking:** All async operations tracked
- **Alert events:** Alert firings logged

### Gaps

- **Missing:** Config changes (who changed what and when)
- **Missing:** Destructive operations (VM deletion, backup deletion)
- **Missing:** Admin actions (user creation, role changes)
- **Missing:** Failed auth attempts (brute force detection)

### Recommendation

**Implement audit logging for:**
1. Authentication failures (IP, username, timestamp)
2. Config changes (old value, new value, user)
3. Destructive operations (resource deleted, user)
4. Admin actions (user management, role changes)

**Implementation approach:**
- Create `internal/middleware/audit.go` - HTTP middleware
- Create `internal/repository/sqlite/audit.go` - Audit log storage
- Create migration `pkg/sqlite/migrations/013_audit.sql` - Audit logs table
- Wire into router after auth middleware

---

## 7. Authentication & Authorization

**Status:** ✅ **IMPLEMENTED**

### JWT Security

- **Secret validation:** Minimum 32 characters, entropy checks
- **Weak secret prevention:** Blocks "secret", "password", "changeme", etc.
- **Token expiry:** 15 minutes (access), 7 days (refresh)
- **MFA support:** TOTP-based 2FA available

### Session Management

- **Session tracking:** All sessions tracked in SQLite
- **Session revocation:** Users can revoke individual sessions
- **API keys:** Scoped keys with expiration

### Files

- `internal/middleware/auth.go` - JWT authentication
- `internal/validator/auth.go` - JWT secret validation
- `internal/handler/auth.go` - Auth handlers

### Recommendation

**Consider adding:**
- Session timeout (force re-authentication after X hours)
- IP-based session binding (optional)
- API key rotation policy

---

## 8. TLS/HTTPS

**Status:** ✅ **IMPLEMENTED**

### Configuration

- **TLS support:** Built-in HTTPS serving
- **Certificate options:**
  - User-provided cert/key files
  - Self-signed (auto-generated)
  - ACME/Let's Encrypt (stubbed for future)

### Files

- `internal/config/config.go` - TLS configuration
- `cmd/server/init_server.go` - TLS server setup

### Recommendation

**For production:**
- Use Let's Encrypt via reverse proxy (Caddy/nginx)
- Enable HSTS headers
- Implement certificate rotation

---

## 9. IP Whitelisting

**Status:** ✅ **IMPLEMENTED**

### Configuration

- **CIDR support:** IPv4 and IPv6
- **Configurable:** `security.allowed_cidrs` in config.yaml
- **Middleware:** Applied before auth for early rejection

### Files

- `internal/middleware/ipwhitelist.go` - IP whitelist middleware
- `internal/config/config.go` - CIDR configuration

### Recommendation

**Document in deployment guide:**
- How to configure allowed CIDRs
- Common CIDR ranges for home networks
- Troubleshooting access issues

---

## 10. Dependencies

**Status:** ✅ **MONITORED**

### Current Dependencies

- **Go:** 1.25.7 (latest)
- **Node.js:** 22 (LTS)
- **libvirt:** System package (tracked by OS)

### Recommendation

**Implement:**
- Dependabot or Renovate for automated dependency updates
- `npm audit` in CI pipeline
- `govulncheck` in CI pipeline

---

## Summary of Recommendations

### Critical (Implement ASAP)

1. **Audit Logging** - Track auth failures, config changes, destructive operations

### High Priority (Next Sprint)

2. **Per-IP Rate Limiting** - Prevent distributed attacks
3. **Dependency Scanning** - Add `govulncheck` and `npm audit` to CI

### Medium Priority (Future)

4. **Session Timeout** - Force re-authentication after extended periods
5. **Rate Limit Headers** - Inform clients of rate limit status
6. **TLS Documentation** - Document production TLS setup

### Low Priority (Nice to Have)

7. **ESLint Rule** - Flag `dangerouslySetInnerHTML` for review
8. **IP-based Session Binding** - Optional security enhancement

---

## Conclusion

The Lab application has a **strong security foundation**:

- ✅ SQL injection prevented via parameterized queries
- ✅ XSS prevented via React escaping (one safe exception)
- ✅ Request timeouts implemented
- ✅ Rate limiting in place
- ✅ Input validation comprehensive
- ✅ Authentication secure with MFA support
- ✅ TLS support available
- ✅ IP whitelisting available

**Primary gap:** Audit logging for compliance and incident response.

**Overall Risk Level:** **LOW** - Suitable for production deployment with recommended improvements.
