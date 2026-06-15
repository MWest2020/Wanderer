package transit

import (
	"bufio"
	"context"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// ToolTracer shells out to an unprivileged path-tracing tool. It
// prefers `tracepath` (uses IP_RECVERR — no root, the tool in the
// motivating example) and falls back to `traceroute`. The output of
// both is parsed by the same tolerant line parser.
type ToolTracer struct {
	// path/args resolved once at construction; empty path = unavailable.
	bin  string
	mode string // "tracepath" | "traceroute"
}

// NewToolTracer picks the first available tracing tool on PATH.
func NewToolTracer() *ToolTracer {
	if p, err := exec.LookPath("tracepath"); err == nil {
		return &ToolTracer{bin: p, mode: "tracepath"}
	}
	if p, err := exec.LookPath("traceroute"); err == nil {
		return &ToolTracer{bin: p, mode: "traceroute"}
	}
	return &ToolTracer{}
}

// Available implements Tracer.
func (t *ToolTracer) Available() bool { return t.bin != "" }

// Trace implements Tracer by running the tool numerically (-n) and
// parsing its hops.
func (t *ToolTracer) Trace(ctx context.Context, ip string, maxHops int) ([]Hop, error) {
	var args []string
	switch t.mode {
	case "tracepath":
		args = []string{"-n", "-m", strconv.Itoa(maxHops), ip}
	default: // traceroute: numeric, one query per hop, 1s wait
		args = []string{"-n", "-q", "1", "-w", "1", "-m", strconv.Itoa(maxHops), ip}
	}
	// No shell: argv is passed directly. t.bin comes from exec.LookPath
	// and the IP arg is validated by net.ParseIP before we get here.
	cmd := exec.CommandContext(ctx, t.bin, args...) //nolint:gosec // G204: not shell-interpreted; bin + IP arg are trusted/validated
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	// Stream lines so a per-probe-timeout kill still leaves us the hops
	// the tool printed before it was stopped (a partial path is useful).
	var buf strings.Builder
	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		buf.WriteString(sc.Text())
		buf.WriteByte('\n')
	}
	scanErr := sc.Err()
	waitErr := cmd.Wait()
	hops := parseTrace(buf.String())
	// Tools exit non-zero when the destination is unreachable or on a
	// timeout kill; only surface an error (process or read) when we
	// parsed nothing at all — a partial path is still useful.
	if len(hops) == 0 {
		if waitErr != nil {
			return nil, waitErr
		}
		if scanErr != nil {
			return nil, scanErr
		}
	}
	return hops, nil
}

var (
	// leading "  N:" (tracepath, optional '?') or "  N " (traceroute).
	hopLineRE = regexp.MustCompile(`^\s*(\d+)[?:]?\s`)
	ipRE      = regexp.MustCompile(`\b(\d{1,3}(?:\.\d{1,3}){3}|[0-9a-fA-F:]{2,}:[0-9a-fA-F:]*)\b`)
	rttRE     = regexp.MustCompile(`([\d.]+)\s*ms`)
)

// parseTrace tolerantly parses tracepath or traceroute -n output into
// ordered hops, one entry per hop number. A hop with no extractable IP
// (or only '*' / "no reply") becomes a NoReply hop. Duplicate lines for
// the same hop number collapse to the first that yields an IP.
func parseTrace(out string) []Hop {
	byNum := map[int]Hop{}
	var order []int
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		m := hopLineRE.FindStringSubmatch(line)
		if m == nil {
			continue // header / Resume: / blank
		}
		num, _ := strconv.Atoi(m[1])
		rest := line[len(m[0]):]
		existing, seen := byNum[num]
		if seen && existing.IP != "" {
			continue // already have a good entry for this hop
		}
		if !seen {
			order = append(order, num)
		}

		hop := Hop{Num: num}
		// "[LOCALHOST]" / "pmtu" pmtu-discovery lines carry no usable IP.
		if ip := ipRE.FindString(rest); ip != "" && !strings.Contains(rest, "LOCALHOST") {
			hop.IP = ip
			if r := rttRE.FindStringSubmatch(rest); r != nil {
				hop.RTTms, _ = strconv.ParseFloat(r[1], 64)
			}
		} else if strings.Contains(rest, "no reply") || strings.Contains(rest, "*") {
			hop.NoReply = true
		} else {
			// pmtu / localhost line with no IP — keep as a placeholder
			// only if we have nothing else for this hop.
			hop.NoReply = true
		}
		byNum[num] = hop
	}
	hops := make([]Hop, 0, len(order))
	for _, n := range order {
		hops = append(hops, byNum[n])
	}
	return hops
}
