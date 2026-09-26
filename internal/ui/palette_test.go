package ui

// Port of openspec/changes/2026-09-26-kleuren-voor-iedereen/meting-kleuren.py:
// WCAG relative-luminance contrast, Machado et al. 2009 colour-vision-deficiency
// simulation on linear RGB, and CIEDE2000. Keep this in lock-step with that
// script rather than inventing a second measurement.

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

type paletteRGB [3]float64

func hexToRGB(t *testing.T, hex string) paletteRGB {
	t.Helper()
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		t.Fatalf("invalid hex colour %q", hex)
	}
	var out paletteRGB
	for i := 0; i < 3; i++ {
		v, err := strconv.ParseInt(hex[i*2:i*2+2], 16, 64)
		if err != nil {
			t.Fatalf("invalid hex colour %q: %v", hex, err)
		}
		out[i] = float64(v) / 255
	}
	return out
}

func linearize(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func delinearize(c float64) float64 {
	if c < 0 {
		c = 0
	}
	if c > 1 {
		c = 1
	}
	if c <= 0.0031308 {
		return 12.92 * c
	}
	return 1.055*math.Pow(c, 1.0/2.4) - 0.055
}

func relLuminance(c paletteRGB) float64 {
	r, g, b := linearize(c[0]), linearize(c[1]), linearize(c[2])
	return 0.2126*r + 0.7152*g + 0.0722*b
}

// contrastRatio implements the WCAG 2.1 contrast formula.
func contrastRatio(a, b paletteRGB) float64 {
	la, lb := relLuminance(a), relLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// cvdMatrices are the Machado et al. 2009 simulation matrices, severity 1.0,
// applied to linear RGB.
var cvdMatrices = map[string][3][3]float64{
	"protan": {
		{0.152286, 1.052583, -0.204868},
		{0.114503, 0.786281, 0.099216},
		{-0.003882, -0.048116, 1.051998},
	},
	"deutan": {
		{0.367322, 0.860646, -0.227968},
		{0.280085, 0.672501, 0.047413},
		{-0.011820, 0.042940, 0.968881},
	},
	"tritan": {
		{1.255528, -0.076749, -0.178779},
		{-0.078411, 0.930809, 0.147602},
		{0.004733, 0.691367, 0.303900},
	},
}

var visions = []string{"normaal", "protan", "deutan", "tritan"}

func simulate(c paletteRGB, vision string) paletteRGB {
	if vision == "normaal" {
		return c
	}
	m := cvdMatrices[vision]
	lin := paletteRGB{linearize(c[0]), linearize(c[1]), linearize(c[2])}
	var out paletteRGB
	for i := 0; i < 3; i++ {
		out[i] = delinearize(m[i][0]*lin[0] + m[i][1]*lin[1] + m[i][2]*lin[2])
	}
	return out
}

type lab [3]float64

func toLab(c paletteRGB) lab {
	r, g, b := linearize(c[0]), linearize(c[1]), linearize(c[2])
	x := (0.4124*r + 0.3576*g + 0.1805*b) / 0.95047
	y := 0.2126*r + 0.7152*g + 0.0722*b
	z := (0.0193*r + 0.1192*g + 0.9505*b) / 1.08883
	f := func(v float64) float64 {
		if v > 0.008856 {
			return math.Cbrt(v)
		}
		return 7.787*v + 16.0/116
	}
	fx, fy, fz := f(x), f(y), f(z)
	return lab{116*fy - 16, 500 * (fx - fy), 200 * (fy - fz)}
}

func degToRad(d float64) float64 { return d * math.Pi / 180 }
func radToDeg(r float64) float64 { return r * 180 / math.Pi }

// ciede2000 implements the CIEDE2000 colour-difference formula.
func ciede2000(a, b lab) float64 {
	L1, a1, b1 := a[0], a[1], a[2]
	L2, a2, b2 := b[0], b[1], b[2]
	C1 := math.Hypot(a1, b1)
	C2 := math.Hypot(a2, b2)
	Cb := (C1 + C2) / 2
	G := 0.5 * (1 - math.Sqrt(math.Pow(Cb, 7)/(math.Pow(Cb, 7)+math.Pow(25, 7))))
	a1p, a2p := (1+G)*a1, (1+G)*a2
	C1p, C2p := math.Hypot(a1p, b1), math.Hypot(a2p, b2)
	h1 := math.Mod(radToDeg(math.Atan2(b1, a1p))+360, 360)
	h2 := math.Mod(radToDeg(math.Atan2(b2, a2p))+360, 360)
	dL := L2 - L1
	dC := C2p - C1p

	var dh float64
	switch {
	case C1p*C2p == 0:
		dh = 0
	case math.Abs(h2-h1) <= 180:
		dh = h2 - h1
	case h2 > h1:
		dh = h2 - h1 - 360
	default:
		dh = h2 - h1 + 360
	}
	dH := 2 * math.Sqrt(C1p*C2p) * math.Sin(degToRad(dh/2))

	Lb := (L1 + L2) / 2
	Cbp := (C1p + C2p) / 2
	var hb float64
	switch {
	case C1p*C2p == 0:
		hb = h1 + h2
	case math.Abs(h1-h2) <= 180:
		hb = (h1 + h2) / 2
	case h1+h2 < 360:
		hb = (h1 + h2 + 360) / 2
	default:
		hb = (h1 + h2 - 360) / 2
	}

	T := 1 - 0.17*math.Cos(degToRad(hb-30)) + 0.24*math.Cos(degToRad(2*hb)) +
		0.32*math.Cos(degToRad(3*hb+6)) - 0.2*math.Cos(degToRad(4*hb-63))
	SL := 1 + 0.015*math.Pow(Lb-50, 2)/math.Sqrt(20+math.Pow(Lb-50, 2))
	SC := 1 + 0.045*Cbp
	SH := 1 + 0.015*Cbp*T
	RT := -2 * math.Sqrt(math.Pow(Cbp, 7)/(math.Pow(Cbp, 7)+math.Pow(25, 7))) *
		math.Sin(degToRad(60*math.Exp(-math.Pow((hb-275)/25, 2))))

	return math.Sqrt(math.Pow(dL/SL, 2) + math.Pow(dC/SC, 2) + math.Pow(dH/SH, 2) + RT*(dC/SC)*(dH/SH))
}

// minDeltaE returns the smallest pairwise CIEDE2000 among pal's colours,
// under the given vision (or "normaal" for unsimulated).
func minDeltaE(t *testing.T, pal map[string]string, vision string) (float64, string, string) {
	t.Helper()
	names := make([]string, 0, len(pal))
	for name := range pal {
		names = append(names, name)
	}
	best := math.Inf(1)
	var bestA, bestB string
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			a := simulate(hexToRGB(t, pal[names[i]]), vision)
			b := simulate(hexToRGB(t, pal[names[j]]), vision)
			de := ciede2000(toLab(a), toLab(b))
			if de < best {
				best, bestA, bestB = de, names[i], names[j]
			}
		}
	}
	return best, bestA, bestB
}

// cssVarPattern matches `--name: #rgb;` or `--name: #rrggbb;` declarations
// in a :root block.
var cssVarPattern = regexp.MustCompile(`--([a-z][a-z-]*):\s*#([0-9a-fA-F]{3}|[0-9a-fA-F]{6});`)

func extractCSSVars(css string) map[string]string {
	vars := map[string]string{}
	for _, m := range cssVarPattern.FindAllStringSubmatch(css, -1) {
		hex := strings.ToLower(m[2])
		if len(hex) == 3 {
			hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
		}
		vars[m[1]] = "#" + hex
	}
	return vars
}

func readMainCSS(t *testing.T) string {
	t.Helper()
	b, err := assets.ReadFile("static/main.css")
	if err != nil {
		t.Fatalf("read main.css: %v", err)
	}
	return string(b)
}

// checkPalette asserts the property from
// openspec/changes/2026-09-26-kleuren-voor-iedereen/specs/web-ui/spec.md:
// every pair of verdict colours SHALL differ by CIEDE2000 >= 20 under normal
// vision and simulated protanopia/deuteranopia/tritanopia, each verdict
// colour SHALL have >= 3:1 contrast against bg, and each verdict's text
// colour SHALL have >= 4.5:1 contrast against its own fill.
func checkPalette(t *testing.T, fills, texts map[string]string, bg string) {
	t.Helper()

	for _, vision := range visions {
		de, a, b := minDeltaE(t, fills, vision)
		if de < 20 {
			t.Errorf("%s: kleinste CIEDE2000 is %.1f (%s/%s), wil >= 20", vision, de, a, b)
		}
	}

	bgRGB := hexToRGB(t, bg)
	for name, fill := range fills {
		c := contrastRatio(hexToRGB(t, fill), bgRGB)
		if c < 3.0 {
			t.Errorf("%s: contrast %.2f met achtergrond, wil >= 3:1", name, c)
		}
	}

	for name, textHex := range texts {
		fillHex, ok := fills[name]
		if !ok {
			t.Errorf("%s: geen bijbehorende vlakkleur gevonden", name)
			continue
		}
		c := contrastRatio(hexToRGB(t, textHex), hexToRGB(t, fillHex))
		if c < 4.5 {
			t.Errorf("%s-tekst: contrast %.2f op eigen vlak, wil >= 4,5:1", name, c)
		}
	}
}

func TestPaletteMeetsAccessibilitySpec(t *testing.T) {
	vars := extractCSSVars(readMainCSS(t))

	verdicts := []string{"soeverein", "voldoende", "afhankelijk", "onbekend"}
	fills := map[string]string{}
	texts := map[string]string{}
	for _, v := range verdicts {
		fill, ok := vars[v]
		if !ok {
			t.Fatalf("--%s niet gevonden in main.css", v)
		}
		text, ok := vars[v+"-tekst"]
		if !ok {
			t.Fatalf("--%s-tekst niet gevonden in main.css", v)
		}
		fills[v] = fill
		texts[v] = text
	}
	bg, ok := vars["bg"]
	if !ok {
		t.Fatalf("--bg niet gevonden in main.css")
	}

	checkPalette(t, fills, texts, bg)
}

// TestOldPaletteFailsAccessibilitySpec proves checkPalette can actually fail:
// the palette in place before this change had voldoende and afhankelijk only
// ΔE 4.2 apart under simulated deuteranopia (see proposal.md).
func TestOldPaletteFailsAccessibilitySpec(t *testing.T) {
	old := map[string]string{
		"soeverein":   "#146c43",
		"voldoende":   "#45700d",
		"afhankelijk": "#b02a37",
		"onbekend":    "#495057",
	}
	de, a, b := minDeltaE(t, old, "deutan")
	if de >= 20 {
		t.Fatalf("verwachtte dat het oude palet faalt onder deutan, kreeg ΔE %.1f (%s/%s)", de, a, b)
	}
	const want = 4.2
	if math.Abs(de-want) > 0.5 {
		t.Fatalf("kleinste ΔE onder deutan = %.1f (%s/%s), wil rond %.1f (zie proposal.md)", de, a, b, want)
	}
}

// TestVerdictTextClearsContrastWhereItIsUsed walks every rule in main.css that
// sets a verdict colour as TEXT and checks it against that rule's own
// background: the fill if the rule sets one, otherwise the page. Run 01 of
// kleuren-voor-iedereen converted the selectors its brief listed and left
// three families on the old pattern (tint + coloured text), where voldoende
// measured 3.5:1. A check on the four variables alone could not see that; this
// one reads the rules that actually render.
func TestVerdictTextClearsContrastWhereItIsUsed(t *testing.T) {
	css := readMainCSS(t)
	vars := extractCSSVars(css)
	rule := regexp.MustCompile(`([^{}]+)\{([^{}]*)\}`)
	textVar := regexp.MustCompile(`(?:^|;)\s*color:\s*var\(--(soeverein|voldoende|afhankelijk|onbekend)\)`)
	bgVar := regexp.MustCompile(`background(?:-color)?:\s*var\(--([a-z-]+)\)`)
	bgTint := regexp.MustCompile(`background(?:-color)?:\s*rgba\(`)
	checked := 0
	for _, m := range rule.FindAllStringSubmatch(css, -1) {
		sel, body := strings.TrimSpace(m[1]), m[2]
		tm := textVar.FindStringSubmatch(body)
		if tm == nil {
			continue
		}
		checked++
		if bgTint.MatchString(body) {
			t.Errorf("%s: verdict colour --%s as text on a tint; use a solid fill with --%s-tekst", sel, tm[1], tm[1])
			continue
		}
		bg := vars["bg"]
		if bm := bgVar.FindStringSubmatch(body); bm != nil {
			bg = vars[bm[1]]
		}
		if r := contrastRatio(hexToRGB(t, vars[tm[1]]), hexToRGB(t, bg)); r < 4.5 {
			t.Errorf("%s: --%s on #%s is %.2f:1, want >= 4.5", sel, tm[1], bg, r)
		}
	}
	if checked == 0 {
		t.Fatal("no rule uses a verdict colour as text; the test checked nothing")
	}
}
