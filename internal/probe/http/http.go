// Package http is the HTTP probe. It fetches the apex URL of the
// target over HTTPS (falling back to HTTP), parses the HTML body, and
// extracts third-party resources. It also fetches
// /.well-known/security.txt over HTTPS (RFC 9116). Robots.txt is
// honoured for the apex fetch. Redirects are capped at 5 and response
// bodies are capped at 2 MiB (64 KiB for security.txt).
package http

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	wprobe "github.com/MWest2020/wanderer/internal/probe"
	"github.com/MWest2020/wanderer/pkg/models"
	"golang.org/x/net/html"
)

const (
	maxRedirects       = 5
	maxBodyBytes       = 2 << 20
	robotsMax          = 64 << 10
	securityTxtMaxSize = 64 << 10
)

// Probe is the HTTP probe.
type Probe struct {
	// Client overrides the HTTP client. Nil means a default client
	// with a redirect cap and 10s timeout.
	Client *http.Client
}

// New returns an HTTP probe with default settings.
func New() *Probe { return &Probe{} }

// ID implements probe.Probe.
func (*Probe) ID() string { return "http" }

// Run implements probe.Probe.
func (p *Probe) Run(ctx context.Context, target models.Target, cfg wprobe.Config) ([]models.Finding, error) {
	client := p.Client
	if client == nil {
		client = &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				DialContext: (&wprobe.SafeDialer{
					Inner:        &net.Dialer{Timeout: 10 * time.Second},
					AllowPrivate: cfg.AllowPrivateTargets,
				}).DialContext,
			},
			CheckRedirect: func(_ *http.Request, via []*http.Request) error {
				if len(via) >= maxRedirects {
					return fmt.Errorf("too many redirects (> %d)", maxRedirects)
				}
				return nil
			},
		}
	}
	ua := cfg.UserAgent
	if ua == "" {
		ua = "Wanderer/0.x"
	}
	var findings []models.Finding
	base := "https://" + target.Domain + "/"
	// robots.txt first
	if blocked, rob := robotsBlocks(ctx, client, ua, target.Domain, "/"); blocked {
		return []models.Finding{{
			ProbeID:    "http.robots_blocked",
			Subject:    target.Domain,
			Severity:   models.SeverityInfo,
			Attributes: map[string]any{"robots_txt_fetched": true},
			Evidence:   rob,
		}}, nil
	}
	findings = append(findings, fetchSecurityTxt(ctx, client, ua, target.Domain))
	resp, body, err := fetch(ctx, client, ua, base)
	if err != nil {
		// Fall back to http://
		httpURL := "http://" + target.Domain + "/"
		if r2, b2, err2 := fetch(ctx, client, ua, httpURL); err2 == nil {
			resp, body = r2, b2
			findings = append(findings, models.Finding{
				ProbeID:       "http.scheme_downgrade",
				DimensionHint: models.DimensionOperationeel,
				Subject:       target.Domain,
				Severity:      models.SeverityConcern,
				Attributes:    map[string]any{"reason": err.Error()},
			})
		} else {
			return append(findings, models.Finding{
				ProbeID:    "http.fetch_failed",
				Subject:    target.Domain,
				Severity:   models.SeverityConcern,
				Attributes: map[string]any{"error": err.Error()},
			}), nil
		}
	}
	defer resp.Body.Close()

	findings = append(findings, responseFindings(target.Domain, resp)...)
	thirdParties, err := extractThirdParties(target.Domain, resp.Request.URL, body)
	if err != nil {
		findings = append(findings, models.Finding{
			ProbeID:    "http.parse_failed",
			Subject:    target.Domain,
			Severity:   models.SeverityInfo,
			Attributes: map[string]any{"error": err.Error()},
		})
		return findings, nil
	}
	for host, kinds := range thirdParties {
		findings = append(findings, models.Finding{
			ProbeID:       "http.third_party",
			DimensionHint: models.DimensionTechnologie,
			Subject:       host,
			Severity:      models.SeverityObservation,
			Attributes: map[string]any{
				"source_domain": target.Domain,
				"kinds":         kinds,
			},
		})
	}
	return findings, nil
}

func fetch(ctx context.Context, client *http.Client, ua, target string) (*http.Response, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml;q=0.9,*/*;q=0.1")
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes+1))
	if err != nil {
		resp.Body.Close()
		return nil, nil, err
	}
	if len(body) > maxBodyBytes {
		body = body[:maxBodyBytes]
	}
	return resp, body, nil
}

func responseFindings(domain string, resp *http.Response) []models.Finding {
	findings := []models.Finding{{
		ProbeID:       "http.response",
		DimensionHint: models.DimensionNone,
		Subject:       domain,
		Severity:      models.SeverityInfo,
		Attributes: map[string]any{
			"status":     resp.StatusCode,
			"final_url":  resp.Request.URL.String(),
			"scheme":     resp.Request.URL.Scheme,
			"server":     resp.Header.Get("Server"),
			"powered_by": resp.Header.Get("X-Powered-By"),
		},
	}}
	// Security headers worth naming explicitly.
	secHeaders := []string{
		"Strict-Transport-Security",
		"Content-Security-Policy",
		"X-Frame-Options",
		"Referrer-Policy",
		"Permissions-Policy",
	}
	present := map[string]string{}
	for _, h := range secHeaders {
		if v := resp.Header.Get(h); v != "" {
			present[h] = v
		}
	}
	findings = append(findings, models.Finding{
		ProbeID:       "http.security_headers",
		DimensionHint: models.DimensionOperationeel,
		Subject:       domain,
		Severity:      models.SeverityObservation,
		Attributes: map[string]any{
			"present": present,
			"missing": diff(secHeaders, keys(present)),
		},
	})
	return findings
}

func extractThirdParties(apex string, base *url.URL, body []byte) (map[string][]string, error) {
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	out := map[string]map[string]struct{}{}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			var ref, kind string
			switch n.Data {
			case "script":
				ref = attr(n, "src")
				kind = "script"
			case "link":
				ref = attr(n, "href")
				kind = "link"
			case "img":
				ref = attr(n, "src")
				kind = "img"
			case "iframe":
				ref = attr(n, "src")
				kind = "iframe"
			case "source":
				ref = attr(n, "src")
				kind = "source"
			}
			if ref != "" {
				if host := resolveHost(base, ref); host != "" && !sameSite(host, apex) {
					if out[host] == nil {
						out[host] = map[string]struct{}{}
					}
					out[host][kind] = struct{}{}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	result := map[string][]string{}
	for host, kinds := range out {
		ks := make([]string, 0, len(kinds))
		for k := range kinds {
			ks = append(ks, k)
		}
		result[host] = ks
	}
	return result, nil
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func resolveHost(base *url.URL, ref string) string {
	u, err := url.Parse(ref)
	if err != nil {
		return ""
	}
	resolved := base.ResolveReference(u)
	if resolved.Host == "" {
		return ""
	}
	return strings.ToLower(resolved.Hostname())
}

func sameSite(host, apex string) bool {
	if host == apex {
		return true
	}
	return strings.HasSuffix(host, "."+apex)
}

func robotsBlocks(ctx context.Context, client *http.Client, ua, domain, path string) (bool, []byte) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+domain+"/robots.txt", nil)
	if err != nil {
		return false, nil
	}
	req.Header.Set("User-Agent", ua)
	resp, err := client.Do(req)
	if err != nil {
		return false, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, robotsMax))
	if err != nil {
		return false, nil
	}
	return disallows(body, ua, path), body
}

// disallows is a minimalistic robots.txt parser. It understands the
// "User-agent" and "Disallow" directives that matter for the MVP. More
// exotic forms (wildcard, allow, sitemap) are ignored by design —
// Wanderer fetches exactly one URL, so false negatives here would only
// over-scan, not under-scan.
func disallows(body []byte, ua, path string) bool {
	lines := strings.Split(string(body), "\n")
	var current string
	applies := func(agents []string) bool {
		for _, a := range agents {
			if a == "*" || strings.EqualFold(a, ua) {
				return true
			}
		}
		return false
	}
	var agents []string
	var disallow []string
	flush := func() {
		if len(agents) > 0 && applies(agents) {
			for _, d := range disallow {
				if d == "" {
					continue
				}
				if strings.HasPrefix(path, d) {
					current = d
				}
			}
		}
		agents = nil
		disallow = nil
	}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k = strings.ToLower(strings.TrimSpace(k))
		v = strings.TrimSpace(v)
		switch k {
		case "user-agent":
			if len(disallow) > 0 {
				flush()
			}
			agents = append(agents, v)
		case "disallow":
			disallow = append(disallow, v)
		}
	}
	flush()
	return current != ""
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func diff(all, have []string) []string {
	seen := map[string]struct{}{}
	for _, h := range have {
		seen[h] = struct{}{}
	}
	var out []string
	for _, a := range all {
		if _, ok := seen[a]; !ok {
			out = append(out, a)
		}
	}
	return out
}

// fetchSecurityTxt fetches /.well-known/security.txt over HTTPS and
// reports its presence, parseability, and RFC 9116 Contact/Expires
// fields. A 404 (or any non-200 status) is a valid observation —
// present=false, not an error — since RFC 9116 does not require the
// file to exist. Only a transport-level failure (DNS, TLS, connect,
// timeout) is unavailable.
func fetchSecurityTxt(ctx context.Context, client *http.Client, ua, domain string) models.Finding {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+domain+"/.well-known/security.txt", nil)
	if err != nil {
		return securityTxtUnavailable(domain, err.Error())
	}
	req.Header.Set("User-Agent", ua)
	resp, err := client.Do(req)
	if err != nil {
		return securityTxtUnavailable(domain, err.Error())
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, securityTxtMaxSize+1))
	if err != nil {
		return securityTxtUnavailable(domain, err.Error())
	}
	truncated := len(body) > securityTxtMaxSize
	if truncated {
		body = body[:securityTxtMaxSize]
	}

	attrs := map[string]any{"status": resp.StatusCode}
	if resp.StatusCode != http.StatusOK {
		attrs["present"] = false
		attrs["parseable"] = false
		return models.Finding{
			ProbeID:       "http.securitytxt",
			DimensionHint: models.DimensionAccountability,
			Subject:       domain,
			Severity:      models.SeverityObservation,
			Attributes:    attrs,
		}
	}
	attrs["present"] = true
	if truncated {
		attrs["truncated"] = true
	}

	if looksLikeHTML(resp.Header.Get("Content-Type"), body) {
		attrs["parseable"] = false
		return models.Finding{
			ProbeID:       "http.securitytxt",
			DimensionHint: models.DimensionAccountability,
			Subject:       domain,
			Severity:      models.SeverityObservation,
			Attributes:    attrs,
			Evidence:      body,
		}
	}

	fields, parseable := parseSecurityTxt(body)
	attrs["parseable"] = parseable
	if contacts := fields["contact"]; len(contacts) > 0 {
		attrs["contact"] = contacts
	}
	if expires := fields["expires"]; len(expires) > 0 {
		attrs["expires"] = expires[0]
	}
	return models.Finding{
		ProbeID:       "http.securitytxt",
		DimensionHint: models.DimensionAccountability,
		Subject:       domain,
		Severity:      models.SeverityObservation,
		Attributes:    attrs,
		Evidence:      body,
	}
}

// looksLikeHTML reports whether the response looks like an HTML error
// page served at the well-known path rather than a text/plain
// security.txt file.
func looksLikeHTML(contentType string, body []byte) bool {
	if strings.Contains(strings.ToLower(contentType), "text/html") {
		return true
	}
	trimmed := strings.ToLower(strings.TrimSpace(string(body)))
	return strings.HasPrefix(trimmed, "<!doctype") || strings.HasPrefix(trimmed, "<html")
}

// parseSecurityTxt is a minimal RFC 9116 line parser: "Field: value"
// pairs, "#" comments, blank lines ignored. It returns the fields
// (lower-cased keys, values in declaration order) and whether at
// least one field was recognised at all — an unparseable body (empty,
// binary, or otherwise not field-shaped) reports false.
func parseSecurityTxt(body []byte) (map[string][]string, bool) {
	fields := map[string][]string{}
	sawField := false
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		val = strings.TrimSpace(val)
		if key == "" || val == "" {
			continue
		}
		fields[key] = append(fields[key], val)
		sawField = true
	}
	return fields, sawField
}

func securityTxtUnavailable(domain, reason string) models.Finding {
	return models.Finding{
		ProbeID:  "http.securitytxt.unavailable",
		Subject:  domain,
		Severity: models.SeverityInfo,
		Attributes: map[string]any{
			"reason": reason,
		},
	}
}
