import { test, expect } from "./fixtures";

/**
 * VM Full Lifecycle Tests
 *
 * These tests run against the real API + libvirt backend (no mocks).
 * Each test that creates a VM performs cleanup in afterEach where possible.
 * Tests are designed to be independent but some share the "create + start" setup.
 */

test.describe("VM Lifecycle — create, start, stop, delete", () => {
  let createdVmid: string | null = null;

  test.afterEach(async ({ cleanupVM }) => {
    if (createdVmid) {
      await cleanupVM(createdVmid).catch(() => {
        // Teardown is best-effort
      });
      createdVmid = null;
    }
  });

  test("full lifecycle: create → start → stop → delete", async ({ page }) => {
    const vmName = `test-lifecycle-${Math.floor(Math.random() * 100000)}`;

    // --- Create VM ---
    console.log(`[lifecycle] creating VM: ${vmName}`);
    await page.goto("/vms");
    await page.waitForLoadState("domcontentloaded");
    await expect(page.getByTestId("vms-page-title")).toBeVisible({
      timeout: 8000,
    });

    await page.getByTestId("create-vm-button").click();
    const modal = page.getByTestId("create-vm-modal");
    await expect(modal).toBeVisible();

    // Select first node in the list
    await page.getByTestId("vm-template-node-select").click();
    await page.getByRole("option").first().click();

    // Select first template (not "Custom Configuration")
    await page.getByTestId("vm-template-select").click();
    // Get all options and find first one that's not "Custom Configuration"
    const options = page.getByRole("option");
    const optionCount = await options.count();
    for (let i = 0; i < optionCount; i++) {
      const optionText = await options.nth(i).textContent();
      if (optionText && !optionText.includes("Custom")) {
        await options.nth(i).click();
        break;
      }
    }

    // Click on the "Basic" tab
    await page.getByRole("tab", { name: "Basic" }).click();

    // Give the VM a name
    await page.getByTestId("vm-name-input").fill(vmName);

    // Click "Create VM"
    await page.getByTestId("vm-create-submit").click();

    // Wait for VM detail page
    await expect(page.getByTestId("vm-detail-vmid")).toBeVisible({
      timeout: 120000,
    });
    createdVmid = page.url().split("/vms/")[1] ?? null;
    console.log(`[lifecycle] created VM: ${vmName} vmid=${createdVmid}`);

    // --- Start VM ---
    console.log(`[lifecycle] starting VM vmid=${createdVmid}`);
    await expect(page.getByTestId("vm-start-button")).toBeVisible({
      timeout: 8000,
    });
    await page.getByTestId("vm-start-button").click();

    const statusLocator = page.getByTestId("vm-detail-status");
    await expect(async () => {
      const status = await statusLocator.textContent();
      expect(status?.toLowerCase()).toContain("running");
    }).toPass({ timeout: 10000, intervals: [500, 1000, 2000] });
    console.log(`[lifecycle] VM vmid=${createdVmid} is running`);

    // Verify action buttons for running state
    await expect(page.getByTestId("vm-pause-button")).toBeVisible();
    await expect(page.getByTestId("vm-shutdown-button")).toBeVisible();
    await expect(page.getByTestId("vm-stop-button")).toBeVisible();
    await expect(page.getByTestId("vm-reboot-button")).toBeVisible();

    // Console button is enabled when running
    await expect(page.getByTestId("vm-console-button")).toBeEnabled();

    // Delete button is disabled when running
    await expect(page.getByTestId("vm-delete-button")).toBeDisabled();

    // --- Stop VM (force stop with confirmation) ---
    console.log(`[lifecycle] force stopping VM vmid=${createdVmid}`);
    await page.getByTestId("vm-stop-button").click();

    // Confirmation dialog appears
    await expect(page.getByTestId("confirm-dialog")).toBeVisible();
    await expect(page.getByTestId("confirm-dialog-title")).toContainText(
      /force stop/i,
    );

    await page.getByTestId("confirm-dialog-confirm").click();

    // Wait until stopped — mutation onSuccess invalidates query, React Query refetches automatically
    await expect(async () => {
      const status = await statusLocator.textContent();
      console.log(`[lifecycle] stop poll: status="${status}"`);
      expect(status?.toLowerCase()).toContain("stopped");
    }).toPass({ timeout: 10000, intervals: [500, 1000, 2000] });
    console.log(`[lifecycle] VM vmid=${createdVmid} is stopped`);

    // Delete is now enabled
    await expect(page.getByTestId("vm-delete-button")).toBeEnabled();

    // Console button is disabled when stopped
    await expect(page.getByTestId("vm-console-button")).toBeDisabled();

    // --- Delete VM ---
    console.log(`[lifecycle] deleting VM vmid=${createdVmid}`);
    await page.getByTestId("vm-delete-button").click();

    await expect(page.getByTestId("confirm-dialog")).toBeVisible();
    await expect(page.getByTestId("confirm-dialog-title")).toContainText(
      /delete/i,
    );

    // The dialog should warn about permanent data loss
    const dialogContent = page.getByTestId("confirm-dialog");
    await expect(dialogContent).toContainText(/permanent/i);

    await page.getByTestId("confirm-dialog-confirm").click();

    // Should redirect to /vms list
    await expect(page).toHaveURL(/\/vms$/, { timeout: 8000 });
    console.log(
      `[lifecycle] VM vmid=${createdVmid} deleted, redirected to /vms`,
    );

    // VM should no longer appear in the list
    if (createdVmid) {
      const vmRow = page.getByTestId(`vm-table-status-${createdVmid}`);
      await expect(vmRow).not.toBeVisible({ timeout: 10000 });
    }

    createdVmid = null; // Prevent afterEach cleanup since already deleted
  });

  test("delete button states during VM lifecycle", async ({
    createAndStartVM,
    page,
  }) => {
    createdVmid = await createAndStartVM();
    console.log(`[delete-btn-states] vmid=${createdVmid} running`);

    // Delete button should be disabled when VM is running
    await expect(page.getByTestId("vm-delete-button")).toBeDisabled();
    await expect(page.getByTestId("vm-delete-button")).toHaveAttribute(
      "data-disabled",
      "true",
    );

    // Stop the VM first
    console.log(`[delete-btn-states] force stopping vmid=${createdVmid}`);
    await page.getByTestId("vm-stop-button").click();
    await expect(page.getByTestId("confirm-dialog")).toBeVisible();
    await page.getByTestId("confirm-dialog-confirm").click();

    // Wait for VM to stop
    const statusLocator = page.getByTestId("vm-detail-status");
    await expect(async () => {
      const status = await statusLocator.textContent();
      console.log(`[delete-btn-states] stop poll: status="${status}"`);
      expect(status?.toLowerCase()).toContain("stopped");
    }).toPass({ timeout: 10000, intervals: [500, 1000, 2000] });
    console.log(`[delete-btn-states] vmid=${createdVmid} stopped`);

    // Delete button should now be enabled
    await expect(page.getByTestId("vm-delete-button")).toBeEnabled();
    await expect(page.getByTestId("vm-delete-button")).not.toHaveAttribute(
      "data-disabled",
      "true",
    );
  });

  test("delete confirmation dialog validation", async ({
    createAndStartVM,
    page,
  }) => {
    createdVmid = await createAndStartVM();
    console.log(`[delete-dialog] vmid=${createdVmid} running`);

    // Stop the VM to enable delete button
    console.log(`[delete-dialog] force stopping vmid=${createdVmid}`);
    await page.getByTestId("vm-stop-button").click();
    await expect(page.getByTestId("confirm-dialog")).toBeVisible();
    await page.getByTestId("confirm-dialog-confirm").click();

    // Wait for VM to stop
    const statusLocator = page.getByTestId("vm-detail-status");
    await expect(async () => {
      const status = await statusLocator.textContent();
      console.log(`[delete-dialog] stop poll: status="${status}"`);
      expect(status?.toLowerCase()).toContain("stopped");
    }).toPass({ timeout: 10000, intervals: [500, 1000, 2000] });
    console.log(`[delete-dialog] vmid=${createdVmid} stopped`);

    // Click delete button
    await page.getByTestId("vm-delete-button").click();

    // Verify delete dialog appears
    await expect(page.getByTestId("confirm-dialog")).toBeVisible();
    await expect(page.getByTestId("confirm-dialog-title")).toContainText(
      /delete/i,
      { timeout: 5000 },
    );

    // Verify dialog contains warning about permanent data loss
    const dialogContent = page.getByTestId("confirm-dialog");
    await expect(dialogContent).toContainText(/permanent/i);
    await expect(dialogContent).toContainText(/cannot be undone/i);

    // Cancel the deletion
    await page.getByTestId("confirm-dialog-cancel").click();
    await expect(page.getByTestId("confirm-dialog")).not.toBeVisible();

    // VM should still exist (dialog closed, still on detail page)
    await expect(page.getByTestId("vm-detail-vmid")).toBeVisible();
  });

  test("confirmation dialog cancel leaves VM unchanged", async ({
    createAndStartVM,
    page,
  }) => {
    createdVmid = await createAndStartVM();

    // Click stop to trigger confirmation
    await page.getByTestId("vm-stop-button").click();
    await expect(page.getByTestId("confirm-dialog")).toBeVisible();

    // Cancel
    await page.getByTestId("confirm-dialog-cancel").click();
    await expect(page.getByTestId("confirm-dialog")).not.toBeVisible();

    // VM should still be running
    const statusLocator = page.getByTestId("vm-detail-status");
    await expect(statusLocator).toContainText(/running/i);

    // Click delete to trigger confirmation
    // (Delete is disabled when running — test cancel on stop dialog only)
    // Verify delete dialog cancel on a stopped VM
    await page.getByTestId("vm-stop-button").click();
    await expect(page.getByTestId("confirm-dialog")).toBeVisible();
    await page.getByTestId("confirm-dialog-cancel").click();
    await expect(page.getByTestId("confirm-dialog")).not.toBeVisible();
  });
});

test.describe("VM Lifecycle — pause and resume", () => {
  let createdVmid: string | null = null;

  test.afterEach(async ({ cleanupVM }) => {
    if (createdVmid) {
      await cleanupVM(createdVmid).catch(() => {});
      createdVmid = null;
    }
  });

  test("can pause a running VM and resume it", async ({
    createAndStartVM,
    page,
  }) => {
    createdVmid = await createAndStartVM();
    console.log(`[pause-resume] vmid=${createdVmid} running, pausing`);

    // VM is running — pause it
    await expect(page.getByTestId("vm-pause-button")).toBeVisible();
    await page.getByTestId("vm-pause-button").click();

    const statusLocator = page.getByTestId("vm-detail-status");
    await expect(async () => {
      const status = await statusLocator.textContent();
      console.log(`[pause-resume] pause poll: status="${status}"`);
      expect(status?.toLowerCase()).toContain("paused");
    }).toPass({ timeout: 10000, intervals: [500, 1000, 2000] });
    console.log(`[pause-resume] vmid=${createdVmid} paused, resuming`);

    // Resume button should appear (not Pause)
    await expect(page.getByTestId("vm-resume-button")).toBeVisible();
    await expect(page.getByTestId("vm-pause-button")).not.toBeVisible();

    // Resume the VM
    await page.getByTestId("vm-resume-button").click();

    await expect(async () => {
      const status = await statusLocator.textContent();
      console.log(`[pause-resume] resume poll: status="${status}"`);
      expect(status?.toLowerCase()).toContain("running");
    }).toPass({ timeout: 10000, intervals: [500, 1000, 2000] });
    console.log(`[pause-resume] vmid=${createdVmid} running again`);

    // Pause button should return
    await expect(page.getByTestId("vm-pause-button")).toBeVisible();
    await expect(page.getByTestId("vm-resume-button")).not.toBeVisible();
  });
});

test.describe("VM Lifecycle — graceful shutdown", () => {
  let createdVmid: string | null = null;

  test.afterEach(async ({ cleanupVM }) => {
    if (createdVmid) {
      await cleanupVM(createdVmid).catch(() => {});
      createdVmid = null;
    }
  });

  test("can gracefully shutdown a running VM", async ({
    createAndStartVM,
    page,
  }) => {
    createdVmid = await createAndStartVM();
    console.log(
      `[graceful-shutdown] vmid=${createdVmid} running, sending shutdown`,
    );

    await expect(page.getByTestId("vm-shutdown-button")).toBeVisible();
    await page.getByTestId("vm-shutdown-button").click();

    // Confirmation dialog
    await expect(page.getByTestId("confirm-dialog")).toBeVisible();
    await expect(page.getByTestId("confirm-dialog-title")).toContainText(
      /shutdown/i,
    );
    await page.getByTestId("confirm-dialog-confirm").click();

    const statusLocator = page.getByTestId("vm-detail-status");
    await expect(async () => {
      const status = await statusLocator.textContent();
      console.log(`[graceful-shutdown] poll: status="${status}"`);
      expect(status?.toLowerCase()).toContain("stopped");
    }).toPass({ timeout: 30000, intervals: [500, 1000, 2000] });
    console.log(`[graceful-shutdown] vmid=${createdVmid} stopped`);
  });
});

test.describe("VM Lifecycle — reboot", () => {
  let createdVmid: string | null = null;

  test.afterEach(async ({ cleanupVM }) => {
    if (createdVmid) {
      await cleanupVM(createdVmid).catch(() => {});
      createdVmid = null;
    }
  });

  test("can reboot a running VM", async ({ createAndStartVM, page }) => {
    createdVmid = await createAndStartVM();
    console.log(`[reboot] vmid=${createdVmid} running, sending reboot`);

    await expect(page.getByTestId("vm-reboot-button")).toBeVisible();
    await page.getByTestId("vm-reboot-button").click();

    // Confirmation dialog
    await expect(page.getByTestId("confirm-dialog")).toBeVisible();
    await expect(page.getByTestId("confirm-dialog-title")).toContainText(
      /reboot/i,
    );
    await page.getByTestId("confirm-dialog-confirm").click();

    // VM should remain/return to running
    const statusLocator = page.getByTestId("vm-detail-status");
    await expect(async () => {
      const status = await statusLocator.textContent();
      console.log(`[reboot] poll: status="${status}"`);
      expect(status?.toLowerCase()).toContain("running");
    }).toPass({ timeout: 30000, intervals: [500, 1000, 2000] });
    console.log(`[reboot] vmid=${createdVmid} is running again`);
  });
});

test.describe("VM button states", () => {
  let createdVmid: string | null = null;

  test.afterEach(async ({ cleanupVM }) => {
    if (createdVmid) {
      await cleanupVM(createdVmid).catch(() => {});
      createdVmid = null;
    }
  });

  test("delete button is disabled when VM is running", async ({
    createAndStartVM,
    page,
  }) => {
    createdVmid = await createAndStartVM();
    await expect(page.getByTestId("vm-delete-button")).toBeDisabled();
  });

  test("console button is disabled when VM is stopped", async ({ page }) => {
    const vmName = `test-btn-${Math.floor(Math.random() * 100000)}`;

    await page.goto("/vms");
    await page.waitForLoadState("domcontentloaded");
    await expect(page.getByTestId("vms-page-title")).toBeVisible({
      timeout: 8000,
    });

    await page.getByTestId("create-vm-button").click();
    await expect(page.getByTestId("create-vm-modal")).toBeVisible();

    // Select node first (required)
    await page.getByTestId("vm-template-node-select").click();
    await page.getByRole("option").first().click();

    // Select Ubuntu 24.04 LTS template from dropdown
    await page.getByTestId("vm-template-select").click();
    await page.getByRole("option", { name: "Ubuntu 24.04 LTS" }).click();

    // Switch to Basic tab
    await page.getByRole("tab", { name: "Basic" }).click();

    await page.getByTestId("vm-name-input").fill(vmName);

    await page.getByTestId("vm-create-submit").click();
    await expect(page.getByTestId("vm-detail-vmid")).toBeVisible({
      timeout: 120000,
    });
    createdVmid = page.url().split("/vms/")[1] ?? null;

    // VM is freshly created (stopped state)
    await expect(page.getByTestId("vm-start-button")).toBeVisible({
      timeout: 8000,
    });
    await expect(page.getByTestId("vm-console-button")).toBeDisabled();
    await expect(page.getByTestId("vm-delete-button")).toBeEnabled();
  });
});

test.describe("VM Console", () => {
  let createdVmid: string | null = null;

  test.afterEach(async ({ cleanupVM }) => {
    if (createdVmid) {
      await cleanupVM(createdVmid).catch(() => {});
      createdVmid = null;
    }
  });

  test("console button opens console dialog when VM is running", async ({
    createAndStartVM,
    page,
  }) => {
    createdVmid = await createAndStartVM();

    // Console button should be enabled
    await expect(page.getByTestId("vm-console-button")).toBeEnabled();

    // Click console button
    await page.getByTestId("vm-console-button").click();

    // Console dialog should open
    await expect(page.getByTestId("vnc-console-dialog")).toBeVisible({
      timeout: 10000,
    });

    // VNC container should be rendered
    await expect(page.getByTestId("vnc-console-container")).toBeVisible();

    // Close the dialog
    // Use ESC key or find the close button
    await page.keyboard.press("Escape");
    await expect(page.getByTestId("vnc-console-dialog")).not.toBeVisible({
      timeout: 5000,
    });
  });
});
