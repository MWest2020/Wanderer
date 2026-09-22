package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/MWest2020/wanderer/internal/agent"
	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"

	"github.com/go-chi/chi/v5"
)

// AgentSecrets resolves a per-agent shared secret. Returning nil
// means "no secret registered for this hostname" — the verify path
// converts that to a 401.
type AgentSecrets interface {
	Lookup(hostname string) []byte
}

// StaticAgentSecrets is a tiny map-backed AgentSecrets useful for
// tests and a simple one-host deployment. Production deployments
// should plug a file- or vault-backed implementation behind the
// AgentSecrets interface.
type StaticAgentSecrets struct {
	mu      sync.RWMutex
	secrets map[string][]byte
}

// NewStaticAgentSecrets returns a StaticAgentSecrets seeded from m.
func NewStaticAgentSecrets(m map[string][]byte) *StaticAgentSecrets {
	cp := make(map[string][]byte, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return &StaticAgentSecrets{secrets: cp}
}

// Lookup implements AgentSecrets.
func (s *StaticAgentSecrets) Lookup(hostname string) []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.secrets[hostname]
}

// StoreAgentSecrets resolves an agent's signing key from the agents
// table, so `serve` can wire real enrolment data into
// FindingsIngestHandler instead of the always-refusing nil default.
// An unknown or revoked hostname returns nil, which the verify path
// converts to a 401 — a revoked agent needs no special case anywhere
// else.
//
// EnrolAgent never persists an agent's plain secret, only
// hex(sha256(secret)) (see internal/store/agent.go), so that digest
// is what both sides actually use as the HMAC key: the agent derives
// the identical value once, right after enrolling (see
// internal/agent.EnsureSecret), and uses it from then on.
type StoreAgentSecrets struct {
	st *store.Store
}

// NewStoreAgentSecrets wraps st as an AgentSecrets.
func NewStoreAgentSecrets(st *store.Store) *StoreAgentSecrets {
	return &StoreAgentSecrets{st: st}
}

// Lookup implements AgentSecrets.
func (s *StoreAgentSecrets) Lookup(hostname string) []byte {
	host, err := models.NormaliseHost(hostname)
	if err != nil {
		return nil
	}
	ag, err := s.st.GetAgentByHostname(context.Background(), host)
	if err != nil || ag.Revoked() {
		return nil
	}
	return []byte(ag.SecretHash)
}

// FindingsIngestHandler returns the http.Handler for
// `POST /scans/{id}/findings`. Agents authenticate via HMAC over the
// timestamp + body. When secrets is nil the route is registered but
// every request is rejected — useful for deployments that have not
// yet set up agent ingestion.
func FindingsIngestHandler(st *store.Store, secrets AgentSecrets) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hostname := r.Header.Get(agent.HeaderHostname)
		timestamp := r.Header.Get(agent.HeaderTimestamp)
		signature := r.Header.Get(agent.HeaderSignature)
		batchID := r.Header.Get(agent.HeaderBatchID)
		body, err := io.ReadAll(io.LimitReader(r.Body, 4*1024*1024))
		if err != nil {
			writeError(w, http.StatusBadRequest, "read_body", err.Error())
			return
		}

		var secret []byte
		if secrets != nil && hostname != "" {
			secret = secrets.Lookup(hostname)
		}
		if err := agent.Verify(secret, body, timestamp, signature, time.Now().UTC()); err != nil {
			// Single 401 surface — do not leak which check failed.
			writeError(w, http.StatusUnauthorized, "unauthorized", "agent authentication failed")
			return
		}

		if batchID == "" {
			writeError(w, http.StatusBadRequest, "missing_batch_id", "X-Wanderer-Batch-Id header is required")
			return
		}

		var payload struct {
			Findings []models.Finding `json:"findings"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		scanID := chi.URLParam(r, "id")
		// Tag every incoming finding with inventory modus by default.
		// Egress agents will set their own modus before sending; we
		// preserve any non-empty SourceModus the agent set.
		for i := range payload.Findings {
			if payload.Findings[i].SourceModus == "" {
				payload.Findings[i].SourceModus = models.SourceModusInventory
			}
		}
		received, duplicate, err := st.AppendBatch(r.Context(), scanID, batchID, payload.Findings)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, "scan_not_found", err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "store_error", err.Error())
			return
		}
		if duplicate {
			// Already stored under this batch ID — answer as received
			// without inserting again. 200 rather than 409: from the
			// agent's perspective this retry is not an error, it is
			// exactly the outcome it wanted (the core has the batch).
			writeJSON(w, http.StatusOK, map[string]any{
				"scan_id":   scanID,
				"received":  0,
				"duplicate": true,
			})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"scan_id":  scanID,
			"received": received,
		})
	})
}
