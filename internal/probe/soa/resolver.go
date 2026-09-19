package soa

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

// NewNetResolver returns a Resolver that queries SOA records with a
// minimal hand-rolled DNS client (net.Resolver exposes no SOA lookup,
// the same gap the DNS probe documents for CAA) and delegates the
// mailbox-domain host/MX checks to r.
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

func (n *netResolver) LookupSOA(ctx context.Context, domain string) (string, string, error) {
	server, err := systemNameserver()
	if err != nil {
		return "", "", err
	}
	return querySOA(ctx, server, domain)
}

// systemNameserver returns the first "nameserver" entry in
// /etc/resolv.conf as a host:port pair. Wanderer is a Linux-only
// operator tool (see the egress probe's eBPF dependency), so reading
// resolv.conf directly is the boring option — no vendored DNS client
// is available offline (see openspec/config.yaml: stdlib first,
// dependencies are vendored).
func systemNameserver() (string, error) {
	f, err := os.Open("/etc/resolv.conf")
	if err != nil {
		return "", fmt.Errorf("soa: read resolv.conf: %w", err)
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
		return "", fmt.Errorf("soa: read resolv.conf: %w", err)
	}
	return "", errors.New("soa: no nameserver found in resolv.conf")
}

// querySOA sends one UDP SOA query for domain to server and parses the
// first SOA record out of the response.
func querySOA(ctx context.Context, server, domain string) (mname, rname string, err error) {
	msg, id, err := encodeSOAQuery(strings.TrimSuffix(domain, "."))
	if err != nil {
		return "", "", err
	}

	var d net.Dialer
	conn, err := d.DialContext(ctx, "udp", server)
	if err != nil {
		return "", "", fmt.Errorf("soa: dial %s: %w", server, err)
	}
	defer conn.Close()

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(5 * time.Second)
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return "", "", fmt.Errorf("soa: set deadline: %w", err)
	}

	if _, err := conn.Write(msg); err != nil {
		return "", "", fmt.Errorf("soa: write query: %w", err)
	}
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return "", "", fmt.Errorf("soa: read response: %w", err)
	}
	return decodeSOAResponse(buf[:n], id)
}

// dnsTypeSOA and dnsClassIN are the wire-format constants for a "SOA
// IN" question (RFC 1035 §3.2.2, §3.2.4).
const (
	dnsTypeSOA = 6
	dnsClassIN = 1
)

func encodeSOAQuery(domain string) ([]byte, uint16, error) {
	id := uint16(rand.Intn(1 << 16))
	var buf bytes.Buffer

	header := make([]byte, 12)
	binary.BigEndian.PutUint16(header[0:2], id)
	binary.BigEndian.PutUint16(header[2:4], 0x0100) // RD=1, standard query
	binary.BigEndian.PutUint16(header[4:6], 1)       // QDCOUNT=1
	buf.Write(header)

	name, err := encodeName(domain)
	if err != nil {
		return nil, 0, err
	}
	buf.Write(name)

	qtype := make([]byte, 4)
	binary.BigEndian.PutUint16(qtype[0:2], dnsTypeSOA)
	binary.BigEndian.PutUint16(qtype[2:4], dnsClassIN)
	buf.Write(qtype)

	return buf.Bytes(), id, nil
}

func encodeName(domain string) ([]byte, error) {
	var buf bytes.Buffer
	if domain != "" {
		for _, label := range strings.Split(domain, ".") {
			if len(label) == 0 || len(label) > 63 {
				return nil, fmt.Errorf("soa: invalid label %q in domain %q", label, domain)
			}
			buf.WriteByte(byte(len(label)))
			buf.WriteString(label)
		}
	}
	buf.WriteByte(0)
	return buf.Bytes(), nil
}

func decodeSOAResponse(msg []byte, wantID uint16) (mname, rname string, err error) {
	if len(msg) < 12 {
		return "", "", errors.New("soa: response too short")
	}
	if binary.BigEndian.Uint16(msg[0:2]) != wantID {
		return "", "", errors.New("soa: response ID mismatch")
	}
	flags := binary.BigEndian.Uint16(msg[2:4])
	if rcode := flags & 0x000F; rcode != 0 {
		return "", "", fmt.Errorf("soa: response rcode %d", rcode)
	}
	qdcount := int(binary.BigEndian.Uint16(msg[4:6]))
	ancount := int(binary.BigEndian.Uint16(msg[6:8]))

	off := 12
	for i := 0; i < qdcount; i++ {
		_, next, err := readName(msg, off)
		if err != nil {
			return "", "", err
		}
		off = next + 4 // QTYPE + QCLASS
	}

	for i := 0; i < ancount; i++ {
		_, next, err := readName(msg, off)
		if err != nil {
			return "", "", err
		}
		off = next
		if off+10 > len(msg) {
			return "", "", errors.New("soa: truncated record header")
		}
		rtype := binary.BigEndian.Uint16(msg[off : off+2])
		rdlen := int(binary.BigEndian.Uint16(msg[off+8 : off+10]))
		off += 10
		if off+rdlen > len(msg) {
			return "", "", errors.New("soa: truncated rdata")
		}
		if rtype == dnsTypeSOA {
			m, mnext, err := readName(msg, off)
			if err != nil {
				return "", "", err
			}
			r, _, err := readName(msg, mnext)
			if err != nil {
				return "", "", err
			}
			return m, r, nil
		}
		off += rdlen
	}
	return "", "", errors.New("soa: no SOA record in response")
}

// readName decodes a (possibly compressed, RFC 1035 §4.1.4) domain
// name starting at offset in msg. It returns the name in RFC 1035
// presentation form — dots and backslashes inside a label are escaped
// with a backslash, so an embedded literal dot in the RNAME's local
// part survives round-tripping through splitRNAME — and the offset in
// msg immediately following the name as it appeared at offset (i.e.
// after a compression pointer, not after the label(s) it points to).
func readName(msg []byte, offset int) (string, int, error) {
	var labels []string
	pos := offset
	endPos := -1
	jumps := 0
	for {
		if pos >= len(msg) {
			return "", 0, errors.New("soa: name out of bounds")
		}
		length := int(msg[pos])
		switch {
		case length == 0:
			pos++
			if endPos == -1 {
				endPos = pos
			}
			return strings.Join(labels, "."), endPos, nil
		case length&0xC0 == 0xC0:
			if pos+1 >= len(msg) {
				return "", 0, errors.New("soa: truncated compression pointer")
			}
			if endPos == -1 {
				endPos = pos + 2
			}
			jumps++
			if jumps > 20 {
				return "", 0, errors.New("soa: too many compression pointers")
			}
			pos = int(binary.BigEndian.Uint16(msg[pos:pos+2]) & 0x3FFF)
		default:
			pos++
			if pos+length > len(msg) {
				return "", 0, errors.New("soa: label out of bounds")
			}
			labels = append(labels, escapeLabel(string(msg[pos:pos+length])))
			pos += length
		}
	}
}

// escapeLabel backslash-escapes dots and backslashes within a single
// wire-format label so the RFC 1035 presentation form built from
// joining labels with "." stays unambiguous.
func escapeLabel(label string) string {
	if !strings.ContainsAny(label, ".\\") {
		return label
	}
	var b strings.Builder
	for i := 0; i < len(label); i++ {
		c := label[i]
		if c == '.' || c == '\\' {
			b.WriteByte('\\')
		}
		b.WriteByte(c)
	}
	return b.String()
}
