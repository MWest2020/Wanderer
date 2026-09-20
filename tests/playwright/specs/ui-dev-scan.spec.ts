// Dev-mode scan form (ADR-0016 / propose-ui-dev-scan).
//
// Runs against the `scan-dev` fixture: a serve instance started with
// -ui-htpasswd (answer-first-ui retired --ui-allow-scan — a signed-in
// user gates the scan route now, see answer-first-flow.spec.ts).
// Asserts the door's scan form renders for that signed-in user. (The
// actual POST triggers a real network scan, so this spec asserts the
// form surface, not a live scan — answer-first-flow.spec.ts covers
// the submit → answer → reasoning flow.)
import { test, expect } from "@playwright/test";

test.describe("Dev-mode scan form", () => {
  test("dashboard shows the scan-a-target form for a signed-in user", async ({ page }) => {
    await page.goto("/ui/");
    await expect(page.locator("form.scan-form")).toBeVisible();
    await expect(page.locator('form.scan-form input[name="domain"]')).toBeVisible();
  });
});
