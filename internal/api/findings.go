package api

import (
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
		if err := st.AppendFindings(r.Context(), scanID, payload.Findings); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, "scan_not_found", err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "store_error", err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"scan_id":  scanID,
			"received": len(payload.Findings),
		})
	})
}
