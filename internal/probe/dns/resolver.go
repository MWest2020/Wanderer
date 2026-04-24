package dns

import (
	"context"
	"net"
)

// NewNetResolver wraps a *net.Resolver with the CAA adapter we need.
// *net.Resolver does not expose CAA lookups directly; we fall back to
// returning an empty slice (not an error) so the probe can continue.
// Wire in a real DNS client if CAA is important for your deployment.
func NewNetResolver(r *net.Resolver) Resolver {
	if r == nil {
		r = net.DefaultResolver
	}
	return &netResolver{r: r}
}

type netResolver struct{ r *net.Resolver }

func (n *netResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	return n.r.LookupHost(ctx, host)
}
func (n *netResolver) LookupMX(ctx context.Context, name string) ([]*net.MX, error) {
	return n.r.LookupMX(ctx, name)
}
func (n *netResolver) LookupNS(ctx context.Context, name string) ([]*net.NS, error) {
	return n.r.LookupNS(ctx, name)
}
func (n *netResolver) LookupCNAME(ctx context.Context, host string) (string, error) {
	return n.r.LookupCNAME(ctx, host)
}
func (n *netResolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	return n.r.LookupTXT(ctx, name)
}

// LookupCAA is not supported by net.Resolver. We return an empty slice
// and no error — the probe records "no CAA records" as a neutral info
// Finding, which is the same shape as actual absence. Operators who
// need strict CAA visibility should swap in a resolver that queries a
// recursive DNS server directly (e.g. via miekg/dns).
func (n *netResolver) LookupCAA(ctx context.Context, name string) ([]CAA, error) {
	return nil, nil
}
