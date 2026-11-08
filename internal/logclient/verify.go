// This file verifies a hub-signed C2SP tlog-checkpoint as a signed note: given
// the verifier key string ResolveVerifierKey produces and the raw checkpoint
// bytes, it confirms the Ed25519 signature via golang.org/x/mod/sumdb/note and
// returns the signed (origin, treeSize, root). A signature that does not match
// the hub's resolved did:web key maps to ErrUnverified (an internally-broken hub,
// ADR-0009) — distinct from the resolution failure ErrUnresolvable in
// didresolve.go. Verification is reused, not reimplemented: note.Open owns the
// signed-note framing and Ed25519 check; this file only parses the verified body.
//
// VerifyCheckpoint is pure (sumdb/note + stdlib only, no net/os/sqlite), so it
// stays shareable with the WASM verify path; the follower seam, not WASM, is its
// production consumer.
package logclient

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/mod/sumdb/note"
)

// ErrUnverified marks a checkpoint whose signature does not match the hub's
// resolved did:web key (an internally-broken hub). Distinct from ErrUnresolvable.
//
// The follower checks errors.Is(err, ErrUnverified) to set hub status
// "unverified". It covers both note.Open failure shapes that mean "not signed by
// the resolved key": an invalid signature for the listed key, and a checkpoint
// carrying no signature line for the listed key at all.
var ErrUnverified = errors.New("checkpoint signature unverified")

// rootBytes is the SHA-256 Merkle tree head length (32 bytes).
const rootBytes = 32

// VerifyCheckpoint verifies raw against vkey and returns the signed checkpoint.
//
// vkey is exactly the string ResolveVerifierKey returns. On success it returns
// the verified body's (origin, treeSize, root); origin equals the signed-note
// signer name. A signature that does not match the key wraps ErrUnverified; a
// well-signed but malformed body wraps a plain parse error (never ErrUnverified),
// keeping the two concerns separable.
func VerifyCheckpoint(vkey string, raw []byte) (origin string, treeSize uint64, root [rootBytes]byte, err error) {
	v, err := note.NewVerifier(vkey)
	if err != nil {
		return "", 0, root, fmt.Errorf("verify checkpoint: bad vkey: %w", err)
	}
	n, err := note.Open(raw, note.VerifierList(v))
	if err != nil {
		// note.Open returns "no verifiable signatures" when no sig line matches the
		// verifier's name/keyhash; either way the checkpoint is not signed by the
		// resolved key.
		return "", 0, root, fmt.Errorf("verify checkpoint: %v: %w", err, ErrUnverified)
	}
	if len(n.Sigs) == 0 || len(n.UnverifiedSigs) != 0 {
		// A listed verifier matched a sig line but the signature did not validate.
		return "", 0, root, fmt.Errorf("verify checkpoint: verified=%d unverified=%d: %w", len(n.Sigs), len(n.UnverifiedSigs), ErrUnverified)
	}
	origin, treeSize, root, err = parseCheckpointBody(n.Text)
	if err != nil {
		return "", 0, root, err
	}
	// The signed-note name MUST equal the checkpoint's first body line (origin);
	// a mismatch means the signed note is not the checkpoint it claims to be.
	if n.Sigs[0].Name != origin {
		return "", 0, root, fmt.Errorf("verify checkpoint: signer name %q != origin %q", n.Sigs[0].Name, origin)
	}
	return origin, treeSize, root, nil
}

// parseCheckpointBody parses a verified checkpoint body into its three fields.
//
// The body is exactly "<origin>\n<tree_size>\n<base64(root)>\n" (C2SP
// tlog-checkpoint). tree_size is decimal, non-negative, with no leading zeros
// (reject "01", accept "0"); root is standard base64 and exactly 32 bytes. It
// returns a plain error (not ErrUnverified) on any malformed field, since the
// signature itself was valid.
func parseCheckpointBody(text string) (origin string, treeSize uint64, root [rootBytes]byte, err error) {
	lines := strings.Split(text, "\n")
	if len(lines) < 3 {
		return "", 0, root, fmt.Errorf("parse checkpoint: body has %d lines, want >=3", len(lines))
	}
	origin = lines[0]
	sizeStr := lines[1]
	if sizeStr != "0" && strings.HasPrefix(sizeStr, "0") {
		return "", 0, root, fmt.Errorf("parse checkpoint: tree size %q has leading zeros", sizeStr)
	}
	treeSize, err = strconv.ParseUint(sizeStr, 10, 64)
	if err != nil {
		return "", 0, root, fmt.Errorf("parse checkpoint: tree size %q: %w", sizeStr, err)
	}
	decoded, err := base64.StdEncoding.DecodeString(lines[2])
	if err != nil {
		return "", 0, root, fmt.Errorf("parse checkpoint: root not base64: %w", err)
	}
	if len(decoded) != rootBytes {
		return "", 0, root, fmt.Errorf("parse checkpoint: root is %d bytes, want %d", len(decoded), rootBytes)
	}
	copy(root[:], decoded)
	return origin, treeSize, root, nil
}
