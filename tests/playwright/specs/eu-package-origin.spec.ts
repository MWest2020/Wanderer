// EU package origin smoke layer.
//
// Runs against the `agent-host` fixture which seeds RPM
// packages with `vendor: "Fedora Project"` (Red Hat-sponsored,
// US-tied) plus one `vendor: "Datadog, Inc."` package. The new
// rule should score afhankelijk and name Red Hat as the parent
// vendor of record for the bulk of the inspected surface.

import { test, expect } from "@playwright/test";

test.describe("EU package origin — catalogue", () => {
  test("Reporting catalogue lists the rule", async ({ page }) => {
    await page.goto("/ui/reporting");
    await expect(
      page.locator("text=wand.host.eu_package_origin"),
    ).toBeVisible();
  });
});

test.describe("EU package origin — agent host classification", () => {
  test("Fedora-vendored host scores afhankelijk with Red Hat parent", async ({
    page,
  }) => {
    await page.goto(
      "/ui/reporting/wand/wand.host.eu_package_origin",
    );
    const row = page.locator("tbody tr", { hasText: "alma" });
    await expect(row.locator("[class*='score-']")).toContainText(
      "afhankelijk",
    );
    // Verdict text on the per-target row exposes the parent
    // organisation so the operator sees the actual upstream.
    await expect(row).toContainText("Red Hat");
  });
});
