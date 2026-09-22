package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// fakeEnrolCore mimics `POST /agents/enrol`: the first call with
// wantToken succeeds and returns plainSecret; anything else is
// refused, mirroring the core's real behaviour for an unknown or
// already-used token.
func fakeEnrolCore(t *testing.T, wantToken, wantHostname, plainSecret string) *httptest.Server {
	t.Helper()
	used := false
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Token    string `json:"token"`
			Hostname string `json:"hostname"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("core: decode request: %v", err)
		}
		if used || body.Token != wantToken || body.Hostname != wantHostname {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"code":"enrolment_failed","message":"enrolment token rejected"}}`))
			return
		}
		used = true
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"hostname": body.Hostname, "secret": plainSecret})
	}))
}

func TestEnsureSecret_ExchangesTokenAndWrites0600(t *testing.T) {
	core := fakeEnrolCore(t, "tok-abc", "webapp-01", "plain-secret-value")
	defer core.Close()

	secretPath := filepath.Join(t.TempDir(), "secret")
	key, err := EnsureSecret(context.Background(), core.Client(), core.URL, "webapp-01", "tok-abc", secretPath)
	if err != nil {
		t.Fatalf("EnsureSecret: %v", err)
	}

	sum := sha256.Sum256([]byte("plain-secret-value"))
	want := hex.EncodeToString(sum[:])
	if string(key) != want {
		t.Errorf("signing key = %q, want %q (hex(sha256(plain secret)) — the same digest the core stores as secret_hash)", key, want)
	}

	info, err := os.Stat(secretPath)
	if err != nil {
		t.Fatalf("stat secret file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("secret file mode = %o, want 0600", perm)
	}
	onDisk, err := os.ReadFile(secretPath)
	if err != nil {
		t.Fatalf("read secret file: %v", err)
	}
	if string(onDisk) != want {
		t.Errorf("secret file contents = %q, want %q", onDisk, want)
	}
}

func TestEnsureSecret_ExistingFileSkipsEnrolment(t *testing.T) {
	// A core that fails the test if it is ever contacted — an agent
	// with a secret already on disk must not re-enrol.
	core := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Fatalf("core: unexpected request %s %s — agent should not re-enrol", r.Method, r.URL.Path)
	}))
	defer core.Close()

	secretPath := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(secretPath, []byte("existing-key"), 0o600); err != nil {
		t.Fatalf("seed secret file: %v", err)
	}

	key, err := EnsureSecret(context.Background(), core.Client(), core.URL, "webapp-01", "some-token-still-in-config", secretPath)
	if err != nil {
		t.Fatalf("EnsureSecret: %v", err)
	}
	if string(key) != "existing-key" {
		t.Errorf("key = %q, want existing-key (token in config must be ignored once a secret file exists)", key)
	}
}

func TestEnsureSecret_NoFileNoTokenFails(t *testing.T) {
	secretPath := filepath.Join(t.TempDir(), "secret")
	if _, err := EnsureSecret(context.Background(), http.DefaultClient, "http://unused.invalid", "webapp-01", "", secretPath); err == nil {
		t.Fatal("expected an error with no secret file and no token")
	}
	if _, err := os.Stat(secretPath); !os.IsNotExist(err) {
		t.Errorf("secret file should not have been created, stat err = %v", err)
	}
}

func TestEnsureSecret_RefusedTokenFails(t *testing.T) {
	core := fakeEnrolCore(t, "the-real-token", "webapp-01", "plain-secret-value")
	defer core.Close()

	secretPath := filepath.Join(t.TempDir(), "secret")
	if _, err := EnsureSecret(context.Background(), core.Client(), core.URL, "webapp-01", "wrong-token", secretPath); err == nil {
		t.Fatal("expected an error for a refused token")
	}
	if _, err := os.Stat(secretPath); !os.IsNotExist(err) {
		t.Errorf("secret file should not have been created on a refused exchange, stat err = %v", err)
	}
}
