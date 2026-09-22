package ui

import (
	"html/template"
	"net/url"
	"regexp"
	"strings"
)

// verdictURLPattern finds http(s) URLs embedded in a rule's Verdict
// sentence — the shape scoreStandardsCategory
// (internal/assessor/wand/standards_rules.go) produces for the
// Internet.nl report URL ("... — see https://…, https://…"), joined
// with ", " when a rule spans more than one imported report. Stops
// at whitespace or a comma so a trailing list separator is never
// swallowed into the link.
var verdictURLPattern = regexp.MustCompile(`https?://[^\s,]+`)

// linkifyVerdict turns http(s) URLs inside a rule verdict into
// clickable links, run 04 task 4.1: "de rapport-URL uit het bewijs is
// aanklikbaar (hij is ondoorzichtig — toon hem, bouw hem nooit zelf
// op)". The URL is Internet.nl's own report.url, carried verbatim
// from an imported file (design.md "Design gate outcome" #5) — never
// constructed here, only made clickable. Everything outside a
// validated http(s) match is HTML-escaped as plain text; a match that
// fails to parse as an absolute http(s) URL is left as plain text
// rather than linked, so this never introduces a javascript: or other
// unsafe href from untrusted import data.
func linkifyVerdict(s string) template.HTML {
	var b strings.Builder
	last := 0
	for _, loc := range verdictURLPattern.FindAllStringIndex(s, -1) {
		start, end := loc[0], loc[1]
		raw := s[start:end]
		u, err := url.Parse(raw)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			continue
		}
		b.WriteString(template.HTMLEscapeString(s[last:start]))
		b.WriteString(`<a href="`)
		b.WriteString(template.HTMLEscapeString(raw))
		b.WriteString(`" rel="noopener noreferrer" target="_blank">`)
		b.WriteString(template.HTMLEscapeString(raw))
		b.WriteString(`</a>`)
		last = end
	}
	b.WriteString(template.HTMLEscapeString(s[last:]))
	return template.HTML(b.String())
}
