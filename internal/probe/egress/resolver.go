package egress

import (
	"context"
	"net"
	"time"

	"github.com/MWest2020/wanderer/internal/probe/ip"
)

// IPResolver adapts an *ip.Probe to the HostResolver interface so the
// egress probe can annotate Findings with ASN / organisation /
// country without depending on the package directly.
type IPResolver struct {
	IP      *ip.Probe
	Timeout time.Duration
}

// Resolve looks up host and returns the first IP's ASN data. The
// boolean is false when no answer is available (probe not
// configured, lookup error, no addresses).
func (r IPResolver) Resolve(host string) (uint, string, string, bool) {
	if r.IP == nil {
		return 0, "", "", false
	}
	timeout := r.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupHost(ctx, host)
	if err != nil || len(addrs) == 0 {
		return 0, "", "", false
	}
	parsed := net.ParseIP(addrs[0])
	if parsed == nil {
		return 0, "", "", false
	}
	out, err := r.IP.Lookup(parsed)
	if err != nil {
		return 0, "", "", false
	}
	return out.ASN, out.Organisation, out.Country, true
}
