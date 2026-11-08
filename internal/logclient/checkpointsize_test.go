// Tests for CheckpointSizeFromRaw, the unverified tree-size reader the dossier
// Exhibit uses. The positive case grounds the size against ground truth: the
// captured sb0 checkpoint fixture's size byte-equals what VerifyCheckpoint (the
// signature-verified path) returns for the same bytes, so the two parsers agree.
// Negative cases prove fail-closed on non-note bytes, a missing size line, and a
// leading-zero size — never a fabricated number.
package logclient

import (
	"testing"

	"golang.org/x/mod/sumdb/note"
)

func TestCheckpointSizeFromRaw(t *testing.T) {
	raw := readCheckpoint(t, "sb0.iscc.id_checkpoint")

	size, ok := CheckpointSizeFromRaw(raw)
	if !ok {
		t.Fatalf("CheckpointSizeFromRaw(sb0 fixture) ok = false, want true")
	}
	if size < 1 {
		t.Errorf("size = %d, want >= 1", size)
	}
	// Ground truth: the unverified read must equal the signature-verified parser's
	// tree size for the same bytes (the two are independent line-2 parses).
	_, wantSize, _, err := VerifyCheckpoint(sb0VKey, raw)
	if err != nil {
		t.Fatalf("VerifyCheckpoint(sb0 fixture): %v", err)
	}
	if size != wantSize {
		t.Errorf("CheckpointSizeFromRaw size = %d, want %d (VerifyCheckpoint)", size, wantSize)
	}
}

func TestCheckpointSizeFromRawFailClosed(t *testing.T) {
	cases := []struct {
		name string
		raw  []byte
	}{
		{"non-note bytes", []byte("not-a-note")},
		{"empty", nil},
		{"single byte", []byte("a")},
		// A well-framed note whose size line carries a leading zero must fail closed
		// (mirrors parseCheckpointBody's reject), so the Exhibit never shows "01".
		{"leading-zero size", buildNote(t, "sb0.iscc.id/log\n01\n"+rootB64+"\n")},
		// A body with fewer than two lines has no size to read.
		{"one-line body", buildNote(t, "sb0.iscc.id/log\n")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if size, ok := CheckpointSizeFromRaw(tc.raw); ok {
				t.Errorf("CheckpointSizeFromRaw(%s) = (%d, true), want (0, false)", tc.name, size)
			}
		})
	}
}

// rootB64 is a valid 32-byte base64-Std root used to frame synthetic checkpoint
// bodies; its exact value is irrelevant since these tests never verify a signature.
const rootB64 = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

// buildNote frames body as a signed note signed by an ephemeral test key so
// note.Open accepts the framing; CheckpointSizeFromRaw reads it WITHOUT verifying
// the signature, so the key need not match any hub. It returns the raw note bytes.
func buildNote(t *testing.T, body string) []byte {
	t.Helper()
	skey, _, err := note.GenerateKey(nil, "sb0.iscc.id/log")
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	signer, err := note.NewSigner(skey)
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	signed, err := note.Sign(&note.Note{Text: body}, signer)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	return signed
}
