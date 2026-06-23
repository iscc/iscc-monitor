// This file reads the tree size from a raw hub-signed checkpoint note WITHOUT
// re-verifying its signature. The dossier Exhibit uses it to display the two
// contradictory checkpoints' sizes (size before → presented): the stored RawA /
// RawB are already signature-verified evidence (the follower freeze path persists
// them only after AcceptCheckpoint), so the Exhibit only needs to read back each
// body's claimed size — it must never pull a vkey / did:web resolution into the
// dossier just to display a number.
//
// It is the size-reading counterpart to KeyIDFromCheckpoint, which recovers the
// signer name + keyhash from the same unverified-but-framed read: note.Open with
// an EMPTY verifier list lands every sig line in UnverifiedSigs and returns
// *note.UnverifiedNoteError whose .Note.Text is the checkpoint body. The line-2
// size parse mirrors VerifyCheckpoint's parseCheckpointBody (reject leading zeros
// except "0", strconv.ParseUint base 10). The file is pure (stdlib + sumdb/note
// only, no net/os/sqlite), so it stays WASM-shareable like its siblings.
package logclient

import (
	"errors"
	"strconv"
	"strings"

	"golang.org/x/mod/sumdb/note"
)

// CheckpointSizeFromRaw reads the tree size from a raw checkpoint note, returning
// ok=false on any malformed or non-note input.
//
// It opens raw with an empty verifier list so note.Open lands every signature line
// in UnverifiedSigs and returns *note.UnverifiedNoteError; the embedded note's Text
// is the checkpoint body. The tree size is body line 2 (decimal, no leading zeros
// except "0"), parsed exactly as VerifyCheckpoint's parseCheckpointBody does. It
// does NOT verify the signature — the caller (the dossier Exhibit) reads back
// already-verified evidence and only displays the size each checkpoint claimed.
//
// It fails closed: any input that is not a well-framed signed note, or a body with
// fewer than two lines, or a size that is not a clean non-negative decimal, returns
// (0, false) so the Exhibit degrades to an honest "size unavailable" rather than
// fabricating a number.
func CheckpointSizeFromRaw(raw []byte) (treeSize uint64, ok bool) {
	_, err := note.Open(raw, note.VerifierList())
	var ue *note.UnverifiedNoteError
	if !errors.As(err, &ue) {
		// A non-UnverifiedNoteError (including a nil error, unreachable with an empty
		// verifier list) means the bytes are not a well-framed signed note.
		return 0, false
	}
	lines := strings.Split(ue.Note.Text, "\n")
	if len(lines) < 2 {
		return 0, false
	}
	sizeStr := lines[1]
	if sizeStr != "0" && strings.HasPrefix(sizeStr, "0") {
		return 0, false
	}
	size, err := strconv.ParseUint(sizeStr, 10, 64)
	if err != nil {
		return 0, false
	}
	return size, true
}
