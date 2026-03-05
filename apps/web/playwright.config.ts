import { defineConfig, devices } from "@playwright/test"

/**
 * See https://playwright.dev/docs/test-configuration.
 */
export default defineConfig({
  testDir: "./e2e",
  globalSetup: "./e2e/global-setup.ts",
  /* Run tests in files in parallel */
  fullyParallel: false,
  /* Fail the build on CI if you accidentally left test.only in the source code. */
  forbidOnly: !!process.env.CI,
  /* Retry on CI only */
  retries: process.env.CI ? 2 : 0,
  /* Opt out of parallel tests on CI. */
  workers: process.env.CI ? 1 : 1,
  /* Reporter to use. See https://playwright.dev/docs/test-reporters */
  reporter: [
    [process.env.CI ? "dot" : "list"],
    // ["html", { open: "never" }],
  ],
  /* Shared settings for all the projects below. See https://playwright.dev/docs/api/class-testoptions. */
  use: {
    /* Base URL to use in actions like `await page.goto('/')`. */
    baseURL: process.env.LAB_BINARY_PATH ? "http://localhost:8080" : "http://localhost:3000",

    /* Collect trace when retrying the failed test. See https://playwright.dev/docs/trace-viewer */
    trace: "on-first-retry",

    /* Capture screenshot on failure */
    screenshot: "only-on-failure",

    /* Capture video on failure */
    video: "on-first-retry",
  },

  /* Configure projects for major browsers */
  projects: [
    {
      name: "auth",
      testMatch: "**/auth.spec.ts",
      timeout: 30000,
      use: { ...devices["Desktop Chrome"] },
    },
    {
      name: "app",
      testMatch: "**/!(auth).spec.ts",
      dependencies: ["auth"],
      timeout: 180000,
      use: {
        ...devices["Desktop Chrome"],
        storageState: "e2e/.auth/user.json",
      },
    },
  ],

  /* Run your local dev servers before starting the tests */
  webServer: process.env.LAB_BINARY_PATH 
    ? [
        {
          command: `${process.env.LAB_BINARY_PATH}`,
          url: "http://localhost:8080/health",
          reuseExistingServer: !process.env.CI,
          timeout: 15000,
        }
      ]
    : [
        {
          command: "cd ../api && go run ./cmd/server",
          url: "http://localhost:8080/health",
          reuseExistingServer: !process.env.CI,
          timeout: 15000,
        },
        {
          command: "pnpm --filter web build && pnpm --filter web start",
          // command: "pnpm --filter web dev",
          url: "http://localhost:3000",
          reuseExistingServer: !process.env.CI,
          timeout: 60000,
          env: {
            TURBO_UI: "0",
            FORCE_COLOR: "0",
          },
        },
      ],
})
