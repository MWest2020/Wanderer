//go:build ignore

// geoip-stub builds a deterministic, empty-but-valid mmdb file
// shaped like GeoLite2-ASN. The IP probe's reader opens the file
// happily; every lookup returns "not found" cleanly. Used by the
// test suite to exercise the configured-but-empty path without a
// real MaxMind license. Run via scripts/geoip-stub.sh.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <output-path>\n", os.Args[0])
		os.Exit(2)
	}
	out := os.Args[1]

	tree, err := mmdbwriter.New(mmdbwriter.Options{
		DatabaseType: "GeoLite2-ASN",
		RecordSize:   24,
	})
	if err != nil {
		log.Fatalf("mmdbwriter.New: %v", err)
	}

	// Empty record map — mmdbwriter requires a Map type but does
	// not require any networks to be inserted. The resulting file
	// is the smallest valid GeoLite2-shaped mmdb the IP probe can
	// open.
	_ = mmdbtype.Map{}

	f, err := os.Create(out)
	if err != nil {
		log.Fatalf("create %s: %v", out, err)
	}
	defer f.Close()

	if _, err := tree.WriteTo(f); err != nil {
		log.Fatalf("write tree: %v", err)
	}
}
