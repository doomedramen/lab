import { test, expect } from "./fixtures"

test.describe("VM Creation and Startup", () => {
  test("should create a new VM using a template", async ({ page }) => {
    const vmName = `test-vm-${Math.floor(Math.random() * 10000)}`

    // Navigate to VMs page
    await page.goto("/vms")
    await page.waitForLoadState("domcontentloaded")
    await expect(page.getByTestId("vms-page-title")).toBeVisible({ timeout: 8000 })

    // Click Create VM button
    await page.getByTestId("create-vm-button").click()

    // Wait for modal to open
    const modal = page.getByTestId("create-vm-modal")
    await expect(modal).toBeVisible()

    // Select first node in the list
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

    // Wait for the VM detail page
    await expect(page.getByTestId("vm-detail-vmid")).toBeVisible({ timeout: 120000 })

    const vmid = page.url().split("/vms/")[1]
    console.log(`Created VM: ${vmName} with ID: ${vmid}`)

    // Verify we're on the VM detail page
    await expect(page.getByTestId("vm-detail-status")).toBeVisible()
  })
})
