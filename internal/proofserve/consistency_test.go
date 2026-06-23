// Conformance test for the computed-consistency-proof HTTP surface: it reuses the
// inclusion test's 300-leaf testonly.Tree mirror fixture (buildMirror) — the single
// source of truth for the tree's roots and proofs — records a prior accepted
// checkpoint at each tested from size, then drives proofserve.Handler's
// /consistency route over the real SQLiteFetcher. For from values straddling the
// 256-leaf tile boundary it asserts the served consistencyProof base64-decodes to
// bytes proof.VerifyConsistency ACCEPTS against the tree's two roots and byte-equals
// tree.ConsistencyProof, with a sharp wrong-from negative.
//
// Non-circularity / non-vacuousness (oracle gate APPLIES — this serves RFC-6962
// consistency crypto). Three independent merkle paths meet: the tree (the prover,
// owning the real consistency proof and both roots), ConsistencyProofFromTiles
// inside the handler (the monitor recompute over the mirror this test wrote), and
// proof.VerifyConsistency (the independent verifier). A green-but-wrong handler
// serving an empty or constant proof for a non-degenerate (from, larger) would fail
// proof.VerifyConsistency; the wrong-from negative pins that the proof for from does
// not verify against a different prior root.
package proofserve

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/transparency-dev/merkle/proof"
	"github.com/transparency-dev/merkle/rfc6962"

	"github.com/iscc/iscc-monitor/internal/dashboard"
	"github.com/iscc/iscc-monitor/internal/store"
)

// recordPriorCheckpoint records an accepted checkpoint at a prior tree size with
// the tree's real root at that size, so serveConsistency's CheckpointAt(from)
// finds a row. buildMirror records only the one checkpoint at the accepted size;
// the consistency route additionally requires a recorded checkpoint at from.
func recordPriorCheckpoint(t *testing.T, m mirrorTree, size uint64) {
	t.Helper()
	if _, _, err := m.store.RecordCheckpoint(context.Background(), store.CheckpointRecord{
		HubID:      m.hubID,
		TreeSize:   size,
		Root:       m.tree.HashAt(size),
		Raw:        []byte("checkpoint-at-prior"),
		ObservedAt: time.Unix(1700000000, 0),
	}); err != nil {
		t.Fatalf("RecordCheckpoint at size %d: %v", size, err)
	}
}

// getConsistency drives the handler's /consistency route with the given query
// string and decodes the JSON body, returning the status code and the parsed
// evidence (the evidence is the zero value on a non-200).
func getConsistency(t *testing.T, h http.Handler, query string) (int, ConsistencyEvidence) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/consistency?"+query, nil))
	if rec.Code != http.StatusOK {
		return rec.Code, ConsistencyEvidence{}
	}
	var ev ConsistencyEvidence
	if err := json.Unmarshal(rec.Body.Bytes(), &ev); err != nil {
		t.Fatalf("decode consistency body %q: %v", rec.Body.String(), err)
	}
	return rec.Code, ev
}

// decodeConsistency base64-Std-decodes the served consistencyProof hashes.
func decodeConsistency(t *testing.T, ev ConsistencyEvidence) [][]byte {
	t.Helper()
	out := make([][]byte, len(ev.ConsistencyProof))
	for i, enc := range ev.ConsistencyProof {
		h, err := base64.StdEncoding.DecodeString(enc)
		if err != nil {
			t.Fatalf("decode consistencyProof[%d] %q: %v", i, enc, err)
		}
		out[i] = h
	}
	return out
}

// fmtFrom renders a from query param.
func fmtFrom(from uint64) string {
	return "from=" + itoa(from)
}

// itoa renders an unsigned int in base 10 without importing strconv into the
// assertions (mirroring the handler's hand-rolled parseUint posture).
func itoa(n uint64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// TestConsistencyServedProofVerifies proves the handler serves a real RFC-6962
// consistency proof: for prior sizes straddling the 256-leaf tile boundary, the
// served proof decodes to bytes proof.VerifyConsistency ACCEPTS against the tree's
// prior and accepted roots, byte-equals the tree's own ConsistencyProof, and does
// NOT verify against a different prior root.
func TestConsistencyServedProofVerifies(t *testing.T) {
	m := buildMirror(t, mirrorLeaves)
	h := Handler(m.store, m.hubID, "sb0.iscc.id", nil, dashboard.Identity{})
	larger := m.size
	largerRoot := m.tree.HashAt(larger)

	for _, from := range []uint64{1, 200, 255, 256, 299} {
		recordPriorCheckpoint(t, m, from)
		fromRoot := m.tree.HashAt(from)

		code, ev := getConsistency(t, h, fmtFrom(from))
		if code != http.StatusOK {
			t.Fatalf("from %d: status = %d, want 200", from, code)
		}
		if ev.Type != "IsccLogConsistencyProof" {
			t.Errorf("from %d: type = %q, want IsccLogConsistencyProof", from, ev.Type)
		}
		if ev.FirstSize != from || ev.SecondSize != larger {
			t.Errorf("from %d: (firstSize, secondSize) = (%d, %d), want (%d, %d)", from, ev.FirstSize, ev.SecondSize, from, larger)
		}

		got := decodeConsistency(t, ev)

		// The served proof must byte-equal the tree's own consistency proof.
		want, err := m.tree.ConsistencyProof(from, larger)
		if err != nil {
			t.Fatalf("from %d: tree.ConsistencyProof: %v", from, err)
		}
		if len(got) != len(want) {
			t.Fatalf("from %d: proof len = %d, want %d", from, len(got), len(want))
		}
		for i := range want {
			if string(got[i]) != string(want[i]) {
				t.Errorf("from %d: proof[%d] = %x, want %x", from, i, got[i], want[i])
			}
		}

		// Note the arg order: proof precedes the two roots, UNLIKE VerifyInclusion.
		if err := proof.VerifyConsistency(rfc6962.DefaultHasher, from, larger, got, fromRoot, largerRoot); err != nil {
			t.Errorf("from %d: served proof does not verify: %v", from, err)
		}

		// The sharp negative: the same proof must NOT verify against a wrong prior
		// root (a green-but-wrong constant proof would also fail here).
		wrongRoot := m.tree.HashAt(from + 1)
		if err := proof.VerifyConsistency(rfc6962.DefaultHasher, from, larger, got, wrongRoot, largerRoot); err == nil {
			t.Errorf("from %d: served proof wrongly verifies against the wrong prior root", from)
		}
	}
}

// TestConsistencyDegenerateBoundaries proves the empty-proof boundaries are served
// as a 200 with an empty consistencyProof array (a valid degenerate proof), not a
// 400: from == 0 (the empty-tree prior, no checkpoint row required) and
// from == LastSize (the same-size roundtrip, which needs the accepted-size row that
// buildMirror already recorded).
func TestConsistencyDegenerateBoundaries(t *testing.T) {
	m := buildMirror(t, mirrorLeaves)
	h := Handler(m.store, m.hubID, "sb0.iscc.id", nil, dashboard.Identity{})

	for _, from := range []uint64{0, m.size} {
		code, ev := getConsistency(t, h, fmtFrom(from))
		if code != http.StatusOK {
			t.Fatalf("from %d: status = %d, want 200", from, code)
		}
		if ev.FirstSize != from || ev.SecondSize != m.size {
			t.Errorf("from %d: (firstSize, secondSize) = (%d, %d), want (%d, %d)", from, ev.FirstSize, ev.SecondSize, from, m.size)
		}
		if len(ev.ConsistencyProof) != 0 {
			t.Errorf("from %d: consistencyProof len = %d, want 0 (degenerate)", from, len(ev.ConsistencyProof))
		}
	}
}

// TestConsistencyMissingFrom asserts a request without from is a 400.
func TestConsistencyMissingFrom(t *testing.T) {
	m := buildMirror(t, 8)
	h := Handler(m.store, m.hubID, "sb0.iscc.id", nil, dashboard.Identity{})
	code, _ := getConsistency(t, h, "")
	if code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", code)
	}
}

// TestConsistencyNonNumericFrom asserts a non-numeric from is a 400.
func TestConsistencyNonNumericFrom(t *testing.T) {
	m := buildMirror(t, 8)
	h := Handler(m.store, m.hubID, "sb0.iscc.id", nil, dashboard.Identity{})
	code, _ := getConsistency(t, h, "from=abc")
	if code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", code)
	}
}

// TestConsistencyFromExceedsAccepted asserts a from larger than the accepted tree
// size is a 400 (RFC-6962 requires M <= N).
func TestConsistencyFromExceedsAccepted(t *testing.T) {
	m := buildMirror(t, 8)
	h := Handler(m.store, m.hubID, "sb0.iscc.id", nil, dashboard.Identity{})
	code, _ := getConsistency(t, h, fmtFrom(m.size+1))
	if code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", code)
	}
}

// TestConsistencyNoAcceptedCheckpoint asserts that with no accepted checkpoint yet
// (LastSize == 0) the route is a 404 — there is nothing to relate the prior root to.
func TestConsistencyNoAcceptedCheckpoint(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/empty.db")
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	hubID, err := st.UpsertHub(context.Background(), "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	h := Handler(st, hubID, "sb0.iscc.id", nil, dashboard.Identity{})
	code, _ := getConsistency(t, h, fmtFrom(1))
	if code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", code)
	}
}

// TestConsistencyUnknownFrom asserts a from with no recorded checkpoint row is a
// 404 (the client's prior root must be one the monitor recorded). buildMirror
// records a checkpoint only at the accepted size, so a mid-tree from has no row.
func TestConsistencyUnknownFrom(t *testing.T) {
	m := buildMirror(t, mirrorLeaves)
	h := Handler(m.store, m.hubID, "sb0.iscc.id", nil, dashboard.Identity{})
	code, _ := getConsistency(t, h, fmtFrom(100)) // no checkpoint recorded at 100
	if code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", code)
	}
}

// TestConsistencyNonGET asserts a non-GET method is a 405.
func TestConsistencyNonGET(t *testing.T) {
	m := buildMirror(t, 8)
	h := Handler(m.store, m.hubID, "sb0.iscc.id", nil, dashboard.Identity{})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/consistency?from=1", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}
