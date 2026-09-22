// Command wanderer is the CLI and API server for the Wanderer digital
// sovereignty monitor. Subcommands:
//
//	wanderer scan <domain>          — run a scan and print a human-readable summary
//	wanderer assess <scan-id>       — score a scan against the DICTU rule set
//	wanderer export <resource>      — export findings/scans/assessments as CSV or JSONL
//	wanderer import internetnl <file> — import a netnl-findings/v1 export (Internet.nl results)
//	wanderer diff <scan-a> <scan-b> — print drift between two scans (no persistence)
//	wanderer serve                  — start the HTTP API (with optional cron schedules)
//	wanderer agent                  — run the host-side inventory/egress inspectors
//	wanderer agent intrekken <host> — revoke an enrolled agent
//	wanderer agent list             — list enrolled agents
//	wanderer agent-token nieuw      — issue a short-lived agent enrolment token
//	wanderer mcp                    — speak the Model Context Protocol over stdio
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
	case "import":
		os.Exit(runImport(args))
	case "diff":
		os.Exit(runDiff(args))
	case "agent":
		os.Exit(runAgent(args))
	case "agent-token":
		os.Exit(runAgentToken(args))
	case "org":
		os.Exit(runOrg(args))
	case "mcp":
		os.Exit(runMCP(args))
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
	fmt.Fprintln(w, "usage: wanderer <scan|assess|export|import|diff|serve|agent|agent-token|org|mcp|version> [args...]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "  scan <domain>      Run a scan and print the findings")
	fmt.Fprintln(w, "  assess <scan-id>   Score a scan against the DICTU rule set")
	fmt.Fprintln(w, "  export <resource>  Export findings/scans/assessments as CSV or JSONL")
	fmt.Fprintln(w, "  import internetnl <file>  Import a netnl-findings/v1 export (Internet.nl results)")
	fmt.Fprintln(w, "  diff <a> <b>       Print drift between two stored scans (read-only)")
	fmt.Fprintln(w, "  serve              Start the HTTP API (with optional cron schedules)")
	fmt.Fprintln(w, "  agent              Run the host-side inventory/egress inspectors (or intrekken|list)")
	fmt.Fprintln(w, "  agent-token        Issue a short-lived agent enrolment token (nieuw)")
	fmt.Fprintln(w, "  org                Manage organisations (add|list|show|rename)")
	fmt.Fprintln(w, "  mcp                Speak the Model Context Protocol over stdio")
	fmt.Fprintln(w, "  version            Print the build version")
}
