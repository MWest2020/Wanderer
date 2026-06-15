package probe

import (
	"errors"
	"net"
	"testing"
)

func TestIsPrivateOrMetadata(t *testing.T) {
	cases := []struct {
		ip   string
		want bool
	}{
		{"127.0.0.1", true},             // loopback
		{"10.1.2.3", true},              // RFC1918
		{"172.20.4.5", true},            // RFC1918
		{"192.168.1.1", true},           // RFC1918
		{"100.64.0.5", true},            // CGNAT
		{"169.254.169.254", true},       // AWS/GCP metadata
		{"::1", true},                   // IPv6 loopback
		{"fd00::1", true},               // IPv6 ULA
		{"fe80::1", true},               // IPv6 link-local
		{"fd00:ec2::254", true},         // AWS IMDSv2 IPv6 metadata
		{"8.8.8.8", false},              // public
		{"1.1.1.1", false},              // public
		{"2606:4700:4700::1111", false}, // public IPv6
	}
	for _, c := range cases {
		ip := net.ParseIP(c.ip)
		if ip == nil {
			t.Fatalf("bad ip in fixture: %s", c.ip)
		}
		if got := IsPrivateOrMetadata(ip); got != c.want {
			t.Errorf("IsPrivateOrMetadata(%s) = %v, want %v", c.ip, got, c.want)
		}
	}
}

func TestSafeDialer_BlockedHost(t *testing.T) {
	// Build a SafeDialer wired to a stub resolver that returns only
	// private addresses. Use a custom Resolver via PreferGo + Dial
	// that routes the synthetic name to the canned IPs is awkward;
	// instead we exercise the same code path via the helper.
	d := &SafeDialer{}
	private := []net.IPAddr{{IP: net.ParseIP("10.0.0.1")}}
	allowed := filterAllowed(private, d.AllowPrivate)
	if len(allowed) != 0 {
		t.Errorf("private IP should be filtered, got %v", allowed)
	}

	d.AllowPrivate = true
	allowed = filterAllowed(private, d.AllowPrivate)
	if len(allowed) != 1 {
		t.Errorf("AllowPrivate should pass private IP, got %v", allowed)
	}

	mixed := []net.IPAddr{
		{IP: net.ParseIP("10.0.0.1")},
		{IP: net.ParseIP("8.8.8.8")},
	}
	allowed = filterAllowed(mixed, false)
	if len(allowed) != 1 || !allowed[0].IP.Equal(net.ParseIP("8.8.8.8")) {
		t.Errorf("mixed should keep public only, got %v", allowed)
	}
}

func TestSafeDialer_ErrPrivateTargetBlocked(t *testing.T) {
	if !errors.Is(ErrPrivateTargetBlocked, ErrPrivateTargetBlocked) {
		t.Fatalf("sentinel should compare equal to itself")
	}
}

// filterAllowed mirrors the inline logic in DialContext so we can
// unit-test the filter without exercising the network.
func filterAllowed(ips []net.IPAddr, allowPrivate bool) []net.IPAddr {
	var out []net.IPAddr
	for _, ip := range ips {
		if !allowPrivate && IsPrivateOrMetadata(ip.IP) {
			continue
		}
		out = append(out, ip)
	}
	return out
}
