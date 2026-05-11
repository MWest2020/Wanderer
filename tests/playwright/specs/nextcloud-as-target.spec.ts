// Nextcloud-as-target smoke layer.
//
// Covers the scenarios from:
// - openspec/changes/add-nextcloud-as-target/specs/inventory/spec.md
//
// Runs against the `agent-host` fixture which seeds, alongside
// the host inventory, a synthetic Nextcloud surface: one US S3
// objectstore (s3.amazonaws.com) and one US OIDC IdP
// (okta.example.com). All three new rules should fire
// `afhankelijk` on the seeded scan.

import { test, expect } from "@playwright/test";

test.describe("Nextcloud as target — rule catalogue", () => {
  test("Reporting catalogue lists the three Nextcloud rules", async ({
    page,
  }) => {
    await page.goto("/ui/reporting");

    await expect(
      page.locator("text=wand.nextcloud.objectstore_eu"),
    ).toBeVisible();
    await expect(
      page.locator("text=wand.nextcloud.oidc_provider_eu"),
    ).toBeVisible();
    await expect(
      page.locator("text=eucsf.sov6.nextcloud_supply_chain"),
    ).toBeVisible();
  });
});

test.describe("Nextcloud as target — agent host hits", () => {
  test("Objectstore rule flags the US S3 backend", async ({ page }) => {
    await page.goto(
      "/ui/reporting/wand/wand.nextcloud.objectstore_eu",
    );
    const row = page.locator("tbody tr", { hasText: "alma" });
    await expect(row.locator("[class*='score-']")).toContainText(
      "afhankelijk",
    );
    await expect(row).toContainText("nextcloud-data");
  });

  test("OIDC rule flags the US IdP", async ({ page }) => {
    await page.goto(
      "/ui/reporting/wand/wand.nextcloud.oidc_provider_eu",
    );
    const row = page.locator("tbody tr", { hasText: "alma" });
    await expect(row.locator("[class*='score-']")).toContainText(
      "afhankelijk",
    );
    await expect(row).toContainText("okta-prod");
  });

  test("SEAL combined rule rolls both hits up", async ({ page }) => {
    await page.goto(
      "/ui/reporting/eucsf/eucsf.sov6.nextcloud_supply_chain",
    );
    const row = page.locator("tbody tr", { hasText: "alma" });
    await expect(row.locator("[class*='score-']")).toContainText(
      "afhankelijk",
    );
  });
});
