package ui_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

// seedRunningScan creates a target + a running scan carrying findings
// (AppendFindings — the same write path the scanner uses), so the
// answer page's live assessment reads exactly what a real in-flight
// scan would have written by now.
func seedRunningScan(t *testing.T, st *store.Store, domain string, findings []models.Finding) (scanID string) {
	t.Helper()
	tgt := &models.Target{Domain: domain}
	if err := st.UpsertTarget(context.Background(), tgt); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	sc, err := st.CreateScan(context.Background(), tgt.ID)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(findings) > 0 {
		if err := st.AppendFindings(context.Background(), sc.ID, findings); err != nil {
			t.Fatalf("findings: %v", err)
		}
	}
	return sc.ID
}

func TestAnswerPage_PartialScanRendersPerFlowState(t *testing.T) {
	// spec.md "Slow probe does not hold the page": DNS/hosting landed,
	// the rest of the flows have not — they must read "nog bezig", not
	// "niet gemeten" (that's only for a finished scan).
	srv, st := newServer(t, "")
	scanID := seedRunningScan(t, st, "partial.nl", []models.Finding{
		{ProbeID: "dns.a", Subject: "partial.nl", Severity: models.SeverityFinding},
		{ProbeID: "ip.asn", Subject: "partial.nl", Severity: models.SeverityFinding, Attributes: map[string]any{"country": "NL"}},
	})
	resp, err := http.Get(srv.URL + "/ui/scans/" + scanID + "/answer")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "score-soeverein") {
		t.Errorf("Hosting should already read an answered soeverein verdict; body:\n%s", bodyStr)
	}
	if !strings.Contains(bodyStr, "nog bezig") {
		t.Errorf("a flow the scan has not reached yet must read 'nog bezig'; body:\n%s", bodyStr)
	}
	if strings.Contains(bodyStr, "niet gemeten") {
		t.Errorf("a still-running scan must not call anything 'niet gemeten' yet; body:\n%s", bodyStr)
	}
	if !strings.Contains(bodyStr, `<meta http-equiv="refresh"`) {
		t.Errorf("a running scan's answer page must keep refreshing; body:\n%s", bodyStr)
	}
}

func TestAnswerPage_CompletedScanStopsRefreshing(t *testing.T) {
	srv, st := newServer(t, "")
	scanID := seedRunningScan(t, st, "done.nl", []models.Finding{
		{ProbeID: "dns.a", Subject: "done.nl", Severity: models.SeverityFinding},
		{ProbeID: "ip.asn", Subject: "done.nl", Severity: models.SeverityFinding, Attributes: map[string]any{"country": "NL"}},
	})
	if err := st.FinishScan(context.Background(), scanID, models.ScanStatusComplete, ""); err != nil {
		t.Fatalf("finish: %v", err)
	}
	resp, err := http.Get(srv.URL + "/ui/scans/" + scanID + "/answer")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if strings.Contains(bodyStr, `<meta http-equiv="refresh"`) {
		t.Errorf("a completed scan's answer page must stop refreshing; body:\n%s", bodyStr)
	}
	if !strings.Contains(bodyStr, "niet gemeten") {
		t.Errorf("a completed scan with a flow that never fired must call it 'niet gemeten'; body:\n%s", bodyStr)
	}
}

func TestAnswerPage_NoFindingsSaysJustStarted(t *testing.T) {
	// A scan with zero findings has not failed to answer anything — it
	// just started. It must not read as a definitive "Nee", nor as the
	// generic "onbekend" copy meant for a scan that measured nothing
	// and is done.
	srv, st := newServer(t, "")
	scanID := seedRunningScan(t, st, "empty.nl", nil)
	resp, err := http.Get(srv.URL + "/ui/scans/" + scanID + "/answer")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "net begonnen") {
		t.Errorf("a scan with zero findings must say it just started; body:\n%s", bodyStr)
	}
	if strings.Contains(bodyStr, "Nee —") {
		t.Errorf("a scan with zero findings must not read as a definitive 'Nee'; body:\n%s", bodyStr)
	}
	if !strings.Contains(bodyStr, `<meta http-equiv="refresh"`) {
		t.Errorf("a just-started scan is still running and must keep refreshing; body:\n%s", bodyStr)
	}
}
