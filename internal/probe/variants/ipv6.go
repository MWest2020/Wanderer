package variants

import (
	"context"
	"net"
	"time"
)

// ipv6ProbeTimeout bounds the short dial detectIPv6 uses to decide
// scanner IPv6 capability.
const ipv6ProbeTimeout = 750 * time.Millisecond

// ipv6ProbeAddr is a stable public IPv6 host:port (Google Public DNS)
// used only as a UDP connect target: for a UDP socket, connect()
// makes the kernel resolve a real route (or fail immediately if none
// exists) without ever putting a packet on the wire.
const ipv6ProbeAddr = "[2001:4860:4860::8888]:53"

// detectIPv6 reports whether the scanner host can reach the public
// internet over IPv6 — not merely whether some IPv6 address is
// configured on an interface. An address alone proves nothing: a
// Tailscale unique-local address (fd7a::/16, within fc00::/7) or a
// CGNAT-style address has no path to the public internet, yet both
// show up in net.InterfaceAddrs(). A short-timeout UDP dial to a
// stable public address forces the kernel to answer the real
// question — is there a route out? — once per scan.
func detectIPv6() bool {
	d := net.Dialer{Timeout: ipv6ProbeTimeout}
	conn, err := d.DialContext(context.Background(), "udp6", ipv6ProbeAddr)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
