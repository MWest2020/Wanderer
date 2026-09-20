# Run report — 04 drempels uit de code (tasks 4.1–4.2)

## 4.1 — `assessor.Rule` draagt zijn grenzen
Added `Thresholds []Threshold` to `assessor.Rule` and a new
`assessor.Threshold` type (`internal/assessor/rule.go`): `Name`,
`Value float64`, `Unit string`, `Explanation string`. Zero value is
`nil` — a rule that doesn't set it (the default for every existing
rule) already satisfies "regels zonder grens houden een lege lijst"
with no per-rule edit needed. No other change to `Rule` or `Match`'s
signature, so this is additive and behaviour-preserving on its own.

## 4.2 — filling in the real thresholds
Before populating anything, I grepped every `wand` and `eucsf` rule
for a numeric comparison inside its own `Match` (`[<>]=? *[0-9]` etc.)
to find which rules actually decide on a fixed number, as opposed to
a presence/absence or count-equality check. Result: across all ~30
rules in both packages, exactly two carry a genuine hard-coded
decision boundary inside their own `Match`:

- `wand.operationeel.domain_expiry` — 90 and 30 days. Extracted to
  `domainExpirySafeDays` / `domainExpiryUrgentDays` constants
  (`accountability_rules.go`), used both in the `switch` comparison
  and in the rule's `Thresholds`, and folded into `Rationale` via
  `fmt.Sprintf` so the prose can't drift from the number either.
- `wand.operationeel.dns_redundancy` — "at least two nameservers".
  Extracted to `dnsRedundancyMinNameservers = 2` (`rules.go`), used in
  the `len(hosts) < …` comparison and in `Thresholds`.

Everything else (`> 0`, `== 0`, `len(a) == len(b)`) is a
presence/all-or-nothing check, not a corpus-shaped assumption someone
could reasonably disagree with — those rules correctly keep
`Thresholds == nil`, per 4.1's "aanwezig-of-niet" carve-out.

### The four rules named in the task-ref — what I found
- `wand.operationeel.domain_expiry` (90/30 dagen) — done, as above.
- `wand.accountability.securitytxt` (verlopen ja/nee) — confirmed
  presence-shaped: the expiry check is `expiresAt.After(time.Now())`,
  a comparison against *now*, not against a fixed day count. No
  number to extract; `Thresholds` correctly stays empty. This is the
  rule the task-ref uses as the "no grens" control case, and it holds.
- `wand.operationeel.variant_convergence` (task-ref cites "8 paden, 24
  verbindingen") — **not populated, and I want to flag why rather than
  fake it.** Those two numbers are real, but they live in
  `internal/probe/variants` (`variants.go`: the probe always attempts
  8 apex/www×v4/v6×http/https paths, `connectionBudget = 24`), not in
  this rule's `Match`. The rule itself never compares against 8 or 24
  — its `total` is `len(paths)`, read off whatever the finding
  contains, and its verdict hinges on origin-count equality and
  presence checks (plain-HTTP found, any path reachable, one origin
  vs. several). Had I added a `Threshold{Value: 8}` here, nothing in
  `Match` would ever compare against it, so the whole point of this
  task — "a test catches a rule's Threshold drifting from what it
  actually compares against" — would be unfalsifiable for this entry:
  the shared-constant test I wrote for the other two rules works
  precisely because it derives its boundary from `Match`'s own
  literal. There is no such literal here to derive from.
- `wand.operationeel.cert_validity` — same shape as
  variant_convergence. Its Description/Rationale say "30 days" and
  the probe (`internal/probe/tls/tls.go`) genuinely computes
  `expiring_soon` at `30*24*time.Hour`, but `cert_validity`'s own
  `Match` never sees that number — it only reads the probe's
  precomputed `expired`/`expiring_soon` booleans (and `days_left` for
  display only). Populating `Thresholds` here would either (a) require
  importing `internal/probe/tls` from the assessor package, which
  breaks the "pure consumer of `models.Finding`" boundary `rule.go`
  states explicitly and that `accountability_rules.go` already goes
  out of its way to preserve for the variants probe (duplicated status
  string constants, not an import), or (b) add a `Threshold.Value`
  with nothing in this package's code checking it against 30, same
  unfalsifiable problem as above. Left `Thresholds` empty.

Both of these are the "regel die een grens gebruikt die nergens is
uitgelegd" case in spirit, inverted: the grens *is* explained (in the
probe's comments and the rule's Rationale prose), it just isn't owned
by the rule the task-ref points at. Fixing it properly — exporting the
probe's constant and either importing it or writing a cross-package
consistency test — touches `internal/probe/variants` and
`internal/probe/tls`, outside "de bestaande regels van de wand- en
eucsf-pakketten" and outside this run's "geen scoringsgedrag
wijzigen" guardrail. Noting it here for whoever picks up the probe
side of this.

## Tests
- `accountability_rules_test.go`:
  `TestDomainExpiryThresholdsMatchComparison` — reads
  `domain_expiry_safe_days` / `domain_expiry_urgent_days` off
  `r.Thresholds` (not a hand-copied 90/30) and asserts the score flips
  at exactly those values. If someone edits the `Match` constant
  without touching `Thresholds` (or vice versa), this test's derived
  boundary and `Match`'s real one disagree and it fails.
- `rules_test.go` (wand): `TestDNSRedundancyThresholdMatchesComparison`
  — same pattern for `dns_redundancy_min_nameservers`.
- `rules_test.go` (wand) and `eucsf/rules_test.go`:
  `TestEveryThresholdIsWellFormed` — for every rule in `DefaultRules()`,
  any `Threshold` present has non-empty `Name`/`Unit`/`Explanation`
  (mirrors the existing `TestEveryRuleHasRationale` pattern). Passes
  today since only two rules populate `Thresholds` yet, but pins the
  contract for whoever adds the next one.

No scoring behaviour changed: the two rewritten `Match` functions use
named constants holding the exact same values (90/30/2) they held as
literals before.

## Verification
- `go build ./...` — clean.
- `go vet ./...` — clean.
- `go test ./...` — all packages `ok`.
- `openspec validate 2026-09-20-vloot-en-regels --strict` — valid.

## Tasks.md
Ticked 4.1 and 4.2.
