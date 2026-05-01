# Design: YAML config file for `wanderer serve`

## Config shape

```yaml
# /etc/wanderer/serve.yaml — every field optional; missing = default
listen: ":8080"
db: "/var/lib/wanderer/wanderer.db"

geoip:
  asn:      "/var/lib/wanderer/geoip/GeoLite2-ASN.mmdb"
  country:  "/var/lib/wanderer/geoip/GeoLite2-Country.mmdb"
  optional: false

ui:
  enabled:  true
  htpasswd: "/etc/wanderer/htpasswd"

schedules: "/etc/wanderer/schedules.yaml"

scan:
  per_probe_timeout:     30s
  budget:                2m
  user_agent:            "Wanderer/1.0"
  allow_private_targets: false
```

Field names match the YAML conventions already in
`internal/agent/config.go` (lowercase, snake_case, durations as
Go-parseable strings). The top-level structure mirrors the flag
surface one-to-one, so the diff between "what flags this binary
accepts" and "what fields the YAML accepts" stays small and
auditable.

## Package surface

```go
package serveconfig

type Config struct {
    Listen    string         `yaml:"listen,omitempty"`
    DB        string         `yaml:"db,omitempty"`
    GeoIP     GeoIPConfig    `yaml:"geoip,omitempty"`
    UI        UIConfig       `yaml:"ui,omitempty"`
    Schedules string         `yaml:"schedules,omitempty"`
    Scan      ScanConfig     `yaml:"scan,omitempty"`
}

type GeoIPConfig struct {
    ASN      string `yaml:"asn,omitempty"`
    Country  string `yaml:"country,omitempty"`
    Optional bool   `yaml:"optional,omitempty"`
}

type UIConfig struct {
    Enabled  bool   `yaml:"enabled,omitempty"`
    Htpasswd string `yaml:"htpasswd,omitempty"`
}

type ScanConfig struct {
    PerProbeTimeout     time.Duration `yaml:"per_probe_timeout,omitempty"`
    Budget              time.Duration `yaml:"budget,omitempty"`
    UserAgent           string        `yaml:"user_agent,omitempty"`
    AllowPrivateTargets bool          `yaml:"allow_private_targets,omitempty"`
}

func Load(path string) (*Config, error)   // file → Parse
func Parse(data []byte) (*Config, error)  // strict YAML unmarshal
func (*Config) Validate() error           // cross-field invariants
```

`Parse` uses `yaml.UnmarshalStrict` so a typo (`htpasswrd`,
`per_proobe_timeout`) returns a parse error instead of being
silently dropped onto an empty default. Restart-to-reload is the
deployment shape; a stale config that "worked" because the parser
ignored a field is the boring failure mode we want to prevent.

## Precedence resolution

`flag.Visit` walks only flags that were explicitly set on the
command line, so we can distinguish "flag at default value" from
"flag not specified". Combined with `os.LookupEnv`, we get a
clean four-layer resolver:

```go
func resolveString(setFlags map[string]bool, flagName, flagVal,
    envName, yamlVal, hardDefault string) string {
    if setFlags[flagName] {
        return flagVal
    }
    if v, ok := os.LookupEnv(envName); ok && v != "" {
        return v
    }
    if yamlVal != "" {
        return yamlVal
    }
    return hardDefault
}
```

Sibling helpers `resolveBool`, `resolveDuration` follow the same
shape. Bool resolution is the only awkward one — Go's `flag.Bool`
doesn't distinguish "user passed `--ui=false`" from "default
false", but `flag.Visit` does. We keep that distinction: if the
user explicitly typed `--ui=false`, that wins over a YAML
`ui.enabled: true`.

## Migration story

No data migration. `--config` is a new optional flag; absent =
behaviour is byte-identical to today. An operator who wants the
config-file shape lays a `serve.yaml` next to the binary, points
`--config` at it, and removes the now-redundant flags from their
systemd unit. An operator who never adopts the YAML continues to
pass flags exactly as before.

## What we don't do

- **No live reload.** A SIGHUP-driven re-parse adds
  concurrency surface (the running scheduler holds references
  to flag-derived state) for operator ergonomics that
  restart-to-reload already provides.
- **No env-var ingestion through YAML.** `${WANDERER_GEOIP}`
  expansion in the YAML is tempting but introduces a parser
  gotcha: which env vars at which time, do we shell-expand or
  Go-expand, etc. The flag/env/yaml/default precedence already
  covers the operator-driven case cleanly; deeper interpolation
  is a separate proposal.
- **No schema versioning.** The config is small and additive;
  every new field gets an `omitempty` zero-value default. If
  the schema ever needs a breaking change, a new `version:` key
  is the time to introduce it.

## Test strategy

- **Parse errors.** A typo in a field name returns a clear error
  from `Parse`; the binary refuses to start.
- **Each precedence layer in isolation.** A test sets only the
  hard default → result is the default. Adds YAML → result is
  YAML. Adds env → result is env. Adds flag → result is flag.
  The same test set runs against `resolveString`,
  `resolveBool`, `resolveDuration` so every type's resolver is
  pinned.
- **Empty config file is valid.** `serve.yaml` with `{}` parses
  cleanly and resolves every value to its hard default — empty
  YAML must be equivalent to no YAML.
- **Strict mode catches typos.** A YAML containing
  `htpasswrd: foo` returns an error mentioning the unknown
  field.

No integration test that boots a real serve process — the unit
seam is the resolver helpers; the existing serve.go tests pin
the rest of the surface.
