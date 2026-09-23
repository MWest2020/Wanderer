package api

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/MWest2020/wanderer/internal/scanner"
	"github.com/MWest2020/wanderer/internal/store"
)

// maxImportBodyBytes caps the request body POST /imports/internetnl
// will read. A netnl-findings/v1 file for a fleet of ~40 domains is
// well under 1 MiB; this leaves headroom without leaving an
// unauthenticated-reachable port an unbounded memory lever.
const maxImportBodyBytes = 4 * 1024 * 1024

// ImportInternetnlHandler returns the http.Handler for
// `POST /imports/internetnl`. The body is a netnl-findings/v1
// document; it is imported through the exact same write path as
// `wanderer import internetnl` (scanner.ImportNetnlDomains) — the
// server is the only writer of its SQLite database, so a CronJob or
// any other second process must post here instead of opening the
// file itself.
//
// token is the value of WANDERER_IMPORT_TOKEN. When empty, the route
// stays registered but refuses every request — the same shape as
// FindingsIngestHandler with a nil AgentSecrets. An import writes
// verdicts that appear in reports as evidence, and the REST API is
// reachable on the tailnet without authentication, so this route may
// not be.
func ImportInternetnlHandler(st *store.Store, logger *slog.Logger, token string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !validImportToken(token, r.Header.Get("Authorization")) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "import token required")
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, maxImportBodyBytes))
		if err != nil {
			writeError(w, http.StatusBadRequest, "read_body", err.Error())
			return
		}
		sum := sha256.Sum256(body)
		fileHash := hex.EncodeToString(sum[:])

		file, err := scanner.ParseNetnlFindings(bytes.NewReader(body), logger)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_file", err.Error())
			return
		}

		imported, skippedUnknown, skippedAlready, err := scanner.ImportNetnlDomains(r.Context(), st, logger, fileHash, file.Domains)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "store_error", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"imported":        imported,
			"skipped_unknown": skippedUnknown,
			"skipped_already": skippedAlready,
		})
	})
}

// validImportToken reports whether header carries "Bearer <token>"
// matching configured, in constant time. An empty configured token
// (WANDERER_IMPORT_TOKEN unset) never matches — the route must fail
// closed, not fall back to "no auth required".
func validImportToken(configured, header string) bool {
	if configured == "" {
		return false
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	supplied := strings.TrimPrefix(header, prefix)
	return subtle.ConstantTimeCompare([]byte(supplied), []byte(configured)) == 1
}
