package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/MWest2020/wanderer/internal/api"
	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

// TestImportInternetnl_CrossDoorIdempotency covers habitat run 01
// decision 2: the CLI hashes the file bytes it reads; the HTTP route
// must hash the request body the same way, so the same bytes are
// "already imported" whether they arrived by CLI or by
// POST /imports/internetnl. Importing by CLI first and then posting
// the identical bytes must report the second door as already
// imported, not import a second time.
func TestImportInternetnl_CrossDoorIdempotency(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "wanderer.db")
	ctx := context.Background()
	fixture := filepath.Join("..", "..", "internal", "scanner", "testdata", "findings-v1-web-20260922.json")

	st, err := store.Open(ctx, "file:"+dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	target := &models.Target{Domain: "westerweel.work"}
	if err := st.UpsertTarget(ctx, target); err != nil {
		t.Fatalf("upsert target: %v", err)
	}
	st.Close()

	if rc := runImportInternetnl([]string{"--db", dbPath, fixture}); rc != 0 {
		t.Fatalf("CLI import exit = %d, want 0", rc)
	}

	st, err = store.Open(ctx, "file:"+dbPath)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer st.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := httptest.NewServer(api.RouterWithImportToken(st, nil, logger, nil, "s3cr3t"))
	defer srv.Close()

	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/imports/internetnl", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer s3cr3t")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body struct {
		Imported       int `json:"imported"`
		SkippedUnknown int `json:"skipped_unknown"`
		SkippedAlready int `json:"skipped_already"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Imported != 0 || body.SkippedAlready != 1 {
		t.Fatalf("body = %+v, want imported=0 skipped_already=1 (same bytes already imported by the CLI)", body)
	}

	var scanCount int
	row := st.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM scans WHERE target_id = ?`, target.ID)
	if err := row.Scan(&scanCount); err != nil {
		t.Fatalf("count scans: %v", err)
	}
	if scanCount != 1 {
		t.Fatalf("scan count after HTTP re-post = %d, want 1 (the CLI's import, not duplicated)", scanCount)
	}
}
