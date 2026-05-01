package serveconfig_test

import (
	"flag"
	"testing"
	"time"

	"github.com/MWest2020/wanderer/internal/serveconfig"
)

// parseHelper builds a flag set, invokes Parse, and returns the
// "explicitly set" map plus the flag set itself. Tests use this
// to mimic the production seam in cmd/wanderer/serve.go.
func parseHelper(t *testing.T, args []string, register func(fs *flag.FlagSet)) (map[string]bool, *flag.FlagSet) {
	t.Helper()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	register(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("parse: %v", err)
	}
	return serveconfig.SetFlags(fs), fs
}

func TestResolveString_FlagWins(t *testing.T) {
	t.Setenv("TEST_LISTEN", "envvalue")
	var listen string
	set, _ := parseHelper(t, []string{"-listen", "flagvalue"}, func(fs *flag.FlagSet) {
		fs.StringVar(&listen, "listen", "", "")
	})
	got := serveconfig.ResolveString(set, "listen", listen, "TEST_LISTEN", "yamlvalue", "default")
	if got != "flagvalue" {
		t.Errorf("got %q, want flagvalue", got)
	}
}

func TestResolveString_EnvBeatsYAML(t *testing.T) {
	t.Setenv("TEST_LISTEN", "envvalue")
	var listen string
	set, _ := parseHelper(t, []string{}, func(fs *flag.FlagSet) {
		fs.StringVar(&listen, "listen", "", "")
	})
	got := serveconfig.ResolveString(set, "listen", listen, "TEST_LISTEN", "yamlvalue", "default")
	if got != "envvalue" {
		t.Errorf("got %q, want envvalue", got)
	}
}

func TestResolveString_YAMLBeatsDefault(t *testing.T) {
	var listen string
	set, _ := parseHelper(t, []string{}, func(fs *flag.FlagSet) {
		fs.StringVar(&listen, "listen", "", "")
	})
	got := serveconfig.ResolveString(set, "listen", listen, "TEST_LISTEN_UNSET", "yamlvalue", "default")
	if got != "yamlvalue" {
		t.Errorf("got %q, want yamlvalue", got)
	}
}

func TestResolveString_DefaultWhenNothingSet(t *testing.T) {
	var listen string
	set, _ := parseHelper(t, []string{}, func(fs *flag.FlagSet) {
		fs.StringVar(&listen, "listen", "", "")
	})
	got := serveconfig.ResolveString(set, "listen", listen, "TEST_LISTEN_UNSET", "", "default")
	if got != "default" {
		t.Errorf("got %q, want default", got)
	}
}

func TestResolveString_EmptyEnvFallsThroughToYAML(t *testing.T) {
	// Env is set but to an empty string — treat as not set so
	// YAML can still apply.
	t.Setenv("TEST_LISTEN", "")
	var listen string
	set, _ := parseHelper(t, []string{}, func(fs *flag.FlagSet) {
		fs.StringVar(&listen, "listen", "", "")
	})
	got := serveconfig.ResolveString(set, "listen", listen, "TEST_LISTEN", "yamlvalue", "default")
	if got != "yamlvalue" {
		t.Errorf("got %q, want yamlvalue (empty env should not block YAML)", got)
	}
}

func TestResolveBool_ExplicitFalseFlagBeatsYAMLTrue(t *testing.T) {
	// The motivating case: operator wants `--ui=false` to override
	// `ui.enabled: true` in the YAML.
	var ui bool
	set, _ := parseHelper(t, []string{"-ui=false"}, func(fs *flag.FlagSet) {
		fs.BoolVar(&ui, "ui", false, "")
	})
	got := serveconfig.ResolveBool(set, "ui", ui, "TEST_UI_UNSET", true /*yaml*/, true /*present*/, false /*hard*/)
	if got != false {
		t.Errorf("got %v, want false (explicit flag must win over YAML)", got)
	}
}

func TestResolveBool_YAMLAppliesWhenFlagAndEnvUnset(t *testing.T) {
	var ui bool
	set, _ := parseHelper(t, []string{}, func(fs *flag.FlagSet) {
		fs.BoolVar(&ui, "ui", false, "")
	})
	got := serveconfig.ResolveBool(set, "ui", ui, "TEST_UI_UNSET", true, true, false)
	if got != true {
		t.Errorf("got %v, want true (YAML must apply)", got)
	}
}

func TestResolveBool_EnvOverridesYAML(t *testing.T) {
	t.Setenv("TEST_UI", "false")
	var ui bool
	set, _ := parseHelper(t, []string{}, func(fs *flag.FlagSet) {
		fs.BoolVar(&ui, "ui", false, "")
	})
	got := serveconfig.ResolveBool(set, "ui", ui, "TEST_UI", true, true, false)
	if got != false {
		t.Errorf("got %v, want false (env false must beat YAML true)", got)
	}
}

func TestResolveDuration_FlagWins(t *testing.T) {
	var budget time.Duration
	set, _ := parseHelper(t, []string{"-budget", "5m"}, func(fs *flag.FlagSet) {
		fs.DurationVar(&budget, "budget", 0, "")
	})
	got := serveconfig.ResolveDuration(set, "budget", budget, "TEST_BUDGET", 2*time.Minute, time.Minute)
	if got != 5*time.Minute {
		t.Errorf("got %s, want 5m", got)
	}
}

func TestResolveDuration_EnvBeatsYAML(t *testing.T) {
	t.Setenv("TEST_BUDGET", "90s")
	var budget time.Duration
	set, _ := parseHelper(t, []string{}, func(fs *flag.FlagSet) {
		fs.DurationVar(&budget, "budget", 0, "")
	})
	got := serveconfig.ResolveDuration(set, "budget", budget, "TEST_BUDGET", 2*time.Minute, time.Minute)
	if got != 90*time.Second {
		t.Errorf("got %s, want 90s", got)
	}
}

func TestResolveDuration_YAMLBeatsDefault(t *testing.T) {
	var budget time.Duration
	set, _ := parseHelper(t, []string{}, func(fs *flag.FlagSet) {
		fs.DurationVar(&budget, "budget", 0, "")
	})
	got := serveconfig.ResolveDuration(set, "budget", budget, "TEST_BUDGET_UNSET", 2*time.Minute, time.Minute)
	if got != 2*time.Minute {
		t.Errorf("got %s, want 2m", got)
	}
}

func TestResolveDuration_DefaultWhenNothingSet(t *testing.T) {
	var budget time.Duration
	set, _ := parseHelper(t, []string{}, func(fs *flag.FlagSet) {
		fs.DurationVar(&budget, "budget", 0, "")
	})
	got := serveconfig.ResolveDuration(set, "budget", budget, "TEST_BUDGET_UNSET", 0, time.Minute)
	if got != time.Minute {
		t.Errorf("got %s, want 1m default", got)
	}
}
