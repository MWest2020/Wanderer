# Design: Agent outbox spool

## Component placement

```
internal/agent/outbox.go        # spool + drain + prune
internal/agent/remote.go        # gains Send-with-spool wrapper
cmd/wanderer/agent.go           # wires the outbox into the loop
```

## Data layout on disk

```
/var/lib/wanderer/agent/outbox/
  20260427T154500Z_a1b2c3.json
  20260427T160500Z_d4e5f6.json
```

Filenames are `<RFC3339-basic>_<random6>.json`. The lexicographic
sort of filenames matches drain order (oldest first). One file per
failed batch — the batch is exactly the JSON body the agent would
have POSTed, including the original timestamp so HMAC re-signing
on retry uses the *current* time, not the original.

## API

```go
type Outbox struct {
    Dir       string
    MaxBytes  int64    // default 100 MiB
}

func (o *Outbox) Spool(ctx context.Context, batch []byte) error
func (o *Outbox) Drain(ctx context.Context, send func(batch []byte) error) error
func (o *Outbox) Prune(ctx context.Context) error
```

`Drain` reads files in sorted order, calls `send`, and only
deletes the file on success. A persistent failure leaves the file
in place for the next tick.

## Backoff inside a single tick

Three tries at 0s, 250ms, 1s with ±25% jitter. After three the
batch goes to the spool; the agent moves on so the inspector
schedule keeps running.

## Failure modes

| Cause                       | Outcome                                            |
| --------------------------- | -------------------------------------------------- |
| Outbox dir not writable     | Hard error at startup; agent exits non-zero        |
| Disk full mid-spool         | Log + drop the batch; emit a `agent.spool_full` log line, do not crash |
| File in outbox is corrupt   | Skip + log; do not block other files               |
| Outbox larger than MaxBytes | Prune oldest files until under the limit           |

## Tests

- Round-trip: Spool a batch, Drain it with a stub `send` that
  succeeds, file is gone.
- Drain with failing send: file stays.
- Prune: spool > MaxBytes worth of batches, prune leaves only the
  newest under the cap.

## Clever valkuil

Tempting: SQLite. Wrong — adds DB lifecycle to the agent for a
fundamentally append-only queue. Files are simpler.
