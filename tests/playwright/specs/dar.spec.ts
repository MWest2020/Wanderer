// DAR nav + org-scope persistence smoke test.
//
// Covers the scenarios from these archived OpenSpec changes:
// - openspec/changes/archive/2026-05-10-restructure-dar-layers/specs/web-ui/spec.md
//   "Analysis layer renders the steering matrix"
//   "Reporting layer is the rule catalogue"
//   "Dashboard is is dit goed of niet"
// - openspec/changes/archive/2026-05-10-fix-nav-org-context/specs/web-ui/spec.md
//   "DAR nav persists the active organisation scope"
//   "Targets page accepts an organisation filter"

import { test, expect } from "@playwright/test";

test.describe("DAR layering", () => {
  test("Dashboard at /ui/ stays slim — verdict pills, no matrix", async ({ page }) => {
    await page.goto("/ui/");
    await expect(page.locator("h1")).toContainText("all organisations");
    await expect(page.getByRole("heading", { name: /Verdict/i })).toBeVisible();
    // Removed sections — must not appear:
    await expect(page.locator("text=External posture")).toHaveCount(0);
    await expect(page.locator("text=Internal posture")).toHaveCount(0);
    await expect(page.locator("text=Top concerns")).toHaveCount(0);
    await expect(page.locator("text=Recent activity")).toHaveCount(0);
  });

  test("Analysis renders the rule × score-counts matrix", async ({ page }) => {
    await page.goto("/ui/analysis");
    await expect(page.locator("h1")).toContainText("steering matrix");
    // Matrix has score-count columns. Look for the column header
    // and at least one badge of any score type.
    await expect(page.locator("th", { hasText: "soeverein" })).toBeVisible();
    await expect(page.locator(".reporting-rules .badge").first()).toBeVisible();
  });

  test("Reporting renders the rule catalogue with status column", async ({ page }) => {
    await page.goto("/ui/reporting");
    await expect(page.locator("h1")).toContainText("rule catalogue");
    await expect(page.locator("th", { hasText: "Current state" })).toBeVisible();
    // Must NOT carry the full per-score matrix (that's Analysis).
    await expect(page.locator("th", { hasText: "soeverein" })).toHaveCount(0);
  });
});

test.describe("Organisation scope persistence", () => {
  test("Per-org dashboard threads slug into nav tabs", async ({ page }) => {
    await page.goto("/ui/orgs/conduction");
    await expect(page.locator("h1")).toContainText("Conduction B.V.");

    const nav = page.locator(".nav-bar");
    await expect(nav.locator("a", { hasText: "Dashboard" })).toHaveAttribute("href", "/ui/orgs/conduction");
    await expect(nav.locator("a", { hasText: "Analysis" })).toHaveAttribute("href", "/ui/analysis?org=conduction");
    // Reporting catalogue is a rule reference — but the status
    // column IS scope-aware, so the link threads the slug too.
    await expect(nav.locator("a", { hasText: "Reporting" })).toHaveAttribute("href", "/ui/reporting?org=conduction");
  });

  test("Analysis with ?org= renders the scope pill", async ({ page }) => {
    await page.goto("/ui/analysis?org=conduction");
    await expect(page.locator(".scope-pill")).toContainText("Conduction");
    await expect(page.locator(".scope-pill a")).toHaveAttribute("href", "/ui/orgs/conduction");
  });

  test("Unknown org slug returns 404", async ({ page }) => {
    const response = await page.goto("/ui/orgs/this-org-does-not-exist", { waitUntil: "domcontentloaded" });
    expect(response?.status()).toBe(404);
  });

  test("Targets page filters by ?org=", async ({ page }) => {
    await page.goto("/ui/targets?org=conduction");
    await expect(page.locator(".scope-pill")).toContainText("Conduction");
    // The seeded conduction org owns conduction.nl + rijksoverheid.nl
    // + mijnoverheid.us + the alma host. Other orgs' targets must
    // not appear.
    await expect(page.locator("text=example.nl")).toHaveCount(0);
  });
});

test.describe("Analysis-page nav carries Reporting tab", () => {
  // Regression test for the bug where scan/assessment/drift/targets
  // pages used to pass HasReporting=false to nav.tmpl.
  test("Scan page nav includes Reporting tab", async ({ page }) => {
    await page.goto("/ui/targets");
    // Pick the first scan link and open it.
    const scanLink = page.locator('a[href*="/ui/scans/"]').first();
    await scanLink.click();
    await expect(page).toHaveURL(/\/ui\/scans\//);
    const nav = page.locator(".nav-bar");
    await expect(nav.locator("a", { hasText: "Reporting" })).toBeVisible();
  });
});
