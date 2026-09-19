package variants

import "net"

// detectIPv6 reports whether the scanner host has a global or
// unique-local IPv6 address configured on any interface — the
// closest offline signal to "the scanner has an IPv6 route" without
// making a network call. Loopback and link-local addresses don't
// count: neither can reach the public internet.
func detectIPv6() bool {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false
	}
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		ip := ipnet.IP
		if ip.To4() != nil {
			continue
		}
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			continue
		}
		return true
	}
	return false
}
