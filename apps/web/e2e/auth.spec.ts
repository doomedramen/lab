import { test, expect } from "./fixtures"

test.describe("Authentication", () => {
  test.describe("Login", () => {
    test("should display login form", async ({ page }) => {
      await page.goto("/login")
      await expect(page.getByTestId("login-page")).toBeVisible()
      await expect(page.getByTestId("login-email-input")).toBeVisible()
      await expect(page.getByTestId("login-password-input")).toBeVisible()
      await expect(page.getByTestId("login-submit-button")).toBeVisible()
    })

    test("should show error for invalid credentials", async ({ page }) => {
      await page.goto("/login")

      await page.getByTestId("login-email-input").fill("invalid@example.com")
      await page.getByTestId("login-password-input").fill("wrongpassword")
      await page.getByTestId("login-submit-button").click()

      await expect(page.getByTestId("login-error-alert")).toBeVisible({ timeout: 8000 })
    })

    test("should show validation for empty fields", async ({ page }) => {
      await page.goto("/login")

      await page.getByTestId("login-submit-button").click()

      await expect(page.getByText("Invalid email address")).toBeVisible()
    })

    test("should successfully login with valid credentials", async ({ page, login }) => {
      await login()
      await expect(page).toHaveURL(/\/(dashboard)?$/, { timeout: 8000 })
    })
  })

  test.describe("Register", () => {
    test("should display registration form", async ({ page }) => {
      await page.goto("/register")
      await expect(page.getByTestId("register-page")).toBeVisible()
      await expect(page.getByTestId("register-email-input")).toBeVisible()
      await expect(page.getByTestId("register-password-input")).toBeVisible()
      await expect(page.getByTestId("register-confirm-password-input")).toBeVisible()
      await expect(page.getByTestId("register-submit-button")).toBeVisible()
    })

    test("should show password requirements", async ({ page }) => {
      await page.goto("/register")

      await expect(page.getByText("At least 8 characters")).toBeVisible()
      await expect(page.getByText("One uppercase letter")).toBeVisible()
      await expect(page.getByText("One lowercase letter")).toBeVisible()
      await expect(page.getByText("One number")).toBeVisible()
      await expect(page.getByText("One special character")).toBeVisible()
    })

    test("should show validation for weak password", async ({ page }) => {
      await page.goto("/register")

      await page.getByTestId("register-email-input").fill("test@example.com")
      await page.getByTestId("register-password-input").fill("weak")
      await page.getByTestId("register-confirm-password-input").fill("weak")
      await page.getByTestId("register-submit-button").click()

      await expect(page.getByText(/must be at least 8 characters/i)).toBeVisible()
    })

    test("should show validation for mismatched passwords", async ({ page }) => {
      await page.goto("/register")

      await page.getByTestId("register-email-input").fill("test@example.com")
      await page.getByTestId("register-password-input").fill("TestP@ssw0rd!")
      await page.getByTestId("register-confirm-password-input").fill("DifferentP@ss!")
      await page.getByTestId("register-submit-button").click()

      await expect(page.getByText(/match/i)).toBeVisible()
    })

    test("should navigate to login from register", async ({ page }) => {
      await page.goto("/register")

      await page.getByRole("link", { name: /sign in/i }).click()

      await expect(page).toHaveURL(/\/login/)
    })

    test("should successfully register a new account", async ({ page }) => {
      const uniqueEmail = `e2e-${Date.now()}@example.com`

      await page.goto("/register")
      await page.getByTestId("register-email-input").fill(uniqueEmail)
      await page.getByTestId("register-password-input").fill("TestP@ssw0rd!")
      await page.getByTestId("register-confirm-password-input").fill("TestP@ssw0rd!")
      await page.getByTestId("register-submit-button").click()

      await expect(page).toHaveURL(/\/(dashboard)?$/, { timeout: 8000 })
    })
  })

  test.describe("Route Guards", () => {
    test("should redirect unauthenticated user from protected route", async ({ page }) => {
      await page.context().clearCookies()
      await page.goto("/dashboard")

      await expect(page).toHaveURL(/\/login/, { timeout: 8000 })
    })
  })

  test.describe("Authenticated User", () => {
    test("should redirect authenticated user from login page", async ({ page, login }) => {
      await login()
      await page.goto("/login")
      await expect(page).toHaveURL(/\/(dashboard)?$/)
    })

    test("should redirect authenticated user from register page", async ({ page, login }) => {
      await login()
      await page.goto("/register")
      await expect(page).toHaveURL(/\/(dashboard)?$/)
    })

    test("should successfully logout", async ({ page, login, logout }) => {
      await login()
      await logout()

      await expect(page).toHaveURL(/\/login/, { timeout: 8000 })

      // After logout, protected routes should redirect back to login
      await page.goto("/dashboard")
      await expect(page).toHaveURL(/\/login/, { timeout: 8000 })
    })
  })
})
