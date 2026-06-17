// Rule catalogue scenarios — now hosted on the Trends (Farmer) tab.
//
// After ADR 0017 the catalogue lives at /ui/trends alongside the
// score matrix. Covers:
//   - Catalogue lists rules with descriptions
//   - Catalogue carries a per-rule status hint

import { test, expect } from "@playwright/test";

test.describe("Trends rule catalogue", () => {
  test("Lists every registered rule with description", async ({ page }) => {
    await page.goto("/ui/trends");
    await expect(page.locator("text=wand.juridisch.cert_issuer_eea")).toBeVisible();
    await expect(page.locator("text=eucsf.sov2.cert_issuer_eu")).toBeVisible();
    await expect(page.locator("text=TLS certificate issued by an authority in the EEA.")).toBeVisible();
  });

  test("Status column shows worst score + target count", async ({ page }) => {
    await page.goto("/ui/trends?org=conduction");
    const row = page.locator("table.rule-catalogue tr", { hasText: "wand.juridisch.cert_issuer_eea" });
    await expect(row).toBeVisible();
    const statusCell = row.locator("td").last();
    await expect(statusCell.locator(".badge")).toBeVisible();
    await expect(statusCell).toContainText(/\d+ of \d+ targets?/);
  });
});
