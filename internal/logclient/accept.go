// This file composes the three pure verification primitives — ResolveVerifierKey
// (did:web key), VerifyCheckpoint (signed-note Ed25519 check), and DIDKey.ValidAt
// (CID 1.0 validity window) — into one decision the stateful follower will
// consume: the four-way hub-status verdict for a single observed checkpoint
// (ADR-0009). It is pure and dependency-injected through the Fetcher seam (no
// SQLite, no persistence, no clock read), so the verdict is deterministic and
// table-testable; the store and the follower loop land in later steps and call
// AcceptCheckpoint.
package logclient

import (
	"context"
	"errors"
	"time"

	"github.com/iscc/iscc-monitor/internal/didweb"
)

// Status is AcceptCheckpoint's four-way verdict for one observed checkpoint.
//
// It maps onto the ADR-0009 hub-status taxonomy. StatusVerified is the only
// outcome that should advance accepted state; the other three are reported,
// while the hub is still mirrored. StatusRotated is the distinct
// rotation/revocation case (a good signature by an out-of-window key) and is
// deliberately neither StatusUnverified nor StatusUnresolvable.
type Status int

const (
	// StatusVerified: signature matches the hub's did:web key and the key is
	// within its CID 1.0 validity window at the observation time.
	StatusVerified Status = iota
	// StatusUnverified: the signature matches no key the hub's did:web document
	// lists (an internally-broken hub).
	StatusUnverified
	// StatusUnresolvable: the hub's did:web document could not be fetched, parsed,
	// or derived into a key (including a malformed validity timestamp).
	StatusUnresolvable
	// StatusRotated: the signature is valid but the signing key is outside its CID
	// 1.0 validity window (rotation/revocation) at the observation time.
	StatusRotated
)

// String returns the lowercase status name for logs and tests.
func (s Status) String() string {
	switch s {
	case StatusVerified:
		return "verified"
	case StatusUnverified:
		return "unverified"
	case StatusUnresolvable:
		return "unresolvable"
	case StatusRotated:
		return "rotated"
	default:
		return "unknown"
	}
}

// CheckpointInfo carries the verified checkpoint identity on StatusVerified.
//
// It is the zero value for every non-verified verdict, since the (origin,
// treeSize, root) are only trustworthy once the signature verified under an
// in-window key.
type CheckpointInfo struct {
	Origin   string
	TreeSize uint64
	Root     [rootBytes]byte
}

// VerifiedContext carries the did:web verifier-key context AcceptCheckpoint
// resolved for one verified poll: the "<name>+<keyid>+<base64>" verifier-key
// string and the parsed DIDKey (public key + CID 1.0 validity window) that the
// signature just verified under.
//
// It is populated only on StatusVerified and is the zero value for every
// non-verified verdict, mirroring CheckpointInfo. The follower reuses it to cache
// the hub key and rebuild the mirror root within the same poll WITHOUT re-fetching
// did.json — the key was resolved and validity-checked this poll (ADR-0009: reuse
// only within one verified poll, never cache across polls).
type VerifiedContext struct {
	VKey string
	Key  didweb.DIDKey
}

// AcceptCheckpoint decides the hub-status verdict for one observed checkpoint.
//
// It composes the three pure primitives in a load-bearing order:
//  1. ResolveVerifierKey — an ErrUnresolvable (unreachable/unparseable did.json,
//     including a malformed validity timestamp) yields StatusUnresolvable.
//  2. VerifyCheckpoint — an ErrUnverified (signature matches no listed key) yields
//     StatusUnverified. A well-signed-but-malformed body is a real error and is
//     returned as a wrapped error, never folded into a status.
//  3. DIDKey.ValidAt(observedAt) — only after a good signature: in-window yields
//     StatusVerified (with CheckpointInfo), out-of-window yields StatusRotated.
//
// observedAt is passed in (never time.Now() here) so the decision is
// deterministic and table-testable, mirroring DIDKey.ValidAt's discipline. On any
// status outcome the returned error is nil; a non-nil error is reserved for a
// genuine fault (a verified-but-garbled body) and pairs with the zero
// CheckpointInfo and StatusUnverified's zero value — callers must check err first.
//
// On StatusVerified it also returns a populated VerifiedContext (the resolved
// verifier-key string + DIDKey it just verified under, already validity-checked
// this poll); on every non-verified verdict the VerifiedContext is the zero value,
// mirroring CheckpointInfo. The follower reuses the context to cache the hub key
// and rebuild the mirror root within the same poll without a second did.json fetch.
func AcceptCheckpoint(ctx context.Context, fetcher Fetcher, baseURL string, raw []byte, observedAt time.Time) (Status, CheckpointInfo, VerifiedContext, error) {
	vkey, key, err := ResolveVerifierKey(ctx, fetcher, baseURL)
	if err != nil {
		if errors.Is(err, ErrUnresolvable) {
			return StatusUnresolvable, CheckpointInfo{}, VerifiedContext{}, nil
		}
		return StatusUnverified, CheckpointInfo{}, VerifiedContext{}, err
	}
	origin, treeSize, root, err := VerifyCheckpoint(vkey, raw)
	if err != nil {
		if errors.Is(err, ErrUnverified) {
			return StatusUnverified, CheckpointInfo{}, VerifiedContext{}, nil
		}
		// A valid signature over a malformed body: a real fault, kept separable
		// from "signature didn't match" exactly as VerifyCheckpoint does.
		return StatusUnverified, CheckpointInfo{}, VerifiedContext{}, err
	}
	if !key.ValidAt(observedAt) {
		return StatusRotated, CheckpointInfo{}, VerifiedContext{}, nil
	}
	return StatusVerified, CheckpointInfo{Origin: origin, TreeSize: treeSize, Root: root}, VerifiedContext{VKey: vkey, Key: key}, nil
}
