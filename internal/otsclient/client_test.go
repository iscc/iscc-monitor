// Tests for the OpenTimestamps calendar-HTTP adapter. They exercise the Upgrader
// closure's parse / upgrade-seam / serialize / classify path fully OFFLINE by
// injecting a fake sequence-upgrade function (no live calendar is ever called in
// go test). The fixtures are the same bundled OpenTimestamps example vectors the
// internal/ots golden test uses (copied verbatim into testdata/), so the closure's
// confirmed verdict is pinned to the same external ground truth: an already-Bitcoin-
// attested fixture round-trips to (Confirmed: true, <exact height>), and a
// calendar-only fixture left unchanged by the fake upgrade stays pending. The Stamp
// helper's serialize round-trip is tested in isolation (no live calendar).
package otsclient

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	opentimestamps "github.com/nbd-wtf/opentimestamps"

	"github.com/iscc/iscc-monitor/internal/ots"
	"github.com/iscc/iscc-monitor/internal/store"
)

// readFixture loads one bundled .ots vector from testdata, failing loudly if it is
// missing so a dropped fixture surfaces rather than as a vacuous pass.
func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

// failUpgrade is a seqUpgrade that fails the test if called: the Upgrader must NOT
// touch the calendar for an already-Bitcoin-confirmed proof (it has no pending
// sequences), so any call here is a regression.
func failUpgrade(t *testing.T) seqUpgrade {
	t.Helper()
	return func(context.Context, opentimestamps.Sequence, []byte) (opentimestamps.Sequence, error) {
		t.Fatalf("upgrade called on a proof with no pending sequences")
		return nil, nil
	}
}

// TestUpgradeAlreadyConfirmed feeds the Upgrader an already-Bitcoin-attested proof:
// it must classify it confirmed at the fixture's exact height WITHOUT calling the
// calendar (no pending sequences), and the returned OTSBytes must re-classify as
// confirmed through ots.Confirmed. The height literals are the bundled example
// ground truth, so the verdict is non-vacuous (a wrong height fails the assertion).
func TestUpgradeAlreadyConfirmed(t *testing.T) {
	cases := []struct {
		name       string
		fixture    string
		wantHeight int64
	}{
		{"hello-world", "hello-world.txt.ots", 358391},
		{"empty", "empty.ots", 129405},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			up := buildUpgrader(failUpgrade(t))
			res, err := up(context.Background(), store.OTSRecord{OTSBytes: readFixture(t, tc.fixture)})
			if err != nil {
				t.Fatalf("Upgrade(%s): unexpected error: %v", tc.fixture, err)
			}
			if !res.Confirmed {
				t.Fatalf("Upgrade(%s): Confirmed = false, want true", tc.fixture)
			}
			if res.BTCHeight != tc.wantHeight {
				t.Errorf("Upgrade(%s): BTCHeight = %d, want %d", tc.fixture, res.BTCHeight, tc.wantHeight)
			}
			confirmed, height, err := ots.Confirmed(res.OTSBytes)
			if err != nil {
				t.Fatalf("re-classify(%s): unexpected error: %v", tc.fixture, err)
			}
			if !confirmed || height != tc.wantHeight {
				t.Errorf("re-classify(%s): got (%v, %d), want (true, %d)", tc.fixture, confirmed, height, tc.wantHeight)
			}
		})
	}
}

// TestUpgradeStillPending feeds the Upgrader a calendar-only (pending) proof with a
// fake upgrade that returns each pending sequence UNCHANGED (the calendar has not
// yet seen a Bitcoin attestation). The reassembled proof reclassifies as pending, so
// the Upgrader returns {Confirmed: false} with a nil error (the loop records a
// back-off, never an error). The fake records being called, proving the pending
// branch ran (not the confirmed short-circuit).
func TestUpgradeStillPending(t *testing.T) {
	var calls int
	stillPending := func(_ context.Context, seq opentimestamps.Sequence, _ []byte) (opentimestamps.Sequence, error) {
		calls++
		return seq, nil // unchanged → still calendar-only → still pending
	}
	up := buildUpgrader(stillPending)
	res, err := up(context.Background(), store.OTSRecord{OTSBytes: readFixture(t, "merkle1.txt.ots")})
	if err != nil {
		t.Fatalf("Upgrade(merkle1 pending): unexpected error: %v", err)
	}
	if res.Confirmed {
		t.Errorf("Upgrade(merkle1 pending): Confirmed = true, want false (still pending)")
	}
	if res.BTCHeight != 0 {
		t.Errorf("Upgrade(merkle1 pending): BTCHeight = %d, want 0", res.BTCHeight)
	}
	if calls == 0 {
		t.Errorf("Upgrade(merkle1 pending): upgrade never called, pending branch did not run")
	}
}

// TestUpgradeTransportFault confirms a calendar transport fault surfaces as a
// wrapped error (best-effort: OTSTick records a back-off and retries, never freezes)
// rather than a silent pending or confirmed. The fake upgrade always errors.
func TestUpgradeTransportFault(t *testing.T) {
	wantErr := errors.New("calendar unreachable")
	faulting := func(context.Context, opentimestamps.Sequence, []byte) (opentimestamps.Sequence, error) {
		return nil, wantErr
	}
	up := buildUpgrader(faulting)
	res, err := up(context.Background(), store.OTSRecord{OTSBytes: readFixture(t, "merkle1.txt.ots")})
	if err == nil {
		t.Fatalf("Upgrade(transport fault): want error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("Upgrade(transport fault): error %v does not wrap %v", err, wantErr)
	}
	if res.Confirmed {
		t.Errorf("Upgrade(transport fault): Confirmed = true on error, want false")
	}
}

// TestUpgradeGarbageProof confirms an unparseable proof fails closed (a wrapped
// error, never a crash or a silent confirmed) — the recoverRead panic guard at the
// library boundary surfaces the fault as a returned error.
func TestUpgradeGarbageProof(t *testing.T) {
	up := buildUpgrader(failUpgrade(t))
	res, err := up(context.Background(), store.OTSRecord{OTSBytes: []byte("not an ots proof")})
	if err == nil {
		t.Fatalf("Upgrade(garbage): want parse error, got nil")
	}
	if res.Confirmed {
		t.Errorf("Upgrade(garbage): Confirmed = true on parse error, want false (fail-closed)")
	}
}

// TestUpgradePanicRecovered confirms a calendar upgrade that PANICS (the
// opentimestamps library panics on parseable-but-uncomputable proofs:
// sha1/reverse/hexlify/keccak256 ops and invalid-instruction paths) surfaces as a
// wrapped fail-closed error rather than crashing the upgrade goroutine (ADR-0004: OTS
// never crashes the follower). The fake panics like the library would on an
// unimplemented op; the merkle1 fixture has a pending sequence so the closure reaches
// safeUpgrade. Removing the recover() in safeUpgrade makes this test panic the binary.
func TestUpgradePanicRecovered(t *testing.T) {
	panicSeq := func(context.Context, opentimestamps.Sequence, []byte) (opentimestamps.Sequence, error) {
		panic("op not implemented")
	}
	up := buildUpgrader(panicSeq)
	res, err := up(context.Background(), store.OTSRecord{OTSBytes: readFixture(t, "merkle1.txt.ots")})
	if err == nil {
		t.Fatalf("Upgrade(panic): want wrapped error, got nil")
	}
	if !strings.Contains(err.Error(), "upgrade sequence") {
		t.Errorf("Upgrade(panic): error %q does not carry the upgrade-sequence wrap", err)
	}
	if !strings.Contains(err.Error(), "panicked") {
		t.Errorf("Upgrade(panic): error %q does not name the recovered panic", err)
	}
	if res.Confirmed {
		t.Errorf("Upgrade(panic): Confirmed = true on recovered panic, want false (fail-closed)")
	}
}

// TestUpgradeBoundsContext confirms each calendar upgrade call carries a bounded
// deadline: safeUpgrade derives a context.WithTimeout before invoking the seam, so a
// stalled calendar GET returns rather than hangs one OTSTick pass. The fake captures
// the ctx it is handed; after driving the closure over a pending-sequence fixture the
// captured ctx must report a deadline. Reverting safeUpgrade to the bare ctx makes
// this FAIL (the parent context.Background() carries no deadline).
func TestUpgradeBoundsContext(t *testing.T) {
	var gotDeadline bool
	var sawCall bool
	captureCtx := func(ctx context.Context, seq opentimestamps.Sequence, _ []byte) (opentimestamps.Sequence, error) {
		sawCall = true
		_, gotDeadline = ctx.Deadline()
		return seq, nil // unchanged → still pending, no further upgrade needed
	}
	up := buildUpgrader(captureCtx)
	if _, err := up(context.Background(), store.OTSRecord{OTSBytes: readFixture(t, "merkle1.txt.ots")}); err != nil {
		t.Fatalf("Upgrade(bounded ctx): unexpected error: %v", err)
	}
	if !sawCall {
		t.Fatalf("Upgrade(bounded ctx): upgrade seam never called, pending branch did not run")
	}
	if !gotDeadline {
		t.Errorf("Upgrade(bounded ctx): upgrade ctx carried no deadline, want a bounded timeout")
	}
}

// TestStampPanicRecovered confirms a calendar stamp submit that PANICS (the
// opentimestamps library parses the calendar response through the same panic-prone
// parseCalendarServerResponse/parseTimestamp family recoverRead guards on the upgrade
// side) surfaces as a wrapped fail-closed error rather than crashing the monitor
// (ADR-0004: OTS never crashes the follower). The fake panics like the library would on
// a malformed response. Removing the recover() in safeStamp makes this test panic the
// binary.
func TestStampPanicRecovered(t *testing.T) {
	digest := [32]byte{}
	panicStamp := func(context.Context, string, [32]byte) (opentimestamps.Sequence, error) {
		panic("malformed calendar response")
	}
	out, err := buildStamper(panicStamp)(context.Background(), DefaultCalendarURL, digest)
	if err == nil {
		t.Fatalf("Stamp(panic): want wrapped error, got nil")
	}
	if out != nil {
		t.Errorf("Stamp(panic): bytes = %x on recovered panic, want nil (fail-closed)", out)
	}
	if !strings.Contains(err.Error(), "otsclient.Stamp") {
		t.Errorf("Stamp(panic): error %q does not carry the otsclient.Stamp wrap", err)
	}
	if !strings.Contains(err.Error(), "panicked") {
		t.Errorf("Stamp(panic): error %q does not name the recovered panic", err)
	}
}

// TestStampBoundsContext confirms the calendar stamp submit carries a bounded
// deadline: safeStamp derives a context.WithTimeout before invoking the seam, so a
// stalled calendar POST returns rather than hangs the OTS goroutine. The fake captures
// the ctx it is handed; after driving the Stamp helper the captured ctx must report a
// deadline. Reverting safeStamp to the bare ctx makes this FAIL (the parent
// context.Background() carries no deadline).
func TestStampBoundsContext(t *testing.T) {
	digest := [32]byte{}
	var gotDeadline bool
	var sawCall bool
	captureCtx := func(ctx context.Context, _ string, _ [32]byte) (opentimestamps.Sequence, error) {
		sawCall = true
		_, gotDeadline = ctx.Deadline()
		return opentimestamps.Sequence{{Attestation: &opentimestamps.Attestation{CalendarServerURL: DefaultCalendarURL}}}, nil
	}
	if _, err := buildStamper(captureCtx)(context.Background(), DefaultCalendarURL, digest); err != nil {
		t.Fatalf("Stamp(bounded ctx): unexpected error: %v", err)
	}
	if !sawCall {
		t.Fatalf("Stamp(bounded ctx): stamp seam never called")
	}
	if !gotDeadline {
		t.Errorf("Stamp(bounded ctx): stamp ctx carried no deadline, want a bounded timeout")
	}
}

// TestStampSerializesRoundTrip checks the Stamp helper's File assembly /
// serialization in isolation: a synthesized single-sequence File serializes and
// reparses with the same digest, the round-trip Stamp performs after a calendar
// submit. It does NOT call a live calendar (no network in go test); only the
// serialize seam is exercised here.
func TestStampSerializesRoundTrip(t *testing.T) {
	digest := [32]byte{}
	for i := range digest {
		digest[i] = byte(i)
	}
	// Mirror Stamp's File assembly with a placeholder calendar sequence (a single
	// pending attestation), then serialize / reparse — the same round-trip Stamp
	// does on the real calendar response, exercised without the network.
	seq := opentimestamps.Sequence{{Attestation: &opentimestamps.Attestation{CalendarServerURL: DefaultCalendarURL}}}
	file := opentimestamps.File{Digest: digest[:], Sequences: []opentimestamps.Sequence{seq}}
	serialized := file.SerializeToFile()

	parsed, err := opentimestamps.ReadFromFile(serialized)
	if err != nil {
		t.Fatalf("reparse serialized stamp: %v", err)
	}
	if string(parsed.Digest) != string(digest[:]) {
		t.Errorf("round-trip digest = %x, want %x", parsed.Digest, digest[:])
	}
	if len(parsed.GetPendingSequences()) != 1 {
		t.Errorf("round-trip pending sequences = %d, want 1", len(parsed.GetPendingSequences()))
	}
}
