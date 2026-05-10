// Playwright config for the Wanderer UI smoke layer.
//
// The Makefile target `make playwright` boots a `wanderer serve`
// instance (via Playwright's webServer block) against a temp
// SQLite seeded with the demo dataset, then runs the specs in
// tests/playwright/specs/.
//
// Chromium headless is the contract — no cross-browser matrix.
// See openspec/changes/archive/2026-05-10-add-playwright-adr-smoke-tests/proposal.md
// for the scope decisions.

import { defineConfig, devices } from "@playwright/test";

const port = process.env.WANDERER_PLAYWRIGHT_PORT ?? "8281";
const baseURL = `http://127.0.0.1:${port}`;

export default defineConfig({
  testDir: "./specs",
  fullyParallel: false, // SQLite is single-writer; keep tests serial
  forbidOnly: !!process.env.CI,
  retries: 0,
  reporter: [["list"], ["html", { outputFolder: "playwright-report", open: "never" }]],
  use: {
    baseURL,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
  webServer: {
    // The Makefile target builds + seeds before invoking this
    // file; the webServer block just attaches to the resulting
    // binary so `npm test` standalone is a no-go without first
    // running `make playwright-fixture`.
    command:
      "../../bin/wanderer serve " +
      `-addr 127.0.0.1:${port} ` +
      `-db ${process.env.WANDERER_PLAYWRIGHT_DB ?? "/tmp/wanderer-playwright.db"} ` +
      "-ui -no-geoip",
    url: `${baseURL}/healthz`,
    reuseExistingServer: false,
    stdout: "pipe",
    stderr: "pipe",
    timeout: 30_000,
  },
});
