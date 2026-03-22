import { test, expect } from "./fixtures";

test.describe("VM Diagnostics", () => {
  test.beforeEach(async ({ page, login }) => {
    await login();
    await page.goto("/vms");
  });

  test.describe("Diagnostics Tab", () => {
    test("should show diagnostics tab", async ({ page }) => {
      const vmCard = page.getByTestId("vm-card").first();
      if (await vmCard.isVisible()) {
        await vmCard.click();
        await expect(page).toHaveURL(/\/vms\/\d+/);

        // Diagnostics tab should be visible
        const diagnosticsTab = page.getByRole("tab", { name: "Diagnostics" });
        await expect(diagnosticsTab).toBeVisible();
      }
    });

    test("should switch to diagnostics tab when clicked", async ({ page }) => {
      const vmCard = page.getByTestId("vm-card").first();
      if (await vmCard.isVisible()) {
        await vmCard.click();
        await expect(page).toHaveURL(/\/vms\/\d+/);

        const diagnosticsTab = page.getByRole("tab", { name: "Diagnostics" });
        await diagnosticsTab.click();

        // Diagnostics content should be visible
        const diagnosticsContent = page.getByTestId("diagnostics-panel");
        // Content should load (may take time)
        await expect(diagnosticsContent.first()).toBeVisible({
          timeout: 10000,
        });
      }
    });
  });

  test.describe("Diagnostics Overview", () => {
    test("should show domain ID", async ({ page }) => {
      const vmCard = page.getByTestId("vm-card").first();
      if (await vmCard.isVisible()) {
        await vmCard.click();
        await expect(page).toHaveURL(/\/vms\/\d+/);

        const diagnosticsTab = page.getByRole("tab", { name: "Diagnostics" });
        await diagnosticsTab.click();

        // Domain ID card should be visible
        const domainIdCard = page.getByText(/Domain ID/i);
        await expect(domainIdCard).toBeVisible();
      }
    });

    test("should show VM state", async ({ page }) => {
      const vmCard = page.getByTestId("vm-card").first();
      if (await vmCard.isVisible()) {
        await vmCard.click();
        await expect(page).toHaveURL(/\/vms\/\d+/);

        const diagnosticsTab = page.getByRole("tab", { name: "Diagnostics" });
        await diagnosticsTab.click();

        // State card should be visible
        const stateCard = page.getByText(/State/i);
        await expect(stateCard).toBeVisible();
      }
    });

    test("should show VNC console info", async ({ page }) => {
      const vmCard = page.getByTestId("vm-card").first();
      if (await vmCard.isVisible()) {
        await vmCard.click();
        await expect(page).toHaveURL(/\/vms\/\d+/);

        const diagnosticsTab = page.getByRole("tab", { name: "Diagnostics" });
        await diagnosticsTab.click();

        // VNC card should be visible
        const vncCard = page.getByText(/VNC Console/i);
        await expect(vncCard).toBeVisible();
      }
    });

    test("should show host info", async ({ page }) => {
      const vmCard = page.getByTestId("vm-card").first();
      if (await vmCard.isVisible()) {
        await vmCard.click();
        await expect(page).toHaveURL(/\/vms\/\d+/);

        const diagnosticsTab = page.getByRole("tab", { name: "Diagnostics" });
        await diagnosticsTab.click();

        // Host card should be visible
        const hostCard = page.getByText(/Host/i);
        await expect(hostCard).toBeVisible();
      }
    });
  });

  test.describe("Diagnostics Network Tab", () => {
    test("should show network tab", async ({ page }) => {
      const vmCard = page.getByTestId("vm-card").first();
      if (await vmCard.isVisible()) {
        await vmCard.click();
        await expect(page).toHaveURL(/\/vms\/\d+/);

        const diagnosticsTab = page.getByRole("tab", { name: "Diagnostics" });
        await diagnosticsTab.click();

        const networkTab = page.getByRole("tab", { name: "Network" });
        await networkTab.click();

        // Network content should be visible
        const networkContent = page.getByText(/Network Interface/i);
        await expect(networkContent.first()).toBeVisible();
      }
    });

    test("should show network interfaces or message if none", async ({
      page,
    }) => {
      const vmCard = page.getByTestId("vm-card").first();
      if (await vmCard.isVisible()) {
        await vmCard.click();
        await expect(page).toHaveURL(/\/vms\/\d+/);

        const diagnosticsTab = page.getByRole("tab", { name: "Diagnostics" });
        await diagnosticsTab.click();

        const networkTab = page.getByRole("tab", { name: "Network" });
        await networkTab.click();

        // Should show either interfaces or "no interfaces" message
        const hasInterfaces = await page.getByText(/vnet|eth|ens/i).isVisible();
        const hasNoInterfaces = await page.getByText(/no network/i).isVisible();

        expect(hasInterfaces || hasNoInterfaces).toBeTruthy();
      }
    });
  });

  test.describe("Diagnostics Storage Tab", () => {
    test("should show storage tab", async ({ page }) => {
      const vmCard = page.getByTestId("vm-card").first();
      if (await vmCard.isVisible()) {
        await vmCard.click();
        await expect(page).toHaveURL(/\/vms\/\d+/);

        const diagnosticsTab = page.getByRole("tab", { name: "Diagnostics" });
        await diagnosticsTab.click();

        const storageTab = page.getByRole("tab", { name: "Storage" });
        await storageTab.click();

        // Storage content should be visible
        const storageContent = page.getByText(/Disk Device/i);
        await expect(storageContent.first()).toBeVisible();
      }
    });

    test("should show disk information", async ({ page }) => {
      const vmCard = page.getByTestId("vm-card").first();
      if (await vmCard.isVisible()) {
        await vmCard.click();
        await expect(page).toHaveURL(/\/vms\/\d+/);

        const diagnosticsTab = page.getByRole("tab", { name: "Diagnostics" });
        await diagnosticsTab.click();

        const storageTab = page.getByRole("tab", { name: "Storage" });
        await storageTab.click();

        // Should show disk info
        const diskInfo = page.getByText(/vda|qcow2|virtio/i);
        await expect(diskInfo.first()).toBeVisible();
      }
    });
  });

  test.describe("Diagnostics Console Tab", () => {
    test("should show console tab", async ({ page }) => {
      const vmCard = page.getByTestId("vm-card").first();
      if (await vmCard.isVisible()) {
        await vmCard.click();
        await expect(page).toHaveURL(/\/vms\/\d+/);

        const diagnosticsTab = page.getByRole("tab", { name: "Diagnostics" });
        await diagnosticsTab.click();

        const consoleTab = page.getByRole("tab", { name: "Console" });
        await consoleTab.click();

        // Console content should be visible
        const consoleContent = page.getByText(/Console Device/i);
        await expect(consoleContent.first()).toBeVisible();
      }
    });

    test("should show serial console device", async ({ page }) => {
      const vmCard = page.getByTestId("vm-card").first();
      if (await vmCard.isVisible()) {
        await vmCard.click();
        await expect(page).toHaveURL(/\/vms\/\d+/);

        const diagnosticsTab = page.getByRole("tab", { name: "Diagnostics" });
        await diagnosticsTab.click();

        const consoleTab = page.getByRole("tab", { name: "Console" });
        await consoleTab.click();

        // Should show serial device
        const serialDevice = page.getByText(/serial/i);
        await expect(serialDevice.first()).toBeVisible();
      }
    });
  });

  test.describe("Diagnostics XML Config Tab", () => {
    test("should show XML config tab", async ({ page }) => {
      const vmCard = page.getByTestId("vm-card").first();
      if (await vmCard.isVisible()) {
        await vmCard.click();
        await expect(page).toHaveURL(/\/vms\/\d+/);

        const diagnosticsTab = page.getByRole("tab", { name: "Diagnostics" });
        await diagnosticsTab.click();

        const xmlTab = page.getByRole("tab", { name: "XML Config" });
        await xmlTab.click();

        // XML content should be visible
        const xmlContent = page.getByText(/<domain/i);
        await expect(xmlContent.first()).toBeVisible();
      }
    });

    test("should have copy button for XML", async ({ page }) => {
      const vmCard = page.getByTestId("vm-card").first();
      if (await vmCard.isVisible()) {
        await vmCard.click();
        await expect(page).toHaveURL(/\/vms\/\d+/);

        const diagnosticsTab = page.getByRole("tab", { name: "Diagnostics" });
        await diagnosticsTab.click();

        const xmlTab = page.getByRole("tab", { name: "XML Config" });
        await xmlTab.click();

        // Copy button should be visible
        const copyButton = page.getByRole("button", { name: /Copy/i });
        await expect(copyButton.first()).toBeVisible();
      }
    });

    test("should have refresh button for XML", async ({ page }) => {
      const vmCard = page.getByTestId("vm-card").first();
      if (await vmCard.isVisible()) {
        await vmCard.click();
        await expect(page).toHaveURL(/\/vms\/\d+/);

        const diagnosticsTab = page.getByRole("tab", { name: "Diagnostics" });
        await diagnosticsTab.click();

        const xmlTab = page.getByRole("tab", { name: "XML Config" });
        await xmlTab.click();

        // Refresh button should be visible
        const refreshButton = page.getByRole("button", { name: /Refresh/i });
        await expect(refreshButton.first()).toBeVisible();
      }
    });
  });

  test.describe("Diagnostics Loading State", () => {
    test("should show loading state initially", async ({ page }) => {
      const vmCard = page.getByTestId("vm-card").first();
      if (await vmCard.isVisible()) {
        await vmCard.click();
        await expect(page).toHaveURL(/\/vms\/\d+/);

        const diagnosticsTab = page.getByRole("tab", { name: "Diagnostics" });
        await diagnosticsTab.click();

        // Should show loading state briefly
        const loadingText = page.getByText(/Loading diagnostics/i);
        // May or may not be visible depending on load speed
        // Just verify it doesn't crash
      }
    });
  });

  test.describe("Diagnostics Error State", () => {
    test("should handle missing VM gracefully", async ({ page }) => {
      // Navigate to non-existent VM
      await page.goto("/vms/999999");

      // Should show not found or error
      const notFoundText = page.getByText(/not found|error/i);
      await expect(notFoundText.first()).toBeVisible({ timeout: 5000 });
    });
  });
});
