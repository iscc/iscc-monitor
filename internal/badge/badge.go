// Package badge renders the HubStatusBadge: a pure, server-rendered template
// partial that shows a hub's status as an inline-SVG silhouette plus a text
// label. It is the grayscale-safe, colorblind-safe status primitive every M-UI
// surface (realm index, hub dossier, log browser, certificate) reuses, ported
// from the design handoff's HubStatusBadge React component (ADR-0010).
//
// Each of the five statuses (verified / unresolvable / unverified / frozen /
// inactive) renders a distinct silhouette (check-circle, question-circle,
// triangle-warning, octagon-x, pause-circle) AND a distinct text label, so the
// status is conveyed by icon + label + silhouette and never by hue alone — these
// pages are screenshotted into legal/audit contexts (ADR-0010 invariant 4).
//
// The package is a pure leaf: its only imports are bytes / embed / html/template
// / io / fmt (stdlib), with no net/http and no internal/store, so it composes
// into any page template and stays trivially testable. The label is mapped in Go
// from a fixed table — the caller's status string is never trusted verbatim for
// the label, so an unknown status fails closed rather than emitting an
// attacker-controlled label.
//
// The oracle/conformance gate is N/A: this is pure static-markup rendering keyed
// on a status string, touching no signature, RFC-6962, Merkle, did:web, fsck, or
// proof path.
package badge

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"io"
)

// PartialName is the defined-template name parent pages invoke via
// {{template "hubStatusBadge" .}} after associating Source.
const PartialName = "hubStatusBadge"

// Source is the embedded template source for the badge partial, parsed once at
// package init so a malformed partial fails the build, not a request. Parent page
// templates associate it with template.Must(parent.Parse(badge.Source)) and then
// invoke it by PartialName with a value exposing .Status and .Label.
//
//go:embed badge.html
var Source string

// tmpl is the parsed badge partial, used by Render for direct rendering.
var tmpl = template.Must(template.New("badge").Parse(Source))

// labels maps each supported status to its display label. It is the single
// source of truth for the label text: the partial renders .Label, which is only
// ever a value from this table, so a caller-supplied status can never inject an
// arbitrary label string.
var labels = map[string]string{
	"verified":     "Verified",
	"unresolvable": "Unresolvable",
	"unverified":   "Unverified",
	"frozen":       "Frozen",
	"inactive":     "Inactive",
}

// view is the badge partial's view-model: the validated status string (selecting
// the silhouette) and its fixed-table label.
type view struct {
	Status string
	Label  string
}

// Label returns the fixed-table display label for status, with ok reporting
// whether status is a supported badge status. A parent page that composes the
// partial via {{template "hubStatusBadge" .}} precomputes its row's .Label from
// this single source of truth, so the label stays single-sourced and fail-closed:
// the caller's status string is never trusted verbatim for the label, and an
// unknown status yields ("", false) rather than an attacker-controlled label.
func Label(status string) (string, bool) {
	label, ok := labels[status]
	return label, ok
}

// Render writes the HubStatusBadge partial for status to w. It fails closed: an
// unknown or empty status returns an error and writes nothing, so the caller can
// never render an arbitrary attacker-controlled label or an unstyled status.
func Render(w io.Writer, status string) error {
	label, ok := labels[status]
	if !ok {
		return fmt.Errorf("badge: unknown hub status %q", status)
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, PartialName, view{Status: status, Label: label}); err != nil {
		return fmt.Errorf("badge: render %q: %w", status, err)
	}
	_, err := buf.WriteTo(w)
	return err
}
