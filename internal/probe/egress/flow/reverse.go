package flow

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"
)

// ReverseDNSResolver maps a destination IP to a best-effort hostname.
// Implementations MUST honour a per-call deadline and MUST return
// ok=false (no error surfaced to the caller) on any failure mode —
// NXDOMAIN, timeout, refused, transport error. Reverse DNS is
// strictly enrichment; failure is normal and not a Finding.
type ReverseDNSResolver interface {
	Reverse(ctx context.Context, ip string) (host string, ok bool)
}

// DefaultReverseTimeout is the per-lookup timeout used when the
// agent config leaves egress.flow.reverse_dns.timeout unset.
const DefaultReverseTimeout = 500 * time.Millisecond

// NewReverseDNSResolver returns a ReverseDNSResolver backed by
// net.DefaultResolver. The timeout is applied per call and clamps
// any individual lookup that the host's DNS path cannot answer
// quickly. A non-positive timeout falls back to DefaultReverseTimeout.
func NewReverseDNSResolver(timeout time.Duration) ReverseDNSResolver {
	if timeout <= 0 {
		timeout = DefaultReverseTimeout
	}
	return &netReverseResolver{timeout: timeout}
}

type netReverseResolver struct {
	timeout time.Duration
}

func (r *netReverseResolver) Reverse(ctx context.Context, ip string) (string, bool) {
	subCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	names, err := net.DefaultResolver.LookupAddr(subCtx, ip)
	if err != nil || len(names) == 0 {
		return "", false
	}
	host := strings.TrimSuffix(names[0], ".")
	if host == "" {
		return "", false
	}
	return host, true
}

// ptrCache memoises ReverseDNSResolver lookups inside one
// sampling window so multiple Findings for the same destination
// IP — for example, port 443 and 8443 to the same host — produce
// exactly one PTR query.
type ptrCache struct {
	mu   sync.Mutex
	seen map[string]ptrResult
}

type ptrResult struct {
	host string
	ok   bool
}

func newPTRCache() *ptrCache {
	return &ptrCache{seen: map[string]ptrResult{}}
}

// Lookup consults the cache and falls through to the resolver on
// miss. The resolver result — including a negative one — is cached
// so repeated misses for the same IP cost one query, not N.
func (c *ptrCache) Lookup(ctx context.Context, resolver ReverseDNSResolver, ip string) (string, bool) {
	if resolver == nil {
		return "", false
	}
	c.mu.Lock()
	hit, found := c.seen[ip]
	c.mu.Unlock()
	if found {
		return hit.host, hit.ok
	}
	host, ok := resolver.Reverse(ctx, ip)
	c.mu.Lock()
	c.seen[ip] = ptrResult{host: host, ok: ok}
	c.mu.Unlock()
	return host, ok
}
