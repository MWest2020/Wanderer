// Accountability answer sheet (run 08/08b, 2026-09-19-propose-accountability-dimension).
//
// Covers the "accountability renders as an answer sheet" requirement
// (specs/web-ui/spec.md): one plain-language question per rule,
// collapsed evidence, and onbekend vs n.v.t. rendered as visually
// distinct states from each other and from a failing "Nee".
//
// Targets voorbeeld.nl's scan specifically (not "the first scan
// link"): internal/fixtures/baseline.go seeds voorbeeld.nl and
// acme.example.com with shared accountability findings, but only
// voorbeeld.nl's ".nl" TLD trips the registry-redaction rule, so its
// report is the one scan guaranteed to carry all four answer states
// (ja via securitytxt, nee via no_reseller, onbekend via the
// evidence-less soa_rname/ns_holder_transparent rules, n.v.t. via
// registrant_identifiable) on one page.
//
// NOTE: this spec was written but NOT executed — the sandbox this run
// (08b) builds in has no network egress at all (even `npm install`
// for the Playwright package itself is denied), so there is no
// Chromium binary to run against. See the run report for run 08b
// (openspec/changes/2026-09-19-propose-accountability-dimension/runs/08b-playwright-wiring.md)
// for the honest status. Task 8.5 (wiring the spec into
// playwright.config.ts's baseline project + the fixture) is done;
// task 8.4 (actually running it) is not.

import { test, expect } from "@playwright/test";
import type { Locator, Page } from "@playwright/test";

async function voorbeeldAssessmentURL(page: Page): Promise<string> {
  await page.goto("/ui/targets");
  const row = page.locator("tr", { hasText: "voorbeeld.nl" });
  await expect(row).toBeVisible();
  const scanLink = row.locator('a[href*="/ui/scans/"]');
  const href = await scanLink.getAttribute("href");
  expect(href).toBeTruthy();
  return `${href}/assessment`;
}

test.describe("Accountability answer sheet", () => {
  test("assessment page renders one question per accountability rule", async ({
    page,
  }) => {
    const url = await voorbeeldAssessmentURL(page);
    await page.goto(url);

    // Sinds 2026-09-22-drie-lagen-ciso run 05 (taak 5.1) staan de
    // frameworktabellen achter één klik: de onderbouwing opent met de
    // zeven vragen, de regels per framework zitten in een dichte
    // <details>. De accountability-sectie zit daarin, dus openen vóór
    // je hem beoordeelt — niet de UI terugdraaien om de spec te
    // plezieren.
    await page.locator("details.framework-details").click();

    const section = page.locator("#wand-accountability");
    await expect(section).toBeVisible();

    // Rule IDs and RDAP jargon stay out of the headline.
    await expect(section.locator(".answer-question").first()).not.toContainText(
      "wand.accountability"
    );

    // Expanding evidence does not navigate away from the report.
    const firstDetails = section.locator("details.answer-evidence").first();
    await firstDetails.locator("summary").click();
    await expect(firstDetails).toHaveAttribute("open", "");
    await expect(page).toHaveURL(new RegExp(`${url}$`));
    await expect(firstDetails.locator("code").first()).toContainText(
      "wand.accountability"
    );
  });

  test("onbekend and n.v.t. answers are visually distinct from a failing nee", async ({
    page,
  }) => {
    const url = await voorbeeldAssessmentURL(page);
    await page.goto(url);

    // Zelfde reden als hierboven: de frameworksectie staat dicht.
    await page.locator("details.framework-details").click();

    const section = page.locator("#wand-accountability");

    // The fixture's registrant_identifiable rule scores n.v.t.
    // (registry_redacted, voorbeeld.nl's ".nl" TLD); soa_rname and
    // ns_holder_transparent score onbekend (no dns.soa / whois.ns_holder
    // evidence); no_reseller scores nee (a reseller finding is present);
    // securitytxt scores ja. All four rows must be present.
    const nvtRow = section.locator(".answer-row.answer-nvt").first();
    const onbekendRow = section.locator(".answer-row.answer-onbekend").first();
    const neeRow = section.locator(".answer-row.answer-nee").first();
    const jaRow = section.locator(".answer-row.answer-ja").first();
    await expect(nvtRow).toBeVisible();
    await expect(onbekendRow).toBeVisible();
    await expect(neeRow).toBeVisible();
    await expect(jaRow).toBeVisible();

    // "Visually distinct" is asserted on the rendered style, not just
    // the class name: each answer state carries its own border-left
    // colour (main.css), so a computed-style diff is real proof, not
    // a class-name coincidence.
    const borderColor = async (row: Locator) =>
      row.evaluate((el) => getComputedStyle(el).borderLeftColor);
    const [nvtColor, onbekendColor, neeColor] = await Promise.all([
      borderColor(nvtRow),
      borderColor(onbekendRow),
      borderColor(neeRow),
    ]);
    expect(nvtColor).not.toBe(onbekendColor);
    expect(nvtColor).not.toBe(neeColor);
    expect(onbekendColor).not.toBe(neeColor);

    // The onbekend and n.v.t. badges also carry distinct labels — the
    // report never dresses up a measurement gap as "not applicable"
    // or vice versa.
    await expect(nvtRow.locator(".answer-badge-nvt")).toContainText("n.v.t.");
    await expect(onbekendRow.locator(".answer-badge-onbekend")).toContainText(
      "Onbekend"
    );
  });
});
