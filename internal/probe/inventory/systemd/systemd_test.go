package systemd

import "testing"

func TestParse(t *testing.T) {
	raw := `[
		{"unit":"sshd.service","load":"loaded","active":"active","sub":"running","description":"OpenSSH server daemon"},
		{"unit":"cron.service","load":"loaded","active":"active","sub":"running","description":"Regular background program processing daemon"}
	]`
	got, err := Parse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2, got %d", len(got))
	}
	if got[0].Subject != "sshd.service" {
		t.Errorf("subject = %s", got[0].Subject)
	}
	if got[0].Attributes["sub_state"] != "running" {
		t.Errorf("sub_state lost")
	}
}

func TestParse_BadJSON(t *testing.T) {
	_, err := Parse("not json")
	if err == nil {
		t.Errorf("expected error on non-JSON")
	}
}
