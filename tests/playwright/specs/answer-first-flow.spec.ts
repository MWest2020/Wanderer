// Answer-first UI (openspec/changes/2026-09-20-answer-first-ui), updated
// for 2026-09-22-drie-lagen-ciso and 2026-09-24-vloot-in-beeld: /ui/ is
// the fleet overview (the vloot is the first layer) with the scan form
// as a compact action at the top of the page, directly under the
// header — where a person looks for it — not the page's headline.
//
// Covers spec.md's four ADDED requirements end to end:
//   - "The entry surface asks for a domain and answers it" — now folded
//     into the fleet overview: the score and domain list lead, the scan
//     form is a lower-page action, no scan IDs / rule IDs shown loose.
//   - "The answer is one sentence with its reason" — including
//     "Unknown is not a yes": an onbekend flow must not round up to
//     a plain "Ja".
//   - "The answer fills in while the scan runs" — progressive
//     rendering after a real submit.
//   - "The reasoning is one click from the answer" — the link to
//     the reasoning page and back.
//
// Runs against the `scan-dev` fixture (playwright.config.ts): a
// `wanderer serve` instance seeded with the `baseline` scenario
// (voorbeeld.nl, acme.example.com — see internal/fixtures/baseline.go)
// and -ui-htpasswd so the door's scan form is enabled for a signed-in
// user (this project's httpCredentials answer the Basic challenge).
//
// The domain-submit test uses a real, stable, long-lived domain
// (example.com) and only waits for the scan to *start* (the answer
// page to render, however incomplete) — not for the traceroute probe
// to finish, which can take up to the scan's global budget. Mirrors
// ui-dev-scan.spec.ts's existing rule of not waiting out a live scan
// in this suite. The content-rich assertions (a completed reasoning
// page, the onbekend-is-not-ja headline) instead read the fixture's
// already-completed voorbeeld.nl scan — deterministic, no network.

import { test, expect } from "@playwright/test";
import type { Page } from "@playwright/test";
import { checkAccessibility } from "../support/axe";

async function voorbeeldAnswerURL(page: Page): Promise<string> {
  await page.goto("/ui/targets");
  const row = page.locator("tr", { hasText: "voorbeeld.nl" });
  await expect(row).toBeVisible();
  const scanLink = row.locator('a[href*="/ui/scans/"]');
  const href = await scanLink.getAttribute("href");
  expect(href).toBeTruthy();
  // index.tmpl links to the scan's raw findings page; the answer
  // page is a sibling route on the same scan ID.
  return (href as string).replace(/\/ui\/scans\/([^/]+).*/, "/ui/scans/$1/answer");
}

test.describe("Het vlootoverzicht", () => {
  test("het scanveld staat bovenaan, de vlootscore en domeinenlijst tonen geen losse rule-ID's", async ({
    page,
  }) => {
    await page.goto("/ui/");

    // spec.md "Het invoerveld staat bovenaan": het scanveld staat vóór
    // de vlootscore in de pagina (2026-09-24-vloot-in-beeld task 1.1),
    // niet — zoals vóór deze change — eronder.
    const scanForm = page.locator("form.scan-form");
    const fleetSummary = page.locator("section.fleet-summary");
    await expect(scanForm).toBeVisible();
    await expect(fleetSummary).toBeVisible();
    const scanBox = await scanForm.boundingBox();
    const fleetBox = await fleetSummary.boundingBox();
    expect(scanBox).not.toBeNull();
    expect(fleetBox).not.toBeNull();
    expect(scanBox!.y).toBeLessThan(fleetBox!.y);
    await expect(page.locator('form.scan-form input[name="domain"]')).toBeVisible();

    // De vlootscore rendert als ring (proposal.md "de vlootscore wordt
    // een ring"), niet meer als tekstbadge — met domeinenlijst en zonder
    // losse rule-ID's (de fixture's baseline scenario seedt voorbeeld.nl
    // en acme.example.com met afgeronde assessments).
    await expect(page.locator("svg.fleet-ring")).toBeVisible();
    const row = page.locator("table.targets tr", { hasText: "voorbeeld.nl" });
    await expect(row).toBeVisible();

    // Geen losse rule-ID's op de pagina zelf (enkel als href in de
    // top-3 "kost de vloot de meeste punten").
    await expect(page.locator("body")).not.toContainText("wand.juridisch");
    await expect(page.locator("body")).not.toContainText("wand.transit");
  });
});

// Run 02 (nakijken op prod-data, 2026-09-24-vloot-in-beeld): screenshots
// on a copy of prod data showed the ring's onbeantwoord count living
// only in its aria-label, and — on a 390px viewport — the per-stroom
// bars collapsing to zero width while the domain grid pushed the whole
// page into horizontal scroll.
test.describe("Vloot-in-beeld: legenda en mobiel (run 02)", () => {
  test("de ring-legenda toont 'onbeantwoord' als tekst, niet alleen in het aria-label", async ({
    page,
  }) => {
    await page.goto("/ui/");
    await expect(page.locator("svg.fleet-ring")).toBeVisible();
    await expect(page.locator(".ring-legend")).toContainText("onbeantwoord");
  });

  test("op 390px blijven de stroombalken zichtbaar en scrollt de pagina niet horizontaal", async ({
    page,
  }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto("/ui/");

    const scrollWidth = await page.evaluate(() => document.documentElement.scrollWidth);
    const innerWidth = await page.evaluate(() => window.innerWidth);
    expect(scrollWidth).toBeLessThanOrEqual(innerWidth);

    const bars = page.locator(".flow-bar");
    const count = await bars.count();
    expect(count).toBeGreaterThan(0);
    for (let i = 0; i < count; i++) {
      const box = await bars.nth(i).boundingBox();
      expect(box).not.toBeNull();
      expect(box!.width).toBeGreaterThan(0);
    }
  });
});

test.describe("Domain in → answer vult zich", () => {
  test("submitting a domain lands on that target's progressive answer page", async ({ page }) => {
    await page.goto("/ui/");

    await page.locator("#scan-domain").fill("example.com");
    await page.locator('form.scan-form button[type="submit"]').click();

    // POST /ui/scan redirects to the scan-status bridge, which
    // meta-refreshes until the background scan's row exists, then
    // redirects to the answer page. Generous timeout: the bridge
    // itself only waits on the Scan row being created (before any
    // probe completes), not on the scan finishing.
    await page.waitForURL(/\/ui\/scans\/[^/]+\/answer$/, { timeout: 20_000 });

    await expect(page.locator("h1")).toHaveText("example.com");
    // Progressive rendering (spec.md "The answer fills in while the
    // scan runs"): either the just-started spinner or a partial
    // flows table is showing — never a scan ID or a rule ID.
    const progressing = page.locator(".scan-progress");
    const flows = page.locator("table.flows");
    await expect(progressing.or(flows)).toBeVisible();
    await expect(page.locator("body")).not.toContainText("wand.juridisch");

    // One link to the reasoning, and it resolves.
    const reasoningLink = page.locator('a:has-text("onderbouwing")');
    await expect(reasoningLink).toBeVisible();
    await reasoningLink.click();
    await expect(page).toHaveURL(/\/ui\/scans\/[^/]+\/assessment$/);
    await expect(page.locator("h1")).toHaveText("example.com");
  });
});

test.describe("Onbekend is geen ja", () => {
  test("a headline with an unanswered flow never reads a plain Ja", async ({ page }) => {
    const answerURL = await voorbeeldAnswerURL(page);
    await page.goto(answerURL);

    // voorbeeld.nl's baseline fixture has no transit.hop or ns-host
    // ip.asn findings, so the transit and DNS flows score onbekend
    // (dns.ns_vendor_jurisdiction has ns hosts but no GeoIP lookup
    // for them; wand.transit.eu_path has no traceroute at all) while
    // hosting, mail and the certificate are soeverein — exactly the
    // "hosting soeverein, DNS onbekend because GeoIP was unavailable"
    // shape spec.md's scenario asks for.
    const verdict = page.locator("p.answer-verdict").first();
    await expect(verdict).toBeVisible();
    // The verdict still reads "ja" (an onbekend flow never flips it
    // to "nee") but the sentence itself is qualified — never the bare
    // "Ja — dit domein staat onder Nederlands of Europees recht."
    // with nothing else, which is what a rounded-up lie would read.
    await expect(verdict.locator(".badge")).toHaveText("ja");
    await expect(verdict).toContainText(/kon(den)? niet worden beantwoord/);
    await checkAccessibility(page, "antwoord");
  });

  test("the reasoning page renders the onbekend flow distinctly, with the rule ID collapsed", async ({
    page,
  }) => {
    const answerURL = await voorbeeldAnswerURL(page);
    await page.goto(answerURL);
    await page.locator('a:has-text("onderbouwing")').click();
    await expect(page).toHaveURL(/\/ui\/scans\/[^/]+\/assessment$/);

    const overview = page.locator("section.sovereignty-overview");
    await expect(overview).toBeVisible();

    const onbekendRow = overview.locator(".answer-row.answer-onbekend").first();
    await expect(onbekendRow).toBeVisible();
    // The rule ID stays out of the question line...
    await expect(onbekendRow.locator(".answer-question")).not.toContainText("wand.");
    // ...and only appears once the evidence is expanded.
    const evidence = onbekendRow.locator("details.answer-evidence");
    await expect(evidence.locator("code").first()).not.toBeVisible();
    await evidence.locator("summary").click();
    await expect(evidence).toHaveAttribute("open", "");
    await expect(evidence.locator("code").first()).toContainText("wand.");

    // One link back to the answer.
    const back = page.locator('a:has-text("antwoord")');
    await expect(back).toBeVisible();
    await back.click();
    await expect(page).toHaveURL(/\/ui\/scans\/[^/]+\/answer$/);
  });
});
