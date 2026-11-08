// Package logclient follows a hub's tlog-tiles transparency log: it derives the
// log origin, fetches and verifies checkpoints, and tracks consistency.
//
// This file holds origin derivation — the single source of truth for a hub's
// signed-note origin string. The origin is the hub domain plus the "/log" path
// (e.g. sb0.iscc.id/log), never the bare domain. It is reused for both the
// checkpoint signed-note name and the verifier-key derivation, so getting it
// wrong silently breaks every signature check.
package logclient

import (
	"fmt"
	"net/url"
	"strings"
)

// origin returns the hub's signed-note origin (<domain>/log) for a base URL.
//
// It accepts a hub base URL such as "https://sb0.iscc.id" (with or without a
// scheme or trailing slash) and returns the scheme-less "<host>/log" string the
// hub uses as its checkpoint name. It returns an error when the host is empty or
// the URL cannot be parsed.
func origin(baseURL string) (string, error) {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return "", fmt.Errorf("origin: empty base URL")
	}
	// url.Parse only populates Host when a scheme is present; default one in so a
	// bare "sb0.iscc.id" still parses into Host rather than Path.
	if !strings.Contains(trimmed, "://") {
		trimmed = "https://" + trimmed
	}
	u, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("origin: parse %q: %w", baseURL, err)
	}
	host := u.Host
	if host == "" {
		return "", fmt.Errorf("origin: no host in %q", baseURL)
	}
	return host + "/log", nil
}

// Origin exposes the package's single origin derivation (<domain>/log) so the
// binary can feed store.UpsertHub its origin argument. It delegates to the
// private origin — there is exactly one deriver — and does not duplicate the
// math, so the golden TestOrigin vectors cover this path too.
func Origin(baseURL string) (string, error) {
	return origin(baseURL)
}
