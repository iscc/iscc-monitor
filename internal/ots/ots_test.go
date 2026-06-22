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
	"math"
	"os"
	"path/filepath"
	"testing"

	opentimestamps "github.com/nbd-wtf/opentimestamps"
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

// TestOTSConfirmedHeightOverflow confirms the adapter fails closed on a Bitcoin
// height above math.MaxInt64: the library's varint reader has no overflow cap, so
// a corrupt/malicious .ots blob can carry such a height that the raw int64() cast
// would wrap NEGATIVE while still reporting confirmed. The fixture is built by
// taking a real confirmed vector, swapping its height to math.MaxInt64+1, and
// reserializing — so the height round-trips through readVarUint exactly. Confirmed
// must return (false, 0, err). This case is non-vacuous: reverting the guard makes
// the cast wrap and the assertion FAILS (confirmed=true, a negative height).
func TestOTSConfirmedHeightOverflow(t *testing.T) {
	overflow := overflowHeightOTS(t, "hello-world.txt.ots")
	confirmed, height, err := Confirmed(overflow)
	if err == nil {
		t.Fatalf("Confirmed(overflow height): want overflow error, got nil")
	}
	if confirmed {
		t.Errorf("Confirmed(overflow height): confirmed = true on overflow, want false (fail-closed)")
	}
	if height != 0 {
		t.Errorf("Confirmed(overflow height): height = %d on overflow, want 0", height)
	}
}

// overflowHeightOTS builds an .ots blob whose Bitcoin attestation height is
// math.MaxInt64+1 by parsing a real confirmed fixture, swapping the height on its
// Bitcoin attestation, and reserializing. The serialized bytes are valid .ots
// (they reparse) but carry an int64-overflowing height, exercising the guard.
func overflowHeightOTS(t *testing.T, name string) []byte {
	t.Helper()
	file, err := opentimestamps.ReadFromFile(readFixture(t, name))
	if err != nil {
		t.Fatalf("parse fixture %s: %v", name, err)
	}
	swapped := false
	for si := range file.Sequences {
		for ii := range file.Sequences[si] {
			att := file.Sequences[si][ii].Attestation
			if att != nil && att.CalendarServerURL == "" {
				att.BitcoinBlockHeight = uint64(math.MaxInt64) + 1
				swapped = true
			}
		}
	}
	if !swapped {
		t.Fatalf("fixture %s carries no Bitcoin attestation to overflow", name)
	}
	return file.SerializeToFile()
}
