package nextcloud_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/MWest2020/wanderer/internal/export/nextcloud"
	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

// davStub records the PUT bodies it receives, keyed by request path,
// and answers MKCOL/PUT the way a real Nextcloud WebDAV endpoint
// does. failPUTs, when >0, makes the first N PUTs fail so the retry
// path can be exercised.
type davStub struct {
	*httptest.Server
	mu          sync.Mutex
	puts        map[string][]byte
	mkcols      []string
	failPUTs    int
	putAttempts int
}

func newDAVStub() *davStub {
	s := &davStub{puts: map[string][]byte{}}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "MKCOL":
			s.mu.Lock()
			s.mkcols = append(s.mkcols, r.URL.Path)
			s.mu.Unlock()
			w.WriteHeader(http.StatusCreated)
		case http.MethodPut:
			s.mu.Lock()
			s.putAttempts++
			if s.failPUTs > 0 {
				s.failPUTs--
				s.mu.Unlock()
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			body, _ := io.ReadAll(r.Body)
			s.puts[r.URL.Path] = body
			s.mu.Unlock()
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	return s
}

func (s *davStub) putBySuffix(suffix string) ([]byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for p, b := range s.puts {
		if strings.HasSuffix(p, suffix) {
			return b, true
		}
	}
	return nil, false
}

func seedScan(t *testing.T, st *store.Store) string {
	t.Helper()
	ctx := context.Background()
	tgt := &models.Target{Domain: "conduction.nl"}
	if err := st.UpsertTarget(ctx, tgt); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	sc, err := st.CreateScan(ctx, tgt.ID)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if err := st.AppendFindings(ctx, sc.ID, []models.Finding{
		{
			ProbeID:  "tls.issuer",
			Subject:  "conduction.nl",
			Severity: models.SeverityFinding,
			Attributes: map[string]any{
				"issuer_country": "NL",
				// A secret-shaped value that the ADR-0008 redaction
				// pass must scrub before the bundle leaves the host.
				"debug_token": "ghp_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
		},
	}); err != nil {
		t.Fatalf("findings: %v", err)
	}
	if err := st.FinishScan(ctx, sc.ID, models.ScanStatusComplete, ""); err != nil {
		t.Fatalf("finish: %v", err)
	}
	return sc.ID
}

func newStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(context.Background(), "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestPublish_DropsJSONLDAndMarkdown(t *testing.T) {
	dav := newDAVStub()
	defer dav.Close()
	st := newStore(t)
	scanID := seedScan(t, st)

	client := nextcloud.NewClient(nextcloud.Config{
		URL:         dav.URL,
		Username:    "wanderer-bot",
		AppPassword: "app-pw",
		TargetDir:   "Wanderer",
	})
	nextcloud.NewPublisher(st, client, nil).Publish(scanID)

	jsonld, ok := dav.putBySuffix(scanID + ".jsonld")
	if !ok {
		t.Fatal("expected a .jsonld PUT")
	}
	if _, ok := dav.putBySuffix(scanID + ".md"); !ok {
		t.Fatal("expected a .md PUT")
	}
	// The bundle filed under the org slug (default org → "default").
	if _, ok := dav.putBySuffix("/Wanderer/default/" + scanID + ".jsonld"); !ok {
		t.Errorf("bundle not filed under /Wanderer/default/; paths=%v", keys(dav))
	}
	// Redaction: the secret-shaped token must be scrubbed.
	if strings.Contains(string(jsonld), "ghp_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa") {
		t.Error("JSON-LD bundle leaked a secret-shaped token — redaction did not run")
	}
	if !strings.Contains(string(jsonld), "conduction.nl") {
		t.Error("JSON-LD bundle missing the scan subject")
	}
}

func TestPublish_RetriesTransientFailure(t *testing.T) {
	dav := newDAVStub()
	defer dav.Close()
	dav.failPUTs = 1 // first PUT 500s, retry must succeed
	st := newStore(t)
	scanID := seedScan(t, st)

	client := nextcloud.NewClient(nextcloud.Config{URL: dav.URL, Username: "bot", AppPassword: "pw", TargetDir: "Wanderer"})
	nextcloud.NewPublisher(st, client, nil).Publish(scanID)

	if _, ok := dav.putBySuffix(scanID + ".jsonld"); !ok {
		t.Fatal("expected the .jsonld PUT to succeed after a retry")
	}
}

func TestPublish_GivesUpAfterBoundedRetries(t *testing.T) {
	dav := newDAVStub()
	defer dav.Close()
	dav.failPUTs = 1000 // every PUT fails
	st := newStore(t)
	scanID := seedScan(t, st)

	client := nextcloud.NewClient(nextcloud.Config{URL: dav.URL, Username: "bot", AppPassword: "pw", TargetDir: "Wanderer"})
	nextcloud.NewPublisher(st, client, nil).Publish(scanID)

	// The first file (.jsonld) is attempted exactly publishAttempts (3)
	// times, then publish gives up — the .md is never attempted.
	dav.mu.Lock()
	attempts := dav.putAttempts
	dav.mu.Unlock()
	if attempts != 3 {
		t.Fatalf("expected exactly 3 PUT attempts (bounded retry), got %d", attempts)
	}
	if _, ok := dav.putBySuffix(scanID + ".jsonld"); ok {
		t.Error("no file should have landed when every PUT fails")
	}
}

func keys(s *davStub) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.puts))
	for k := range s.puts {
		out = append(out, k)
	}
	return out
}
