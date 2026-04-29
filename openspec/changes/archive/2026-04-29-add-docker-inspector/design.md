# Design: Docker inspector

## Component placement

```
internal/probe/inventory/docker/
  docker.go         # existing placeholder, replace with real impl
  client.go         # http.Client wired to a unix socket dialer
  client_test.go    # httptest server over a unix socket fixture
```

## Transport

Stdlib only. The Docker Engine API speaks HTTP; we just route the
TCP socket through a unix domain socket dialer:

```go
func newClient(socket string) *http.Client {
    return &http.Client{
        Transport: &http.Transport{
            DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
                return (&net.Dialer{}).DialContext(ctx, "unix", socket)
            },
        },
        Timeout: 10 * time.Second,
    }
}
```

Requests are issued against `http://docker/v1.41/containers/json`
and `http://docker/v1.41/images/json`. The host portion is
irrelevant — the dialer routes everything to the socket.

## API surface used

| Endpoint                        | Purpose                              |
| ------------------------------- | ------------------------------------ |
| `GET /v1.41/containers/json?all=true` | List containers, all states    |
| `GET /v1.41/images/json`        | List images with digest + repo tags  |

We do not call `/info`, `/version`, or per-container `/inspect` in
the MVP. They are richer but optional.

## Failure modes

| Cause                  | Outcome                                                         |
| ---------------------- | --------------------------------------------------------------- |
| Socket missing         | `inventory.docker.unavailable` with `reason: not present`       |
| Permission denied      | `inventory.docker.unavailable` with the OS error in `reason`    |
| API responds non-200   | `inventory.docker.error` with `status_code` and `body` snippet  |
| Body decodes badly     | `inventory.docker.error` with the JSON error                    |

## Tests

- `httptest.NewUnstartedServer` bound to a temporary unix socket;
  hand-rolled JSON fixtures for `/containers/json` and
  `/images/json`.
- One test per failure mode (404, 500, malformed JSON).

## Clever valkuil

Tempting: pull `github.com/docker/docker/client`. Wrong — the
SDK is large, drags in CGo helpers in some configurations, and
gives us features we explicitly do not want (mutating calls,
swarm primitives). Two stdlib HTTP calls is the right size.
