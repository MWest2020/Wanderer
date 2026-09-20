package api_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/MWest2020/wanderer/internal/agent"
	"github.com/MWest2020/wanderer/internal/api"
	"github.com/MWest2020/wanderer/internal/probe"
	"github.com/MWest2020/wanderer/internal/scanner"
	"github.com/MWest2020/wanderer/internal/store"
)

// storeSecretsServer wires a router backed by StoreAgentSecrets — the
// same construction `wanderer serve` uses — instead of the
// StaticAgentSecrets fixture the rest of this package's tests use.
func storeSecretsServer(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	st, err := store.Open(context.Background(), "file::memory:?cache=shared&"+t.Name())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	sc := scanner.New(st, []probe.Probe{stubProbe{}}, probe.Config{})
	sc.Logger = slog.New(slog.NewTextHandler(nopWriter{}, nil))
	srv := httptest.NewServer(api.RouterWithSecrets(st, sc, sc.Logger, api.NewStoreAgentSecrets(st)))
	t.Cleanup(srv.Close)
	return srv, st
}

// keyFor mirrors internal/agent.EnsureSecret's key derivation: the
// core never stores an agent's plain secret, only hex(sha256(secret)),
// so that digest is the actual shared HMAC key both sides use.
func keyFor(plainSecret string) []byte {
	sum := sha256.Sum256([]byte(plainSecret))
	return []byte(hex.EncodeToString(sum[:]))
}

// TestStoreAgentSecrets_EnrolledAgentDelivers exercises the full path
// end to end: an operator-issued token, the real `agent.EnsureSecret`
// exchange (as `wanderer agent` would run it), and a signed findings
// POST that must land on the scan — the "Enrolled agent delivers"
// scenario in specs/scanner/spec.md.
func TestStoreAgentSecrets_EnrolledAgentDelivers(t *testing.T) {
	srv, st := storeSecretsServer(t)
	ctx := context.Background()

	plainToken, _, err := st.CreateEnrolmentToken(ctx, time.Hour)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	secretPath := filepath.Join(t.TempDir(), "secret")
	key, err := agent.EnsureSecret(ctx, srv.Client(), srv.URL, "webapp-01", plainToken, secretPath)
	if err != nil {
		t.Fatalf("EnsureSecret: %v", err)
	}

	scanID := seedScan(t, st)
	body := []byte(`{"findings":[{"probe_id":"inventory.systemd.service","subject":"sshd.service","severity":"info","attributes":{"active_state":"active"}}]}`)
	ts, sig := agent.Sign(key, body, time.Now().UTC())
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/scans/"+scanID+"/findings", bytes.NewReader(body))
	req.Header.Set(agent.HeaderHostname, "webapp-01")
	req.Header.Set(agent.HeaderTimestamp, ts)
	req.Header.Set(agent.HeaderSignature, sig)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}

	scan, err := st.GetScan(ctx, scanID)
	if err != nil {
		t.Fatalf("get scan: %v", err)
	}
	if len(scan.Findings) != 1 {
		t.Fatalf("want 1 persisted finding, got %d", len(scan.Findings))
	}
}

// TestStoreAgentSecrets_RevokedAgentRefusedOtherUnaffected pins the
// "One agent revoked" scenario: revoking one enrolled agent refuses
// its findings while another enrolled agent keeps working.
func TestStoreAgentSecrets_RevokedAgentRefusedOtherUnaffected(t *testing.T) {
	srv, st := storeSecretsServer(t)
	ctx := context.Background()

	tokA, _, err := st.CreateEnrolmentToken(ctx, time.Hour)
	if err != nil {
		t.Fatalf("create token a: %v", err)
	}
	secretA, _, err := st.EnrolAgent(ctx, tokA, "webapp-a")
	if err != nil {
		t.Fatalf("enrol a: %v", err)
	}
	tokB, _, err := st.CreateEnrolmentToken(ctx, time.Hour)
	if err != nil {
		t.Fatalf("create token b: %v", err)
	}
	secretB, _, err := st.EnrolAgent(ctx, tokB, "webapp-b")
	if err != nil {
		t.Fatalf("enrol b: %v", err)
	}
	if err := st.RevokeAgent(ctx, "webapp-a"); err != nil {
		t.Fatalf("revoke a: %v", err)
	}

	scanID := seedScan(t, st)
	body := []byte(`{"findings":[]}`)

	postAs := func(hostname string, secret string) *http.Response {
		ts, sig := agent.Sign(keyFor(secret), body, time.Now().UTC())
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/scans/"+scanID+"/findings", bytes.NewReader(body))
		req.Header.Set(agent.HeaderHostname, hostname)
		req.Header.Set(agent.HeaderTimestamp, ts)
		req.Header.Set(agent.HeaderSignature, sig)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("post as %s: %v", hostname, err)
		}
		return resp
	}

	respA := postAs("webapp-a", secretA)
	defer respA.Body.Close()
	if respA.StatusCode != http.StatusUnauthorized {
		t.Errorf("revoked agent status = %d, want 401", respA.StatusCode)
	}

	respB := postAs("webapp-b", secretB)
	defer respB.Body.Close()
	if respB.StatusCode != http.StatusCreated {
		t.Errorf("other agent status = %d, want 201", respB.StatusCode)
	}

	respUnknown := postAs("never-enrolled", "whatever")
	defer respUnknown.Body.Close()
	if respUnknown.StatusCode != http.StatusUnauthorized {
		t.Errorf("unknown agent status = %d, want 401", respUnknown.StatusCode)
	}
}

// TestStoreAgentSecrets_NoAgentsEnrolled_RouteStaysClosed pins the
// "No agents enrolled" scenario: with an empty agents table, every
// request to the findings route is refused, matching pre-enrolment
// behaviour exactly.
func TestStoreAgentSecrets_NoAgentsEnrolled_RouteStaysClosed(t *testing.T) {
	srv, st := storeSecretsServer(t)
	scanID := seedScan(t, st)
	body := []byte(`{"findings":[]}`)
	ts, sig := agent.Sign([]byte("anything"), body, time.Now().UTC())
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/scans/"+scanID+"/findings", bytes.NewReader(body))
	req.Header.Set(agent.HeaderHostname, "webapp-01")
	req.Header.Set(agent.HeaderTimestamp, ts)
	req.Header.Set(agent.HeaderSignature, sig)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}
