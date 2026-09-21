package dns

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"
)

// NewNetResolver wraps a *net.Resolver with the CAA adapter we need.
// *net.Resolver does not expose CAA lookups directly, so LookupCAA is a
// hand-rolled UDP query — the same gap and the same fix the SOA probe
// already applies (internal/probe/soa/resolver.go).
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

// LookupCAA queries CAA records for name directly (no tree-walk — that
// is the DNS probe's job, see caaFindings in dns.go). net.Resolver
// exposes no CAA lookup, so this sends one hand-rolled UDP query to the
// system's configured nameserver and parses the answer section itself.
func (n *netResolver) LookupCAA(ctx context.Context, name string) ([]CAA, error) {
	server, err := systemNameserver()
	if err != nil {
		return nil, err
	}
	return queryCAA(ctx, server, name)
}

// systemNameserver returns the first "nameserver" entry in
// /etc/resolv.conf as a host:port pair. Wanderer is a Linux-only
// operator tool (see the egress probe's eBPF dependency), so reading
// resolv.conf directly is the boring option — no vendored DNS client
// is available offline (see openspec/config.yaml: stdlib first,
// dependencies are vendored). Identical to the SOA probe's helper of
// the same name; kept local so the DNS probe stays self-contained
// (config.yaml: "each probe is isolated").
func systemNameserver() (string, error) {
	f, err := os.Open("/etc/resolv.conf")
	if err != nil {
		return "", fmt.Errorf("dns: read resolv.conf: %w", err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 || fields[0] != "nameserver" {
			continue
		}
		ip := net.ParseIP(fields[1])
		if ip == nil {
			continue
		}
		return net.JoinHostPort(ip.String(), "53"), nil
	}
	if err := sc.Err(); err != nil {
		return "", fmt.Errorf("dns: read resolv.conf: %w", err)
	}
	return "", errors.New("dns: no nameserver found in resolv.conf")
}

// queryCAA sends one UDP "CAA IN <name>" query to server and returns
// every CAA record in the response's answer section.
func queryCAA(ctx context.Context, server, name string) ([]CAA, error) {
	msg, id, err := encodeCAAQuery(strings.TrimSuffix(name, "."))
	if err != nil {
		return nil, err
	}

	var d net.Dialer
	conn, err := d.DialContext(ctx, "udp", server)
	if err != nil {
		return nil, fmt.Errorf("dns: dial %s: %w", server, err)
	}
	defer conn.Close()

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(5 * time.Second)
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return nil, fmt.Errorf("dns: set deadline: %w", err)
	}

	if _, err := conn.Write(msg); err != nil {
		return nil, fmt.Errorf("dns: write query: %w", err)
	}
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("dns: read response: %w", err)
	}
	return decodeCAAResponse(buf[:n], id)
}

// dnsTypeCAA and dnsClassIN are the wire-format constants for a
// "CAA IN" question (RFC 8659 §4, RFC 1035 §3.2.4).
const (
	dnsTypeCAA = 257
	dnsClassIN = 1
)

func encodeCAAQuery(name string) ([]byte, uint16, error) {
	id := uint16(rand.Intn(1 << 16))
	var buf bytes.Buffer

	header := make([]byte, 12)
	binary.BigEndian.PutUint16(header[0:2], id)
	binary.BigEndian.PutUint16(header[2:4], 0x0100) // RD=1, standard query
	binary.BigEndian.PutUint16(header[4:6], 1)      // QDCOUNT=1
	buf.Write(header)

	qname, err := encodeName(name)
	if err != nil {
		return nil, 0, err
	}
	buf.Write(qname)

	qtype := make([]byte, 4)
	binary.BigEndian.PutUint16(qtype[0:2], dnsTypeCAA)
	binary.BigEndian.PutUint16(qtype[2:4], dnsClassIN)
	buf.Write(qtype)

	return buf.Bytes(), id, nil
}

func encodeName(name string) ([]byte, error) {
	var buf bytes.Buffer
	if name != "" {
		for _, label := range strings.Split(name, ".") {
			if len(label) == 0 || len(label) > 63 {
				return nil, fmt.Errorf("dns: invalid label %q in name %q", label, name)
			}
			buf.WriteByte(byte(len(label)))
			buf.WriteString(label)
		}
	}
	buf.WriteByte(0)
	return buf.Bytes(), nil
}

// decodeCAAResponse parses the answer section of a DNS response to a
// CAA query, returning every CAA record it finds. NXDOMAIN (RCODE=3)
// is reported as "no records, no error" — RFC 8659 §3's tree-walk
// treats a non-existent intermediate name the same as an empty one and
// keeps climbing.
func decodeCAAResponse(msg []byte, wantID uint16) ([]CAA, error) {
	if len(msg) < 12 {
		return nil, errors.New("dns: response too short")
	}
	if binary.BigEndian.Uint16(msg[0:2]) != wantID {
		return nil, errors.New("dns: response ID mismatch")
	}
	flags := binary.BigEndian.Uint16(msg[2:4])
	if rcode := flags & 0x000F; rcode != 0 {
		if rcode == 3 { // NXDOMAIN
			return nil, nil
		}
		return nil, fmt.Errorf("dns: response rcode %d", rcode)
	}
	qdcount := int(binary.BigEndian.Uint16(msg[4:6]))
	ancount := int(binary.BigEndian.Uint16(msg[6:8]))

	off := 12
	for i := 0; i < qdcount; i++ {
		next, err := skipName(msg, off)
		if err != nil {
			return nil, err
		}
		off = next + 4 // QTYPE + QCLASS
	}

	var out []CAA
	for i := 0; i < ancount; i++ {
		next, err := skipName(msg, off)
		if err != nil {
			return nil, err
		}
		off = next
		if off+10 > len(msg) {
			return nil, errors.New("dns: truncated record header")
		}
		rtype := binary.BigEndian.Uint16(msg[off : off+2])
		rdlen := int(binary.BigEndian.Uint16(msg[off+8 : off+10]))
		off += 10
		if off+rdlen > len(msg) {
			return nil, errors.New("dns: truncated rdata")
		}
		if rtype == dnsTypeCAA {
			c, err := decodeCAARecord(msg[off : off+rdlen])
			if err != nil {
				return nil, err
			}
			out = append(out, c)
		}
		off += rdlen
	}
	return out, nil
}

// decodeCAARecord parses a CAA RDATA blob (RFC 8659 §4.1): a 1-byte
// flag, a 1-byte tag length, the tag itself, and the remaining bytes as
// the value. Unlike an SOA's MNAME/RNAME, none of this is a domain
// name, so no name decompression is involved.
func decodeCAARecord(rdata []byte) (CAA, error) {
	if len(rdata) < 2 {
		return CAA{}, errors.New("dns: caa rdata too short")
	}
	tagLen := int(rdata[1])
	if 2+tagLen > len(rdata) {
		return CAA{}, errors.New("dns: caa tag length out of bounds")
	}
	return CAA{
		Flag:  rdata[0],
		Tag:   string(rdata[2 : 2+tagLen]),
		Value: string(rdata[2+tagLen:]),
	}, nil
}

// skipName returns the offset in msg immediately following the
// (possibly compressed, RFC 1035 §4.1.4) domain name starting at
// offset. Answer owner names are skipped, never inspected, so this
// does not need to follow compression pointers or build the name —
// only recognise where it ends.
func skipName(msg []byte, offset int) (int, error) {
	pos := offset
	for {
		if pos >= len(msg) {
			return 0, errors.New("dns: name out of bounds")
		}
		length := int(msg[pos])
		switch {
		case length == 0:
			return pos + 1, nil
		case length&0xC0 == 0xC0:
			if pos+1 >= len(msg) {
				return 0, errors.New("dns: truncated compression pointer")
			}
			return pos + 2, nil
		default:
			if pos+1+length > len(msg) {
				return 0, errors.New("dns: label out of bounds")
			}
			pos += 1 + length
		}
	}
}
