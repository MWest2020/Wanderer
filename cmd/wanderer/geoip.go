package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

// registerGeoIPFlags adds --geoip, --geoip-country, and --no-geoip to
// the given FlagSet, defaulting to the WANDERER_GEOIP_ASN and
// WANDERER_GEOIP_COUNTRY env vars respectively. The returned
// pointers are wired into the rest of the command. Helper exists so
// scan.go and serve.go register identical flags without copy-paste.
func registerGeoIPFlags(fs *flag.FlagSet) (asn, country *string, noGeoIP *bool) {
	asn = fs.String("geoip", envOr("WANDERER_GEOIP_ASN", ""), "Path to GeoLite2-ASN mmdb (ASN + country)")
	country = fs.String("geoip-country", envOr("WANDERER_GEOIP_COUNTRY", ""), "Optional GeoLite2-Country mmdb (defaults to --geoip)")
	noGeoIP = fs.Bool("no-geoip", false, "Silence the startup warning when GeoLite2 is intentionally absent (CI, offline labs)")
	return
}

// warnIfGeoIPMissing emits one stderr line when the operator has not
// configured GeoLite2 and has not opted out via --no-geoip /
// WANDERER_GEOIP_OPTIONAL=1. It returns nothing — the warning is
// fire-and-forget; the IP probe's existing ip.unavailable Finding
// path handles the runtime degradation. `out` defaults to os.Stderr;
// tests inject a buffer.
func warnIfGeoIPMissing(out io.Writer, asnPath string, noGeoIP bool) {
	if out == nil {
		out = os.Stderr
	}
	if noGeoIP || os.Getenv("WANDERER_GEOIP_OPTIONAL") == "1" {
		return
	}
	if asnPath != "" {
		return
	}
	fmt.Fprintln(out, "warning: GeoLite2 ASN database is not configured — scan will continue with reduced assessment coverage. Pass --geoip <path> (or set WANDERER_GEOIP_ASN), or pass --no-geoip to silence this warning. See docs/operator.md for setup.")
}
