package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/MWest2020/wanderer/internal/store"
)

// runAgentToken dispatches `wanderer agent-token <verb>`.
func runAgentToken(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "wanderer agent-token: usage: wanderer agent-token nieuw [flags]")
		return 2
	}
	verb := args[0]
	rest := args[1:]
	switch verb {
	case "nieuw":
		return runAgentTokenNieuw(rest)
	default:
		fmt.Fprintf(os.Stderr, "wanderer agent-token: unknown verb %q\n", verb)
		return 2
	}
}

// runAgentTokenNieuw handles `wanderer agent-token nieuw`. It prints
// the plain token exactly once — the store keeps only its hash, so
// this is the only chance to capture it.
func runAgentTokenNieuw(args []string) int {
	fs := flag.NewFlagSet("agent-token nieuw", flag.ContinueOnError)
	dbPath := fs.String("db", envOr("WANDERER_DB", "wanderer.db"), "Path to SQLite database")
	ttl := fs.Duration("ttl", time.Hour, "How long the token stays valid (e.g. 1h, 30m)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	st, err := openAgentStore(*dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer st.Close()
	plainToken, tok, err := st.CreateEnrolmentToken(context.Background(), *ttl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wanderer agent-token nieuw: %v\n", err)
		return 1
	}
	fmt.Printf("token:      %s\n", plainToken)
	fmt.Printf("expires_at: %s\n", tok.ExpiresAt.UTC().Format(time.RFC3339))
	fmt.Println("This token is shown once and cannot be retrieved again — hand it to the agent now.")
	return 0
}

// runAgentIntrekken handles `wanderer agent intrekken <hostname>`.
func runAgentIntrekken(args []string) int {
	fs := flag.NewFlagSet("agent intrekken", flag.ContinueOnError)
	dbPath := fs.String("db", envOr("WANDERER_DB", "wanderer.db"), "Path to SQLite database")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "wanderer agent intrekken: usage: wanderer agent intrekken <hostname>")
		return 2
	}
	hostname := fs.Arg(0)
	st, err := openAgentStore(*dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer st.Close()
	if err := st.RevokeAgent(context.Background(), hostname); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			fmt.Fprintf(os.Stderr, "wanderer agent intrekken: no agent with hostname %q\n", hostname)
			return 1
		}
		fmt.Fprintf(os.Stderr, "wanderer agent intrekken: %v\n", err)
		return 1
	}
	fmt.Printf("agent %s revoked\n", hostname)
	return 0
}

// runAgentList handles `wanderer agent list`.
func runAgentList(args []string) int {
	fs := flag.NewFlagSet("agent list", flag.ContinueOnError)
	dbPath := fs.String("db", envOr("WANDERER_DB", "wanderer.db"), "Path to SQLite database")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	st, err := openAgentStore(*dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer st.Close()
	agents, err := st.ListAgents(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "wanderer agent list: %v\n", err)
		return 1
	}
	tw := tabwriter.NewWriter(os.Stdout, 2, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "HOSTNAME\tENROLLED_AT\tREVOKED_AT")
	for _, a := range agents {
		revoked := "-"
		if a.RevokedAt != nil {
			revoked = a.RevokedAt.UTC().Format("2006-01-02 15:04 UTC")
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", a.Hostname, a.EnrolledAt.UTC().Format("2006-01-02 15:04 UTC"), revoked)
	}
	tw.Flush()
	return 0
}

func openAgentStore(path string) (*store.Store, error) {
	st, err := store.Open(context.Background(), "file:"+filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("wanderer agent: open store: %w", err)
	}
	return st, nil
}
