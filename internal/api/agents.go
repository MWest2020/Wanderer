package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/MWest2020/wanderer/internal/store"
)

// EnrolHandler returns the http.Handler for `POST /agents/enrol`. An
// agent posts the enrolment token an operator created for it plus
// its own hostname, and receives a freshly generated secret — shown
// exactly once, in this response. An unknown, expired, or
// already-used token is refused with the same response, so a caller
// cannot learn which of the three happened.
func EnrolHandler(st *store.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Token    string `json:"token"`
			Hostname string `json:"hostname"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		if body.Token == "" || body.Hostname == "" {
			writeError(w, http.StatusBadRequest, "missing_field", "token and hostname are required")
			return
		}
		secret, ag, err := st.EnrolAgent(r.Context(), body.Token, body.Hostname)
		if err != nil {
			if errors.Is(err, store.ErrEnrolmentFailed) {
				writeError(w, http.StatusUnauthorized, "enrolment_failed", "enrolment token rejected")
				return
			}
			writeError(w, http.StatusInternalServerError, "store_error", err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"hostname":    ag.Hostname,
			"secret":      secret,
			"enrolled_at": ag.EnrolledAt,
		})
	})
}
