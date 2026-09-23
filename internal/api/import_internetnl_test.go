package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/internal/api"
	"github.com/MWest2020/wanderer/internal/probe"
	"github.com/MWest2020/wanderer/internal/scanner"
	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

const importFixture = "../scanner/testdata/findings-v1-web-20260922.json"

// importServer wires a router with POST /imports/internetnl active
// under importToken — the shape `wanderer serve` builds once
// WANDERER_IMPORT_TOKEN is set.
func importServer(t *testing.T, importToken string) (*httptest.Server, *store.Store) {
	t.Helper()
	st, err := store.Open(context.Background(), "file::memory:?cache=shared&"+t.Name())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	sc := scanner.New(st, []probe.Probe{stubProbe{}}, probe.Config{})
	sc.Logger = slog.New(slog.NewTextHandler(nopWriter{}, nil))
	srv := httptest.NewServer(api.RouterWithImportToken(st, sc, sc.Logger, nil, importToken))
	t.Cleanup(srv.Close)
	return srv, st
}

func importRequest(t *testing.T, srv *httptest.Server, body []byte, authHeader string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/imports/internetnl", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	return resp
}

func decodeImportResponse(t *testing.T, resp *http.Response) struct {
	Imported       int `json:"imported"`
	SkippedUnknown int `json:"skipped_unknown"`
	SkippedAlready int `json:"skipped_already"`
} {
	t.Helper()
	var body struct {
		Imported       int `json:"imported"`
		SkippedUnknown int `json:"skipped_unknown"`
		SkippedAlready int `json:"skipped_already"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return body
}

// TestImportInternetnl_ValidFileWithTokenIsImported pins spec.md "A
// valid file with the token is imported".
func TestImportInternetnl_ValidFileWithTokenIsImported(t *testing.T) {
	srv, st := importServer(t, "s3cr3t")
	ctx := context.Background()
	target := &models.Target{Domain: "westerweel.work"}
	if err := st.UpsertTarget(ctx, target); err != nil {
		t.Fatalf("upsert target: %v", err)
	}

	data, err := os.ReadFile(importFixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	resp := importRequest(t, srv, data, "Bearer s3cr3t")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body := decodeImportResponse(t, resp)
	if body.Imported != 1 || body.SkippedUnknown != 0 || body.SkippedAlready != 0 {
		t.Fatalf("body = %+v, want imported=1 skipped_unknown=0 skipped_already=0", body)
	}

	scan, err := st.LatestScanByModus(ctx, target.ID, models.SourceModusImport)
	if err != nil {
		t.Fatalf("LatestScanByModus: %v", err)
	}
	if len(scan.Findings) != 38 {
		t.Fatalf("findings = %d, want 38", len(scan.Findings))
	}
}

// TestImportInternetnl_NoOrWrongTokenRefused pins spec.md "No token,
// no import": with a configured token, a missing or wrong
// Authorization header is refused and nothing is stored — checked
// with the fix removed (validImportToken always returning true) to
// confirm the assertion actually fails without it.
func TestImportInternetnl_NoOrWrongTokenRefused(t *testing.T) {
	srv, st := importServer(t, "s3cr3t")
	ctx := context.Background()
	target := &models.Target{Domain: "westerweel.work"}
	if err := st.UpsertTarget(ctx, target); err != nil {
		t.Fatalf("upsert target: %v", err)
	}

	data, err := os.ReadFile(importFixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	respNoAuth := importRequest(t, srv, data, "")
	defer respNoAuth.Body.Close()
	if respNoAuth.StatusCode != http.StatusUnauthorized {
		t.Errorf("no-auth status = %d, want 401", respNoAuth.StatusCode)
	}

	respWrong := importRequest(t, srv, data, "Bearer wrong-token")
	defer respWrong.Body.Close()
	if respWrong.StatusCode != http.StatusUnauthorized {
		t.Errorf("wrong-token status = %d, want 401", respWrong.StatusCode)
	}

	if _, err := st.LatestScanByModus(ctx, target.ID, models.SourceModusImport); err == nil {
		t.Error("want no import-kind scan stored after unauthenticated requests")
	}
}

// TestImportInternetnl_UnconfiguredServerRefusesEverything pins
// spec.md "An unconfigured server refuses everything": with
// WANDERER_IMPORT_TOKEN unset (importToken == ""), every request is
// refused, with or without a token header.
func TestImportInternetnl_UnconfiguredServerRefusesEverything(t *testing.T) {
	srv, st := importServer(t, "")
	ctx := context.Background()
	target := &models.Target{Domain: "westerweel.work"}
	if err := st.UpsertTarget(ctx, target); err != nil {
		t.Fatalf("upsert target: %v", err)
	}

	data, err := os.ReadFile(importFixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	respNoAuth := importRequest(t, srv, data, "")
	defer respNoAuth.Body.Close()
	if respNoAuth.StatusCode != http.StatusUnauthorized {
		t.Errorf("no-auth status = %d, want 401", respNoAuth.StatusCode)
	}

	respWithHeader := importRequest(t, srv, data, "Bearer anything")
	defer respWithHeader.Body.Close()
	if respWithHeader.StatusCode != http.StatusUnauthorized {
		t.Errorf("with-header status = %d, want 401 (no token configured)", respWithHeader.StatusCode)
	}

	if _, err := st.LatestScanByModus(ctx, target.ID, models.SourceModusImport); err == nil {
		t.Error("want no import-kind scan stored when the route is unconfigured")
	}
}

// TestImportInternetnl_SameFileTwiceImportedOnce pins spec.md "The
// same file twice is imported once".
func TestImportInternetnl_SameFileTwiceImportedOnce(t *testing.T) {
	srv, st := importServer(t, "s3cr3t")
	ctx := context.Background()
	target := &models.Target{Domain: "westerweel.work"}
	if err := st.UpsertTarget(ctx, target); err != nil {
		t.Fatalf("upsert target: %v", err)
	}

	data, err := os.ReadFile(importFixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	resp1 := importRequest(t, srv, data, "Bearer s3cr3t")
	defer resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("first status = %d, want 200", resp1.StatusCode)
	}
	body1 := decodeImportResponse(t, resp1)
	if body1.Imported != 1 {
		t.Fatalf("first body = %+v, want imported=1", body1)
	}

	resp2 := importRequest(t, srv, data, "Bearer s3cr3t")
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("second status = %d, want 200", resp2.StatusCode)
	}
	body2 := decodeImportResponse(t, resp2)
	if body2.Imported != 0 || body2.SkippedAlready != 1 {
		t.Fatalf("second body = %+v, want imported=0 skipped_already=1", body2)
	}

	var scanCount int
	row := st.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM scans WHERE target_id = ?`, target.ID)
	if err := row.Scan(&scanCount); err != nil {
		t.Fatalf("count scans: %v", err)
	}
	if scanCount != 1 {
		t.Fatalf("scan count after re-post = %d, want 1 (not stored twice)", scanCount)
	}
}

// TestImportInternetnl_WrongSchemaVersionRefusedWhole pins spec.md "A
// wrong schema version is refused whole".
func TestImportInternetnl_WrongSchemaVersionRefusedWhole(t *testing.T) {
	srv, st := importServer(t, "s3cr3t")
	ctx := context.Background()
	target := &models.Target{Domain: "westerweel.work"}
	if err := st.UpsertTarget(ctx, target); err != nil {
		t.Fatalf("upsert target: %v", err)
	}

	body := []byte(`{"schema":"netnl-findings/v2","domains":[]}`)
	resp := importRequest(t, srv, body, "Bearer s3cr3t")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	var errBody struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&errBody); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if !strings.Contains(errBody.Error.Message, scanner.NetnlSchemaV1) {
		t.Errorf("error message %q does not name the supported version %q", errBody.Error.Message, scanner.NetnlSchemaV1)
	}

	if _, err := st.LatestScanByModus(ctx, target.ID, models.SourceModusImport); err == nil {
		t.Error("want no import-kind scan stored for a rejected schema version")
	}
}
