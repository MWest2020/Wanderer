package tmpcontrast

import "testing"

func TestCurrentBadgeContrast(t *testing.T) {
	pairs := map[string]string{
		"soeverein":   "198754",
		"voldoende":   "84cc16",
		"afhankelijk": "d9534f",
		"onbekend":    "6c757d",
		"accent-nvt":  "1a73e8",
	}
	for name, fg := range pairs {
		bg := blend(fg, 0.12, "ffffff")
		c := contrast(fg, bg)
		t.Logf("%-12s fg=#%s bg(12%%tint over white)=#%s contrast=%.3f", name, fg, bg, c)
	}
}

func TestCurrentStatusTextContrast(t *testing.T) {
	pairs := map[string]string{
		"soeverein(complete)": "198754",
		"observation":         "ffc107",
		"concern":             "fd7e14",
		"finding":             "d9534f",
		"info":                "6c757d",
		"accent":              "1a73e8",
		"muted":               "666666",
	}
	for name, fg := range pairs {
		c := contrast(fg, "ffffff")
		t.Logf("%-20s fg=#%s on white contrast=%.3f", name, fg, c)
	}
}

func TestScopePillAndNavActive(t *testing.T) {
	c := contrast("874d00", "fff7e6")
	t.Logf("scope-pill        fg=#874d00 bg=#fff7e6 contrast=%.3f", c)
	c2 := contrast("ffffff", "1a73e8")
	t.Logf("nav-link.is-active fg=#ffffff bg=#1a73e8 contrast=%.3f", c2)
}

func TestVoldoendeOnWhite(t *testing.T) {
	t.Logf("voldoende fg=#84cc16 on white contrast=%.3f", contrast("84cc16", "ffffff"))
}

func TestCandidates(t *testing.T) {
	type cand struct {
		name string
		fg   string
		tint float64 // badge tint opacity
	}
	cands := []cand{
		{"soeverein", "146c43", 0.12},
		{"soeverein-alt", "0f5132", 0.12},
		{"voldoende-lime800", "4d7c0f", 0.15},
		{"voldoende-lime900", "3f6212", 0.15},
		{"afhankelijk", "b02a37", 0.12},
		{"afhankelijk-alt", "a71d2a", 0.12},
		{"onbekend", "495057", 0.12},
		{"onbekend-alt", "3d4247", 0.12},
		{"nvt-blue", "0b5ed7", 0.1},
		{"nvt-blue-alt", "1558b0", 0.1},
		{"observation-amber", "997404", 0.12},
		{"observation-amber-alt", "7a5b00", 0.12},
	}
	for _, c := range cands {
		bg := blend(c.fg, c.tint, "ffffff")
		onWhite := contrast(c.fg, "ffffff")
		onTint := contrast(c.fg, bg)
		t.Logf("%-24s fg=#%s tint=%.2f bg=#%s onWhite=%.3f onTintBg=%.3f", c.name, c.fg, c.tint, bg, onWhite, onTint)
	}
}

func TestAccentPrecision(t *testing.T) {
	l1 := lum("1a73e8")
	l2 := lum("ffffff")
	t.Logf("lum(accent)=%.6f lum(white)=%.6f contrast=%.6f", l1, l2, (l2+0.05)/(l1+0.05))
}

func TestFlowStateBezigNew(t *testing.T) {
	bg := blend("1a73e8", 0.08, "ffffff")
	t.Logf("flow-state-bezig NEW fg=#495057 bg=#%s contrast=%.3f", bg, contrast("495057", bg))
}

func TestVoldoendeTighten(t *testing.T) {
	for _, fg := range []string{"4d7c0f", "45700d", "3f6212", "436b0c"} {
		bg := blend(fg, 0.08, "ffffff")
		t.Logf("fg=#%s bg=#%s onWhite=%.3f onTintBg=%.3f", fg, bg, contrast(fg, "ffffff"), contrast(fg, bg))
	}
}

func TestFinalPalette08(t *testing.T) {
	type cand struct{ name, fg string }
	cands := []cand{
		{"soeverein", "146c43"},
		{"voldoende", "4d7c0f"},
		{"afhankelijk", "b02a37"},
		{"onbekend", "495057"},
		{"nvt", "0b5ed7"},
		{"observation", "7a5b00"},
	}
	for _, tint := range []float64{0.08, 0.10} {
		for _, c := range cands {
			bg := blend(c.fg, tint, "ffffff")
			t.Logf("tint=%.2f %-12s fg=#%s bg=#%s onTintBg=%.3f", tint, c.name, c.fg, bg, contrast(c.fg, bg))
		}
	}
}

func TestFlowStateAndKindHost(t *testing.T) {
	// flow-state-bezig: color var(--info)=#6c757d on rgba(26,115,232,0.08) over white
	bezigBg := blend("1a73e8", 0.08, "ffffff")
	t.Logf("flow-state-bezig fg=#6c757d bg=#%s contrast=%.3f", bezigBg, contrast("6c757d", bezigBg))
	// flow-state-niet_gemeten: color var(--muted)=#666666 on #eeeeee
	t.Logf("flow-state-niet_gemeten fg=#666666 bg=#eeeeee contrast=%.3f", contrast("666666", "eeeeee"))
}
