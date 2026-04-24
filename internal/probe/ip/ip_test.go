package ip_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	ipprobe "github.com/MWest2020/wanderer/internal/probe/ip"
)

func TestOpenMissingDB(t *testing.T) {
	_, err := ipprobe.Open("/does/not/exist.mmdb", "")
	if err == nil {
		t.Fatal("expected error for missing DB")
	}
	if !strings.Contains(err.Error(), "asn DB") {
		t.Errorf("error = %v, want mention of asn DB", err)
	}
}

func TestOpenCorruptDB(t *testing.T) {
	f := filepath.Join(t.TempDir(), "bad.mmdb")
	if err := os.WriteFile(f, []byte("not a real mmdb"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ipprobe.Open(f, "")
	if err == nil {
		t.Fatal("expected error for corrupt DB")
	}
}

func TestOpenEmptyPath(t *testing.T) {
	if _, err := ipprobe.Open("", ""); err == nil {
		t.Fatal("expected error for empty path")
	}
}
