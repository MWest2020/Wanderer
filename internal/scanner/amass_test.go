package scanner

import (
	"bytes"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
)

// discardWriter is a tiny io.Writer that drops everything. Used in
// tests where we do not need to assert on log output.
type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

func TestParseAmass_HappyPath(t *testing.T) {
	in := strings.NewReader(`{"name":"mail.example.nl","domain":"example.nl"}
{"name":"www.example.nl","domain":"example.nl"}
{"name":"www.example.nl","domain":"example.nl"}
`)
	got, err := parseAmass(in, slog.New(slog.NewTextHandler(discardWriter{}, nil)))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("want 2 dedup'd entries, got %d (%v)", len(got), got)
	}
	if got[0] != "mail.example.nl" || got[1] != "www.example.nl" {
		t.Errorf("sort order broken: %v", got)
	}
}

func TestParseAmass_SkipsMalformedLines(t *testing.T) {
	in := strings.NewReader(`{"name":"mail.example.nl","domain":"example.nl"}
not json at all
{"name":"vpn.example.nl"}
`)
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	got, err := parseAmass(in, logger)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("want 2, got %d (%v)", len(got), got)
	}
	if !strings.Contains(buf.String(), "amass.malformed_line") {
		t.Errorf("malformed line should produce a warn log; got %s", buf.String())
	}
}

func TestParseAmass_DropsBareLabels(t *testing.T) {
	in := strings.NewReader(`{"name":"localhost"}
{"name":"foo.example.nl"}
`)
	got, _ := parseAmass(in, slog.New(slog.NewTextHandler(discardWriter{}, nil)))
	if len(got) != 1 || got[0] != "foo.example.nl" {
		t.Errorf("bare labels should be dropped; got %v", got)
	}
}

func TestLoadAmassFQDNs_MissingFileIsError(t *testing.T) {
	_, err := LoadAmassFQDNs(filepath.Join("does", "not", "exist.json"), nil)
	if err == nil {
		t.Errorf("missing file should error")
	}
}
