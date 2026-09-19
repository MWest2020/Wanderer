package soa

import (
	"bytes"
	"context"
	"encoding/binary"
	"net"
	"testing"
	"time"
)

// buildSOAResponse assembles a wire-format DNS response to a "SOA IN
// <domain>" question with the given id, using compression pointers
// for the answer's owner name and for the RNAME's domain suffix (the
// same shape real authoritative servers send), so the test exercises
// readName's pointer-following, not just flat names.
func buildSOAResponse(t *testing.T, id uint16, domain, mname, rnameLocal string) []byte {
	t.Helper()
	qname, err := encodeName(domain)
	if err != nil {
		t.Fatalf("encodeName: %v", err)
	}
	const qnameOffset = 12 // right after the 12-byte header

	var buf bytes.Buffer
	header := make([]byte, 12)
	binary.BigEndian.PutUint16(header[0:2], id)
	binary.BigEndian.PutUint16(header[2:4], 0x8180) // QR=1, RA=1, RCODE=0
	binary.BigEndian.PutUint16(header[4:6], 1)      // QDCOUNT
	binary.BigEndian.PutUint16(header[6:8], 1)      // ANCOUNT
	buf.Write(header)
	buf.Write(qname)
	qtype := make([]byte, 4)
	binary.BigEndian.PutUint16(qtype[0:2], dnsTypeSOA)
	binary.BigEndian.PutUint16(qtype[2:4], dnsClassIN)
	buf.Write(qtype)

	// Answer: NAME is a pointer back to the question's name (the SOA's
	// owner is the zone apex).
	buf.Write(pointerTo(qnameOffset))
	rrHead := make([]byte, 8)
	binary.BigEndian.PutUint16(rrHead[0:2], dnsTypeSOA)
	binary.BigEndian.PutUint16(rrHead[2:4], dnsClassIN)
	binary.BigEndian.PutUint32(rrHead[4:8], 0) // TTL
	buf.Write(rrHead)

	var rdata bytes.Buffer
	// MNAME: partial label + pointer to the zone name.
	writeLabel(&rdata, firstLabel(mname))
	rdata.Write(pointerTo(qnameOffset))
	// RNAME: local-part label + pointer to the zone name as the domain.
	writeLabel(&rdata, rnameLocal)
	rdata.Write(pointerTo(qnameOffset))
	rdata.Write(make([]byte, 20)) // SERIAL/REFRESH/RETRY/EXPIRE/MINIMUM

	rdlen := make([]byte, 2)
	binary.BigEndian.PutUint16(rdlen, uint16(rdata.Len()))
	buf.Write(rdlen)
	buf.Write(rdata.Bytes())

	return buf.Bytes()
}

func pointerTo(offset int) []byte {
	p := make([]byte, 2)
	binary.BigEndian.PutUint16(p, uint16(offset)|0xC000)
	return p
}

func writeLabel(buf *bytes.Buffer, label string) {
	buf.WriteByte(byte(len(label)))
	buf.WriteString(label)
}

func firstLabel(fqdn string) string {
	for i := 0; i < len(fqdn); i++ {
		if fqdn[i] == '.' {
			return fqdn[:i]
		}
	}
	return fqdn
}

func TestQuerySOA(t *testing.T) {
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer pc.Close()

	errCh := make(chan error, 1)
	go func() {
		buf := make([]byte, 512)
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			errCh <- err
			return
		}
		id := binary.BigEndian.Uint16(buf[0:2])
		resp := buildSOAResponse(t, id, "voorbeeld.nl", "ns1.voorbeeld.nl", "hostmaster")
		_, err = pc.WriteTo(resp, addr)
		errCh <- err
		_ = n
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	mname, rname, err := querySOA(ctx, pc.LocalAddr().String(), "voorbeeld.nl")
	if err != nil {
		t.Fatalf("querySOA: %v", err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("server write: %v", err)
	}
	if mname != "ns1.voorbeeld.nl" {
		t.Errorf("mname = %q, want ns1.voorbeeld.nl", mname)
	}
	if rname != "hostmaster.voorbeeld.nl" {
		t.Errorf("rname = %q, want hostmaster.voorbeeld.nl", rname)
	}
}

func TestQuerySOA_Timeout(t *testing.T) {
	// A listener that never replies stands in for a lame delegation.
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer pc.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, _, err := querySOA(ctx, pc.LocalAddr().String(), "voorbeeld.nl"); err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestEscapeLabel(t *testing.T) {
	cases := map[string]string{
		"hostmaster": "hostmaster",
		"john.doe":   `john\.doe`,
		`back\slash`: `back\\slash`,
	}
	for in, want := range cases {
		if got := escapeLabel(in); got != want {
			t.Errorf("escapeLabel(%q) = %q, want %q", in, got, want)
		}
	}
}
