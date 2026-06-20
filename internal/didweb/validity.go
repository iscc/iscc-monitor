// This file enforces the CID 1.0 key validity window parsed by
// ParseDIDDocument. ParseDIDDocument surfaces the validFrom/validUntil/revoked
// timestamps but does no now-vs-window check; DIDKey.ValidAt is the pure
// predicate that decides whether a resolved key is valid at a given observation
// time. The follower's checkpoint-acceptance path consumes it: a signature
// whose verifying key is outside its validity window must not be `verified`
// (ADR-0009 — validity windows are the domain owner's rotation/revocation
// mechanism). It imports only time so it stays WASM-pure like the rest of
// internal/didweb.
package didweb

import "time"

// ValidAt reports whether the key is valid at observation time now.
//
// The window is the CID 1.0 wall-clock window parsed by ParseDIDDocument:
// active in the half-open interval [ValidFrom, ValidUntil) and not revoked
// before now. A zero-valued field means "no constraint" (matching parseTime
// and the live testnet docs that omit these fields, which are "currently
// valid"), so a zero-value DIDKey is always valid. The boundaries are
// half-open: now == ValidUntil is expired and now == Revoked is already
// revoked. now is taken as an argument so the predicate never reads the wall
// clock and stays deterministic; the caller (the follower) decides what an
// out-of-window key means for hub status.
func (k DIDKey) ValidAt(now time.Time) bool {
	if !k.ValidFrom.IsZero() && now.Before(k.ValidFrom) {
		return false
	}
	if !k.ValidUntil.IsZero() && !now.Before(k.ValidUntil) {
		return false
	}
	if !k.Revoked.IsZero() && !now.Before(k.Revoked) {
		return false
	}
	return true
}
