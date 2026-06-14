// OIDC login-gate smoke layer.
//
// Covers the scenario from:
// - openspec/changes/propose-nextcloud-as-oidc/specs/web-ui/spec.md
//
// Runs against the `oidc` fixture: a wanderer serve instance with
// an oidc: block configured against an unreachable provider.
// Discovery is lazy, so the server boots and the gate can issue
// the unauthenticated redirect without ever contacting Nextcloud.
//
// The request API is used (rather than page navigation) so the
// 302 → /ui/login redirect is asserted directly, independent of
// what the (deliberately unreachable) login page then renders.

import { test, expect } from "@playwright/test";

test.describe("OIDC login gate", () => {
  test("unauthenticated /ui/ redirects to /ui/login", async ({ request }) => {
    const res = await request.get("/ui/", { maxRedirects: 0 });
    expect(res.status()).toBe(302);
    expect(res.headers()["location"]).toBe("/ui/login");
  });

  test("the login route bypasses the gate (no redirect loop)", async ({
    request,
  }) => {
    // With the provider unreachable, /ui/login returns 503 — but it
    // must NOT redirect back to itself, which would prove the gate
    // is wrongly protecting its own login route.
    const res = await request.get("/ui/login", { maxRedirects: 0 });
    expect(res.status()).not.toBe(302);
  });
});
