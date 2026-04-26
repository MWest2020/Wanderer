package scheduler

import (
	"strings"
	"testing"
)

func TestParseConfig_Valid(t *testing.T) {
	yaml := `
schedules:
  - name: daily
    target:
      domain: example.nl
    cron: "0 6 * * *"
    timeout: 5m
`
	c, err := ParseConfig([]byte(yaml))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(c.Schedules) != 1 {
		t.Fatalf("want 1 schedule, got %d", len(c.Schedules))
	}
}

func TestParseConfig_InvalidCron(t *testing.T) {
	yaml := `
schedules:
  - name: bad
    target: {domain: example.nl}
    cron: "foo bar"
`
	_, err := ParseConfig([]byte(yaml))
	if err == nil || !strings.Contains(err.Error(), "invalid cron") {
		t.Errorf("want invalid cron error, got %v", err)
	}
}

func TestParseConfig_MissingTarget(t *testing.T) {
	yaml := `
schedules:
  - name: noTarget
    cron: "0 6 * * *"
`
	_, err := ParseConfig([]byte(yaml))
	if err == nil || !strings.Contains(err.Error(), "target.domain") {
		t.Errorf("want target.domain error, got %v", err)
	}
}

func TestParseConfig_DuplicateName(t *testing.T) {
	yaml := `
schedules:
  - name: a
    target: {domain: example.nl}
    cron: "0 6 * * *"
  - name: a
    target: {domain: example.nl}
    cron: "0 7 * * *"
`
	_, err := ParseConfig([]byte(yaml))
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("want duplicate error, got %v", err)
	}
}

func TestParseConfig_MissingName(t *testing.T) {
	yaml := `
schedules:
  - target: {domain: example.nl}
    cron: "0 6 * * *"
`
	_, err := ParseConfig([]byte(yaml))
	if err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Errorf("want name-required error, got %v", err)
	}
}
