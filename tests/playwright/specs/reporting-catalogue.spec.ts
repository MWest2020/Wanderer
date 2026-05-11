// Reporting catalogue scenarios.
//
// Covers the scenarios from:
// - openspec/changes/archive/2026-05-10-add-reporting-status-column/specs/web-ui/spec.md
//   "Reporting layer is the rule catalogue" (modified)
//   - Catalogue lists rules with descriptions
//   - Catalogue carries a per-rule status hint
//   - Rules without rationale render an explicit placeholder

import { test, expect } from "@playwright/test";

test.describe("Reporting catalogue", () => {
  test("Lists every registered rule with description", async ({ page }) => {
    await page.goto("/ui/reporting");
    // At least one rule from each registered pack must be present.
    await expect(page.locator("text=wand.juridisch.cert_issuer_eea")).toBeVisible();
    await expect(page.locator("text=eucsf.sov2.cert_issuer_eu")).toBeVisible();
    // The description for the wand rule is rendered.
    await expect(page.locator("text=TLS certificate issued by an authority in the EEA.")).toBeVisible();
  });

  test("Status column shows worst score + target count", async ({ page }) => {
    await page.goto("/ui/reporting?org=conduction");
    // The cert_issuer_eea rule in the demo data fires on multiple
    // targets, at least one of which is afhankelijk (the demo data
    // has conduction.nl + mijnoverheid.us + example.nl scoring
    // afhankelijk on US-issued certs).
    const row = page.locator("tr", { hasText: "wand.juridisch.cert_issuer_eea" });
    await expect(row).toBeVisible();
    // Last cell is the "Current state" column — pin to it
    // because the description column also carries a `.muted.small`
    // (the "why this matters" details summary).
    const statusCell = row.locator("td").last();
    await expect(statusCell.locator(".badge")).toBeVisible();
    await expect(statusCell).toContainText(/\d+ of \d+ targets?/);
  });

  // The "no rationale yet" placeholder assertion moved to
  // empty-org-state.spec.ts now that the hermetic fixture
  // suite has an empty-org scenario.
});
