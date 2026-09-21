package dns

import (
	"bytes"
	"context"
	"encoding/binary"
	"net"
	"testing"
	"time"
)

// buildCAAResponse assembles a wire-format DNS response to a
// "CAA IN <name>" question with the given id, one answer per record.
// The answer owner name is a compression pointer back to the question
// (the shape real authoritative servers send), so the test exercises
// skipName's pointer handling, not just flat names.
func buildCAAResponse(t *testing.T, id uint16, name string, records []CAA) []byte {
	t.Helper()
	qname, err := encodeName(name)
	if err != nil {
		t.Fatalf("encodeName: %v", err)
	}
	const qnameOffset = 12 // right after the 12-byte header

	var buf bytes.Buffer
	header := make([]byte, 12)
	binary.BigEndian.PutUint16(header[0:2], id)
	binary.BigEndian.PutUint16(header[2:4], 0x8180) // QR=1, RA=1, RCODE=0
	binary.BigEndian.PutUint16(header[4:6], 1)      // QDCOUNT
	binary.BigEndian.PutUint16(header[6:8], uint16(len(records)))
	buf.Write(header)
	buf.Write(qname)
	qtype := make([]byte, 4)
	binary.BigEndian.PutUint16(qtype[0:2], dnsTypeCAA)
	binary.BigEndian.PutUint16(qtype[2:4], dnsClassIN)
	buf.Write(qtype)

	for _, rec := range records {
		buf.Write(pointerTo(qnameOffset))
		rrHead := make([]byte, 8)
		binary.BigEndian.PutUint16(rrHead[0:2], dnsTypeCAA)
		binary.BigEndian.PutUint16(rrHead[2:4], dnsClassIN)
		binary.BigEndian.PutUint32(rrHead[4:8], 0) // TTL
		buf.Write(rrHead)

		var rdata bytes.Buffer
		rdata.WriteByte(rec.Flag)
		rdata.WriteByte(byte(len(rec.Tag)))
		rdata.WriteString(rec.Tag)
		rdata.WriteString(rec.Value)

		rdlen := make([]byte, 2)
		binary.BigEndian.PutUint16(rdlen, uint16(rdata.Len()))
		buf.Write(rdlen)
		buf.Write(rdata.Bytes())
	}

	return buf.Bytes()
}

func pointerTo(offset int) []byte {
	p := make([]byte, 2)
	binary.BigEndian.PutUint16(p, uint16(offset)|0xC000)
	return p
}

// TestQueryCAA drives queryCAA against a local UDP responder that
// returns real CAA records — no real DNS, no network egress. Before
// this change netResolver.LookupCAA unconditionally returned nil, nil;
// a test built the same way against that code would have failed here
// (0 records, not 2), which is exactly the bug the proposal describes.
func TestQueryCAA(t *testing.T) {
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer pc.Close()

	want := []CAA{
		{Flag: 0, Tag: "issue", Value: "certsign.ro"},
		{Flag: 0, Tag: "iodef", Value: "mailto:security@voorbeeld.nl"},
	}

	errCh := make(chan error, 1)
	go func() {
		buf := make([]byte, 512)
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			errCh <- err
			return
		}
		id := binary.BigEndian.Uint16(buf[0:2])
		resp := buildCAAResponse(t, id, "voorbeeld.nl", want)
		_, err = pc.WriteTo(resp, addr)
		errCh <- err
		_ = n
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	got, err := queryCAA(ctx, pc.LocalAddr().String(), "voorbeeld.nl")
	if err != nil {
		t.Fatalf("queryCAA: %v", err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("server write: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("records = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("record[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestQueryCAA_Timeout(t *testing.T) {
	// A listener that never replies stands in for a lame delegation.
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer pc.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := queryCAA(ctx, pc.LocalAddr().String(), "voorbeeld.nl"); err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

// TestDecodeCAAResponse_NXDOMAIN pins the RFC 8659 §3 tree-walk
// requirement at the wire-decoding layer: an NXDOMAIN on an
// intermediate name is "nothing here", not an error, so the probe's
// climb keeps going instead of aborting.
func TestDecodeCAAResponse_NXDOMAIN(t *testing.T) {
	id := uint16(4242)
	header := make([]byte, 12)
	binary.BigEndian.PutUint16(header[0:2], id)
	binary.BigEndian.PutUint16(header[2:4], 0x8183) // QR=1, RA=1, RCODE=3 (NXDOMAIN)

	got, err := decodeCAAResponse(header, id)
	if err != nil {
		t.Fatalf("decodeCAAResponse: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("records = %d, want 0", len(got))
	}
}
