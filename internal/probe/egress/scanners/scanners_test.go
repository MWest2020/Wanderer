package scanners

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParseEnvFile(t *testing.T) {
	body := `# comment
DATABASE_URL=postgres://app:secret@db.example:5432/app
FOO="bar baz"
EMPTY=
`
	got := parseEnvFile("/tmp/x.env", body)
	if len(got) != 3 {
		t.Fatalf("want 3, got %d (%+v)", len(got), got)
	}
	if got[0].Key != "DATABASE_URL" {
		t.Errorf("first key = %s", got[0].Key)
	}
	if got[1].Value != "bar baz" {
		t.Errorf("quoted value lost: %s", got[1].Value)
	}
}

func TestConfigFiles_AvailableRequiresPaths(t *testing.T) {
	c := ConfigFiles{}
	if ok, _ := c.Available(); ok {
		t.Errorf("default ConfigFiles should be unavailable")
	}
	c = ConfigFiles{Paths: []string{"/tmp"}}
	if ok, _ := c.Available(); !ok {
		t.Errorf("with paths configured should be available")
	}
}

func TestConfigFiles_Scan(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, "app.env")
	if err := os.WriteFile(envPath, []byte("S3_ENDPOINT=https://s3.eu-west-1.amazonaws.com\nDB_USER=app\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	c := ConfigFiles{Paths: []string{dir}}
	got, err := c.Scan(context.Background())
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 candidates, got %d", len(got))
	}
}

func TestConfigFiles_DoesNotFollowSymlinkOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	target := filepath.Join(outside, "secret.env")
	if err := os.WriteFile(target, []byte("SHOULD_NOT_LEAK=value"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	link := filepath.Join(root, "linked.env")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	c := ConfigFiles{Paths: []string{root}}
	got, _ := c.Scan(context.Background())
	for _, cand := range got {
		if cand.Key == "SHOULD_NOT_LEAK" {
			t.Errorf("symlink outside root was followed: %+v", cand)
		}
	}
}

func TestParseEnviron(t *testing.T) {
	body := []byte("DATABASE_URL=postgres://x\x00API_KEY=secret\x00")
	got := parseEnviron("123", body)
	if len(got) != 2 {
		t.Fatalf("want 2, got %d", len(got))
	}
	if got[1].Key != "API_KEY" || got[1].Value != "secret" {
		t.Errorf("second entry mismatch: %+v", got[1])
	}
}

func TestProcEnv_AvailableWhenRootExists(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "1"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "1", "environ"), []byte("FOO=bar\x00"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	p := ProcEnv{Root: dir}
	if ok, _ := p.Available(); !ok {
		t.Fatalf("should be available")
	}
	got, err := p.Scan(context.Background())
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(got) != 1 || got[0].Value != "bar" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestParseUnit_EnvironmentDirective(t *testing.T) {
	body := `[Service]
Environment="DATABASE_URL=postgres://x"
Environment="API_KEY=hunter2"
ExecStart=/bin/true
`
	got := parseUnit("/tmp/x.service", body)
	if len(got) != 2 {
		t.Fatalf("want 2, got %d", len(got))
	}
	if got[0].Key != "DATABASE_URL" {
		t.Errorf("first key = %s", got[0].Key)
	}
}
