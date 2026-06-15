// Playwright config for the Wanderer UI smoke layer.
//
// `make playwright` boots three independent `wanderer serve`
// instances (one per hermetic fixture DB) and runs the matching
// spec files against each. Each project is pinned to one
// fixture scenario by `testMatch`.
//
// Scenarios are seeded by `internal/fixtures/main` via
// `make playwright-fixture`. The DBs live under
// `tests/playwright/fixtures/` (gitignored).
//
// Chromium headless is the contract — no cross-browser matrix.
// See openspec/changes/archive/2026-05-10-add-playwright-adr-smoke-tests/proposal.md
// for the original scope decisions.

import { defineConfig, devices } from "@playwright/test";

const baselinePort = "8281";
const agentHostPort = "8282";
const emptyOrgPort = "8283";
const oidcPort = "8284";
const scanDevPort = "8285";

const fixtureDir = "./fixtures";
const wandererBin = "../../bin/wanderer";

function serve(port: string, db: string): string {
  return (
    `${wandererBin} serve ` +
    `-addr 127.0.0.1:${port} ` +
    `-db ${fixtureDir}/${db} ` +
    "-ui -no-geoip"
  );
}

// serveOIDC adds a --config carrying the oidc block. The provider
// is unreachable on purpose: discovery is lazy, so the server boots
// and the unauthenticated-redirect test never contacts it.
function serveOIDC(port: string, db: string): string {
  return serve(port, db) + ` -config ${fixtureDir}/oidc-serve.yaml`;
}

export default defineConfig({
  testDir: "./specs",
  fullyParallel: false, // SQLite is single-writer; keep tests serial
  forbidOnly: !!process.env.CI,
  retries: 0,
  reporter: [["list"], ["html", { outputFolder: "playwright-report", open: "never" }]],
  use: {
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  projects: [
    {
      name: "baseline",
      testMatch: ["dar.spec.ts", "reporting-catalogue.spec.ts", "sovereignty-overview.spec.ts"],
      use: {
        ...devices["Desktop Chrome"],
        baseURL: `http://127.0.0.1:${baselinePort}`,
      },
    },
    {
      name: "agent-host",
      testMatch: [
        "host-side-scoring.spec.ts",
        "nextcloud-as-target.spec.ts",
        "container-image-sovereignty.spec.ts",
        "eu-package-origin.spec.ts",
      ],
      use: {
        ...devices["Desktop Chrome"],
        baseURL: `http://127.0.0.1:${agentHostPort}`,
      },
    },
    {
      name: "empty-org",
      testMatch: ["empty-org-state.spec.ts"],
      use: {
        ...devices["Desktop Chrome"],
        baseURL: `http://127.0.0.1:${emptyOrgPort}`,
      },
    },
    {
      name: "oidc",
      testMatch: ["oidc-login.spec.ts"],
      use: {
        ...devices["Desktop Chrome"],
        baseURL: `http://127.0.0.1:${oidcPort}`,
      },
    },
    {
      name: "scan-dev",
      testMatch: ["ui-dev-scan.spec.ts"],
      use: {
        ...devices["Desktop Chrome"],
        baseURL: `http://127.0.0.1:${scanDevPort}`,
      },
    },
  ],
  webServer: [
    {
      command: serve(baselinePort, "baseline.db"),
      url: `http://127.0.0.1:${baselinePort}/healthz`,
      reuseExistingServer: false,
      stdout: "pipe",
      stderr: "pipe",
      timeout: 30_000,
    },
    {
      command: serve(agentHostPort, "agent-host.db"),
      url: `http://127.0.0.1:${agentHostPort}/healthz`,
      reuseExistingServer: false,
      stdout: "pipe",
      stderr: "pipe",
      timeout: 30_000,
    },
    {
      command: serve(emptyOrgPort, "empty-org.db"),
      url: `http://127.0.0.1:${emptyOrgPort}/healthz`,
      reuseExistingServer: false,
      stdout: "pipe",
      stderr: "pipe",
      timeout: 30_000,
    },
    {
      command: serveOIDC(oidcPort, "oidc.db"),
      url: `http://127.0.0.1:${oidcPort}/healthz`,
      reuseExistingServer: false,
      stdout: "pipe",
      stderr: "pipe",
      timeout: 30_000,
    },
    {
      command: serve(scanDevPort, "scan.db") + " -ui-allow-scan",
      url: `http://127.0.0.1:${scanDevPort}/healthz`,
      reuseExistingServer: false,
      stdout: "pipe",
      stderr: "pipe",
      timeout: 30_000,
    },
  ],
});
