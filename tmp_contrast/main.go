package tmpcontrast

import (
	"fmt"
	"math"
	"strconv"
)

func lin(c float64) float64 {
	c = c / 255.0
	if c <= 0.03928 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func lum(hexColor string) float64 {
	r, _ := strconv.ParseInt(hexColor[0:2], 16, 64)
	g, _ := strconv.ParseInt(hexColor[2:4], 16, 64)
	b, _ := strconv.ParseInt(hexColor[4:6], 16, 64)
	return 0.2126*lin(float64(r)) + 0.7152*lin(float64(g)) + 0.0722*lin(float64(b))
}

func contrast(h1, h2 string) float64 {
	l1, l2 := lum(h1), lum(h2)
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

func blend(fgHex string, alpha float64, bgHex string) string {
	fr, _ := strconv.ParseInt(fgHex[0:2], 16, 64)
	fg, _ := strconv.ParseInt(fgHex[2:4], 16, 64)
	fb, _ := strconv.ParseInt(fgHex[4:6], 16, 64)
	br, _ := strconv.ParseInt(bgHex[0:2], 16, 64)
	bg, _ := strconv.ParseInt(bgHex[2:4], 16, 64)
	bb, _ := strconv.ParseInt(bgHex[4:6], 16, 64)
	r := math.Round(float64(fr)*alpha + float64(br)*(1-alpha))
	g := math.Round(float64(fg)*alpha + float64(bg)*(1-alpha))
	b := math.Round(float64(fb)*alpha + float64(bb)*(1-alpha))
	return fmt.Sprintf("%02x%02x%02x", int(r), int(g), int(b))
}
