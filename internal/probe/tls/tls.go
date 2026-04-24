// Package tls is the TLS probe. It dials the target on :443, inspects
// the certificate chain presented during the handshake, records issuer
// identity, SANs, validity, and then attempts a Certificate Transparency
// lookup via crt.sh. The CT lookup is best-effort: rate-limits or
// outages never fail the probe.
package tls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/MWest2020/wanderer/internal/probe"
	"github.com/MWest2020/wanderer/pkg/models"
)

// Probe is the TLS probe.
type Probe struct {
	// Dialer lets tests swap in an in-memory tls listener. Nil means
	// use a default net.Dialer + tls.Client.
	Dialer func(ctx context.Context, addr string, cfg *tls.Config) (*tls.ConnectionState, error)

	// CrtShURL overrides the crt.sh endpoint used for CT lookups; if
	// empty, the default public endpoint is used.
	CrtShURL string
}

// New returns a TLS probe with default networking.
func New() *Probe { return &Probe{} }

// ID implements probe.Probe.
func (*Probe) ID() string { return "tls" }

// Run implements probe.Probe.
func (p *Probe) Run(ctx context.Context, target models.Target, cfg probe.Config) ([]models.Finding, error) {
	if target.Domain == "" {
		return nil, errors.New("tls: empty domain")
	}
	findings := p.inspect(ctx, target.Domain)
	findings = append(findings, p.crtShLookup(ctx, target.Domain, cfg)...)
	return findings, nil
}

func (p *Probe) inspect(ctx context.Context, domain string) []models.Finding {
	addr := net.JoinHostPort(domain, "443")
	cfg := &tls.Config{
		ServerName:         domain,
		InsecureSkipVerify: false, //nolint:gosec // we report verification failures as findings
	}
	state, err := p.dial(ctx, addr, cfg)
	if err != nil {
		// Retry with verification off so we can still inspect a
		// misissued / expired cert and record it as a Finding.
		insecureCfg := cfg.Clone()
		insecureCfg.InsecureSkipVerify = true
		state, err = p.dial(ctx, addr, insecureCfg)
		if err != nil {
			return []models.Finding{{
				ProbeID:       "tls.handshake",
				DimensionHint: models.DimensionOperationeel,
				Subject:       domain,
				Severity:      models.SeverityConcern,
				Attributes: map[string]any{
					"error": err.Error(),
					"kind":  classifyTLSErr(err),
				},
			}}
		}
		// Handshake succeeded only with verification off: record
		// verification failure as a Finding.
		return append(verificationFailure(domain, state), inspectState(domain, state)...)
	}
	return inspectState(domain, state)
}

func (p *Probe) dial(ctx context.Context, addr string, cfg *tls.Config) (*tls.ConnectionState, error) {
	if p.Dialer != nil {
		return p.Dialer(ctx, addr, cfg)
	}
	d := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := tls.DialWithDialer(d, "tcp", addr, cfg)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	// Block on handshake completion with context.
	if ctx != nil {
		deadline, ok := ctx.Deadline()
		if ok {
			_ = conn.SetDeadline(deadline)
		}
	}
	if err := conn.HandshakeContext(ctx); err != nil {
		return nil, err
	}
	state := conn.ConnectionState()
	return &state, nil
}

func inspectState(domain string, state *tls.ConnectionState) []models.Finding {
	var findings []models.Finding
	if len(state.PeerCertificates) == 0 {
		return []models.Finding{{
			ProbeID:       "tls.handshake",
			DimensionHint: models.DimensionOperationeel,
			Subject:       domain,
			Severity:      models.SeverityConcern,
			Attributes:    map[string]any{"error": "no peer certificates"},
		}}
	}
	leaf := state.PeerCertificates[0]
	findings = append(findings, models.Finding{
		ProbeID:       "tls.issuer",
		DimensionHint: models.DimensionJuridisch,
		Subject:       domain,
		Severity:      models.SeverityFinding,
		Attributes: map[string]any{
			"issuer_cn":       leaf.Issuer.CommonName,
			"issuer_o":        leaf.Issuer.Organization,
			"issuer_country":  leaf.Issuer.Country,
			"subject_cn":      leaf.Subject.CommonName,
			"serial":          leaf.SerialNumber.String(),
			"not_before":      leaf.NotBefore.UTC(),
			"not_after":       leaf.NotAfter.UTC(),
			"signature_algo":  leaf.SignatureAlgorithm.String(),
			"public_key_algo": leaf.PublicKeyAlgorithm.String(),
		},
		Evidence: pemEncode(leaf),
	})
	// SANs
	findings = append(findings, models.Finding{
		ProbeID:       "tls.san",
		DimensionHint: models.DimensionDataAI,
		Subject:       domain,
		Severity:      models.SeverityObservation,
		Attributes: map[string]any{
			"dns_names":    leaf.DNSNames,
			"ip_addresses": ipStrings(leaf.IPAddresses),
		},
	})
	// Validity
	sev := models.SeverityInfo
	attrs := map[string]any{
		"not_before": leaf.NotBefore.UTC(),
		"not_after":  leaf.NotAfter.UTC(),
		"days_left":  int(time.Until(leaf.NotAfter).Hours() / 24),
	}
	if time.Now().After(leaf.NotAfter) {
		sev = models.SeverityConcern
		attrs["expired"] = true
	} else if time.Until(leaf.NotAfter) < 30*24*time.Hour {
		sev = models.SeverityObservation
		attrs["expiring_soon"] = true
	}
	findings = append(findings, models.Finding{
		ProbeID:       "tls.validity",
		DimensionHint: models.DimensionOperationeel,
		Subject:       domain,
		Severity:      sev,
		Attributes:    attrs,
	})
	// Chain length + intermediates
	if len(state.PeerCertificates) > 1 {
		intermediates := make([]string, 0, len(state.PeerCertificates)-1)
		for _, c := range state.PeerCertificates[1:] {
			intermediates = append(intermediates, c.Subject.CommonName)
		}
		findings = append(findings, models.Finding{
			ProbeID:       "tls.chain",
			DimensionHint: models.DimensionOperationeel,
			Subject:       domain,
			Severity:      models.SeverityInfo,
			Attributes: map[string]any{
				"length":        len(state.PeerCertificates),
				"intermediates": intermediates,
			},
		})
	}
	return findings
}

func verificationFailure(domain string, state *tls.ConnectionState) []models.Finding {
	attrs := map[string]any{"verified": false}
	if len(state.PeerCertificates) > 0 {
		attrs["subject_cn"] = state.PeerCertificates[0].Subject.CommonName
	}
	return []models.Finding{{
		ProbeID:       "tls.verify",
		DimensionHint: models.DimensionOperationeel,
		Subject:       domain,
		Severity:      models.SeverityConcern,
		Attributes:    attrs,
	}}
}

func pemEncode(c *x509.Certificate) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: c.Raw})
}

func ipStrings(ips []net.IP) []string {
	out := make([]string, 0, len(ips))
	for _, ip := range ips {
		out = append(out, ip.String())
	}
	return out
}

func classifyTLSErr(err error) string {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "timeout"
	}
	return "other"
}

// crtShLookup queries crt.sh for certificates issued for this domain.
// Any error degrades to a single "unavailable" Finding; the probe never
// fails because of this.
func (p *Probe) crtShLookup(ctx context.Context, domain string, cfg probe.Config) []models.Finding {
	endpoint := p.CrtShURL
	if endpoint == "" {
		endpoint = "https://crt.sh/"
	}
	q := url.Values{}
	q.Set("q", domain)
	q.Set("output", "json")
	u := endpoint + "?" + q.Encode()
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return []models.Finding{ctUnavailable(domain, err)}
	}
	req.Header.Set("User-Agent", userAgent(cfg))
	resp, err := client.Do(req)
	if err != nil {
		return []models.Finding{ctUnavailable(domain, err)}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return []models.Finding{ctUnavailable(domain, fmt.Errorf("crt.sh HTTP %d", resp.StatusCode))}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return []models.Finding{ctUnavailable(domain, err)}
	}
	var entries []struct {
		IssuerName   string `json:"issuer_name"`
		NotBefore    string `json:"not_before"`
		NotAfter     string `json:"not_after"`
		NameValue    string `json:"name_value"`
		SerialNumber string `json:"serial_number"`
	}
	if err := json.Unmarshal(body, &entries); err != nil {
		return []models.Finding{ctUnavailable(domain, err)}
	}
	issuers := map[string]int{}
	for _, e := range entries {
		issuers[e.IssuerName]++
	}
	return []models.Finding{{
		ProbeID:       "tls.ct",
		DimensionHint: models.DimensionJuridisch,
		Subject:       domain,
		Severity:      models.SeverityObservation,
		Attributes: map[string]any{
			"total_entries": len(entries),
			"issuer_counts": issuers,
		},
	}}
}

func ctUnavailable(domain string, err error) models.Finding {
	return models.Finding{
		ProbeID:       "tls.ct",
		DimensionHint: models.DimensionNone,
		Subject:       domain,
		Severity:      models.SeverityInfo,
		Attributes: map[string]any{
			"unavailable": true,
			"error":       err.Error(),
		},
	}
}

func userAgent(cfg probe.Config) string {
	if cfg.UserAgent != "" {
		return cfg.UserAgent
	}
	return "Wanderer/0.x"
}
