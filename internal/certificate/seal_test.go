// Tests for the ISCC Trust Seal: a canonical ISCC-ID and the UTC date fill both places
// each appears in the brand template (the lettering and the aria-label), no placeholder
// survives, and an id of any other shape yields no seal rather than misaligned lettering.
package certificate

import (
	"strings"
	"testing"
	"time"
)

func TestTrustSealPersonalises(t *testing.T) {
	at := time.Date(2026, 11, 7, 23, 30, 0, 0, time.FixedZone("CET", 3600))
	seal := string(trustSeal("ISCC:MEIGKUAF6B6AUMAC", at))

	if want := `aria-label="ISCC Trust Seal: verifiable evidence for ISCC:MEIGKUAF6B6AUMAC, 2026-11-07"`; !strings.Contains(seal, want) {
		t.Errorf("seal missing its personalised accessible name %q", want)
	}
	if n := strings.Count(seal, "ISCC:MEIGKUAF6B6AUMAC"); n != 2 {
		t.Errorf("seal names the id %d times, want 2 (lettering and aria-label)", n)
	}
	if n := strings.Count(seal, "2026-11-07"); n != 2 {
		t.Errorf("seal names the UTC date %d times, want 2 (lettering and aria-label)", n)
	}
	if strings.Contains(seal, "{{") {
		t.Errorf("seal still holds a placeholder")
	}
	if !strings.HasPrefix(seal, "<svg ") {
		t.Errorf("seal does not start with the inline <svg> element")
	}
}

func TestTrustSealRejectsOtherShapes(t *testing.T) {
	for _, id := range []string{
		"",
		"MEIGKUAF6B6AUMAC",          // no prefix
		"ISCC:MEIGKUAF6B6AUMA",      // 15 characters
		"ISCC:MEIGKUAF6B6AUMACA",    // 17 characters
		"ISCC:meigkuaf6b6aumac",     // lower case
		"ISCC:MEIGKUAF6B6AUMA1",     // 1 is not base32
		`ISCC:MEIGKUAF6B6AUM"><svg`, // markup
	} {
		if got := trustSeal(id, time.Now()); got != "" {
			t.Errorf("trustSeal(%q) = %d bytes, want no seal", id, len(got))
		}
	}
}
