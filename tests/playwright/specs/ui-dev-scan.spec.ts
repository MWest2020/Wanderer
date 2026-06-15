// Dev-mode scan form (ADR-0016 / propose-ui-dev-scan).
//
// Runs against the `scan-dev` fixture: a serve instance started with
// --ui-allow-scan. Asserts the "Scan a target" form renders. (The
// actual POST triggers a real network scan, so the spec asserts the
// form surface, not a live scan.)
import { test, expect } from "@playwright/test";

test.describe("Dev-mode scan form", () => {
  test("dashboard shows the scan-a-target form when --ui-allow-scan", async ({ page }) => {
    await page.goto("/ui/");
    await expect(page.locator("form.scan-form")).toBeVisible();
    await expect(page.locator('form.scan-form input[name="domain"]')).toBeVisible();
  });
});
