// Answer-first UI (openspec/changes/2026-09-20-answer-first-ui).
//
// Covers spec.md's four ADDED requirements end to end:
//   - "The entry surface asks for a domain and answers it" — the
//     door's input, no scan IDs / rule IDs on that surface.
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
// (conduction.nl, acme.example.com — see internal/fixtures/baseline.go)
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
// already-completed conduction.nl scan — deterministic, no network.

import { test, expect } from "@playwright/test";
import type { Page } from "@playwright/test";
import { checkAccessibility } from "../support/axe";

async function conductionAnswerURL(page: Page): Promise<string> {
  await page.goto("/ui/targets");
  const row = page.locator("tr", { hasText: "conduction.nl" });
  await expect(row).toBeVisible();
  const scanLink = row.locator('a[href*="/ui/scans/"]');
  const href = await scanLink.getAttribute("href");
  expect(href).toBeTruthy();
  // index.tmpl links to the scan's raw findings page; the answer
  // page is a sibling route on the same scan ID.
  return (href as string).replace(/\/ui\/scans\/([^/]+).*/, "/ui/scans/$1/answer");
}

test.describe("The door", () => {
  test("the entry surface has one input and no scan IDs or rule IDs", async ({ page }) => {
    await page.goto("/ui/");

    await expect(page.locator("form.door-form")).toBeVisible();
    await expect(page.locator('form.door-form input[name="domain"]')).toBeVisible();

    // No fleet table, no matrix, no scan-ID/rule-ID surface on the
    // door — that moved to /ui/trends (spec.md 2.1).
    await expect(page.locator("table")).toHaveCount(0);
    await expect(page.locator("body")).not.toContainText("wand.juridisch");
    await expect(page.locator("body")).not.toContainText("wand.transit");

    // The recently-answered list reads as one-line verdicts, not a
    // second scans table (the fixture's baseline scenario seeds
    // conduction.nl and acme.example.com with completed assessments).
    const recent = page.locator("li.answer-line", { hasText: "conduction.nl" });
    await expect(recent).toBeVisible();
    await expect(recent.locator(".badge")).toBeVisible();
  });
});

test.describe("Domain in → answer vult zich", () => {
  test("submitting a domain lands on that target's progressive answer page", async ({ page }) => {
    await page.goto("/ui/");

    await page.locator("#scan-domain").fill("example.com");
    await page.locator('form.door-form button[type="submit"]').click();

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
    const answerURL = await conductionAnswerURL(page);
    await page.goto(answerURL);

    // conduction.nl's baseline fixture has no transit.hop or ns-host
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
    const answerURL = await conductionAnswerURL(page);
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
