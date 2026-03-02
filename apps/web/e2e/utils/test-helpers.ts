import { type Page, expect } from "@playwright/test"

export interface VMFormData {
  name: string
  node: string
  os?: string
  cpuCores?: number
  memoryGb?: number
  diskGb?: number
  description?: string
  tags?: string
}

/**
 * Fill in the Create VM modal form
 */
export async function fillCreateVMForm(page: Page, data: VMFormData): Promise<void> {
  // Fill in basic info
  await page.getByTestId("vm-name-input").fill(data.name)

  // Select node
  await page.getByTestId("vm-node-select").click()
  await page.getByRole("option", { name: data.node }).click()

  // Select OS if provided
  if (data.os) {
    await page.getByTestId("vm-os-select").click()
    await page.getByRole("option", { name: new RegExp(data.os, "i") }).click()
  }

  // Fill in resources
  if (data.cpuCores) {
    await page.getByTestId("vm-cpu-input").fill(String(data.cpuCores))
  }
  if (data.memoryGb) {
    await page.getByTestId("vm-memory-input").fill(String(data.memoryGb))
  }
  if (data.diskGb) {
    await page.getByTestId("vm-disk-input").fill(String(data.diskGb))
  }

  // Optional fields
  if (data.description) {
    await page.getByTestId("vm-description-input").fill(data.description)
  }
  if (data.tags) {
    await page.getByTestId("vm-tags-input").fill(data.tags)
  }
}

/**
 * Wait for a VM to appear in the table by name
 */
export async function waitForVMInTable(page: Page, name: string, timeout = 10000): Promise<string> {
  // Find the row containing the VM name
  const vmRow = page.locator("[data-testid^='vm-table-row-']").filter({
    has: page.locator("[data-testid^='vm-table-name-']").filter({ hasText: name }),
  })

  await expect(vmRow).toBeVisible({ timeout })

  // Extract VMID from the row
  const vmidCell = vmRow.locator("[data-testid^='vm-table-vmid-']")
  const vmidText = await vmidCell.textContent()

  if (!vmidText) {
    throw new Error(`Could not find VMID for VM "${name}"`)
  }

  return vmidText.trim()
}

/**
 * Wait for VM status to change to expected value
 */
export async function waitForVMStatus(
  page: Page,
  vmid: string,
  expectedStatus: "running" | "stopped" | "paused",
  timeout = 30000,
): Promise<void> {
  const statusLocator = page.getByTestId(`vm-table-status-${vmid}`)

  await expect(async () => {
    const statusText = await statusLocator.textContent()
    expect(statusText?.toLowerCase()).toContain(expectedStatus)
  }).toPass({ timeout })
}

/**
 * Wait for VM detail status to show expected value
 */
export async function waitForVMDetailStatus(
  page: Page,
  expectedStatus: "running" | "stopped" | "paused",
  timeout = 30000,
): Promise<void> {
  const statusLocator = page.getByTestId("vm-detail-status")

  await expect(async () => {
    const statusText = await statusLocator.textContent()
    expect(statusText?.toLowerCase()).toContain(expectedStatus)
  }).toPass({ timeout })
}

/**
 * Start a VM from the detail page
 */
export async function startVMFromDetail(page: Page): Promise<void> {
  const startButton = page.getByTestId("vm-start-button")
  await expect(startButton).toBeVisible()
  await startButton.click()
}

/**
 * Stop a VM from the detail page
 */
export async function stopVMFromDetail(page: Page): Promise<void> {
  const stopButton = page.getByTestId("vm-stop-button")
  await expect(stopButton).toBeVisible()
  await stopButton.click()
}

/**
 * Generate a unique VM name for testing
 */
export function generateVMName(prefix = "test-vm"): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).substring(2, 7)}`
}
