package assessor

import (
	"strings"
	"testing"
)

func TestContainerRegistryEntries_NotEmpty(t *testing.T) {
	entries := ContainerRegistryEntries()
	if len(entries) == 0 {
		t.Fatal("ContainerRegistryEntries returned zero entries")
	}
	for i, e := range entries {
		if e.Host == "" {
			t.Errorf("entry %d: empty host", i)
		}
		if e.Vendor == "" {
			t.Errorf("entry %d (host %q): empty vendor", i, e.Host)
		}
	}
}

func TestContainerRegistryMatch_Cases(t *testing.T) {
	cases := []struct {
		name         string
		imageRef     string
		wantMatch    bool
		wantHost     string
		wantImplicit bool
		wantVendor   string
	}{
		{name: "bare name → docker.io implicit", imageRef: "nginx", wantMatch: true, wantHost: "docker.io", wantImplicit: true, wantVendor: "Docker"},
		{name: "library shorthand → docker.io implicit", imageRef: "library/nginx:1.27", wantMatch: true, wantHost: "docker.io", wantImplicit: true, wantVendor: "Docker"},
		{name: "explicit docker.io", imageRef: "docker.io/library/nginx:1.27", wantMatch: true, wantHost: "docker.io", wantImplicit: false, wantVendor: "Docker"},
		{name: "gcr.io", imageRef: "gcr.io/foo/bar:v1", wantMatch: true, wantHost: "gcr.io", wantImplicit: false, wantVendor: "Google"},
		{name: "ghcr.io", imageRef: "ghcr.io/anthropics/claude-code:latest", wantMatch: true, wantHost: "ghcr.io", wantImplicit: false, wantVendor: "GitHub"},
		{name: "EU self-hosted miss", imageRef: "harbor.example.de/team/app:v3", wantMatch: false, wantHost: "harbor.example.de", wantImplicit: false},
		{name: "localhost dev miss", imageRef: "localhost:5000/dev:latest", wantMatch: false, wantHost: "localhost:5000", wantImplicit: false},
		{name: "digest reference", imageRef: "gcr.io/foo/bar@sha256:abc123", wantMatch: true, wantHost: "gcr.io", wantImplicit: false, wantVendor: "Google"},
		{name: "case insensitive", imageRef: "DOCKER.IO/library/nginx", wantMatch: true, wantHost: "DOCKER.IO", wantImplicit: false, wantVendor: "Docker"},
		{name: "empty", imageRef: "", wantMatch: true, wantHost: "docker.io", wantImplicit: true, wantVendor: "Docker"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ContainerRegistryMatch(tc.imageRef)
			if ok != tc.wantMatch {
				t.Fatalf("match = %v, want %v (resolved host %q)", ok, tc.wantMatch, got.ResolvedHost)
			}
			if got.ResolvedHost != tc.wantHost {
				t.Errorf("host = %q, want %q", got.ResolvedHost, tc.wantHost)
			}
			if got.ImpliedDockerIO != tc.wantImplicit {
				t.Errorf("implicit = %v, want %v", got.ImpliedDockerIO, tc.wantImplicit)
			}
			if tc.wantMatch && !strings.Contains(got.Registry.Vendor, tc.wantVendor) {
				t.Errorf("vendor = %q, want substring %q", got.Registry.Vendor, tc.wantVendor)
			}
		})
	}
}
