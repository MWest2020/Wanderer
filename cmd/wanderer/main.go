// Command wanderer is the CLI and API server for the Wanderer digital
// sovereignty monitor. Subcommands:
//
//	wanderer scan <domain>          — run a scan and print a human-readable summary
//	wanderer assess <scan-id>       — score a scan against the DICTU rule set
//	wanderer export <resource>      — export findings/scans/assessments as CSV or JSONL
//	wanderer serve                  — start the HTTP API
//	wanderer version                — print the build version
package main

import (
	"fmt"
	"os"
)

// Version is set at build time via -ldflags.
var Version = "0.0.0-dev"

func main() {
	if len(os.Args) < 2 {
		usage(os.Stderr)
		os.Exit(2)
	}
	cmd := os.Args[1]
	args := os.Args[2:]
	switch cmd {
	case "version", "-v", "--version":
		fmt.Println(Version)
	case "scan":
		os.Exit(runScan(args))
	case "assess":
		os.Exit(runAssess(args))
	case "export":
		os.Exit(runExport(args))
	case "serve":
		os.Exit(runServe(args))
	case "help", "-h", "--help":
		usage(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "wanderer: unknown command %q\n", cmd)
		usage(os.Stderr)
		os.Exit(2)
	}
}

func usage(w *os.File) {
	fmt.Fprintln(w, "usage: wanderer <scan|assess|export|serve|version> [args...]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "  scan <domain>      Run a scan and print the findings")
	fmt.Fprintln(w, "  assess <scan-id>   Score a scan against the DICTU rule set")
	fmt.Fprintln(w, "  export <resource>  Export findings/scans/assessments as CSV or JSONL")
	fmt.Fprintln(w, "  serve              Start the HTTP API")
	fmt.Fprintln(w, "  version            Print the build version")
}
