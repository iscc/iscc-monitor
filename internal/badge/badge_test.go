package badge

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

// statusFixtures pins each status to its distinct label and its single
// distinguishing SVG element, so a silhouette regression (e.g. two statuses
// collapsing to the same icon) fails the test rather than rendering an ambiguous
// badge.
var statusFixtures = []struct {
	status      string
	wantLabel   string
	wantDistSVG string // a marker present only in this status's silhouette
}{
	{"verified", "Verified", `M8.4 12.3l2.5 2.5 4.7-5.2`},           // check-circle
	{"unresolvable", "Unresolvable", `M9.2 9.3a3 3 0 0 1 5.6 1.2`},  // question-circle
	{"unverified", "Unverified", `M12 3.4 21 19H3z`},                // triangle-warning
	{"frozen", "Frozen", `M8.2 3.3h7.6L20.7 8.2v7.6L15.8 20.7H8.2`}, // octagon-x
	{"inactive", "Inactive", `x1="9.7" y1="9" x2="9.7" y2="15"`},    // pause-circle
}

// TestRenderLabelAndSilhouette asserts every status renders its own distinct
// text label AND its own distinguishing inline-SVG element — proving status is
// conveyed by icon + label + silhouette, not hue alone (ADR-0010 invariant 4).
func TestRenderLabelAndSilhouette(t *testing.T) {
	for _, f := range statusFixtures {
		var buf bytes.Buffer
		if err := Render(&buf, f.status); err != nil {
			t.Fatalf("Render(%q) error: %v", f.status, err)
		}
		out := buf.String()
		if !strings.Contains(out, ">"+f.wantLabel+"<") {
			t.Errorf("Render(%q): missing label %q in:\n%s", f.status, f.wantLabel, out)
		}
		if !strings.Contains(out, f.wantDistSVG) {
			t.Errorf("Render(%q): missing distinguishing SVG %q in:\n%s", f.status, f.wantDistSVG, out)
		}
		if !strings.Contains(out, "<svg") {
			t.Errorf("Render(%q): no inline <svg> rendered:\n%s", f.status, out)
		}
	}
}

// TestPairwiseDistinct collects all five rendered outputs into a set and asserts
// they are pairwise unique — no two statuses collapse to the same markup, so each
// is distinguishable in grayscale.
func TestPairwiseDistinct(t *testing.T) {
	seen := make(map[string]string, len(statusFixtures))
	for _, f := range statusFixtures {
		var buf bytes.Buffer
		if err := Render(&buf, f.status); err != nil {
			t.Fatalf("Render(%q) error: %v", f.status, err)
		}
		out := buf.String()
		if prev, dup := seen[out]; dup {
			t.Errorf("statuses %q and %q rendered identical markup", prev, f.status)
		}
		seen[out] = f.status
	}
	if len(seen) != len(statusFixtures) {
		t.Fatalf("expected %d distinct renderings, got %d", len(statusFixtures), len(seen))
	}
}

// TestUnknownStatusFailsClosed asserts an unknown or empty status returns an
// error and writes nothing — the caller's string is never echoed back as a label,
// so an attacker-controlled status cannot inject markup or a bogus label.
func TestUnknownStatusFailsClosed(t *testing.T) {
	for _, status := range []string{"", "pwned", "VERIFIED", "rotated", "<script>"} {
		var buf bytes.Buffer
		err := Render(&buf, status)
		if err == nil {
			t.Errorf("Render(%q): want error, got nil (rendered %q)", status, buf.String())
		}
		if buf.Len() != 0 {
			t.Errorf("Render(%q): want no output on error, got %q", status, buf.String())
		}
	}
}

// TestPartialComposesIntoParent asserts a parent page template can associate
// Source and invoke the partial by PartialName — the html/template
// partial-include idiom later M-UI pages use to embed the badge inline.
func TestPartialComposesIntoParent(t *testing.T) {
	parent := template.Must(template.New("page").Parse(
		`<li>{{template "hubStatusBadge" .}}</li>`,
	))
	template.Must(parent.Parse(Source))

	var buf bytes.Buffer
	if err := parent.ExecuteTemplate(&buf, "page", view{Status: "frozen", Label: "Frozen"}); err != nil {
		t.Fatalf("parent execute error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "<li>") || !strings.Contains(out, ">Frozen<") {
		t.Errorf("partial did not compose into parent:\n%s", out)
	}
	if !strings.Contains(out, `M8.2 3.3h7.6`) {
		t.Errorf("parent-composed partial missing frozen silhouette:\n%s", out)
	}
}
