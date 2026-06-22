// Tests for the OpenTimestamps adapter. They pin Confirmed's verdict to the
// OpenTimestamps ecosystem's own bundled .ots example vectors (copied verbatim
// into testdata/ so the test is hermetic and never reads the module cache). These
// fixtures carry real Bitcoin attestations produced by the OTS ecosystem, so they
// are the external `ots verify` oracle this milestone demands — the expected
// (confirmed, height) values below are ground truth literals, NOT derived from the
// adapter's own code. The table asserts both legs of each verdict (confirmed flag
// AND exact height; pending flag AND zero height) so the gate is non-vacuous:
// reverting Confirmed to always-pending fails the confirmed rows, and dropping the
// height assertion would let a height regression pass — so the height is pinned.
package ots

import (
	"os"
	"path/filepath"
	"testing"
)

// readFixture loads one bundled .ots vector from testdata, failing the test if it
// is missing so a dropped fixture surfaces loudly rather than as a vacuous pass.
func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

// TestOTSConfirmed pins Confirmed against the bundled OpenTimestamps example
// vectors: the Bitcoin-attested proofs report (true, <exact height>), and a
// calendar-only proof reports (false, 0) without an error. The fixture set spans
// two confirmed heights and one pending case so the verdict is anchored to ground
// truth on both legs.
func TestOTSConfirmed(t *testing.T) {
	cases := []struct {
		name          string
		fixture       string
		wantConfirmed bool
		wantHeight    int64
	}{
		{"hello-world confirmed", "hello-world.txt.ots", true, 358391},
		{"empty confirmed", "empty.ots", true, 129405},
		{"merkle1 pending", "merkle1.txt.ots", false, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			confirmed, height, err := Confirmed(readFixture(t, tc.fixture))
			if err != nil {
				t.Fatalf("Confirmed(%s): unexpected error: %v", tc.fixture, err)
			}
			if confirmed != tc.wantConfirmed {
				t.Errorf("Confirmed(%s): confirmed = %v, want %v", tc.fixture, confirmed, tc.wantConfirmed)
			}
			if height != tc.wantHeight {
				t.Errorf("Confirmed(%s): height = %d, want %d", tc.fixture, height, tc.wantHeight)
			}
		})
	}
}

// TestOTSConfirmedParseError confirms the adapter fails closed on unparseable
// bytes: a proof carrying an unsupported attestation type errors out of
// ReadFromFile, and Confirmed must wrap that error and report NOT confirmed —
// never a silent confirmed. unknown-notary.txt.ots is a deliberately-unsupported
// vector from the bundled examples.
func TestOTSConfirmedParseError(t *testing.T) {
	confirmed, height, err := Confirmed(readFixture(t, "unknown-notary.txt.ots"))
	if err == nil {
		t.Fatalf("Confirmed(unknown-notary): want parse error, got nil")
	}
	if confirmed {
		t.Errorf("Confirmed(unknown-notary): confirmed = true on parse error, want false (fail-closed)")
	}
	if height != 0 {
		t.Errorf("Confirmed(unknown-notary): height = %d on parse error, want 0", height)
	}
}

// TestOTSConfirmedGarbage confirms wholly invalid bytes (not an .ots file at all)
// also fail closed rather than panicking or reporting confirmed.
func TestOTSConfirmedGarbage(t *testing.T) {
	confirmed, height, err := Confirmed([]byte("not an ots proof"))
	if err == nil {
		t.Fatalf("Confirmed(garbage): want parse error, got nil")
	}
	if confirmed || height != 0 {
		t.Errorf("Confirmed(garbage): got (%v, %d), want (false, 0) on parse error", confirmed, height)
	}
}
