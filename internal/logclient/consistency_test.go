// Tests for the pure shrink-, fork-, and equivocation-trigger detection. They
// drive CheckShrink, CheckFork, and CheckEquivocation over tables that pin every
// load-bearing boundary: a strict decrease is a shrink, a differing root at equal
// size is a fork, and a failing RFC-6962 consistency proof across a growing pair
// is an equivocation, while equal size (re-observation), growth, and the
// fresh-store prev==0 zero are otherwise neither — so the fresh
// FollowState{}.LastSize zero is never misread as a violation. The equivocation
// golden vector is built in-test from a real RFC-6962 tree (transparency-dev/
// merkle's testonly.Tree over rfc6962.DefaultHasher), so its roots and proof are
// ground truth from the hasher, never author-asserted magic values.
package logclient

import (
	"fmt"
	"testing"

	"github.com/transparency-dev/merkle/rfc6962"
	"github.com/transparency-dev/merkle/testonly"
)

func TestCheckShrink(t *testing.T) {
	cases := []struct {
		name string
		prev uint64
		next uint64
		want bool
	}{
		{name: "strict shrink", prev: 10183, next: 10182, want: true},
		{name: "equal size is re-observation not shrink", prev: 10183, next: 10183, want: false},
		{name: "growth", prev: 10183, next: 10184, want: false},
		{name: "fresh store no prior accepted size", prev: 0, next: 5, want: false},
		{name: "both zero", prev: 0, next: 0, want: false},
		{name: "shrink to zero", prev: 1, next: 0, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CheckShrink(tc.prev, tc.next); got != tc.want {
				t.Errorf("CheckShrink(%d, %d) = %v, want %v", tc.prev, tc.next, got, tc.want)
			}
		})
	}
}

// TestViolationShrinkKind pins the kind string this slice owns to the exact value
// the store.Violation.Kind field and the violations.kind column expect.
func TestViolationShrinkKind(t *testing.T) {
	if string(ViolationShrink) != "shrink" {
		t.Errorf("ViolationShrink = %q, want %q", ViolationShrink, "shrink")
	}
}

func TestCheckFork(t *testing.T) {
	var (
		rootA    = [rootBytes]byte{0: 0x01, 1: 0x01, 31: 0x01}
		rootB    = [rootBytes]byte{0: 0x02, 1: 0x02, 31: 0x02}
		zeroRoot [rootBytes]byte
	)
	if rootA == rootB {
		t.Fatal("test setup: rootA and rootB must differ")
	}
	cases := []struct {
		name     string
		prevSize uint64
		prevRoot [rootBytes]byte
		nextSize uint64
		nextRoot [rootBytes]byte
		want     bool
	}{
		{name: "same size different root", prevSize: 10183, prevRoot: rootA, nextSize: 10183, nextRoot: rootB, want: true},
		{name: "same size identical root is re-observation", prevSize: 10183, prevRoot: rootA, nextSize: 10183, nextRoot: rootA, want: false},
		{name: "shrink is shrinks concern not fork", prevSize: 10183, prevRoot: rootA, nextSize: 10182, nextRoot: rootB, want: false},
		{name: "growth even with differing root", prevSize: 10183, prevRoot: rootA, nextSize: 10184, nextRoot: rootB, want: false},
		{name: "fresh store no prior accepted size", prevSize: 0, prevRoot: zeroRoot, nextSize: 5, nextRoot: rootB, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CheckFork(tc.prevSize, tc.prevRoot, tc.nextSize, tc.nextRoot); got != tc.want {
				t.Errorf("CheckFork(%d, %x, %d, %x) = %v, want %v", tc.prevSize, tc.prevRoot, tc.nextSize, tc.nextRoot, got, tc.want)
			}
		})
	}
}

// TestViolationForkKind pins the fork kind string to the exact value the
// store.Violation.Kind field and the violations.kind column expect.
func TestViolationForkKind(t *testing.T) {
	if string(ViolationFork) != "fork" {
		t.Errorf("ViolationFork = %q, want %q", ViolationFork, "fork")
	}
}

// rootArray converts a hasher-produced 32-byte root slice into the [rootBytes]byte
// array the CheckEquivocation signature takes, failing the test on a wrong length.
func rootArray(t *testing.T, h []byte) [rootBytes]byte {
	t.Helper()
	if len(h) != rootBytes {
		t.Fatalf("root is %d bytes, want %d", len(h), rootBytes)
	}
	var a [rootBytes]byte
	copy(a[:], h)
	return a
}

// TestCheckEquivocation drives the merkle-backed equivocation trigger against a
// real RFC-6962 tree. testonly.Tree (over rfc6962.DefaultHasher) supplies the
// ground-truth roots at sizes M and N and a VALID consistency proof between them:
// the valid proof for the growing pair must verify (violated=false), while
// corrupting the next root or a proof element flips the verdict to violated=true.
// The fresh-store (prevSize==0), same-size (fork's), and shrinking (shrink's)
// boundaries return (false, nil) and must never reach VerifyConsistency.
func TestCheckEquivocation(t *testing.T) {
	const (
		sizeM = 7  // M: a non-perfect prior size, so the proof has interior nodes.
		sizeN = 11 // N > M: the growing case the trigger fires on.
	)
	tree := testonly.New(rfc6962.DefaultHasher)
	for i := range sizeN {
		tree.AppendData([]byte(fmt.Sprintf("leaf-%d", i)))
	}
	rootM := rootArray(t, tree.HashAt(sizeM))
	rootN := rootArray(t, tree.HashAt(sizeN))
	validProof, err := tree.ConsistencyProof(sizeM, sizeN)
	if err != nil {
		t.Fatalf("ConsistencyProof(%d, %d): %v", sizeM, sizeN, err)
	}
	if len(validProof) == 0 {
		t.Fatalf("ConsistencyProof(%d, %d) returned an empty proof", sizeM, sizeN)
	}

	// Non-vacuousness: the consistent and corrupted next roots must actually
	// differ, mirroring TestCheckFork's "rootA != rootB" guard, so the
	// consistent-vs-corrupted verdicts below are a real opposite, not a no-op.
	corruptRootN := rootN
	corruptRootN[0] ^= 0xff
	if corruptRootN == rootN {
		t.Fatal("test setup: corrupted rootN must differ from rootN")
	}

	// A corrupted proof element (flip a byte of the first hash) must also fail to
	// verify against the genuine roots.
	corruptProof := make([][]byte, len(validProof))
	for i, h := range validProof {
		c := make([]byte, len(h))
		copy(c, h)
		corruptProof[i] = c
	}
	corruptProof[0][0] ^= 0xff

	cases := []struct {
		name         string
		prevSize     uint64
		prevRoot     [rootBytes]byte
		nextSize     uint64
		nextRoot     [rootBytes]byte
		proof        [][]byte
		wantViolated bool
		wantErr      bool
	}{
		{name: "valid consistency proof for growing pair", prevSize: sizeM, prevRoot: rootM, nextSize: sizeN, nextRoot: rootN, proof: validProof, wantViolated: false},
		{name: "corrupted next root fails to verify", prevSize: sizeM, prevRoot: rootM, nextSize: sizeN, nextRoot: corruptRootN, proof: validProof, wantViolated: true},
		{name: "corrupted proof element fails to verify", prevSize: sizeM, prevRoot: rootM, nextSize: sizeN, nextRoot: rootN, proof: corruptProof, wantViolated: true},
		{name: "fresh store no prior accepted size", prevSize: 0, prevRoot: [rootBytes]byte{}, nextSize: sizeN, nextRoot: rootN, proof: validProof, wantViolated: false},
		{name: "same size is forks concern", prevSize: sizeN, prevRoot: rootN, nextSize: sizeN, nextRoot: corruptRootN, proof: validProof, wantViolated: false},
		{name: "shrink is shrinks concern", prevSize: sizeN, prevRoot: rootN, nextSize: sizeM, nextRoot: rootM, proof: validProof, wantViolated: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CheckEquivocation(tc.prevSize, tc.prevRoot, tc.nextSize, tc.nextRoot, tc.proof)
			if tc.wantErr != (err != nil) {
				t.Fatalf("CheckEquivocation err = %v, wantErr %v", err, tc.wantErr)
			}
			if got != tc.wantViolated {
				t.Errorf("CheckEquivocation = %v, want %v", got, tc.wantViolated)
			}
		})
	}
}

// TestCheckEquivocationBoundariesSkipVerify proves the non-growing boundaries
// return (false, nil) WITHOUT calling proof.VerifyConsistency: each is given a
// deliberately garbage proof and mismatched roots that VerifyConsistency would
// reject, yet the result stays (false, nil) because the growing guard short-
// circuits before the proof is ever consulted.
func TestCheckEquivocationBoundariesSkipVerify(t *testing.T) {
	garbage := [][]byte{{0x00}, {0x01, 0x02}}
	var rootA = [rootBytes]byte{0: 0x01, 31: 0x01}
	var rootB = [rootBytes]byte{0: 0x02, 31: 0x02}
	cases := []struct {
		name     string
		prevSize uint64
		nextSize uint64
	}{
		{name: "fresh store prevSize zero", prevSize: 0, nextSize: 5},
		{name: "same size fork boundary", prevSize: 7, nextSize: 7},
		{name: "shrink boundary", prevSize: 11, nextSize: 7},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CheckEquivocation(tc.prevSize, rootA, tc.nextSize, rootB, garbage)
			if err != nil {
				t.Fatalf("CheckEquivocation err = %v, want nil", err)
			}
			if got {
				t.Errorf("CheckEquivocation = true, want false (boundary must skip VerifyConsistency)")
			}
		})
	}
}

// TestViolationEquivocationKind pins the equivocation kind string to the exact
// value the store.Violation.Kind field and the violations.kind column expect.
func TestViolationEquivocationKind(t *testing.T) {
	if string(ViolationEquivocation) != "equivocation" {
		t.Errorf("ViolationEquivocation = %q, want %q", ViolationEquivocation, "equivocation")
	}
}
