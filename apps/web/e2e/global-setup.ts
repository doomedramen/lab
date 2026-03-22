import { chromium, FullConfig } from "@playwright/test";
import { mkdirSync } from "fs";

const TEST_EMAIL = process.env.E2E_TEST_EMAIL || "test@example.com";
const TEST_PASSWORD = process.env.E2E_TEST_PASSWORD || "TestP@ssw0rd!";
const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

async function tryLogin(
  email: string,
  password: string,
): Promise<{ accessToken?: string; ok: boolean }> {
  try {
    const res = await fetch(`${API_URL}/lab.v1.AuthService/Login`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Connect-Protocol-Version": "1",
      },
      body: JSON.stringify({ email, password }),
    });
    if (res.ok) {
      const body = await res.json();
      return { ok: true, accessToken: body.accessToken };
    }
    return { ok: false };
  } catch {
    return { ok: false };
  }
}

async function tryRegister(email: string, password: string): Promise<boolean> {
  try {
    const res = await fetch(`${API_URL}/lab.v1.AuthService/Register`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Connect-Protocol-Version": "1",
      },
      body: JSON.stringify({ email, password }),
    });
    if (!res.ok) {
      const body = await res.text();
      console.warn("[global-setup] Registration failed:", body);
      return false;
    }
    return true;
  } catch (err) {
    console.warn("[global-setup] Could not register test user:", err);
    return false;
  }
}

export default async function globalSetup(_config: FullConfig) {
  mkdirSync("e2e/.auth", { recursive: true });

  // Try to login first; register only if that fails
  const loginResult = await tryLogin(TEST_EMAIL, TEST_PASSWORD);
  if (!loginResult.ok) {
    console.log("[global-setup] Login failed, attempting registration...");
    await tryRegister(TEST_EMAIL, TEST_PASSWORD);
    const retryResult = await tryLogin(TEST_EMAIL, TEST_PASSWORD);
    if (!retryResult.ok) {
      throw new Error(
        `[global-setup] Could not authenticate as ${TEST_EMAIL} — check credentials or API availability`,
      );
    }
  }

  // Use real browser login to capture full storageState (localStorage + cookies)
  const browser = await chromium.launch();
  const page = await browser.newPage();
  try {
    await page.goto(`${API_URL}/login`, { waitUntil: "domcontentloaded" });
    await page.getByTestId("login-email-input").fill(TEST_EMAIL);
    await page.getByTestId("login-password-input").fill(TEST_PASSWORD);
    await page.getByTestId("login-submit-button").click();
    await page.waitForURL(/\/(dashboard|vms)/, { timeout: 8000 });
    await page.context().storageState({ path: "e2e/.auth/user.json" });
    console.log("[global-setup] Auth state saved to e2e/.auth/user.json");
  } finally {
    await browser.close();
  }
}
