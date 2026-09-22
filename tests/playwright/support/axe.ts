// Accessibility gate (openspec/changes/2026-09-22-toegankelijkheid, run 01).
//
// checkAccessibility() runs axe-core against the page's current DOM and
// fails the test on any `serious` or `critical` finding. `moderate` and
// `minor` findings are logged but do not fail — axe-core's own docs call
// those two levels "should fix", not "must fix", and this gate is meant
// to catch the failures a Dutch-government buyer's own accessibility
// statement would flag, not to relitigate every WCAG advisory.
//
// A green run here means "no automatically-detectable finding of this
// type" — not "toegankelijk". axe-core finds roughly a third of what can
// be wrong; keyboard paths, screen-reader text and focus order still
// need a human. See the change's proposal.md ("Wat dit eerlijk houdt").

import AxeBuilder from "@axe-core/playwright";
import type { Page } from "@playwright/test";

const FAILING_IMPACTS = new Set(["serious", "critical"]);

// checkAccessibility scans `page` and throws if any violation has impact
// "serious" or "critical". `screenName` identifies the screen in the
// failure message (e.g. "trends", "antwoord — conduction.nl") so a
// reader can find the right template without re-running the suite.
export async function checkAccessibility(page: Page, screenName: string): Promise<void> {
  const results = await new AxeBuilder({ page }).analyze();
  const failing = results.violations.filter((violation) =>
    FAILING_IMPACTS.has(violation.impact ?? ""),
  );
  if (failing.length === 0) {
    return;
  }

  const lines = failing.flatMap((violation) =>
    violation.nodes.map((node) => {
      const target = node.target.join(" ");
      const detail = (node.failureSummary ?? "").replace(/\n/g, " ");
      return `  [${violation.impact}] ${violation.id} on "${screenName}" — element ${target}: ${detail}`;
    }),
  );

  throw new Error(
    `axe found ${failing.length} serious/critical accessibility violation(s) ` +
      `on "${screenName}":\n${lines.join("\n")}`,
  );
}
