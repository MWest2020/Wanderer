// Overview slimness + org-scope persistence smoke test.
//
// After the Tourist/Explorer/Farmer restructure (ADR 0017) the nav is
// two tabs — Overview / Trends. The fleet table, Trends consolidation,
// and legacy redirects are covered by ui-personas.spec.ts; this file
// keeps the slimness + org-scope-threading scenarios.

import { test, expect } from "@playwright/test";

test.describe("Overview slimness", () => {
  test("Overview at /ui/ stays slim — fleet + verdict pills, no steering blocks", async ({ page }) => {
    await page.goto("/ui/");
    await expect(page.locator("h1")).toContainText("all organisations");
    await expect(page.locator("section.targets-fleet")).toBeVisible();
    await expect(page.getByRole("heading", { name: /Verdict/i })).toBeVisible();
    // Removed/relocated sections — must not appear on the Overview:
    await expect(page.locator("text=External posture")).toHaveCount(0);
    await expect(page.locator("text=Internal posture")).toHaveCount(0);
    await expect(page.locator("text=Top concerns")).toHaveCount(0);
    await expect(page.locator("text=Recent activity")).toHaveCount(0);
    // The score matrix lives on Trends, not the Overview.
    await expect(page.locator("table.reporting-rules")).toHaveCount(0);
  });
});

test.describe("Organisation scope persistence", () => {
  test("Per-org Overview threads slug into the two nav tabs", async ({ page }) => {
    await page.goto("/ui/orgs/conduction");
    await expect(page.locator("h1")).toContainText("Conduction B.V.");

    const nav = page.locator(".nav-bar");
    await expect(nav.locator("a", { hasText: "Overview" })).toHaveAttribute("href", "/ui/orgs/conduction");
    await expect(nav.locator("a", { hasText: "Trends" })).toHaveAttribute("href", "/ui/trends?org=conduction");
    // The retired tabs are gone.
    await expect(nav.locator("a", { hasText: "Analysis" })).toHaveCount(0);
    await expect(nav.locator("a", { hasText: "Reporting" })).toHaveCount(0);
  });

  test("Trends with ?org= renders the scope pill", async ({ page }) => {
    await page.goto("/ui/trends?org=conduction");
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
    await expect(page.locator("text=example.nl")).toHaveCount(0);
  });
});
