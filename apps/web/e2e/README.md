# E2E Tests

This directory contains Playwright end-to-end tests for the web application.

## Setup

### Prerequisites

1. **Start the API server** (must be running before tests):
   ```bash
   cd apps/api
   go run ./cmd/server
   ```

2. **Start the web app**:
   ```bash
   cd apps/web
   pnpm dev
   ```

The global setup (`global-setup.ts`) runs automatically before tests and will:
- Register the test user if they don't exist
- Log in and save browser state to `e2e/.auth/user.json`

`e2e/.auth/` is gitignored — it's generated at runtime.

### Test Credentials

By default, tests use these credentials:
- Email: `test@example.com`
- Password: `TestP@ssw0rd!`

Override with environment variables:
```bash
E2E_TEST_EMAIL=myuser@example.com E2E_TEST_PASSWORD=MyP@ssword pnpm test:e2e
```

## Running Tests

### Headless (default)
```bash
pnpm test:e2e
```

### With UI Mode (interactive)
```bash
pnpm test:e2e:ui
```

### Headed (visible browser)
```bash
pnpm test:e2e:headed
```

### With Custom Credentials
```bash
E2E_TEST_EMAIL=admin@example.com E2E_TEST_PASSWORD=AdminP@ss! pnpm test:e2e
```

## Test Structure

- `global-setup.ts` - Runs once before all tests: creates test user and saves auth state
- `fixtures.ts` - Extended Playwright fixtures with authentication helpers
- `auth.spec.ts` - Authentication tests (login, register, route guards, logout)
- `vm-create-start.spec.ts` - Tests for VM creation and startup flow

## Projects

Tests are split into two Playwright projects:

### `auth` project
- Runs `auth.spec.ts` only
- No pre-authenticated state — tests the full auth flow from scratch

### `app` project
- Runs all other spec files
- Depends on `auth` completing first
- Uses pre-authenticated state from `e2e/.auth/user.json` (set by global setup)
- No manual login needed in individual tests

## Fixtures

### Authentication Fixtures

- `login(email?, password?)` - Logs in a user with optional custom credentials (used in auth tests)
- `logout()` - Logs out the current user via the user menu

### Usage in auth tests

```typescript
import { test, expect } from "./fixtures"

test("should login", async ({ page, login }) => {
  await login()
  await expect(page).toHaveURL(/\/dashboard/)
})
```

### Usage in app tests

App tests (non-auth) run with pre-authenticated state — no login fixture needed:

```typescript
import { test, expect } from "./fixtures"

test("should see vms page", async ({ page }) => {
  await page.goto("/vms")
  await expect(page.getByTestId("vms-page-title")).toBeVisible()
})
```

## Selectors

All tests use `data-testid` attributes for stable selectors. Key selectors:

### Authentication
- `login-page` - Login page container
- `login-email-input` - Email input field
- `login-password-input` - Password input field
- `login-submit-button` - Sign in button
- `login-mfa-input` - MFA code input (when MFA is enabled)
- `login-error-alert` - Error message alert
- `register-page` - Registration page container
- `register-email-input` - Email input field
- `register-password-input` - Password input field
- `register-confirm-password-input` - Confirm password field
- `register-submit-button` - Create account button
- `register-error-alert` - Error message alert

### VM Creation
- `create-vm-button` - Button to open create VM modal
- `create-vm-modal` - The create VM modal dialog
- `vm-name-input` - VM name input field
- `vm-node-select` - Node selection dropdown
- `vm-os-select` - OS template selection
- `vm-cpu-input` - CPU cores input
- `vm-memory-input` - Memory (GB) input
- `vm-disk-input` - Disk (GB) input
- `vm-create-submit` - Submit/create button

### VM List
- `vms-page-title` - VMs page heading
- `vms-table` - VM table container
- `vm-table-row-{vmid}` - Table row for a specific VM
- `vm-table-status-{vmid}` - Status badge for a VM

### VM Detail
- `vm-detail-vmid` - VM ID badge
- `vm-detail-status` - VM status badge
- `vm-start-button` - Start VM button (when stopped)
- `vm-stop-button` - Stop VM button (when running)
- `vm-pause-button` - Pause VM button
- `vm-reboot-button` - Reboot VM button

## Writing New Tests

### Auth tests (need login)

```typescript
import { test, expect } from "./fixtures"

test.describe("My Auth Feature", () => {
  test("should do something after login", async ({ page, login }) => {
    await login()
    // ... test code
  })
})
```

### App tests (pre-authenticated via storageState)

```typescript
import { test, expect } from "./fixtures"

test.describe("My Feature", () => {
  test("should do something", async ({ page }) => {
    await page.goto("/my-page")
    // Already logged in via storageState
    // ... test code
  })
})
```

### Best Practices

1. **Use `data-testid` attributes** for stable selectors
2. **Use fixtures** (`login`, `gotoVMs`, etc.) for common operations
3. **Wait for network idle** when navigating: `await page.waitForLoadState("networkidle")`
4. **Use `toPass`** for async operations that take time
5. **Clean up test data** (VMs, etc.) after tests when possible

## Troubleshooting

### Tests fail with "Login page not found"
- Ensure the API server is running
- Check that a test user exists with the correct credentials
- Verify `NEXT_PUBLIC_API_URL` is set correctly

### Tests fail with "Timeout exceeded"
- Increase timeout in the test: `await expect(...).toBeVisible({ timeout: 30000 })`
- Check if the API is responding
- Run in headed mode to see what's happening: `pnpm test:e2e:headed`

### MFA is required but tests fail
- Disable MFA for the test user, or
- Update the test to include MFA code entry
