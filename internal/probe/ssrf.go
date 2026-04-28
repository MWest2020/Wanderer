package probe

import (
	"context"
	"errors"
	"net"
)

// ErrPrivateTargetBlocked is returned by SafeDialer when every
// resolved address for the requested host falls inside one of the
// private / metadata ranges and `--allow-private-targets` is not
// set. The error wording is deliberately generic so it can be
// surfaced verbatim in a Finding's Attributes without leaking
// which family blocked the connection.
var ErrPrivateTargetBlocked = errors.New("scanner: target resolves only to private or cloud-metadata addresses")

// SafeDialer wraps net.Dialer with a pre-connect IP check. The zero
// value is safe to use; AllowPrivate=false means the dialer rejects
// private and metadata destinations.
type SafeDialer struct {
	// Inner is the dialer used for the actual TCP connect. nil ⇒
	// a fresh net.Dialer with no extra options.
	Inner *net.Dialer
	// Resolver overrides the DNS resolver. nil ⇒ net.DefaultResolver.
	Resolver *net.Resolver
	// AllowPrivate disables the private-address block. Operators
	// scanning internal infrastructure flip this on with
	// `--allow-private-targets`.
	AllowPrivate bool
}

// DialContext implements the standard dialer signature so the dialer
// can plug straight into http.Transport and the tls.Dialer. It
// resolves the host, filters private/metadata addresses, and dials
// only the survivors.
func (d *SafeDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	resolver := d.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	ips, err := resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	var allowed []net.IPAddr
	for _, ip := range ips {
		if !d.AllowPrivate && IsPrivateOrMetadata(ip.IP) {
			continue
		}
		allowed = append(allowed, ip)
	}
	if len(allowed) == 0 {
		return nil, ErrPrivateTargetBlocked
	}
	inner := d.Inner
	if inner == nil {
		inner = &net.Dialer{}
	}
	for _, ip := range allowed {
		conn, dialErr := inner.DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
		if dialErr == nil {
			return conn, nil
		}
		err = dialErr
	}
	return nil, err
}

// blockedRanges is the static set of IP networks the SafeDialer
// refuses to connect to by default. Two reviewers should be able
// to eyeball this slice and agree it covers loopback, link-local,
// private (RFC1918), CGNAT, IPv6 ULA, IPv6 link-local, and the
// well-known cloud-metadata addresses.
var blockedRanges = func() []*net.IPNet {
	cidrs := []string{
		"127.0.0.0/8",       // IPv4 loopback
		"169.254.0.0/16",    // IPv4 link-local incl. 169.254.169.254 metadata
		"10.0.0.0/8",        // RFC1918
		"172.16.0.0/12",     // RFC1918
		"192.168.0.0/16",    // RFC1918
		"100.64.0.0/10",     // CGNAT
		"::1/128",           // IPv6 loopback
		"fc00::/7",          // IPv6 ULA
		"fe80::/10",         // IPv6 link-local
		"fd00:ec2::254/128", // AWS IMDSv2 IPv6 metadata
	}
	out := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			panic("scanner: bad CIDR in blockedRanges: " + c)
		}
		out = append(out, n)
	}
	return out
}()

// IsPrivateOrMetadata reports whether ip falls inside one of the
// blocked ranges. Exported so the API layer can pre-flight a
// requested target before kicking off a scan.
func IsPrivateOrMetadata(ip net.IP) bool {
	for _, n := range blockedRanges {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}
