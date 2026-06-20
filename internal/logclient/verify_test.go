// Tests for the signed-note checkpoint verifier. They drive VerifyCheckpoint
// against the live sb0/sb1 checkpoint fixtures (testdata/live/) and their golden
// did:web-derived verifier keys, asserting the verified (origin, treeSize, root)
// without hard-coding the live size/root (the tree grows). Negative cases prove a
// non-matching key and a one-char-tampered root both surface as errors.Is(err,
// ErrUnverified); a separate parseCheckpointBody table proves a malformed body
// after a valid signature is a plain parse error, never ErrUnverified.
package logclient

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// sb0VKey is the golden verifier key for sb0.iscc.id/log, derived from sb0's
// current did:web key (z6Mkqb…) and matching derive_vkey.py byte-for-byte. It
// matches the signature on the captured sb0 checkpoint fixture.
const sb0VKey = "sb0.iscc.id/log+40b74463+AaV+ivnly67hhzQSQfGqCBP3PlOV2NBcmfGyzGdE2ZE5"

// sb1VKey is the verifier key derived from sb1's CURRENT live did:web key
// (z6Mkmwqg…, keyhash 069d0f14), which is what the captured sb1 checkpoint is
// signed with. sb1 rotated its key since the earlier did.json fixture (z6MkiNW…,
// keyhash 22b08f3e) was captured; that stale key is exercised as sb1StaleVKey
// below to prove key rotation surfaces as ErrUnverified.
const sb1VKey = "sb1.amlet.id/log+069d0f14+AW9UGZSxDvYFeewtbNU74zEMv12ChQPcuE4veN80nNtb"

// sb1StaleVKey is sb1's PRIOR did:web key (keyhash 22b08f3e), no longer the
// signer. Verifying the current sb1 checkpoint against it must yield ErrUnverified
// (a key the hub no longer signs with — a real-world rotation case).
const sb1StaleVKey = "sb1.amlet.id/log+22b08f3e+ATo2ruguSdJGh11PS76osrQf6OZKrufzzwH/HMwE3a8/"

// readCheckpoint loads a captured live checkpoint fixture from the module-root
// testdata/live/ directory (two levels up from this package).
func readCheckpoint(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "live", name))
	if err != nil {
		t.Fatalf("read checkpoint fixture %q: %v", name, err)
	}
	return data
}

// isZeroRoot reports whether every byte of root is zero.
func isZeroRoot(root [rootBytes]byte) bool {
	for _, b := range root {
		if b != 0 {
			return false
		}
	}
	return true
}

func TestVerifyCheckpoint(t *testing.T) {
	cases := []struct {
		name       string
		vkey       string
		fixture    string
		wantOrigin string
	}{
		{
			name:       "sb0",
			vkey:       sb0VKey,
			fixture:    "sb0.iscc.id_checkpoint",
			wantOrigin: "sb0.iscc.id/log",
		},
		{
			name:       "sb1",
			vkey:       sb1VKey,
			fixture:    "sb1.amlet.id_checkpoint",
			wantOrigin: "sb1.amlet.id/log",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw := readCheckpoint(t, tc.fixture)
			origin, treeSize, root, err := VerifyCheckpoint(tc.vkey, raw)
			if err != nil {
				t.Fatalf("VerifyCheckpoint(%q) error: %v", tc.fixture, err)
			}
			if origin != tc.wantOrigin {
				t.Errorf("origin = %q, want %q", origin, tc.wantOrigin)
			}
			// The tree grows, so assert >=1 rather than a fixed size.
			if treeSize < 1 {
				t.Errorf("treeSize = %d, want >= 1", treeSize)
			}
			if isZeroRoot(root) {
				t.Errorf("root is all zero, want 32 non-zero bytes")
			}
		})
	}
}

func TestVerifyCheckpointUnverified(t *testing.T) {
	sb0 := readCheckpoint(t, "sb0.iscc.id_checkpoint")
	sb1 := readCheckpoint(t, "sb1.amlet.id_checkpoint")

	// (a) sb0 checkpoint against the sb1 key: the sb1 signer name does not match the
	// "sb0.iscc.id/log" sig line, so note.Open returns "no verifiable signatures".
	t.Run("wrong hub key", func(t *testing.T) {
		_, _, _, err := VerifyCheckpoint(sb1VKey, sb0)
		if !errors.Is(err, ErrUnverified) {
			t.Errorf("error = %v, want errors.Is(err, ErrUnverified)", err)
		}
	})

	// (a') sb1 checkpoint against sb1's STALE (pre-rotation) key: the name matches
	// but the keyhash (22b08f3e) does not match the sig line's keyhash (069d0f14),
	// so no sig line is verifiable. A rotated-away key must surface as ErrUnverified.
	t.Run("stale rotated key", func(t *testing.T) {
		_, _, _, err := VerifyCheckpoint(sb1StaleVKey, sb1)
		if !errors.Is(err, ErrUnverified) {
			t.Errorf("error = %v, want errors.Is(err, ErrUnverified)", err)
		}
	})

	// (b) sb0 checkpoint with one base64 char flipped in the root line, against the
	// correct sb0 key: the verifier matches the sig line but the signature is now
	// invalid, so note.Open populates UnverifiedSigs.
	t.Run("tampered root", func(t *testing.T) {
		tampered := tamperRoot(t, sb0)
		_, _, _, err := VerifyCheckpoint(sb0VKey, tampered)
		if !errors.Is(err, ErrUnverified) {
			t.Errorf("error = %v, want errors.Is(err, ErrUnverified)", err)
		}
	})
}

// tamperRoot flips one character on the third (base64 root) line of a checkpoint.
//
// It keeps the signed-note framing intact so note.Open reaches the Ed25519 check
// and reports the signature as invalid (UnverifiedSigs), rather than failing to
// parse the note structure.
func tamperRoot(t *testing.T, raw []byte) []byte {
	t.Helper()
	out := make([]byte, len(raw))
	copy(out, raw)
	// Find the start of the third line (after the second newline).
	nl := 0
	idx := -1
	for i, b := range out {
		if b == '\n' {
			nl++
			if nl == 2 {
				idx = i + 1
				break
			}
		}
	}
	if idx < 0 || idx >= len(out) {
		t.Fatalf("tamperRoot: could not locate root line")
	}
	// Flip one base64 character ('A' <-> 'B') so the line stays valid base64 but
	// decodes to a different root, invalidating the signature.
	if out[idx] == 'A' {
		out[idx] = 'B'
	} else {
		out[idx] = 'A'
	}
	return out
}

func TestParseCheckpointBody(t *testing.T) {
	// A known-good 32-byte root encoded as standard base64.
	const goodRoot = "uir3z5T1yZVfmzWWPyagl0lifPjFdVxnGFquCSOOC8E="

	t.Run("valid", func(t *testing.T) {
		origin, size, root, err := parseCheckpointBody("sb0.iscc.id/log\n10183\n" + goodRoot + "\n")
		if err != nil {
			t.Fatalf("parseCheckpointBody error: %v", err)
		}
		if origin != "sb0.iscc.id/log" {
			t.Errorf("origin = %q, want sb0.iscc.id/log", origin)
		}
		if size != 10183 {
			t.Errorf("size = %d, want 10183", size)
		}
		if isZeroRoot(root) {
			t.Errorf("root is all zero")
		}
	})

	t.Run("zero size accepted", func(t *testing.T) {
		_, size, _, err := parseCheckpointBody("origin/log\n0\n" + goodRoot + "\n")
		if err != nil {
			t.Fatalf("parseCheckpointBody error: %v", err)
		}
		if size != 0 {
			t.Errorf("size = %d, want 0", size)
		}
	})

	// Malformed bodies must NOT map to ErrUnverified: the signature was valid, the
	// body is just unparseable. Keep the two concerns separable.
	bad := []struct {
		name string
		body string
	}{
		{"leading zero size", "origin/log\n01\n" + goodRoot + "\n"},
		{"non-numeric size", "origin/log\nabc\n" + goodRoot + "\n"},
		{"root not base64", "origin/log\n1\n!!!notbase64!!!\n"},
		{"root wrong length", "origin/log\n1\nYWJj\n"}, // "abc" -> 3 bytes
		{"too few lines", "origin/log\n1\n"},
	}
	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, err := parseCheckpointBody(tc.body)
			if err == nil {
				t.Fatalf("parseCheckpointBody(%q) = nil error, want parse error", tc.body)
			}
			if errors.Is(err, ErrUnverified) {
				t.Errorf("error = %v, want a plain parse error, not ErrUnverified", err)
			}
		})
	}
}
