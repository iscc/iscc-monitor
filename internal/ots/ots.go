// Package ots is the monitor's OpenTimestamps adapter: it wraps
// github.com/nbd-wtf/opentimestamps to answer the one question the background
// upgrade loop, the .ots HTTP route, and certificate §5 BITCOIN ANCHOR all need —
// given serialized .ots proof bytes, is this root Bitcoin-confirmed yet, and at
// what block height? It parses an already-serialized proof and classifies it; it
// performs NO network I/O (no calendar submission, no proof upgrade, no Bitcoin
// RPC), so calendar/upgrade transport stays out of this leaf and the follower is
// never blocked (OTS is best-effort, ADR-0004). Submission/upgrade against a
// calendar belongs to the Upgrader closure that calls Confirmed.
//
// The verdict is fail-closed: a parse error — including a panic the library
// raises on some malformed inputs, recovered into an error here — is wrapped and
// returned (never a silent confirmed, never a crash), and a proof with no Bitcoin
// attestation classifies as pending — not an error — so a still-pending root
// re-polls instead of faulting.
// The Confirmed verdict is the milestone's `ots verify` oracle gate: it is pinned
// in tests to the OpenTimestamps ecosystem's own bundled .ots example vectors
// (real Bitcoin attestations), which are ground truth, not derived from this code.
//
// This is not a WASM-pure leaf like internal/proof/verify (OTS confirmation is
// server-side only): it imports only the library's parse/classify symbols, so its
// closure stays free of net/http and database/sql, but it must not be imported by
// any WASM-shared package.
package ots

import (
	"fmt"

	opentimestamps "github.com/nbd-wtf/opentimestamps"
)

// Confirmed classifies serialized OpenTimestamps proof bytes: it returns whether
// the stamped root has been upgraded to a Bitcoin-confirmed attestation and, when
// confirmed, the confirming Bitcoin block height. It parses the bytes with
// opentimestamps.ReadFromFile and inspects the Bitcoin-attested sequences:
//
//   - A proof terminating in a Bitcoin attestation → (true, <block height>, nil),
//     the height taken from the first attested sequence (a proof carries one root,
//     so the attested height is the same across sequences).
//   - A proof still terminating only in a calendar (pending) attestation →
//     (false, 0, nil). Still-pending is NOT an error: the upgrade loop records a
//     backed-off retry rather than faulting.
//   - Unparseable bytes (truncated, wrong magic, unsupported attestation type) →
//     (false, 0, wrapped error). Fail-closed: a parse failure never reports
//     confirmed.
//
// The returned int64 height matches follower.UpgradeResult.BTCHeight and
// store.OTSRecord.BTCHeight so the upgrade loop plugs in with no conversion seam;
// the library's uint64 height never overflows int64 for a real Bitcoin height.
//
// The parse is wrapped in recoverParse because opentimestamps.ReadFromFile reads
// the proof without bounds-checking and panics (slice out of range) on some
// malformed inputs instead of returning an error. Since these bytes can come from
// an untrusted .ots blob, that panic is converted into the same wrapped, fail-
// closed error so a garbage proof can never crash the upgrade-loop goroutine — OTS
// must never block the follower (ADR-0004).
func Confirmed(otsBytes []byte) (confirmed bool, height int64, err error) {
	file, parseErr := recoverParse(otsBytes)
	if parseErr != nil {
		return false, 0, fmt.Errorf("ots.Confirmed: parse: %w", parseErr)
	}
	attested := file.GetBitcoinAttestedSequences()
	if len(attested) == 0 {
		return false, 0, nil
	}
	att := attested[0].GetAttestation()
	return true, int64(att.BitcoinBlockHeight), nil
}

// recoverParse calls opentimestamps.ReadFromFile and converts any panic it raises
// on malformed input into an error. The library's parser over-reads its buffer on
// truncated / non-.ots bytes (a slice-bounds panic) rather than failing cleanly;
// recovering here keeps the fail-closed contract — a parse fault is always a
// returned error, never a crash — at this external-library boundary.
func recoverParse(otsBytes []byte) (file *opentimestamps.File, err error) {
	defer func() {
		if r := recover(); r != nil {
			file = nil
			err = fmt.Errorf("opentimestamps.ReadFromFile panicked: %v", r)
		}
	}()
	return opentimestamps.ReadFromFile(otsBytes)
}
