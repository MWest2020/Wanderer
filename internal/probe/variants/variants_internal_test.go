package variants

import (
	"context"
	"errors"
	"net"
	"testing"

	wprobe "github.com/MWest2020/wanderer/internal/probe"
	"github.com/MWest2020/wanderer/pkg/models"
)

// erroringResolver fails every lookup so Run reaches its terminal
// http.variants finding cheaply (every path ends unreachable at
// resolveAndGuard, before any dial), without needing a live listener.
type erroringResolver struct{}

func (erroringResolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) {
	return nil, errors.New("variants internal test: dns disabled")
}

// TestPathCountAndConnectionBudgetDeriveFromConstants covers tasks
// 1.2/1.3: the http.variants finding's path_count and
// connection_budget attributes must equal len(paths) and
// connectionBudget — the same values Run's loop and the walker's
// budget are themselves built from — derived here from those
// identifiers rather than literal 8/24, so a change to the attribute
// that stops tracking the real constant fails this test instead of
// drifting silently.
func TestPathCountAndConnectionBudgetDeriveFromConstants(t *testing.T) {
	p := &Probe{
		Resolver: erroringResolver{},
		HasIPv6:  func() bool { return true },
	}
	findings, err := p.Run(context.Background(), models.Target{Domain: "example.nl"}, wprobe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	var f *models.Finding
	for i := range findings {
		if findings[i].ProbeID == "http.variants" {
			f = &findings[i]
		}
	}
	if f == nil {
		t.Fatalf("no http.variants finding, got %+v", findings)
	}

	gotPathCount, ok := f.Attributes["path_count"].(int)
	if !ok || gotPathCount != len(paths) {
		t.Errorf("path_count = %v (ok=%v), want %d (len(paths))", f.Attributes["path_count"], ok, len(paths))
	}
	gotBudget, ok := f.Attributes["connection_budget"].(int)
	if !ok || gotBudget != connectionBudget {
		t.Errorf("connection_budget = %v (ok=%v), want %d", f.Attributes["connection_budget"], ok, connectionBudget)
	}
}
