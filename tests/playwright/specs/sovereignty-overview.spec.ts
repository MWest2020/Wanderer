// Sovereignty-overview synthesis panel.
//
// Covers ADR-0015 / propose-sovereignty-overview. Runs against the
// baseline fixture, whose seeded scan has a wand assessment containing
// the flow rules (apex/mx/ns/hyperscaler/third-parties), so the
// "Onderbouwing — soevereiniteit" panel renders on the assessment page.
//
// The panel's rows were rewritten by answer-first-ui (run 04, task 4.1)
// from a `table.flows` into the shared answer-row article markup (one
// plain-language question per flow, evidence collapsed) — this spec was
// updated to match that markup instead of the old table.

import { test, expect } from "@playwright/test";

test.describe("Sovereignty overview", () => {
  test("assessment page shows the synthesis panel with flow rows", async ({
    page,
  }) => {
    // Reach a scan from the targets page, then its assessment page.
    await page.goto("/ui/targets");
    const scanLink = page.locator('a[href*="/ui/scans/"]').first();
    await expect(scanLink).toBeVisible();
    const href = await scanLink.getAttribute("href");
    expect(href).toBeTruthy();
    await page.goto(`${href}/assessment`);

    // The synthesis panel + at least the Hosting / Mail / DNS flows,
    // rendered as answer-row articles (one plain-language question per
    // flow), not a table.
    await expect(
      page.locator("section.sovereignty-overview h2", {
        hasText: /Onderbouwing/i,
      }),
    ).toBeVisible();
    const flows = page.locator("section.sovereignty-overview .answer-row");
    await expect(flows.first()).toBeVisible();
    await expect(
      page
        .locator("section.sovereignty-overview .answer-question", { hasText: /mail|dns|hosting/i })
        .first(),
    ).toBeVisible();

    // The hub-and-spoke SVG renders alongside the table.
    await expect(page.locator("svg.sov-diagram")).toBeVisible();
    await expect(page.locator("svg.sov-diagram circle.hub")).toBeVisible();
    expect(await page.locator("svg.sov-diagram circle.node").count()).toBeGreaterThan(0);
  });

  test("Trends rolls flows up across targets (moved off the door by answer-first-ui)", async ({ page }) => {
    await page.goto("/ui/trends");
    await expect(
      page.locator("section.sovereignty-rollup h2", { hasText: /Sovereignty by flow/i }),
    ).toBeVisible();
    await expect(
      page.locator("section.sovereignty-rollup th", { hasText: /Hosting|Mail|DNS/ }).first(),
    ).toBeVisible();
  });
});
