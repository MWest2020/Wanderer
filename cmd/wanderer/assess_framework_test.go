package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/pkg/models"
)

func TestSelectedFrameworks_WandIsCanonical(t *testing.T) {
	got, err := selectedFrameworks("wand")
	if err != nil {
		t.Fatalf("wand: %v", err)
	}
	if len(got) != 1 || got[0] != models.FrameworkWand {
		t.Errorf("got %+v, want [wand]", got)
	}
}

func TestSelectedFrameworks_DictuAliasResolvesToWand(t *testing.T) {
	got, err := selectedFrameworks("dictu")
	if err != nil {
		t.Fatalf("dictu: %v", err)
	}
	if len(got) != 1 || got[0] != models.FrameworkWand {
		t.Errorf("dictu alias should resolve to wand; got %+v", got)
	}
}

func TestSelectedFrameworks_BothIncludesWand(t *testing.T) {
	got, err := selectedFrameworks("both")
	if err != nil {
		t.Fatalf("both: %v", err)
	}
	if len(got) != 2 || got[0] != models.FrameworkWand || got[1] != models.FrameworkEUCSF {
		t.Errorf("both: got %+v, want [wand, eucsf]", got)
	}
}

func TestSelectedFrameworks_UnknownFails(t *testing.T) {
	_, err := selectedFrameworks("not-a-framework")
	if err == nil {
		t.Fatal("expected error for unknown framework")
	}
	if !strings.Contains(err.Error(), "wand|eucsf|both") {
		t.Errorf("error message should hint at the canonical values, got %v", err)
	}
}

func TestWarnIfDeprecatedFramework_DictuEmits(t *testing.T) {
	var buf bytes.Buffer
	warnIfDeprecatedFramework(&buf, "dictu")
	got := buf.String()
	if !strings.HasPrefix(got, "warning:") {
		t.Errorf("expected warning prefix, got %q", got)
	}
	if !strings.Contains(got, "--framework wand") {
		t.Errorf("warning should suggest --framework wand, got %q", got)
	}
}

func TestWarnIfDeprecatedFramework_WandSilent(t *testing.T) {
	var buf bytes.Buffer
	warnIfDeprecatedFramework(&buf, "wand")
	if buf.Len() != 0 {
		t.Errorf("--framework wand should not warn, got %q", buf.String())
	}
}

func TestWarnIfDeprecatedFramework_EucsfSilent(t *testing.T) {
	var buf bytes.Buffer
	warnIfDeprecatedFramework(&buf, "eucsf")
	if buf.Len() != 0 {
		t.Errorf("--framework eucsf should not warn, got %q", buf.String())
	}
}
