import { test, expect } from "./fixtures"

/**
 * Stacks E2E Tests
 *
 * Tests run against the real API + Docker backend (no mocks).
 * Requires stacks_dir to be configured in the API config and Docker to be available.
 * Tests that create stacks perform cleanup in afterEach.
 */

test.describe("Stacks — list page", () => {
  test("renders stacks page title and create button", async ({ gotoStacks, page }) => {
    await gotoStacks()
    await expect(page.getByTestId("stacks-page-title")).toBeVisible()
    await expect(page.getByTestId("create-stack-button")).toBeVisible()
  })

  test("stacks list container is present", async ({ gotoStacks, page }) => {
    await gotoStacks()
    await expect(page.getByTestId("stacks-list")).toBeVisible()
  })
})

test.describe("Stacks — create modal", () => {
  test("opens create stack modal on button click", async ({ gotoStacks, page }) => {
    await gotoStacks()
    await page.getByTestId("create-stack-button").click()
    await expect(page.getByTestId("create-stack-modal")).toBeVisible()
    await expect(page.getByTestId("stack-name-input")).toBeVisible()
    await expect(page.getByTestId("stack-create-submit")).toBeVisible()
  })

  test("submit button is disabled when name is empty", async ({ gotoStacks, page }) => {
    await gotoStacks()
    await page.getByTestId("create-stack-button").click()
    await expect(page.getByTestId("create-stack-modal")).toBeVisible()
    // Name field is empty — submit should be disabled
    await expect(page.getByTestId("stack-create-submit")).toBeDisabled()
  })

  test("shows validation error for names with invalid characters", async ({ gotoStacks, page }) => {
    await gotoStacks()
    await page.getByTestId("create-stack-button").click()
    await expect(page.getByTestId("create-stack-modal")).toBeVisible()

    // Type a name with a space (invalid)
    await page.getByTestId("stack-name-input").fill("invalid name!")
    await expect(page.getByText(/only letters/i)).toBeVisible()
    // Submit should remain disabled after invalid input
    await expect(page.getByTestId("stack-create-submit")).toBeDisabled()
  })

  test("valid name enables the submit button", async ({ gotoStacks, page }) => {
    await gotoStacks()
    await page.getByTestId("create-stack-button").click()
    await expect(page.getByTestId("create-stack-modal")).toBeVisible()

    await page.getByTestId("stack-name-input").fill("valid-name-123")
    await expect(page.getByTestId("stack-create-submit")).toBeEnabled()
  })

  test("compose tab shows Monaco editor", async ({ gotoStacks, page }) => {
    await gotoStacks()
    await page.getByTestId("create-stack-button").click()
    await expect(page.getByTestId("create-stack-modal")).toBeVisible()

    // Compose tab should be active by default
    await expect(page.getByRole("tab", { name: /docker-compose/i })).toBeVisible()
    await expect(page.locator(".monaco-editor").first()).toBeVisible({ timeout: 10000 })
  })

  test("env tab shows Monaco editor when clicked", async ({ gotoStacks, page }) => {
    await gotoStacks()
    await page.getByTestId("create-stack-button").click()
    await expect(page.getByTestId("create-stack-modal")).toBeVisible()

    await page.getByRole("tab", { name: /\.env/i }).click()
    await expect(page.locator(".monaco-editor").first()).toBeVisible({ timeout: 10000 })
  })

  test("cancel button closes modal without navigating", async ({ gotoStacks, page }) => {
    await gotoStacks()
    await page.getByTestId("create-stack-button").click()
    await expect(page.getByTestId("create-stack-modal")).toBeVisible()

    await page.getByRole("button", { name: "Cancel" }).click()
    await expect(page.getByTestId("create-stack-modal")).not.toBeVisible()
    await expect(page).toHaveURL(/\/stacks$/)
  })
})

test.describe("Stacks — create and detail page", () => {
  let createdStackId: string | null = null

  test.afterEach(async ({ cleanupStack }) => {
    if (createdStackId) {
      await cleanupStack(createdStackId).catch(() => {})
      createdStackId = null
    }
  })

  test("creating a stack navigates to detail page with correct name", async ({ page, gotoStacks }) => {
    const stackName = `test-detail-${Math.floor(Math.random() * 100000)}`
    console.log(`[stack-detail] creating: ${stackName}`)

    await gotoStacks()
    await page.getByTestId("create-stack-button").click()
    await expect(page.getByTestId("create-stack-modal")).toBeVisible()

    await page.getByTestId("stack-name-input").fill(stackName)
    await page.getByTestId("stack-create-submit").click()

    // Modal closes after successful creation
    await expect(page.getByTestId("create-stack-modal")).not.toBeVisible({ timeout: 20000 })

    // New stack card appears in list — click it
    const stackCard = page.locator('[data-testid^="stack-card-"]').filter({ hasText: stackName })
    await expect(stackCard).toBeVisible({ timeout: 8000 })
    await stackCard.click()

    // Detail page renders
    await expect(page.getByTestId("stack-detail-name")).toBeVisible({ timeout: 8000 })
    createdStackId = page.url().split("/stacks/")[1] ?? null
    console.log(`[stack-detail] id=${createdStackId}`)

    await expect(page.getByTestId("stack-detail-name")).toContainText(stackName)
    await expect(page.getByTestId("stack-detail-id")).toBeVisible()
    await expect(page.getByTestId("stack-detail-status")).toBeVisible()
  })

  test("detail page shows all four tabs", async ({ createStack, page }) => {
    createdStackId = await createStack()

    await expect(page.getByRole("tab", { name: /Containers/i })).toBeVisible()
    await expect(page.getByRole("tab", { name: /docker-compose/i })).toBeVisible()
    await expect(page.getByRole("tab", { name: /\.env/i })).toBeVisible()
    await expect(page.getByRole("tab", { name: /Logs/i })).toBeVisible()
  })

  test("detail page shows all action buttons", async ({ createStack, page }) => {
    createdStackId = await createStack()

    // A freshly created (stopped) stack shows Start but not Restart
    await expect(page.getByTestId("stack-start-button")).toBeVisible()
    await expect(page.getByTestId("stack-restart-button")).not.toBeVisible()
    // Down and Delete are always present
    await expect(page.getByTestId("stack-down-button")).toBeVisible()
    await expect(page.getByTestId("stack-delete-button")).toBeVisible()
  })

  test("back link navigates to stacks list", async ({ createStack, page }) => {
    createdStackId = await createStack()

    await page.getByRole("link", { name: /back to stacks/i }).click()
    await expect(page).toHaveURL(/\/stacks$/)
  })

  test("compose tab renders Monaco editor with YAML content", async ({ createStack, page }) => {
    createdStackId = await createStack()

    await page.getByRole("tab", { name: /docker-compose/i }).click()
    await expect(page.locator(".monaco-editor").first()).toBeVisible({ timeout: 10000 })
    // Should contain some YAML content from the default template
    const editorContent = await page.locator(".monaco-editor").first().textContent()
    expect(editorContent).toBeTruthy()
  })

  test("env tab renders Monaco editor", async ({ createStack, page }) => {
    createdStackId = await createStack()

    await page.getByRole("tab", { name: /\.env/i }).click()
    await expect(page.locator(".monaco-editor").first()).toBeVisible({ timeout: 10000 })
  })

  test("created stack appears in the stacks list", async ({ createStack, page }) => {
    const stackName = `test-list-appear-${Math.floor(Math.random() * 100000)}`
    createdStackId = await createStack(stackName)

    await page.goto("/stacks")
    await page.waitForLoadState("domcontentloaded")
    await expect(page.getByTestId("stacks-page-title")).toBeVisible({ timeout: 8000 })

    // Stack card should be present
    const stackCard = page.locator('[data-testid^="stack-card-"]').filter({ hasText: stackName })
    await expect(stackCard).toBeVisible({ timeout: 8000 })
  })
})

test.describe("Stacks — lifecycle: start, stop, delete", () => {
  let createdStackId: string | null = null

  test.afterEach(async ({ cleanupStack }) => {
    if (createdStackId) {
      await cleanupStack(createdStackId).catch(() => {})
      createdStackId = null
    }
  })

  test("start → running → stop → stopped → delete", async ({ createStack, page }) => {
    createdStackId = await createStack()
    console.log(`[stack-lifecycle] id=${createdStackId} — starting`)

    // --- Start ---
    await expect(page.getByTestId("stack-start-button")).toBeVisible({ timeout: 8000 })
    await page.getByTestId("stack-start-button").click()

    const statusLocator = page.getByTestId("stack-detail-status")

    // Poll until running (image pull + container start can take a while)
    await expect(async () => {
      await page.reload({ waitUntil: "networkidle" })
      const status = await statusLocator.textContent()
      console.log(`[stack-lifecycle] start poll: "${status}"`)
      expect(status?.toLowerCase()).toContain("running")
    }).toPass({ timeout: 120000, intervals: [2000, 3000, 5000] })
    console.log(`[stack-lifecycle] running`)

    // When running: Restart button visible, Start button hidden, Stop button visible
    await expect(page.getByTestId("stack-restart-button")).toBeVisible()
    await expect(page.getByTestId("stack-start-button")).not.toBeVisible()
    await expect(page.getByTestId("stack-stop-button")).toBeVisible()

    // --- Stop ---
    console.log(`[stack-lifecycle] stopping`)
    await page.getByTestId("stack-stop-button").click()

    await expect(async () => {
      await page.reload({ waitUntil: "networkidle" })
      const status = await statusLocator.textContent()
      console.log(`[stack-lifecycle] stop poll: "${status}"`)
      expect(status?.toLowerCase()).toContain("stopped")
    }).toPass({ timeout: 30000, intervals: [500, 1000, 2000] })
    console.log(`[stack-lifecycle] stopped`)

    // When stopped: Start button visible again, Restart and Stop hidden
    await expect(page.getByTestId("stack-start-button")).toBeVisible()
    await expect(page.getByTestId("stack-restart-button")).not.toBeVisible()

    // --- Delete ---
    console.log(`[stack-lifecycle] deleting`)
    await page.getByTestId("stack-delete-button").click()
    await expect(page.getByTestId("stack-delete-confirm")).toBeVisible()
    await page.getByTestId("stack-delete-confirm").click()

    await expect(page).toHaveURL(/\/stacks$/, { timeout: 15000 })
    console.log(`[stack-lifecycle] deleted, back at /stacks`)
    createdStackId = null // already deleted — skip afterEach cleanup
  })

  test("restart button is visible when stack is running", async ({ createStack, page }) => {
    createdStackId = await createStack()
    console.log(`[stack-restart-btn] id=${createdStackId}`)

    await expect(page.getByTestId("stack-start-button")).toBeVisible({ timeout: 8000 })
    await page.getByTestId("stack-start-button").click()

    const statusLocator = page.getByTestId("stack-detail-status")
    await expect(async () => {
      await page.reload({ waitUntil: "networkidle" })
      const status = await statusLocator.textContent()
      console.log(`[stack-restart-btn] start poll: "${status}"`)
      expect(status?.toLowerCase()).toContain("running")
    }).toPass({ timeout: 120000, intervals: [2000, 3000, 5000] })

    await expect(page.getByTestId("stack-restart-button")).toBeVisible()
    await expect(page.getByTestId("stack-start-button")).not.toBeVisible()
  })
})

test.describe("Stacks — confirmation dialogs", () => {
  let createdStackId: string | null = null

  test.afterEach(async ({ cleanupStack }) => {
    if (createdStackId) {
      await cleanupStack(createdStackId).catch(() => {})
      createdStackId = null
    }
  })

  test("down dialog: cancel keeps stack unchanged", async ({ createStack, page }) => {
    createdStackId = await createStack()
    console.log(`[down-dialog] id=${createdStackId}`)

    // Click Down
    await page.getByTestId("stack-down-button").click()

    // Confirmation dialog appears
    await expect(page.getByTestId("stack-down-confirm")).toBeVisible()

    // Cancel — dialog should close, stack remains
    await page.getByRole("button", { name: "Cancel" }).click()
    await expect(page.getByTestId("stack-down-confirm")).not.toBeVisible()

    // Still on detail page, stack detail still shows
    await expect(page.getByTestId("stack-detail-name")).toBeVisible()
    await expect(page).toHaveURL(/\/stacks\/.+/)
  })

  test("down dialog contains informative description", async ({ createStack, page }) => {
    createdStackId = await createStack()

    await page.getByTestId("stack-down-button").click()
    await expect(page.getByTestId("stack-down-confirm")).toBeVisible()

    // Dialog should mention "docker compose down"
    const dialog = page.locator('[role="alertdialog"]')
    await expect(dialog).toContainText(/docker compose down/i)

    await page.getByRole("button", { name: "Cancel" }).click()
  })

  test("delete dialog: cancel keeps stack unchanged", async ({ createStack, page }) => {
    createdStackId = await createStack()
    console.log(`[delete-dialog] id=${createdStackId}`)

    // Click Delete
    await page.getByTestId("stack-delete-button").click()

    // Confirmation dialog appears
    await expect(page.getByTestId("stack-delete-confirm")).toBeVisible()

    // Cancel
    await page.getByRole("button", { name: "Cancel" }).click()
    await expect(page.getByTestId("stack-delete-confirm")).not.toBeVisible()

    // Still on detail page
    await expect(page.getByTestId("stack-detail-name")).toBeVisible()
    await expect(page).toHaveURL(/\/stacks\/.+/)
  })

  test("delete dialog contains permanent-deletion warning", async ({ createStack, page }) => {
    createdStackId = await createStack()

    await page.getByTestId("stack-delete-button").click()
    await expect(page.getByTestId("stack-delete-confirm")).toBeVisible()

    const dialog = page.locator('[role="alertdialog"]')
    await expect(dialog).toContainText(/cannot be undone/i)

    await page.getByRole("button", { name: "Cancel" }).click()
  })

  test("deleting a stack redirects to the stacks list", async ({ createStack, page }) => {
    createdStackId = await createStack()
    console.log(`[delete-redirect] id=${createdStackId}`)

    await page.getByTestId("stack-delete-button").click()
    await expect(page.getByTestId("stack-delete-confirm")).toBeVisible()
    await page.getByTestId("stack-delete-confirm").click()

    await expect(page).toHaveURL(/\/stacks$/, { timeout: 15000 })
    console.log(`[delete-redirect] redirected to /stacks`)
    createdStackId = null
  })
})

test.describe("Stacks — compose and env editing", () => {
  let createdStackId: string | null = null

  test.afterEach(async ({ cleanupStack }) => {
    if (createdStackId) {
      await cleanupStack(createdStackId).catch(() => {})
      createdStackId = null
    }
  })

  test("compose tab has a Save Compose button", async ({ createStack, page }) => {
    createdStackId = await createStack()

    await page.getByRole("tab", { name: /docker-compose/i }).click()
    await expect(page.getByRole("button", { name: /save compose/i })).toBeVisible()
  })

  test("env tab has a Save Env button", async ({ createStack, page }) => {
    createdStackId = await createStack()

    await page.getByRole("tab", { name: /\.env/i }).click()
    await expect(page.getByRole("button", { name: /save env/i })).toBeVisible()
  })
})
