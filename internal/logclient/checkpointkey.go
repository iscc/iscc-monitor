// This file recovers the signed-note (name, keyhash) directly from a raw
// checkpoint's signature line, WITHOUT resolving the hub's did.json. The verified
// PollHub path can then consult store.LookupHubKey(hubID, keyID) before
// re-resolving the key, breaking the chicken-and-egg where today the only source
// of the cache-lookup key id is ResolveVerifierKey — the very fetch we want to
// skip. It is the raw-checkpoint counterpart to keyid.go, which recovers the same
// keyhash from the vkey STRING after resolution; both yield the same uint32 for
// the same hub (the vkey's middle "+<hex>+" field IS this sig-line keyhash).
//
// The keyhash is not re-parsed by hand: note.Open owns the signed-note framing and
// already decodes each sig line's leading BE-uint32 keyhash. With an empty
// verifier list no sig can verify, so every sig line lands in UnverifiedSigs and
// Open returns *note.UnverifiedNoteError; the embedded note exposes the decoded
// Name and Hash. The file is pure (stdlib + sumdb/note only, no net/os/sqlite).
package logclient

import (
	"errors"
	"fmt"

	"golang.org/x/mod/sumdb/note"
)

// KeyIDFromCheckpoint recovers the signer name and signed-note key id from raw.
//
// It opens raw with an empty verifier list, so note.Open lands every signature
// line in UnverifiedSigs and returns *note.UnverifiedNoteError; the first
// unverified sig carries the signer name and the key id note already decoded from
// the sig line (the BE-uint32 keyhash). The returned keyID equals
// KeyIDFromVerifier of the hub's resolved vkey, so a caller can look up the cached
// key by id before re-resolving did.json. A genuinely malformed note (any error
// that is not *note.UnverifiedNoteError, or a note with zero unverified sigs) is
// wrapped and returned, never panicked.
//
// v1 checkpoints carry a single hub signature, so the first unverified sig is the
// hub key. A future M7 cosigner could add a second unverified sig line; selecting
// among multiple unverified sigs is out of scope here and would need its own logic.
func KeyIDFromCheckpoint(raw []byte) (name string, keyID uint32, err error) {
	_, err = note.Open(raw, note.VerifierList())
	var ue *note.UnverifiedNoteError
	if !errors.As(err, &ue) {
		// A non-UnverifiedNoteError (including a nil error, unreachable with an empty
		// verifier list) means the bytes are not a well-framed signed note.
		return "", 0, fmt.Errorf("key id from checkpoint: %w", err)
	}
	if len(ue.Note.UnverifiedSigs) < 1 {
		return "", 0, fmt.Errorf("key id from checkpoint: note carries no signature line")
	}
	sig := ue.Note.UnverifiedSigs[0]
	return sig.Name, sig.Hash, nil
}
