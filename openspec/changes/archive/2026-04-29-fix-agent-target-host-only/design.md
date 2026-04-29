# Design: Agent accepts host-only Targets

## Today

`pkg/models/target.go`:

```go
func (t *Target) Validate() error {
    // ...
    if !strings.Contains(t.Domain, ".") {
        return fmt.Errorf("domain: %q has no TLD", t.Domain)
    }
    // ...
}
```

The check is correct for `wanderer scan example.nl` but wrong for
`wanderer agent` where `cfg.Hostname` is `webapp-01` on most Linux
boxes.

## Decision

Add a `Kind` field to `Target` with two values: `domain` (default,
unchanged) and `host` (host-only, no TLD requirement). The
validator dispatches on Kind:

```go
type TargetKind string
const (
    TargetKindDomain TargetKind = "domain"
    TargetKindHost   TargetKind = "host"
)

type Target struct {
    ID, Domain string
    Related    []string
    Kind       TargetKind  // empty defaults to domain
    CreatedAt  time.Time
}

func (t *Target) Validate() error {
    if t.Domain == "" { return errors.New("...") }
    switch t.Kind {
    case "", TargetKindDomain:
        // existing TLD check
    case TargetKindHost:
        // require Domain be a non-empty label, no TLD requirement
    default:
        return fmt.Errorf("unknown target kind %q", t.Kind)
    }
    // ...
}
```

The store gains a `kind` column with default `domain`; the
migration is idempotent (`ALTER TABLE ADD COLUMN ... DEFAULT
'domain'`).

The agent sets `Kind: TargetKindHost` when it builds its Target.

## Failure modes

- Existing scan databases continue to work — the migration's
  default value covers every existing row.
- An external consumer importing `pkg/models.Target` and not
  setting Kind continues to validate as `domain` (backwards
  compatible).

## Tests

- `Target.Validate` round-trips a host target with no TLD.
- Scan/Agent integration: agent writes a target with
  `Kind: host`; perimeter scan against a domain still rejects bare
  hostnames.
