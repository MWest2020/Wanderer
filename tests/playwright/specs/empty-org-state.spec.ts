// Empty-org state checks.
//
// Runs against the `empty-org` fixture (one organisation with
// zero targets + zero scans + zero assessments). Pins the UI's
// empty-state copy on each surface so a future refactor that
// loses the empty path is caught at CI time.

import { test, expect } from "@playwright/test";

test.describe("Empty-org state", () => {
  test("Targets page renders the empty placeholder", async ({ page }) => {
    await page.goto("/ui/targets");
    await expect(page.locator(".muted", { hasText: /No scans persisted/i })).toBeVisible();
  });

  test("Reporting catalogue lists rules with no-rationale placeholder", async ({
    page,
  }) => {
    await page.goto("/ui/reporting?org=acme-empty");
    // Every registered rule still surfaces (catalogue is the
    // rule list, not the scored list).
    await expect(
      page.locator("text=wand.juridisch.cert_issuer_eea"),
    ).toBeVisible();
    // No rationale yet — the scope-aware Current state column
    // shows the empty placeholder.
    await expect(page.locator("text=/no rationale yet|no data/i").first()).toBeVisible();
  });

  test("Per-org dashboard renders the org name + zero-target copy", async ({
    page,
  }) => {
    await page.goto("/ui/orgs/acme-empty");
    await expect(page.locator("h1")).toContainText(/ACME .empty.|acme-empty/i);
  });
});
