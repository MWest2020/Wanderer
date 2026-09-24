// Vloot en regels (openspec/changes/2026-09-20-vloot-en-regels, run 06).
//
// Proves three requirements from specs/web-ui/spec.md end to end:
//   - "Domeinen zijn bij te houden als vloot" — a signed-in user adds a
//     domain to an organisation's fleet without scanning it.
//   - "Het vlootscherm scoort x van n, niet ja of nee" plus its sort
//     controls — sorting by score and by last scan reorders the rows.
//   - "Een regel legt zichzelf uit, inclusief de drempel" and "Bij elk
//     oordeel 'nee' staat wat je eraan doet" — the regelpagina shows a
//     rule's threshold and, for a failing target, its remediation.
//
// Runs against the `scan-dev` fixture (playwright.config.ts): a
// `wanderer serve` instance seeded with the `baseline` scenario and
// -ui-htpasswd, so this project's httpCredentials answer the Basic
// challenge and every request acts as a signed-in operator — required
// for the fleet-edit route (internal/ui/ui.go's allowFleetEdit).
//
// The sort assertions and the rule page's threshold/remediation both
// depend on internal/fixtures/baseline.go: voorbeeld's fleet carries
// a second domain (tweede.nl, afhankelijk on the certificate
// dimension, scanned two days before voorbeeld.nl) so "sort by
// score" and "sort by last scan" produce different first rows, and
// acme.example.com carries a whois.expiry finding inside
// wand.operationeel.domain_expiry's 30-day urgent window so that
// rule's page has a target on the "afhankelijk" side of its
// threshold.

import { test, expect } from "@playwright/test";
import { checkAccessibility } from "../support/axe";

test.describe("Domein toevoegen aan de vloot", () => {
  test("een nieuw domein verschijnt zonder te scannen", async ({ page }) => {
    await page.goto("/ui/orgs/voorbeeld/fleet");

    await page.locator("#fleet-domain").fill("vloot-playwright.nl");
    await page.locator('form.fleet-add-form button[type="submit"]').click();

    await expect(page).toHaveURL(/\/ui\/orgs\/voorbeeld\/fleet$/);
    const row = page.locator("table tbody tr", { hasText: "vloot-playwright.nl" });
    await expect(row).toBeVisible();
    await expect(row).toContainText("nog niet gescand");
  });
});

test.describe("Het vlootscherm sorteren", () => {
  test("op score zet het slechtst scorende domein bovenaan", async ({ page }) => {
    await page.goto("/ui/orgs/voorbeeld/fleet");

    await page.locator("p.fleet-sort a", { hasText: "Score" }).click();
    await expect(page).toHaveURL(/[?&]sort=score/);
    // The clicked link becomes the active, unlinked label.
    await expect(page.locator("p.fleet-sort strong", { hasText: "Score" })).toBeVisible();

    const firstRow = page.locator("table tbody tr").first();
    await expect(firstRow.locator("td").first()).toHaveText("tweede.nl");
    await expect(firstRow).toContainText("4/5");
  });

  test("op laatste scan zet het meest recent gescande domein bovenaan", async ({ page }) => {
    await page.goto("/ui/orgs/voorbeeld/fleet");

    await page.locator("p.fleet-sort a", { hasText: "Laatste scan" }).click();
    await expect(page).toHaveURL(/[?&]sort=last_scan/);
    await expect(page.locator("p.fleet-sort strong", { hasText: "Laatste scan" })).toBeVisible();

    const firstRow = page.locator("table tbody tr").first();
    await expect(firstRow.locator("td").first()).toHaveText("voorbeeld.nl");
  });
});

// vloot-beheren-bereikbaar: /ui/ links straight to the fleet manager
// when it shows exactly one organisation's fleet — none of the
// scan-dev project's baseline-derived fixtures qualify (they all
// carry voorbeeld + acme alongside the migration's default org), so
// this describe block talks to the single-org fixture directly
// (playwright.config.ts's singleOrgPort, port 8286) instead of the
// project's default baseURL.
test.describe("Vlootbeheer bereikbaar vanaf /ui/ (één organisatie)", () => {
  test.use({ baseURL: "http://127.0.0.1:8286" });

  test("vloot beheren linkt rechtstreeks naar de vloot, en een domein is te verwijderen", async ({
    page,
  }) => {
    await page.goto("/ui/");

    await page.locator("a", { hasText: "vloot beheren" }).click();
    await expect(page).toHaveURL(/\/ui\/orgs\/default\/fleet$/);

    const row = page.locator("table tbody tr", { hasText: "solo.nl" });
    await expect(row).toBeVisible();
    await row.getByRole("button", { name: "Verwijderen" }).click();
    await expect(page.locator("table tbody tr", { hasText: "solo.nl" })).toHaveCount(0);
  });
});

test.describe("De regelpagina toont een grens en een handeling", () => {
  test("wand.operationeel.domain_expiry toont de drempel en de handeling bij afhankelijk", async ({
    page,
  }) => {
    await page.goto("/ui/reporting/wand/wand.operationeel.domain_expiry");

    await expect(page.locator("h1 code")).toHaveText("wand.operationeel.domain_expiry");

    // The threshold ("grens"): the rule's judgement flips at 90 and
    // at 30 days, shown as an explicit value, not just buried in the
    // comparison (spec.md "Een drempel SHALL niet alleen in de
    // vergelijking in de code bestaan").
    const thresholds = page.locator(".rule-thresholds");
    await expect(thresholds).toBeVisible();
    await expect(thresholds).toContainText("90");
    await expect(thresholds).toContainText("30");

    // The handeling ("wat je eraan doet") on acme.example.com's
    // failing row, naming the domain, not a repeated goal.
    const row = page.locator("table tbody tr", { hasText: "acme.example.com" });
    await expect(row).toBeVisible();
    await expect(row.locator(".badge")).toHaveClass(/score-afhankelijk/);
    await expect(row.locator(".answer-remediation")).toContainText(
      "Verleng de domeinregistratie van acme.example.com",
    );
    await checkAccessibility(page, "regelpagina");
  });
});
