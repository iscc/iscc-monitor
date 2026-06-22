// Package otsclient is the monitor's OpenTimestamps calendar-HTTP transport
// adapter: it provides the real follower.Upgrader closure (and a Stamp helper)
// the background OTS upgrade loop drives, wiring github.com/nbd-wtf/opentimestamps'
// networked calendar calls to internal/ots' pure Bitcoin-confirmation classifier.
//
// It lives OUTSIDE internal/follower on purpose: the Upgrader is a func seam so
// internal/follower imports no anchoring package (it stays import-free of both
// opentimestamps and internal/ots). The real closure must import BOTH the network
// library AND internal/ots, so it belongs here; cmd/iscc-monitor constructs the
// closure and passes it to the follower as a follower.Upgrader. This split keeps
// internal/ots a pure classify leaf and internal/otsclient the network transport.
//
// Like internal/ots this is NOT a WASM-pure leaf (it imports net/http transitively
// via opentimestamps and internal/ots): OTS confirmation is server-side only, so no
// WASM-shared package may import it. It imports no database/sql either — the
// Upgrader takes a store.OTSRecord by value, never the store's DB handle — so the
// store stays an uncoupled leaf.
//
// OTS is best-effort and NEVER blocks the follower (ADR-0004): all calendar HTTP
// happens here in the Upgrader/Stamp calls, which the loop runs off the poll path.
// A transport fault is returned as a wrapped error so OTSTick records a back-off and
// retries — it never freezes a hub.
package otsclient

import (
	"context"
	"fmt"
	"time"

	opentimestamps "github.com/nbd-wtf/opentimestamps"

	"github.com/iscc/iscc-monitor/internal/follower"
	"github.com/iscc/iscc-monitor/internal/ots"
	"github.com/iscc/iscc-monitor/internal/store"
)

// DefaultCalendarURL is the public OpenTimestamps calendar Stamp submits a digest
// to by default. A single calendar is fine for v1; redundancy (multiple calendars
// with independent back-off) is a later refinement, so this is the lone default.
const DefaultCalendarURL = "https://alice.btc.calendar.opentimestamps.org"

// upgradeTimeout bounds one calendar upgrade request so a stalled calendar GET can
// never hang a single OTSTick pass indefinitely (starving later pending rows). It is
// a best-effort transport bound, not a safety gate — OTS is best-effort and the loop
// backs off and retries on a deadline-exceeded error like any other transport fault.
const upgradeTimeout = 30 * time.Second

// seqUpgrade upgrades one pending calendar sequence against its calendar, returning
// the lengthened sequence. It is the injectable network seam: production wires
// opentimestamps.UpgradeSequence (which does the calendar HTTP GET), while tests
// inject a fake so the Upgrader's classify/serialize path runs fully offline.
type seqUpgrade func(ctx context.Context, seq opentimestamps.Sequence, initial []byte) (opentimestamps.Sequence, error)

// NewUpgrader returns the production follower.Upgrader: a closure that upgrades a
// stamped root's pending OpenTimestamps proof against the calendar and classifies
// the result with internal/ots. It wires the real opentimestamps.UpgradeSequence
// (the calendar HTTP transport), so cmd/iscc-monitor passes its result straight to
// the follower as the Upgrader — the first production caller of follower.OTSTick.
func NewUpgrader() follower.Upgrader {
	return buildUpgrader(opentimestamps.UpgradeSequence)
}

// buildUpgrader builds the follower.Upgrader over an injectable sequence-upgrade
// function so the closure is testable offline. The closure satisfies the
// follower.Upgrader contract:
//
//   - It parses r.OTSBytes (the serialized initial pending sequence(s) a future
//     stampRoot calendar-submit will persist; this step does not yet persist them,
//     but the closure's shape is fixed against that contract now) into an
//     opentimestamps.File, recovering the library's malformed-input panic into a
//     fail-closed error so a garbage proof never crashes the upgrade goroutine.
//   - For each still-pending (calendar-only) sequence it calls upgrade(ctx, seq,
//     file.Digest) against the calendar; already-Bitcoin-attested sequences are
//     kept verbatim (upgrading them would hit a malformed empty-host URL). It
//     reassembles the upgraded File and serializes it.
//   - It classifies the serialized bytes with ots.Confirmed (the oracle-gated
//     classifier — never a re-implemented attestation walk): confirmed →
//     {Confirmed: true, OTSBytes, BTCHeight}; still pending → {Confirmed: false}
//     with a nil error (the loop records a back-off); a transport or parse fault →
//     a wrapped error (best-effort, the loop backs off and retries, never freezes).
func buildUpgrader(upgrade seqUpgrade) follower.Upgrader {
	return func(ctx context.Context, r store.OTSRecord) (follower.UpgradeResult, error) {
		file, err := recoverRead(r.OTSBytes)
		if err != nil {
			return follower.UpgradeResult{}, fmt.Errorf("otsclient.Upgrade: read proof: %w", err)
		}

		// Keep already-confirmed sequences verbatim; upgrade only the pending ones
		// (calling UpgradeSequence on a Bitcoin-attested sequence GETs an empty-host
		// URL and fails).
		sequences := append([]opentimestamps.Sequence(nil), file.GetBitcoinAttestedSequences()...)
		for _, seq := range file.GetPendingSequences() {
			upgraded, err := safeUpgrade(ctx, upgrade, seq, file.Digest)
			if err != nil {
				return follower.UpgradeResult{}, fmt.Errorf("otsclient.Upgrade: upgrade sequence: %w", err)
			}
			sequences = append(sequences, upgraded)
		}

		upgradedFile := opentimestamps.File{Digest: file.Digest, Sequences: sequences}
		upgradedBytes := upgradedFile.SerializeToFile()

		confirmed, height, err := ots.Confirmed(upgradedBytes)
		if err != nil {
			return follower.UpgradeResult{}, fmt.Errorf("otsclient.Upgrade: classify: %w", err)
		}
		if !confirmed {
			return follower.UpgradeResult{Confirmed: false}, nil
		}
		return follower.UpgradeResult{Confirmed: true, OTSBytes: upgradedBytes, BTCHeight: height}, nil
	}
}

// Stamp submits digest to calendarURL and returns the serialized initial pending
// OpenTimestamps proof bytes. It wraps opentimestamps.Stamp, builds the single-
// sequence File{Digest, Sequences}, and returns file.SerializeToFile() — the bytes
// a future stampRoot will persist into OTSRecord.OTSBytes so the Upgrader can later
// upgrade them. It is exposed now and called by stampRoot in a follow-up sub-step;
// the follower does not call it yet. A calendar transport fault is returned wrapped
// (best-effort, the caller backs off, never freezes).
func Stamp(ctx context.Context, calendarURL string, digest [32]byte) ([]byte, error) {
	seq, err := opentimestamps.Stamp(ctx, calendarURL, digest)
	if err != nil {
		return nil, fmt.Errorf("otsclient.Stamp: %q: %w", calendarURL, err)
	}
	file := opentimestamps.File{Digest: digest[:], Sequences: []opentimestamps.Sequence{seq}}
	return file.SerializeToFile(), nil
}

// recoverRead calls opentimestamps.ReadFromFile and converts any panic it raises on
// malformed input into an error, mirroring internal/ots' recoverParse. The library
// over-reads its buffer on some truncated / non-.ots bytes (a slice-bounds panic)
// rather than failing cleanly; recovering here keeps the fail-closed contract — a
// parse fault is always a returned error, never a crash — at this external-library
// boundary, so an untrusted .ots blob can never crash the upgrade goroutine.
func recoverRead(otsBytes []byte) (file *opentimestamps.File, err error) {
	defer func() {
		if r := recover(); r != nil {
			file = nil
			err = fmt.Errorf("opentimestamps.ReadFromFile panicked: %v", r)
		}
	}()
	return opentimestamps.ReadFromFile(otsBytes)
}

// safeUpgrade runs one calendar sequence-upgrade under a bounded deadline and a
// panic guard, mirroring recoverRead at this external-library boundary. It derives a
// per-request context.WithTimeout(ctx, upgradeTimeout) so a stalled calendar GET
// returns rather than hangs (opentimestamps.UpgradeSequence honors ctx deadlines on
// its GET), cancelling it before return (one explicit defer per call, no defer-in-loop
// leak). It recovers any panic seq.Compute raises on a parseable-but-uncomputable
// proof (opentimestamps' sha1/reverse/hexlify/keccak256 ops and invalid-instruction
// paths panic rather than erroring) into a wrapped fail-closed error, so an untrusted
// calendar response can never crash the upgrade goroutine (ADR-0004: OTS never crashes
// the follower). Both guards surface the fault as a returned error the loop wraps and
// OTSTick treats as a best-effort back-off — never a //nolint or swallow.
func safeUpgrade(ctx context.Context, upgrade seqUpgrade, seq opentimestamps.Sequence, digest []byte) (upgraded opentimestamps.Sequence, err error) {
	ctx, cancel := context.WithTimeout(ctx, upgradeTimeout)
	defer cancel()
	defer func() {
		if r := recover(); r != nil {
			upgraded = nil
			err = fmt.Errorf("otsclient.Upgrade: upgrade sequence panicked: %v", r)
		}
	}()
	return upgrade(ctx, seq, digest)
}
