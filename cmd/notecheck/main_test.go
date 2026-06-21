// Tests for the notecheck signature-parity oracle. They drive the pure run helper
// (no subprocess, no os.Exit) against the live sb0 checkpoint fixture and its
// golden did:web-derived verifier key, asserting run accepts the genuine signed
// checkpoint and rejects a one-byte-corrupted body. A bad-vkey case covers the
// NewVerifier (exit-2) branch. The embedded vkey is reproducible from the
// fully-independent .claude/derive_vkey.py oracle.
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sb0VKey is the golden verifier key for sb0.iscc.id/log, derived from sb0's
// current did:web key and matching .claude/derive_vkey.py byte-for-byte. Its
// middle "+40b74463+" field is the signed-note keyhash of the captured fixture.
const sb0VKey = "sb0.iscc.id/log+40b74463+AaV+ivnly67hhzQSQfGqCBP3PlOV2NBcmfGyzGdE2ZE5"

// readCheckpoint loads the captured live sb0 checkpoint fixture from the
// module-root testdata/live/ directory (two levels up from this package).
func readCheckpoint(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "live", name))
	if err != nil {
		t.Fatalf("read checkpoint fixture %q: %v", name, err)
	}
	return data
}

func TestNotecheckGolden(t *testing.T) {
	raw := readCheckpoint(t, "sb0.iscc.id_checkpoint")
	var out bytes.Buffer
	name, err := run(sb0VKey, bytes.NewReader(raw), &out)
	if err != nil {
		t.Fatalf("run(sb0VKey, live fixture) error: %v", err)
	}
	if name != "sb0.iscc.id/log" {
		t.Errorf("name = %q, want sb0.iscc.id/log", name)
	}
}

func TestNotecheckRejectsCorruptedBody(t *testing.T) {
	raw := readCheckpoint(t, "sb0.iscc.id_checkpoint")
	// Flip one base64 char on the signature line so the note still parses but the
	// Ed25519 signature no longer validates — note.Open then populates
	// UnverifiedSigs and the strict reject must fire. A green-but-wrong oracle that
	// accepts anything is worthless.
	corrupt := corruptSignatureLine(t, raw)
	if bytes.Equal(corrupt, raw) {
		t.Fatal("corruptSignatureLine did not change the body")
	}
	var out bytes.Buffer
	_, err := run(sb0VKey, bytes.NewReader(corrupt), &out)
	if err == nil {
		t.Fatal("run(corrupted body) = nil error, want a reject")
	}
}

func TestNotecheckBadVKey(t *testing.T) {
	raw := readCheckpoint(t, "sb0.iscc.id_checkpoint")
	// An empty/garbage vkey must fail at the NewVerifier leg (the exit-2 setup
	// path), independently of the checkpoint body.
	for _, vkey := range []string{"", "not-a-valid-vkey"} {
		var out bytes.Buffer
		_, err := run(vkey, bytes.NewReader(raw), &out)
		if err == nil {
			t.Errorf("run(%q) = nil error, want a NewVerifier error", vkey)
			continue
		}
		if exitCode(err) != 2 {
			t.Errorf("run(%q): exitCode = %d, want 2 (setup error)", vkey, exitCode(err))
		}
	}
}

// corruptSignatureLine flips one base64 character on the last line of a
// checkpoint (the C2SP "— <name> <base64sig>" line), keeping the note framing
// intact so note.Open reaches the Ed25519 check and reports the signature invalid.
func corruptSignatureLine(t *testing.T, raw []byte) []byte {
	t.Helper()
	text := string(raw)
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	last := lines[len(lines)-1]
	// The sig line is "— <name> <base64sig>"; flip a char near the end of the
	// base64 signature so it stays valid base64 but decodes to a different sig.
	idx := len(last) - 5
	if idx < 0 || idx >= len(last) {
		t.Fatalf("corruptSignatureLine: signature line too short: %q", last)
	}
	b := []byte(last)
	if b[idx] == 'A' {
		b[idx] = 'B'
	} else {
		b[idx] = 'A'
	}
	lines[len(lines)-1] = string(b)
	return []byte(strings.Join(lines, "\n") + "\n")
}
