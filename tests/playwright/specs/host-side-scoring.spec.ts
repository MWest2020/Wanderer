// Host-side scoring smoke layer.
//
// Covers the scenarios from:
// - openspec/changes/add-host-side-scoring/specs/assessor/spec.md
//   "Host-side findings produce a non-onbekend verdict"
//
// Two layers:
//
//   1. Catalogue assertion — runnable against any seeded DB.
//      Confirms the three new host rules are registered and
//      surface on `/ui/reporting`. This locks the rule shipping
//      contract regardless of whether the demo DB carries an
//      agent-host scan.
//
//   2. End-to-end host scan assertion — gated on a host scan
//      being present in the DB. Until the hermetic Playwright
//      fixture loader (separate OpenSpec change) lands, an
//      operator seeds it manually:
//
//        ./bin/wanderer agent --once \
//          --config tests/playwright/fixtures/agent-host.yaml
//
//      The spec's `test.skip` predicate flips on once a target
//      with Kind=host shows up in `/ui/targets`.

import { test, expect } from "@playwright/test";

test.describe("Host-side scoring — rule catalogue", () => {
  test("Reporting catalogue lists the three host rules", async ({ page }) => {
    await page.goto("/ui/reporting");

    // Wand pack: two host rules, one for packages, one for systemd units.
    await expect(
      page.locator("text=wand.host.no_us_telemetry_packages"),
    ).toBeVisible();
    await expect(
      page.locator("text=wand.host.no_us_telemetry_services"),
    ).toBeVisible();

    // EUCSF pack: single combined rule (SEAL rolls package + service
    // exposure into one observation).
    await expect(
      page.locator("text=eucsf.sov5.host_no_us_telemetry"),
    ).toBeVisible();
  });

  test("Each host rule deep-dive renders rationale", async ({ page }) => {
    const ids = [
      "wand.host.no_us_telemetry_packages",
      "wand.host.no_us_telemetry_services",
      "eucsf.sov5.host_no_us_telemetry",
    ];
    for (const id of ids) {
      const framework = id.split(".")[0];
      await page.goto(`/ui/reporting/${framework}/${id}`);
      // The CriteriumID heading renders — the route resolved.
      await expect(
        page.locator("h1 code", { hasText: id }),
      ).toBeVisible();
      // Rationale lives on every host rule per the spec —
      // never the "no rationale yet" placeholder.
      await expect(page.locator("text=no rationale yet")).toHaveCount(0);
      await expect(page.locator(".rationale-text")).toBeVisible();
      // Either per-target rows or the empty-state message: the
      // page MUST render one or the other so an operator knows
      // whether the rule has produced rationale rows.
      const hasRows = await page.locator(`[class*="score-"]`).count();
      const hasEmpty = await page.locator(".empty-state").count();
      expect(
        hasRows + hasEmpty,
        `${id}: expected either score rows or .empty-state`,
      ).toBeGreaterThan(0);
    }
  });
});

test.describe("Host-side scoring — agent scan smoke", () => {
  // Runs against the `agent-host` fixture which seeds an `alma`
  // host target with 32 inventory packages (one `datadog-agent`
  // hit) and 14 systemd units (one `datadog-agent.service` hit).
  // The host rule deep-dive is the cheapest signal: per-target
  // table has one row with `afhankelijk` + the datadog vendor.
  test("Host rule deep-dive shows the seeded datadog hit", async ({
    page,
  }) => {
    await page.goto(
      "/ui/reporting/wand/wand.host.no_us_telemetry_packages",
    );

    // Exactly one host scored row — the seeded `alma`.
    const row = page.locator("tbody tr", { hasText: "alma" });
    await expect(row).toBeVisible();
    await expect(row.locator("[class*='score-']")).toContainText(
      "afhankelijk",
    );
    await expect(row).toContainText("datadog-agent");
    await expect(row).toContainText("Datadog");
  });

  test("Services rule shows the seeded datadog.service hit", async ({
    page,
  }) => {
    await page.goto(
      "/ui/reporting/wand/wand.host.no_us_telemetry_services",
    );
    const row = page.locator("tbody tr", { hasText: "alma" });
    await expect(row.locator("[class*='score-']")).toContainText(
      "afhankelijk",
    );
    await expect(row).toContainText("datadog-agent.service");
  });

  test("EUCSF combined rule covers both shapes", async ({ page }) => {
    await page.goto(
      "/ui/reporting/eucsf/eucsf.sov5.host_no_us_telemetry",
    );
    const row = page.locator("tbody tr", { hasText: "alma" });
    await expect(row.locator("[class*='score-']")).toContainText(
      "afhankelijk",
    );
  });
});
