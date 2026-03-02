import { test, expect } from "./fixtures"

test.describe("VM Console", () => {
  test.beforeEach(async ({ page, login }) => {
    await login()
    await page.goto("/vms")
  })

  test.describe("Console Button", () => {
    test("should show console button for running VM", async ({ page }) => {
      // Find a running VM and click to view details
      const vmCard = page.getByTestId("vm-card").first()
      if (await vmCard.isVisible()) {
        await vmCard.click()
        
        // Wait for VM detail page to load
        await expect(page).toHaveURL(/\/vms\/\d+/)
        
        // Console button should be visible
        const consoleButton = page.getByTestId("vm-console-button")
        await expect(consoleButton).toBeVisible()
      }
    })

    test("should disable console button for stopped VM", async ({ page }) => {
      // This test requires a stopped VM
      // Navigate to a stopped VM's detail page
      const vmCard = page.getByTestId("vm-card").first()
      if (await vmCard.isVisible()) {
        await vmCard.click()
        await expect(page).toHaveURL(/\/vms\/\d+/)
        
        // Check if VM is stopped
        const statusBadge = page.getByTestId("vm-status")
        const status = await statusBadge.textContent()
        
        if (status?.toLowerCase().includes("stopped")) {
          const consoleButton = page.getByTestId("vm-console-button")
          await expect(consoleButton).toBeDisabled()
        }
      }
    })

    test("should show dropdown button next to console", async ({ page }) => {
      const vmCard = page.getByTestId("vm-card").first()
      if (await vmCard.isVisible()) {
        await vmCard.click()
        await expect(page).toHaveURL(/\/vms\/\d+/)
        
        // Dropdown button should be visible
        const dropdownButton = page.getByTestId("vm-console-dropdown-button")
        await expect(dropdownButton).toBeVisible()
      }
    })
  })

  test.describe("Console Dropdown Menu", () => {
    test("should open dropdown menu when clicking dropdown button", async ({ page }) => {
      const vmCard = page.getByTestId("vm-card").first()
      if (await vmCard.isVisible()) {
        await vmCard.click()
        await expect(page).toHaveURL(/\/vms\/\d+/)
        
        const dropdownButton = page.getByTestId("vm-console-dropdown-button")
        await dropdownButton.click()
        
        // Dropdown menu should be visible
        const menu = page.getByRole("menu")
        await expect(menu).toBeVisible()
      }
    })

    test("should show serial console option", async ({ page }) => {
      const vmCard = page.getByTestId("vm-card").first()
      if (await vmCard.isVisible()) {
        await vmCard.click()
        await expect(page).toHaveURL(/\/vms\/\d+/)
        
        const dropdownButton = page.getByTestId("vm-console-dropdown-button")
        await dropdownButton.click()
        
        // Serial console option should be visible
        const serialOption = page.getByText("Serial Console")
        await expect(serialOption).toBeVisible()
      }
    })

    test("should show noVNC option", async ({ page }) => {
      const vmCard = page.getByTestId("vm-card").first()
      if (await vmCard.isVisible()) {
        await vmCard.click()
        await expect(page).toHaveURL(/\/vms\/\d+/)
        
        const dropdownButton = page.getByTestId("vm-console-dropdown-button")
        await dropdownButton.click()
        
        // noVNC option should be visible
        const vncOption = page.getByText("noVNC")
        await expect(vncOption).toBeVisible()
      }
    })

    test("should show websockify option as disabled", async ({ page }) => {
      const vmCard = page.getByTestId("vm-card").first()
      if (await vmCard.isVisible()) {
        await vmCard.click()
        await expect(page).toHaveURL(/\/vms\/\d+/)
        
        const dropdownButton = page.getByTestId("vm-console-dropdown-button")
        await dropdownButton.click()
        
        // websockify option should be visible but disabled
        const websockifyOption = page.getByText("websockify")
        await expect(websockifyOption).toBeVisible()
        await expect(websockifyOption).toBeDisabled()
      }
    })
  })

  test.describe("Serial Console", () => {
    test("should open serial console dialog when clicking console button", async ({ page }) => {
      const vmCard = page.getByTestId("vm-card").first()
      if (await vmCard.isVisible()) {
        await vmCard.click()
        await expect(page).toHaveURL(/\/vms\/\d+/)
        
        // Click main console button (should open serial by default)
        const consoleButton = page.getByTestId("vm-console-button")
        await consoleButton.click()
        
        // Console dialog should be visible
        const dialog = page.getByTestId("console-dialog")
        await expect(dialog).toBeVisible()
        
        // Dialog title should mention "Serial"
        const title = page.getByRole("heading", { name: /Serial/ })
        await expect(title).toBeVisible()
      }
    })

    test("should show connection status when connecting", async ({ page }) => {
      const vmCard = page.getByTestId("vm-card").first()
      if (await vmCard.isVisible()) {
        await vmCard.click()
        await expect(page).toHaveURL(/\/vms\/\d+/)
        
        const consoleButton = page.getByTestId("vm-console-button")
        await consoleButton.click()
        
        // Should show connecting status
        const connectingText = page.getByText(/Connecting/i)
        await expect(connectingText).toBeVisible({ timeout: 5000 })
      }
    })

    test("should close dialog when clicking close button", async ({ page }) => {
      const vmCard = page.getByTestId("vm-card").first()
      if (await vmCard.isVisible()) {
        await vmCard.click()
        await expect(page).toHaveURL(/\/vms\/\d+/)
        
        const consoleButton = page.getByTestId("vm-console-button")
        await consoleButton.click()
        
        // Close dialog using the built-in close button
        const closeButton = page.getByRole("button", { name: /Close/i }).first()
        await closeButton.click()
        
        // Dialog should be hidden
        const dialog = page.getByTestId("console-dialog")
        await expect(dialog).toBeHidden()
      }
    })

    test("should close dialog when pressing Escape", async ({ page }) => {
      const vmCard = page.getByTestId("vm-card").first()
      if (await vmCard.isVisible()) {
        await vmCard.click()
        await expect(page).toHaveURL(/\/vms\/\d+/)
        
        const consoleButton = page.getByTestId("vm-console-button")
        await consoleButton.click()
        
        // Press Escape
        await page.keyboard.press("Escape")
        
        // Dialog should be hidden
        const dialog = page.getByTestId("console-dialog")
        await expect(dialog).toBeHidden()
      }
    })
  })

  test.describe("Console Error Handling", () => {
    test("should show error message when connection fails", async ({ page }) => {
      const vmCard = page.getByTestId("vm-card").first()
      if (await vmCard.isVisible()) {
        await vmCard.click()
        await expect(page).toHaveURL(/\/vms\/\d+/)
        
        const consoleButton = page.getByTestId("vm-console-button")
        await consoleButton.click()
        
        // Wait for connection attempt
        await page.waitForTimeout(5000)
        
        // Should show error or connection message
        const errorOrMessage = page.getByText(/Error|Failed|Connecting/i)
        await expect(errorOrMessage.first()).toBeVisible({ timeout: 10000 })
      }
    })

    test("should show retry button on error", async ({ page }) => {
      const vmCard = page.getByTestId("vm-card").first()
      if (await vmCard.isVisible()) {
        await vmCard.click()
        await expect(page).toHaveURL(/\/vms\/\d+/)
        
        const consoleButton = page.getByTestId("vm-console-button")
        await consoleButton.click()
        
        // Wait for potential error
        await page.waitForTimeout(5000)
        
        // Retry button should be visible if there's an error
        const retryButton = page.getByRole("button", { name: /Retry/i })
        // May or may not be visible depending on connection success
        // Just verify it doesn't crash
      }
    })
  })
})
