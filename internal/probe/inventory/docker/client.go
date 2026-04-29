package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// dockerAPIVersion pins the Engine API. v1.41 ships with
// Docker 20.10 (released Dec 2020) — older daemons return an error
// on this version path, which surfaces cleanly as an
// inventory.docker.error rather than wrong data.
const dockerAPIVersion = "v1.41"

// container is the subset of /containers/json we read.
type container struct {
	ID      string            `json:"Id"`
	Names   []string          `json:"Names"`
	Image   string            `json:"Image"`
	ImageID string            `json:"ImageID"`
	Created int64             `json:"Created"`
	State   string            `json:"State"`
	Status  string            `json:"Status"`
	Labels  map[string]string `json:"Labels"`
}

// image is the subset of /images/json we read.
type image struct {
	ID       string   `json:"Id"`
	RepoTags []string `json:"RepoTags"`
	Created  int64    `json:"Created"`
	Size     int64    `json:"Size"`
}

// client wraps an http.Client whose transport routes every request
// through a unix domain socket. The host portion of request URLs is
// irrelevant.
type client struct {
	http   *http.Client
	socket string
}

func newClient(socket string) *client {
	return &client{
		socket: socket,
		http: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "unix", socket)
				},
			},
			Timeout: 10 * time.Second,
		},
	}
}

// listContainers calls GET /containers/json?all=true.
func (c *client) listContainers(ctx context.Context) ([]container, error) {
	var out []container
	if err := c.get(ctx, "/containers/json?all=true", &out); err != nil {
		return nil, fmt.Errorf("docker: list containers: %w", err)
	}
	return out, nil
}

// listImages calls GET /images/json.
func (c *client) listImages(ctx context.Context) ([]image, error) {
	var out []image
	if err := c.get(ctx, "/images/json", &out); err != nil {
		return nil, fmt.Errorf("docker: list images: %w", err)
	}
	return out, nil
}

// get is the shared GET-and-decode helper. The inspector intentionally
// has no other verb — see the read-only contract in
// docs/agent.md and the inventory-probe spec.
func (c *client) get(ctx context.Context, path string, into any) error {
	url := "http://docker/" + dockerAPIVersion + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("docker: build request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("docker: do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// Read up to 4 KiB of error body so the operator can see
		// what the daemon said without dragging in megabytes of
		// unexpected payload.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return &apiError{
			Path:       path,
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(body)),
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(into); err != nil {
		return fmt.Errorf("docker: decode %s: %w", path, err)
	}
	return nil
}

// apiError is returned for non-2xx responses. The inspector
// translates it into an inventory.docker.error Finding with the
// status code attached, so the operator can distinguish "daemon
// crashed" (5xx) from "API path moved" (404) at a glance.
type apiError struct {
	Path       string
	StatusCode int
	Body       string
}

func (e *apiError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("docker: %s returned %d", e.Path, e.StatusCode)
	}
	return fmt.Sprintf("docker: %s returned %d: %s", e.Path, e.StatusCode, e.Body)
}
