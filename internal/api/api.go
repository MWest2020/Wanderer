// Package api exposes Wanderer's HTTP surface: POST /scans,
// GET /scans/{id}, GET /healthz, GET /metrics. The MVP is single-
// tenant and trusted-network; authentication is a separate change.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/MWest2020/wanderer/internal/metrics"
	"github.com/MWest2020/wanderer/internal/scanner"
	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Router builds the HTTP router. It does NOT start the server — that is
// the caller's responsibility.
func Router(st *store.Store, sc *scanner.Scanner, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(slogMiddleware(logger))
	r.Use(middleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Handle("/metrics", promhttp.Handler())

	r.Post("/scans", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Domain  string   `json:"domain"`
			Related []string `json:"related,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		if body.Domain == "" {
			writeError(w, http.StatusBadRequest, "missing_field", "domain is required")
			return
		}
		metrics.ScansStarted.Inc()
		// Enqueue synchronously in MVP — one scan at a time is fine,
		// we have not yet promised a queue.
		ctx, cancel := context.WithTimeout(context.Background(), sc.GlobalBudget+10*time.Second)
		defer cancel()
		result, err := sc.Scan(ctx, models.Target{Domain: body.Domain, Related: body.Related})
		if err != nil {
			writeError(w, http.StatusBadRequest, "scan_failed", err.Error())
			return
		}
		metrics.ScansEnded.WithLabelValues(string(result.Status)).Inc()
		writeJSON(w, http.StatusCreated, result)
	})

	r.Get("/scans/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		scan, err := st.GetScan(r.Context(), id)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, "not_found", "scan not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "store_error", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, scan)
	})

	return r
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// errorResponse is the shape returned for every error.
type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	var body errorResponse
	body.Error.Code = code
	body.Error.Message = message
	writeJSON(w, status, body)
}

// slogMiddleware emits an access log line per request with the
// request-ID propagated to downstream handlers via r.Context().
func slogMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rid := middleware.GetReqID(r.Context())
			rl := logger.With("req_id", rid, "method", r.Method, "path", r.URL.Path)
			ctx := withLogger(r.Context(), rl)
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r.WithContext(ctx))
			rl.Info("http.request",
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"ms", time.Since(start).Milliseconds())
		})
	}
}

type loggerKey struct{}

func withLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, l)
}

// LoggerFrom returns the request-scoped logger, or slog.Default() if
// none was installed.
func LoggerFrom(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}
