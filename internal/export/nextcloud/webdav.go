// Package nextcloud publishes completed-scan artefacts into a
// Nextcloud instance over WebDAV (direction 2 of the Nextcloud
// integration — "Nextcloud as output"). It is opt-in: nothing
// publishes unless `serve.yaml`'s nextcloud: block is enabled.
//
// WebDAV file drop is the cheapest, most universal surface — every
// Nextcloud install exposes `remote.php/dav` and authenticates an
// app password with HTTP Basic, so no Nextcloud app needs to be
// installed on the customer side. Talk/Deck adapters can follow as
// opt-in surfaces later; this package is deliberately just PUT +
// MKCOL.
package nextcloud

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Config is the resolved nextcloud: block. AppPassword is the
// secret itself (the serve layer reads it from app_password_file).
// TargetDir is relative to the bot user's Files root, e.g.
// "Wanderer" surfaces as /Wanderer in the user's Nextcloud.
type Config struct {
	URL         string
	Username    string
	AppPassword string
	TargetDir   string
}

// Client is a minimal Nextcloud WebDAV client: ensure-collection
// (MKCOL) and upload (PUT), with HTTP Basic auth.
type Client struct {
	cfg  Config
	http *http.Client
}

// NewClient returns a WebDAV client with a bounded HTTP timeout so
// a hung Nextcloud cannot stall the post-scan hook indefinitely.
func NewClient(cfg Config) *Client {
	return &Client{cfg: cfg, http: &http.Client{Timeout: 15 * time.Second}}
}

// davBase is the WebDAV root for the configured user's files:
// <url>/remote.php/dav/files/<username>.
func (c *Client) davBase() string {
	return strings.TrimRight(c.cfg.URL, "/") + "/remote.php/dav/files/" + url.PathEscape(c.cfg.Username)
}

// segments returns the directory segments (TargetDir first) under
// which an artefact for orgSlug is filed, each path-escaped.
func (c *Client) segments(orgSlug string) []string {
	var out []string
	for _, raw := range []string{c.cfg.TargetDir, orgSlug} {
		raw = strings.Trim(raw, "/")
		if raw == "" {
			continue
		}
		for _, part := range strings.Split(raw, "/") {
			if part != "" {
				out = append(out, url.PathEscape(part))
			}
		}
	}
	return out
}

// ensureDirs MKCOLs each cumulative collection so a subsequent PUT
// does not 409 on a missing parent. An existing collection returns
// 405 (Method Not Allowed), which is treated as success.
func (c *Client) ensureDirs(ctx context.Context, segments []string) error {
	path := c.davBase()
	for _, seg := range segments {
		path += "/" + seg
		if err := c.mkcol(ctx, path); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) mkcol(ctx context.Context, fullURL string) error {
	req, err := http.NewRequestWithContext(ctx, "MKCOL", fullURL, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.cfg.Username, c.cfg.AppPassword)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("nextcloud: mkcol: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusCreated, http.StatusOK, http.StatusMethodNotAllowed:
		// 201 created, or 405 — the collection already exists.
		return nil
	default:
		return fmt.Errorf("nextcloud: mkcol %s: unexpected status %d", fullURL, resp.StatusCode)
	}
}

// PutFile uploads body to <TargetDir>/<orgSlug>/<name> under the
// bot user's Files root, creating the parent collections first.
func (c *Client) PutFile(ctx context.Context, orgSlug, name, contentType string, body []byte) error {
	segs := c.segments(orgSlug)
	if err := c.ensureDirs(ctx, segs); err != nil {
		return err
	}
	fullURL := c.davBase()
	for _, seg := range segs {
		fullURL += "/" + seg
	}
	fullURL += "/" + url.PathEscape(name)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, fullURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.cfg.Username, c.cfg.AppPassword)
	req.Header.Set("Content-Type", contentType)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("nextcloud: put %s: %w", name, err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusCreated, http.StatusNoContent, http.StatusOK:
		// 201 new file, 204 overwrite, 200 some servers.
		return nil
	default:
		return fmt.Errorf("nextcloud: put %s: unexpected status %d", name, resp.StatusCode)
	}
}
