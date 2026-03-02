# Authentication Guide

This document describes the authentication system for the Lab (Proxmox Clone) application.

## Overview

The authentication system is built into the Go API backend and provides:

- **User Authentication**: Email/password login with secure password hashing (bcrypt)
- **JWT Tokens**: Short-lived access tokens (15 min) + long-lived refresh tokens (7 days)
- **MFA Support**: TOTP-based multi-factor authentication (compatible with Google Authenticator, Authy, etc.)
- **API Keys**: For CLI and automation tools
- **Role-Based Access Control (RBAC)**: Admin, Operator, and Viewer roles
- **Audit Logging**: All authentication events are logged

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Go API (Auth Owner)                     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │   Connect    │──│   Auth       │──│  SQLite      │       │
│  │  Handlers    │  │  Middleware  │  │  (Sessions)  │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
└─────────────────────────────────────────────────────────────┘
         ▲
         │ JWT / API Key
         │
    ┌────┴─────────────────────────────────────────┐
    │                                              │
    ▼                                              ▼
┌──────────────┐                          ┌──────────────┐
│   Next.js    │                          │   CLI/       │
│    (Web)     │                          │  Terraform   │
└──────────────┘                          └──────────────┘
```

## Configuration

### Option 1: Config File (Recommended)

Create `config.yaml` (copy from `config.example.yaml`):

```yaml
auth:
  # REQUIRED: Generate a secure random string
  # openssl rand -base64 32
  jwt_secret: "your-secure-secret-here"
  
  access_token_expiry: "15m"
  refresh_token_expiry: "168h"
  issuer: "lab-api"
  
  mfa:
    issuer_name: "Lab"
    required_for_admin: false
```

### Option 2: Environment Variable

```bash
export JWT_SECRET="your-secure-random-string"
export ACCESS_TOKEN_EXPIRY="15m"
export REFRESH_TOKEN_EXPIRY="168h"
export JWT_ISSUER="lab-api"
```

Environment variables override config file values.

### Generate a Secure JWT Secret

```bash
openssl rand -base64 32
```

Example output: `xVjK9mN2pL5qR8sT1uW4yZ7aB3cD6eF0gH2iJ5kM8nO=`

## Security Considerations

### Production Setup

1. **Generate a secure JWT secret**:
   ```bash
   openssl rand -base64 32
   ```

2. **Use HTTPS**: All authentication should happen over HTTPS in production.

3. **Set `required_for_admin: true`**: Require MFA for admin accounts.

4. **Regular key rotation**: Rotate JWT secrets periodically.

## API Reference

### User Registration

The first user to register becomes an admin. Subsequent users default to viewer role.

```typescript
// TypeScript example using Connect RPC
const { data } = await authService.register({
  email: "admin@example.com",
  password: "SecureP@ssw0rd!",
});

// Response: { user, accessToken, refreshToken }
```

Password requirements:
- Minimum 8 characters
- At least one uppercase letter
- At least one lowercase letter
- At least one number
- At least one special character

### Login

```typescript
const { data } = await authService.login({
  email: "admin@example.com",
  password: "SecureP@ssw0rd!",
  // mfaCode: "123456" // if MFA is enabled
});

// If MFA is required: { user, mfaRequired: true }
// Otherwise: { user, accessToken, refreshToken }
```

### Token Refresh

Access tokens expire after 15 minutes. Use the refresh token to get new tokens:

```typescript
const { data } = await authService.refreshToken({
  refreshToken: "lab_xxxxx",
});
```

### Logout

Invalidates all refresh tokens for the user:

```typescript
await authService.logout({});
```

## MFA Setup

### Step 1: Generate MFA Secret

```typescript
const { data } = await authService.setupMFA({});

// Returns: { secret, qrCodeUrl, manualKey, backupCodes }
```

Display the QR code to the user for scanning with an authenticator app.

### Step 2: Enable MFA

User enters the 6-digit code from their authenticator app:

```typescript
await authService.enableMFA({
  mfaCode: "123456",
});
```

### Step 3: Store Backup Codes

Backup codes are shown only once! Store them securely.

### Disable MFA

```typescript
await authService.disableMFA({
  mfaCode: "123456", // or a backup code
});
```

## API Keys

API keys are for CLI tools and automation. They can have granular permissions.

### Create API Key

```typescript
const { data } = await authService.createAPIKey({
  name: "My CLI Key",
  permissions: ["vm:read", "vm:start", "vm:stop"],
  // expiresAt: "2025-12-31T23:59:59Z", // optional
});

// IMPORTANT: data.rawKey is shown only once!
```

### Use API Key

Include the API key in the Authorization header:

```bash
curl -H "Authorization: labkey_abc123..." https://api.example.com/...
```

### List API Keys

```typescript
const { data } = await authService.listAPIKeys({});
```

### Revoke API Key

```typescript
await authService.revokeAPIKey({ id: "key-id" });
```

## User Roles

| Role | Permissions |
|------|-------------|
| **Admin** | Full access to all resources |
| **Operator** | Can start/stop VMs, read resources |
| **Viewer** | Read-only access |

## Error Handling

| Error Code | Description |
|------------|-------------|
| `UNAUTHENTICATED` | Invalid/missing credentials |
| `PERMISSION_DENIED` | Insufficient permissions |
| `INVALID_ARGUMENT` | Invalid input (e.g., wrong MFA code) |
| `ALREADY_EXISTS` | Email already registered |

## Frontend Integration (Next.js)

See the Next.js authentication implementation in `apps/web/lib/auth/`.

Key components:
- `AuthContext`: Provides authentication state
- `useAuth`: Hook for accessing auth state
- Connect RPC client with auth header injection

## Audit Logging

All authentication events are logged to the `audit_logs` table:

- `user.login` / `user.logout`
- `api_key.create` / `api_key.use`
- `mfa.enabled` / `mfa.disabled`

Query audit logs:

```sql
SELECT * FROM audit_logs 
WHERE action = 'user.login' 
ORDER BY created_at DESC 
LIMIT 100;
```

## Database Schema

### Tables

- `users`: User accounts
- `refresh_tokens`: Active sessions
- `api_keys`: API keys for automation
- `audit_logs`: Audit trail

### Views

- `active_sessions`: Current user sessions
- `active_api_keys`: Valid API keys

## Troubleshooting

### "Invalid credentials" error

- Check email/password spelling
- Password is case-sensitive

### "Token expired" error

- Refresh token may have expired (7 days)
- Log in again to get new tokens

### MFA not working

- Check time synchronization on device
- Use backup codes if needed
- Contact admin to reset MFA

## Migration from No Auth

If you're adding auth to an existing deployment:

1. Back up your database
2. Run migrations (auth tables are created automatically)
3. Set `JWT_SECRET` environment variable
4. Restart the API
5. First user to register becomes admin
