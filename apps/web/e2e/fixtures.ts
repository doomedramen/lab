import { test as base, expect, type Page } from "@playwright/test"

export interface Fixtures {
  gotoVMs: () => Promise<void>
  gotoVMDetail: (vmid: string | number) => Promise<void>
  login: (email?: string, password?: string) => Promise<void>
  logout: () => Promise<void>
  createAndStartVM: () => Promise<string>
  waitForVMStatus: (vmid: string, status: string, timeout?: number) => Promise<void>
  cleanupVM: (vmid: string) => Promise<void>
  gotoStacks: () => Promise<void>
  gotoStackDetail: (stackId: string) => Promise<void>
  createStack: (name?: string) => Promise<string>
  cleanupStack: (stackId: string) => Promise<void>
}

// Test credentials
const TEST_EMAIL = process.env.E2E_TEST_EMAIL || "test@example.com"
const TEST_PASSWORD = process.env.E2E_TEST_PASSWORD || "TestP@ssw0rd!"

export const test = base.extend<Fixtures>({
  // Login helper that navigates to login page and authenticates
  login: async ({ page }, use) => {
    await use(async (email = TEST_EMAIL, password = TEST_PASSWORD) => {
      // Navigate to login page
      await page.goto("/login", { waitUntil: "domcontentloaded" })

      // Wait for login form to be visible
      await expect(page.getByTestId("login-page")).toBeVisible({ timeout: 8000 })

      // Fill in credentials
      await page.getByTestId("login-email-input").fill(email)
      await page.getByTestId("login-password-input").fill(password)

      // Submit login form
      await page.getByTestId("login-submit-button").click()

      // Wait for redirect to dashboard (or home page)
      await page.waitForURL(/\/(dashboard|vms)/, { timeout: 8000 })
    })
  },

  // Logout helper
  logout: async ({ page }, use) => {
    await use(async () => {
      await page.getByTestId("logout-button").click()
      await page.waitForURL(/\/login/, { timeout: 8000 })
    })
  },

  gotoVMs: async ({ page }, use) => {
    await use(async () => {
      await page.goto("/vms", { waitUntil: "domcontentloaded" })
      await expect(page.getByTestId("vms-page-title")).toBeVisible({ timeout: 8000 })
    })
  },

  gotoVMDetail: async ({ page }, use) => {
    await use(async (vmid: string | number) => {
      await page.goto(`/vms/${vmid}`, { waitUntil: "domcontentloaded" })
      await expect(page.getByTestId("vm-detail-vmid")).toBeVisible({ timeout: 8000 })
    })
  },

  // Creates a VM via the UI using a template, starts it, and waits until running.
  // Returns the vmid string.
  createAndStartVM: async ({ page }, use) => {
    await use(async (): Promise<string> => {
      const vmName = `test-vm-${Math.floor(Math.random() * 100000)}`
      console.log(`[createAndStartVM] creating VM: ${vmName}`)

      await page.goto("/vms")
      await page.waitForLoadState("domcontentloaded")
      await expect(page.getByTestId("vms-page-title")).toBeVisible({ timeout: 8000 })

      // Open create modal
      await page.getByTestId("create-vm-button").click()
      const modal = page.getByTestId("create-vm-modal")
      await expect(modal).toBeVisible()

      // Select first node
      await page.getByTestId("vm-template-node-select").click()
      await page.getByRole("option").first().click()

      // Select first template (not "Custom Configuration")
      await page.getByTestId("vm-template-select").click()
      // Get all options and find first one that's not "Custom Configuration"
      const options = page.getByRole("option")
      const optionCount = await options.count()
      for (let i = 0; i < optionCount; i++) {
        const optionText = await options.nth(i).textContent()
        if (optionText && !optionText.includes("Custom")) {
          await options.nth(i).click()
          break
        }
      }

      // Click on the "Basic" tab
      await page.getByRole("tab", { name: "Basic" }).click()

      // Give the VM a name
      await page.getByTestId("vm-name-input").fill(vmName)

      // Click "Create VM"
      await page.getByTestId("vm-create-submit").click()

      // Wait for redirect to VM detail page
      await expect(page.getByTestId("vm-detail-vmid")).toBeVisible({ timeout: 120000 })
      const vmid = page.url().split("/vms/")[1] ?? ""
      console.log(`[createAndStartVM] created vmid=${vmid}, starting`)

      // Start the VM
      await expect(page.getByTestId("vm-start-button")).toBeVisible({ timeout: 8000 })
      await page.getByTestId("vm-start-button").click()

      // Wait until running
      const statusLocator = page.getByTestId("vm-detail-status")
      await expect(async () => {
        const status = await statusLocator.textContent()
        console.log(`[createAndStartVM] start poll: status="${status}"`)
        expect(status?.toLowerCase()).toContain("running")
      }).toPass({ timeout: 10000, intervals: [500, 1000, 2000] })
      console.log(`[createAndStartVM] vmid=${vmid} is running`)

      return vmid
    })
  },

  // Polls the VM detail page until the status badge matches `status`.
  waitForVMStatus: async ({ page }, use) => {
    await use(async (vmid: string, status: string, timeout = 10000) => {
      await page.goto(`/vms/${vmid}`)
      await expect(page.getByTestId("vm-detail-vmid")).toBeVisible({ timeout: 8000 })
      const statusLocator = page.getByTestId("vm-detail-status")
      await expect(async () => {
        // Reload to get fresh data; networkidle ensures the API response has been received
        await page.reload({ waitUntil: "networkidle" })
        const text = await statusLocator.textContent()
        expect(text?.toLowerCase()).toContain(status.toLowerCase())
      }).toPass({ timeout, intervals: [500, 1000, 2000] })
    })
  },

  // Stops a VM if running, then deletes it. Best-effort for test teardown.
  cleanupVM: async ({ page }, use) => {
    await use(async (vmid: string) => {
      await page.goto(`/vms/${vmid}`)
      await page.waitForLoadState("domcontentloaded")

      // If a stop button is visible, stop the VM first
      const stopBtn = page.getByTestId("vm-stop-button")
      if (await stopBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
        await stopBtn.click()
        // Confirm the stop dialog if it appears
        const confirmBtn = page.getByTestId("confirm-dialog-confirm")
        if (await confirmBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
          await confirmBtn.click()
        }
        // Wait for VM to reach stopped state — mutation triggers React Query refetch automatically
        const stopStatusLocator = page.getByTestId("vm-detail-status")
        await expect(async () => {
          const text = await stopStatusLocator.textContent()
          expect(text?.toLowerCase()).toContain("stopped")
        }).toPass({ timeout: 10000, intervals: [500, 1000, 2000] })
      }

      // Click delete
      const deleteBtn = page.getByTestId("vm-delete-button")
      if (await deleteBtn.isEnabled({ timeout: 3000 }).catch(() => false)) {
        await deleteBtn.click()
        const confirmDeleteBtn = page.getByTestId("confirm-dialog-confirm")
        if (await confirmDeleteBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
          await confirmDeleteBtn.click()
        }
      }
    })
  },

  // Navigate to the stacks list page and wait for it to render.
  gotoStacks: async ({ page }, use) => {
    await use(async () => {
      await page.goto("/stacks", { waitUntil: "domcontentloaded" })
      await expect(page.getByTestId("stacks-page-title")).toBeVisible({ timeout: 8000 })
    })
  },

  // Navigate to a specific stack detail page and wait for it to render.
  gotoStackDetail: async ({ page }, use) => {
    await use(async (stackId: string) => {
      await page.goto(`/stacks/${stackId}`, { waitUntil: "domcontentloaded" })
      await expect(page.getByTestId("stack-detail-name")).toBeVisible({ timeout: 8000 })
    })
  },

  // Creates a stack via the UI modal. Navigates to the stacks list, opens the
  // create modal, fills in the name (random if not provided), submits, then
  // clicks the new stack card to navigate to the detail page.
  // Returns the stack ID string.
  createStack: async ({ page }, use) => {
    await use(async (name?: string): Promise<string> => {
      const stackName = name ?? `test-stack-${Math.floor(Math.random() * 100000)}`
      console.log(`[createStack] creating: ${stackName}`)

      await page.goto("/stacks")
      await page.waitForLoadState("domcontentloaded")
      await expect(page.getByTestId("stacks-page-title")).toBeVisible({ timeout: 8000 })

      await page.getByTestId("create-stack-button").click()
      await expect(page.getByTestId("create-stack-modal")).toBeVisible()

      await page.getByTestId("stack-name-input").fill(stackName)
      await page.getByTestId("stack-create-submit").click()

      // Wait for modal to close (onSuccess fires after API confirms creation)
      await expect(page.getByTestId("create-stack-modal")).not.toBeVisible({ timeout: 20000 })

      // The stacks list is now refreshed — find the new card and extract its ID from the href
      const stackCard = page.locator('[data-testid^="stack-card-"]').filter({ hasText: stackName })
      await expect(stackCard).toBeVisible({ timeout: 8000 })
      const href = await stackCard.getAttribute("href")
      const stackId = href?.split("/stacks/")[1] ?? ""
      await stackCard.click()

      // Confirm we landed on the detail page
      await expect(page.getByTestId("stack-detail-name")).toBeVisible({ timeout: 8000 })
      console.log(`[createStack] created: ${stackName} id=${stackId}`)
      return stackId
    })
  },

  // Runs docker compose down then deletes the stack. Best-effort for teardown.
  cleanupStack: async ({ page }, use) => {
    await use(async (stackId: string) => {
      try {
        await page.goto(`/stacks/${stackId}`)
        await page.waitForLoadState("domcontentloaded")

        // Dismiss any open dialogs first
        await page.keyboard.press("Escape")

        // Down the stack (stops + removes containers) if the button is present
        const downBtn = page.getByTestId("stack-down-button")
        if (await downBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
          await downBtn.click()
          const downConfirm = page.getByTestId("stack-down-confirm")
          if (await downConfirm.isVisible({ timeout: 3000 }).catch(() => false)) {
            await downConfirm.click()
            // Wait for dialog to close — confirms the mutation succeeded
            await expect(downConfirm).not.toBeVisible({ timeout: 30000 }).catch(() => {})
          }
        }

        // Delete the stack folder
        const deleteBtn = page.getByTestId("stack-delete-button")
        if (await deleteBtn.isEnabled({ timeout: 5000 }).catch(() => false)) {
          await deleteBtn.click()
          const deleteConfirm = page.getByTestId("stack-delete-confirm")
          if (await deleteConfirm.isVisible({ timeout: 3000 }).catch(() => false)) {
            await deleteConfirm.click()
            await page.waitForURL(/\/stacks$/, { timeout: 15000 }).catch(() => {})
          }
        }
      } catch {
        // Best-effort — if the stack is already gone, ignore errors
      }
    })
  },
})

export { expect }
