// Sovereignty-overview synthesis panel.
//
// Covers ADR-0015 / propose-sovereignty-overview. Runs against the
// baseline fixture, whose seeded scan has a wand assessment containing
// the flow rules (apex/mx/ns/hyperscaler/third-parties), so the
// "Sovereignty overview" panel renders on the assessment page.

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

    // The synthesis panel + at least the Hosting / Mail / DNS flows.
    await expect(
      page.locator("section.sovereignty-overview h2", {
        hasText: /Sovereignty overview/i,
      }),
    ).toBeVisible();
    const flows = page.locator("section.sovereignty-overview table.flows tbody tr");
    await expect(flows.first()).toBeVisible();
    await expect(
      page.locator("section.sovereignty-overview th", { hasText: /Mail|DNS|Hosting/ }).first(),
    ).toBeVisible();

    // The hub-and-spoke SVG renders alongside the table.
    await expect(page.locator("svg.sov-diagram")).toBeVisible();
    await expect(page.locator("svg.sov-diagram circle.hub")).toBeVisible();
    expect(await page.locator("svg.sov-diagram circle.node").count()).toBeGreaterThan(0);
  });

  test("instance dashboard rolls flows up across targets", async ({ page }) => {
    await page.goto("/ui/");
    await expect(
      page.locator("section.sovereignty-rollup h2", { hasText: /Sovereignty by flow/i }),
    ).toBeVisible();
    await expect(
      page.locator("section.sovereignty-rollup th", { hasText: /Hosting|Mail|DNS/ }).first(),
    ).toBeVisible();
  });
});
