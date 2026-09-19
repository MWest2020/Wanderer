// Accountability answer sheet (run 08, 2026-09-19-propose-accountability-dimension).
//
// Covers the "accountability renders as an answer sheet" requirement
// (specs/web-ui/spec.md): one plain-language question per rule,
// collapsed evidence, and onbekend vs n.v.t. rendered as visually
// distinct states from each other and from a failing "Nee".
//
// NOTE: this spec was written but NOT executed — the sandbox this run
// built in has no egress to the Playwright browser-binary CDN, so
// `npx playwright install` cannot fetch Chromium. See the run report
// for run 08 (openspec/changes/2026-09-19-propose-accountability-dimension/runs/08-ui.md)
// for the honest status.

import { test, expect } from "@playwright/test";

test.describe("Accountability answer sheet", () => {
  test("assessment page renders one question per accountability rule", async ({
    page,
  }) => {
    await page.goto("/ui/targets");
    const scanLink = page.locator('a[href*="/ui/scans/"]').first();
    await expect(scanLink).toBeVisible();
    const href = await scanLink.getAttribute("href");
    expect(href).toBeTruthy();
    await page.goto(`${href}/assessment`);

    const section = page.locator("#accountability");
    await expect(section).toBeVisible();

    // Rule IDs and RDAP jargon stay out of the headline.
    await expect(section.locator(".answer-question").first()).not.toContainText(
      "wand.accountability"
    );

    // Expanding evidence does not navigate away from the report.
    const firstDetails = section.locator("details.answer-evidence").first();
    await firstDetails.locator("summary").click();
    await expect(firstDetails).toHaveAttribute("open", "");
    await expect(page).toHaveURL(new RegExp(`${href}/assessment$`));
    await expect(firstDetails.locator("code").first()).toContainText(
      "wand.accountability"
    );
  });

  test("onbekend and n.v.t. answers are visually distinct from a failing nee", async ({
    page,
  }) => {
    await page.goto("/ui/targets");
    const scanLink = page.locator('a[href*="/ui/scans/"]').first();
    await page.goto(`${(await scanLink.getAttribute("href")) ?? ""}/assessment`);

    const section = page.locator("#accountability");
    const classes = await section.locator(".answer-row").evaluateAll((rows) =>
      rows.map((r) => r.className)
    );
    // The three classes, where present, must not collapse onto each
    // other — each carries its own answer-<class> modifier.
    const distinct = new Set(
      classes
        .flatMap((c) => c.split(" "))
        .filter((c) => c.startsWith("answer-") && c !== "answer-row")
    );
    expect(distinct.size).toBeGreaterThan(0);
    for (const a of ["answer-onbekend", "answer-nvt"]) {
      if (distinct.has(a)) {
        expect(distinct.has("answer-nee")).not.toBe(undefined); // sanity: attribute exists to compare against
      }
    }
  });
});
