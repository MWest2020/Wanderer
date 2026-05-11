// Container image sovereignty smoke layer.
//
// Runs against the `agent-host` fixture which seeds three Docker
// images (two harbor.example.de, one gcr.io/foo/bar) plus two
// running containers (one harbor.example.de, one gcr.io). Both
// wand rules + the SEAL combined rule should fire afhankelijk on
// the alma host.

import { test, expect } from "@playwright/test";

test.describe("Container image sovereignty — rule catalogue", () => {
  test("Reporting catalogue lists the three Docker rules", async ({
    page,
  }) => {
    await page.goto("/ui/reporting");
    await expect(
      page.locator("text=wand.docker.images_us_registry"),
    ).toBeVisible();
    await expect(
      page.locator("text=wand.docker.containers_us_registry"),
    ).toBeVisible();
    await expect(
      page.locator("text=eucsf.sov6.container_supply_chain"),
    ).toBeVisible();
  });
});

test.describe("Container image sovereignty — agent host hits", () => {
  test("Images rule flags the gcr.io image", async ({ page }) => {
    await page.goto(
      "/ui/reporting/wand/wand.docker.images_us_registry",
    );
    const row = page.locator("tbody tr", { hasText: "alma" });
    await expect(row.locator("[class*='score-']")).toContainText(
      "afhankelijk",
    );
    await expect(row).toContainText("gcr.io/foo/bar");
  });

  test("Containers rule flags the running gcr.io container", async ({
    page,
  }) => {
    await page.goto(
      "/ui/reporting/wand/wand.docker.containers_us_registry",
    );
    const row = page.locator("tbody tr", { hasText: "alma" });
    await expect(row.locator("[class*='score-']")).toContainText(
      "afhankelijk",
    );
  });

  test("SEAL combined rule covers both image + container surfaces", async ({
    page,
  }) => {
    await page.goto(
      "/ui/reporting/eucsf/eucsf.sov6.container_supply_chain",
    );
    const row = page.locator("tbody tr", { hasText: "alma" });
    await expect(row.locator("[class*='score-']")).toContainText(
      "afhankelijk",
    );
  });
});
