package egress

import (
	"context"
	"os"

	"github.com/MWest2020/wanderer/internal/probe/egress/scanners"
	"github.com/MWest2020/wanderer/pkg/models"
)

// HostResolver annotates a Finding with ASN/country information.
// Implementations may wrap the IP probe; nil disables annotation.
type HostResolver interface {
	Resolve(host string) (asn uint, organisation, country string, ok bool)
}

// Probe coordinates the registered scanners, classification,
// redaction, and host-resolution and emits Findings tagged with
// SourceModusEgress.
type Probe struct {
	Scanners []scanners.Scanner
	Resolver HostResolver
}

// Inspect runs every registered scanner and returns the merged
// Finding list, ready for persistence.
func (p Probe) Inspect(ctx context.Context) []models.Finding {
	host := hostname()
	var findings []models.Finding
	resolverOK := p.Resolver != nil

	for _, sc := range p.Scanners {
		ok, reason := sc.Available()
		if !ok {
			findings = append(findings, models.Finding{
				ProbeID:     "egress." + sc.ID() + ".unconfigured",
				SourceModus: models.SourceModusEgress,
				Subject:     host,
				Severity:    models.SeverityInfo,
				Attributes:  map[string]any{"reason": reason},
			})
			continue
		}
		cands, err := sc.Scan(ctx)
		if err != nil {
			findings = append(findings, models.Finding{
				ProbeID:     "egress." + sc.ID() + ".error",
				SourceModus: models.SourceModusEgress,
				Subject:     host,
				Severity:    models.SeverityInfo,
				Attributes:  map[string]any{"error": err.Error()},
			})
			continue
		}
		for _, cand := range cands {
			if f, ok := buildFinding(cand, p.Resolver); ok {
				findings = append(findings, f)
			}
		}
	}
	if !resolverOK && containsAnyEgressFinding(findings) {
		findings = append(findings, models.Finding{
			ProbeID:     "egress.host_resolution.unavailable",
			SourceModus: models.SourceModusEgress,
			Subject:     host,
			Severity:    models.SeverityInfo,
			Attributes:  map[string]any{"reason": "IP probe not configured"},
		})
	}
	return findings
}

// buildFinding turns one Candidate into a Finding by classifying its
// (key, value), redacting any secret material, and resolving the
// host when a resolver is wired. The bool result is false when the
// value is not URL-shaped and the classifier failed to bucket it —
// keeping plain `DEBUG=true` style entries out of the Finding stream.
func buildFinding(cand scanners.Candidate, resolver HostResolver) (models.Finding, bool) {
	cls := Classify(cand.Key, cand.Value)
	if cls.Category == "unknown" && !urlSchemeRE.MatchString(cand.Value) {
		return models.Finding{}, false
	}
	if cls.Host == "" && cls.Category == "unknown" {
		return models.Finding{}, false
	}
	redactedValue, _ := Apply(cand.Key, cand.Value)
	probeID := "egress." + cls.Category
	if cls.Category == "unknown" {
		probeID = "egress.unknown"
	}
	severity := models.SeverityObservation
	if cls.Category == "unknown" {
		severity = models.SeverityInfo
	}
	attrs := map[string]any{
		"config_source":   cand.Source,
		"config_key":      cand.Key,
		"value":           redactedValue,
		"classifier_rule": cls.Rule,
		"confidence":      string(cls.Confidence),
	}
	if cls.Provider != "" {
		attrs["provider"] = cls.Provider
	}
	if cls.Region != "" {
		attrs["region"] = cls.Region
	}
	if cls.Port != "" {
		attrs["port"] = cls.Port
	}
	if resolver != nil && cls.Host != "" {
		if asn, org, country, ok := resolver.Resolve(cls.Host); ok {
			attrs["asn"] = asn
			attrs["organisation"] = org
			attrs["country"] = country
		}
	}
	return models.Finding{
		ProbeID:       probeID,
		SourceModus:   models.SourceModusEgress,
		DimensionHint: cls.Dimension,
		Subject:       cls.Host,
		Severity:      severity,
		Attributes:    attrs,
		Evidence:      []byte(cand.Key + "=" + redactedValue),
	}, true
}

// containsAnyEgressFinding returns true when at least one finding in
// findings is a real egress observation that benefited (or would
// have benefited) from host resolution. The unavailable / error /
// unconfigured / no-host findings do not need ASN annotation, so
// they do not trigger the host_resolution.unavailable companion.
func containsAnyEgressFinding(findings []models.Finding) bool {
	for _, f := range findings {
		switch {
		case f.ProbeID == "":
			continue
		case f.ProbeID == "egress.host_resolution.unavailable":
			continue
		}
		if f.SourceModus == models.SourceModusEgress &&
			!stringHasSuffix(f.ProbeID, ".unconfigured") &&
			!stringHasSuffix(f.ProbeID, ".error") {
			return true
		}
	}
	return false
}

func stringHasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func hostname() string {
	if h, err := os.Hostname(); err == nil {
		return h
	}
	return "unknown"
}
