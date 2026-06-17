// Tourist / Explorer / Farmer information architecture.
//
// Covers ADR docs/decisions/0017-ui-personas.md and the
// restructure-ui-tourist-farmer-explorer OpenSpec change:
//   - Overview leads with the target fleet (Tourist)
//   - The scan assessment is the report (Explorer)
//   - Trends consolidates catalogue + matrix (Farmer)
//   - Nav collapses to two tabs; legacy routes redirect

import { test, expect } from "@playwright/test";

test.describe("Overview = Tourist", () => {
  test("Overview leads with the target fleet, linking to reports", async ({ page }) => {
    await page.goto("/ui/");
    const fleet = page.locator("section.targets-fleet");
    await expect(fleet).toBeVisible();
    // Each row carries a verdict badge and a report link.
    await expect(fleet.locator("table.targets")).toBeVisible();
    await expect(fleet.locator("a", { hasText: "report →" }).first()).toBeVisible();
    // The fleet appears before the verdict-pill section.
    const fleetIdx = await page.locator("section.targets-fleet").evaluate((el) => {
      return Array.from(document.querySelectorAll("main > section")).indexOf(el);
    });
    expect(fleetIdx).toBe(0);
  });
});

test.describe("Report = Explorer", () => {
  test("Clicking a target opens its domain-titled report", async ({ page }) => {
    await page.goto("/ui/");
    const reportLink = page.locator("section.targets-fleet a", { hasText: "report →" }).first();
    await reportLink.click();
    await expect(page).toHaveURL(/\/ui\/scans\/.+\/assessment/);
    // The heading is a domain, not "scan s_...".
    await expect(page.locator("h1")).not.toContainText("scan s_");
  });
});

test.describe("Trends = Farmer", () => {
  test("Trends consolidates the catalogue and the score matrix", async ({ page }) => {
    await page.goto("/ui/trends");
    await expect(page.locator("h1")).toContainText("rules across your fleet");
    await expect(page.getByRole("heading", { name: /Rule catalogue/i })).toBeVisible();
    await expect(page.getByRole("heading", { name: /Score matrix/i })).toBeVisible();
    await expect(page.locator("table.rule-catalogue")).toBeVisible();
    await expect(page.locator("table.reporting-rules")).toBeVisible();
  });
});

test.describe("Two-tab nav + legacy redirects", () => {
  test("Nav shows exactly Overview and Trends", async ({ page }) => {
    await page.goto("/ui/");
    const nav = page.locator(".nav-bar");
    await expect(nav.locator("a", { hasText: "Overview" })).toBeVisible();
    await expect(nav.locator("a", { hasText: "Trends" })).toBeVisible();
    await expect(nav.locator("a", { hasText: "Analysis" })).toHaveCount(0);
    await expect(nav.locator("a", { hasText: "Reporting" })).toHaveCount(0);
  });

  test("/ui/analysis and /ui/reporting redirect to /ui/trends", async ({ page }) => {
    await page.goto("/ui/analysis");
    await expect(page).toHaveURL(/\/ui\/trends$/);
    await page.goto("/ui/reporting");
    await expect(page).toHaveURL(/\/ui\/trends$/);
  });
});
