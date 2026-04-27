# Design: Externalised vendor list

## Layout

```
internal/probe/egress/
  vendors.go            # loader + schema
  vendors.yaml          # default data, embedded
  vendors_test.go
  classify.go           # consumes the loaded table via a small interface
```

## YAML schema

```yaml
log_shippers:
  - host_contains: datadoghq.com
    rule_id: datadog
  - host_contains: papertrailapp.com
    rule_id: papertrail

webhooks:
  - host_contains: hooks.slack.com
    rule_id: slack
  - host_contains: outlook.office.com/webhook
    rule_id: teams

object_storage:
  aws_regional_regex: '^s3[.-]([a-z]{2}-[a-z]+-\d)\.amazonaws\.com$'
  gcs_host_contains:  storage.googleapis.com
  azure_host_contains: blob.core.windows.net

us_hyperscaler_organisation_substrings:
  - amazon
  - google
  - microsoft
  - cloudflare
  - akamai
  - fastly
```

The loader returns a typed `Vendors` struct that the classifier
consults instead of the current inline constants.

## Override precedence

1. `--vendors /etc/wanderer/vendors.yaml` (CLI flag)
2. `WANDERER_VENDORS` env var
3. The embedded default

A loaded file replaces the defaults wholesale; merging files is a
follow-up if anyone asks for it.

## Tests

- Round-trip: parse the embedded file, assert the classifier still
  produces the same verdicts as before for the existing test cases.
- Override: pass a tiny vendor file with a custom log_shipper
  entry; assert the new rule fires.
- Schema: malformed YAML at load time is a fatal error with a
  clear message that names the offending key.

## Failure modes

| Cause                           | Outcome                                                         |
| ------------------------------- | --------------------------------------------------------------- |
| Override file not readable      | Fatal at agent start; the operator sees the failure immediately |
| Override file YAML invalid      | Fatal with a parse-error line/column                            |
| Override file missing keys      | Empty list for that category; no findings rather than crash     |
