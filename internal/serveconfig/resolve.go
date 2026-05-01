package serveconfig

import (
	"flag"
	"os"
	"time"
)

// SetFlags returns the set of flag names that were explicitly
// passed on the command line — the seam that distinguishes
// "user typed --ui=false" from "default false".
//
// Call AFTER fs.Parse(args). flag.Visit walks only flags that
// were set, regardless of value (so --ui=false counts as set).
func SetFlags(fs *flag.FlagSet) map[string]bool {
	out := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { out[f.Name] = true })
	return out
}

// ResolveString applies the precedence flag > env > yaml > default
// for a string-valued setting. If the user passed --flagName on
// the command line, flagVal wins. Else if the named env var is
// set and non-empty, the env wins. Else if the YAML value is
// non-empty, the YAML wins. Else the hard default.
func ResolveString(setFlags map[string]bool, flagName, flagVal, envName, yamlVal, hardDefault string) string {
	if setFlags[flagName] {
		return flagVal
	}
	if v, ok := os.LookupEnv(envName); ok && v != "" {
		return v
	}
	if yamlVal != "" {
		return yamlVal
	}
	return hardDefault
}

// ResolveBool applies the same precedence for a boolean setting.
// flag.Visit treats --ui=false as "set" — so an explicit
// flag-false correctly overrides a YAML true. There is no
// "yaml unset" sentinel for bools the way "" works for strings,
// so the caller passes yamlPresent: true when the field appeared
// in the parsed YAML or when its zero value is meaningful, and
// false when the YAML did not carry the field. In practice the
// caller passes `yamlPresent = (cfg != nil)` because the YAML
// either provides a Config or doesn't; once provided, every bool
// is meaningful (zero = "the operator chose not to set this").
func ResolveBool(setFlags map[string]bool, flagName string, flagVal bool, envName string, yamlVal bool, yamlPresent bool, hardDefault bool) bool {
	if setFlags[flagName] {
		return flagVal
	}
	if v, ok := os.LookupEnv(envName); ok && v != "" {
		return envBool(v, hardDefault)
	}
	if yamlPresent {
		return yamlVal
	}
	return hardDefault
}

// ResolveDuration applies the precedence for time.Duration. The
// YAML side already parses durations natively (Go's
// `time.Duration` UnmarshalYAML), so we treat zero as "not set".
// An operator who wants an explicit zero passes the flag.
func ResolveDuration(setFlags map[string]bool, flagName string, flagVal time.Duration, envName string, yamlVal time.Duration, hardDefault time.Duration) time.Duration {
	if setFlags[flagName] {
		return flagVal
	}
	if v, ok := os.LookupEnv(envName); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	if yamlVal != 0 {
		return yamlVal
	}
	return hardDefault
}

// envBool parses common boolean string forms from env vars —
// "1", "true", "yes" → true; "0", "false", "no", "" → false;
// anything else falls back to the hard default rather than
// silently mis-parsing.
func envBool(v string, fallback bool) bool {
	switch v {
	case "1", "true", "TRUE", "yes", "YES", "on", "ON":
		return true
	case "0", "false", "FALSE", "no", "NO", "off", "OFF":
		return false
	default:
		return fallback
	}
}
