package ui_test

// Covers openspec change 2026-09-26-stylesheet-versie: the stylesheet
// link carries a version derived from main.css's content, so a
// release that changes the file gets a new address and no cache
// (browser or edge) can serve the stale one back.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/internal/ui"
	"github.com/MWest2020/wanderer/pkg/models"

	"golang.org/x/crypto/bcrypt"
)

// wantStylesheetHref computes the expected `<link>` href straight from
// the on-disk static/main.css — the same file go:embed bakes into the
// binary — so the test tracks the file's actual content instead of a
// hard-coded hash that would silently go stale.
func wantStylesheetHref(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("static/main.css")
	if err != nil {
		t.Fatalf("read static/main.css: %v", err)
	}
	sum := sha256.Sum256(data)
	return `href="/ui/static/main.css?v=` + hex.EncodeToString(sum[:])[:12] + `"`
}

// TestStylesheet_LinksCarryContentHash renders every template that
// links main.css and asserts the link carries the hash of the current
// embedded file — pins proposal.md's fix for the Cloudflare HIT that
// kept serving v0.11's stylesheet 4046 seconds into v0.12.
func TestStylesheet_LinksCarryContentHash(t *testing.T) {
	want := wantStylesheetHref(t)

	dir := t.TempDir()
	htpasswd := filepath.Join(dir, "passwd")
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct horse battery staple"), bcrypt.MinCost)
	if err := os.WriteFile(htpasswd, []byte("op:"+string(hash)+"\n"), 0o600); err != nil {
		t.Fatalf("write htpasswd: %v", err)
	}
	st := newTestStore(t)
	h, err := ui.Handler(st, ui.Options{HtpasswdPath: htpasswd, Scanner: stubScanner{st: st}})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/ui/", http.StripPrefix("/ui", h))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	client := basicAuthClient("op", "correct horse battery staple")

	targetID, scanID := seed(t, st)
	if err := st.FinishScan(context.Background(), scanID, models.ScanStatusComplete, ""); err != nil {
		t.Fatalf("finish scan: %v", err)
	}
	seedAssessment(t, st, scanID, "wand")

	o := &models.Organisation{Slug: "stylesheet-test", Name: "Stylesheet Test"}
	if err := st.UpsertOrganisation(context.Background(), o); err != nil {
		t.Fatal(err)
	}

	paths := map[string]string{
		"dashboard.tmpl":      "/ui/",
		"index.tmpl":          "/ui/targets",
		"scan.tmpl":           "/ui/scans/" + scanID,
		"answer.tmpl":         "/ui/scans/" + scanID + "/answer",
		"assessment.tmpl":     "/ui/scans/" + scanID + "/assessment",
		"drift.tmpl":          "/ui/targets/" + targetID + "/drift",
		"trends.tmpl":         "/ui/trends",
		"fleet.tmpl":          "/ui/orgs/stylesheet-test/fleet",
		"scan-status.tmpl":    "/ui/scan-status?domain=nooitgescand.nl",
		"reporting_rule.tmpl": "/ui/reporting/wand/wand.juridisch.cert_issuer_eea",
	}

	for name, path := range paths {
		resp, err := client.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("%s: get %s: %v", name, path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if !strings.Contains(string(body), want) {
			t.Errorf("%s (%s): expected stylesheet link %s; body:\n%s", name, path, want, body)
		}
	}
}

// TestStylesheet_NoTemplateLinksBareCSSPath scans every template for a
// literal, unversioned `href="/ui/static/main.css"` so a future
// template cannot reintroduce the address a cache can never learn has
// changed.
func TestStylesheet_NoTemplateLinksBareCSSPath(t *testing.T) {
	matches, err := filepath.Glob("templates/*.tmpl")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("no templates found — glob pattern is wrong")
	}
	bare := regexp.MustCompile(`href="/ui/static/main\.css"`)
	for _, m := range matches {
		src, err := os.ReadFile(m)
		if err != nil {
			t.Fatalf("read %s: %v", m, err)
		}
		if bare.Match(src) {
			t.Errorf("%s: links main.css without a content version", m)
		}
	}
}
