// Package metrics exposes Prometheus counters for the scanner and
// probes. Keep this surface small — new counters should be added only
// when an operator will actually watch them.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ProbeRuns counts every probe invocation by probe ID.
	ProbeRuns = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "wanderer",
		Subsystem: "probe",
		Name:      "runs_total",
		Help:      "Number of probe invocations.",
	}, []string{"probe"})

	// ProbeFailures counts probe failures by probe ID and reason
	// (timeout, panic, error).
	ProbeFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "wanderer",
		Subsystem: "probe",
		Name:      "failures_total",
		Help:      "Number of probe failures, labelled by reason.",
	}, []string{"probe", "reason"})

	// ScansStarted counts scan starts.
	ScansStarted = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "wanderer",
		Subsystem: "scan",
		Name:      "started_total",
		Help:      "Number of scans started.",
	})

	// ScansEnded counts scan completions by terminal status.
	ScansEnded = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "wanderer",
		Subsystem: "scan",
		Name:      "ended_total",
		Help:      "Number of scans ended, labelled by terminal status.",
	}, []string{"status"})
)
