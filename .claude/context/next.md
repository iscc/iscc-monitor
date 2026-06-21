# Next Work Package

## Step: Land the pure inclusion cross-check `VerifyInclusionEvidence` (decode hub `IsccLogInclusionProof` → compute from tiles → byte-compare)

## Goal
Build the pure, golden-testable core of M2's SECOND Verify half: a function that decodes a hub's
`IsccLogInclusionProof` and asserts the monitor's own `InclusionProofFromTiles` byte-equals the
hub-supplied proof for that leaf. This is `InclusionProofFromTiles`'s first production-shaped caller
and the second true *external* oracle (the hub-computed proof), kept pure so the `PollHub` wiring +
`iscc_index` lookup can follow in a later, separate slice.

## Scope
- **Create**: `internal/logclient/inclusioncheck.go` — the decode + cross-check function.
- **Create**: `internal/logclient/inclusioncheck_test.go` — golden + mutation test (test, not counted).
- **Modify**: (none — strictly additive; no existing production file changes)
- **Reference**:
  - `/workspace/iscc-monitor/cauldron/iscc-hub/iscc_hub/log_tree.py` lines ~208-232
    (`inclusion_evidence`) — the EXACT `IsccLogInclusionProof` shape the hub emits:
    `{type, checkpoint, treeSize, leafIndex, inclusionProof: [base64(h) for h in proof]}`.
    `base64.b64encode` = **standard** padding (`base64.StdEncoding`).
  - `/workspace/iscc-monitor/cauldron/iscc-hub/iscc_hub/schema.py` lines ~193-206 (`Evidence` model) —
    field names/types (`treeSize ge=1`, `leafIndex ge=0`, `inclusionProof list[str]`).
  - `/workspace/iscc-monitor/internal/logclient/proofbuilder.go` lines 86-127 —
    `InclusionProofFromTiles(ctx, fetch, index, size)` and the `TileFetcher` type (line 39):
    `func(ctx, level, index uint64, p uint8) ([]byte, error)`, byte-identical to
    `store.SQLiteFetcher.ReadTile`.
  - `/workspace/iscc-monitor/internal/logclient/inclusionproof_test.go` — the `testonly.Tree` +
    `tileFetcherFor` + `equalProof` golden pattern to reuse (same package).
  - `/workspace/iscc-monitor/internal/logclient/verify.go` line 16 — `encoding/base64` is already
    imported in this package (precedent for stdlib `StdEncoding`).

## Not In Scope
- **Do NOT wire this into `PollHub`/`follower.go`.** That needs the `iscc_index` writer to resolve a
  sampled `iscc_id` → `leafIndex`, plus bundle fixtures — its own later slice.
- **Do NOT add the `iscc_index` projection writer** (the schema-agnostic `iscc_id → seq` table is a
  separate M2 item; this step does not read or write it).
- **Do NOT re-verify the embedded `checkpoint` signature here** — this function cross-checks the
  *proof hashes* only; signature + treeSize/root verification is already `AcceptCheckpoint`'s job. You
  MAY decode `treeSize`/`leafIndex` from the JSON, but do not re-run `note.Open` on the bundled
  checkpoint string.
- **Do NOT capture live sb0/sb1 tiles or real `IsccLogInclusionProof` fixtures.** Use the in-process
  `testonly.Tree` mirror, consistent with the fsck slice precedent (handoff 2026-06-21: live leaf
  preimages were never captured).
- **Do NOT touch `go.mod`/`go.sum`/`schema.sql`** — `encoding/json` + `encoding/base64` are stdlib;
  `InclusionProofFromTiles`, `proof`, `tessera/api` are already in the package closure.

## Implementation Notes
- Define an exported struct mirroring the hub VC member, e.g.:
  ```go
  type InclusionEvidence struct {
      Type           string   `json:"type"`
      Checkpoint     string   `json:"checkpoint"`
      TreeSize       uint64   `json:"treeSize"`
      LeafIndex      uint64   `json:"leafIndex"`
      InclusionProof []string `json:"inclusionProof"`
  }
  ```
- Add `ParseInclusionEvidence(raw []byte) (InclusionEvidence, error)` — a thin `json.Unmarshal`,
  reject a wrong `Type` (`!= "IsccLogInclusionProof"`) and `TreeSize == 0`. Match the
  `ParseDIDDocument` style (`%w`-wrapped errors).
- Add the cross-check `VerifyInclusionEvidence(ctx context.Context, fetch TileFetcher, ev
  InclusionEvidence) error`:
  1. Guard `ev.LeafIndex < ev.TreeSize` — else a clear non-nil error WITHOUT reaching the fetcher
     (same posture as `TestInclusionProofFromTilesIndexOutOfRange`).
  2. `got, err := InclusionProofFromTiles(ctx, fetch, ev.LeafIndex, ev.TreeSize)` — propagate `%w`
     (a missing tile must keep `errors.Is(err, os.ErrNotExist)`, like the builder's own tests).
  3. base64-**Std**-decode each `ev.InclusionProof[i]` → `[]byte` (mirror `inclusion_evidence`'s
     `base64.b64encode`); a bad base64 element → wrapped error.
  4. Compare lengths first, then each hash with `bytes.Equal`. On any mismatch return a **descriptive
     sentinel** (package var `ErrInclusionMismatch`) wrapped with context so the future `PollHub`
     caller can `errors.Is` on it; on full match return `nil`.
- Keep the file **net-free at the file level** (imports exactly `bytes`/`context`/`encoding/base64`/
  `encoding/json`/`errors`/`fmt`). The package already pulls `net/http` via `didresolve.go`, so the
  load-bearing purity invariant is the `internal/didweb` WASM build (unaffected), NOT this package —
  do not try to make `go list -deps ./internal/logclient | grep net/http` empty (it can't be, and
  that is documented in learnings).
- **Correctness rule (learnings — oracle gate APPLIES, RFC-6962 inclusion crypto).** Make the golden
  non-circular: build the proof with `testonly.Tree.InclusionProof(index, treeLeaves)` (the hub's
  role), base64-Std-encode it into an `InclusionEvidence` exactly as `log_tree.py` does, serve tiles
  via the existing `tileFetcherFor`, and assert `VerifyInclusionEvidence` returns `nil` across the
  256-leaf boundary (reuse indices like `{0, 5, 255, 256, 299}` against the 300-leaf `testonly.Tree`).
  Prover (`testonly.Tree.InclusionProof`), tile-builder (`InclusionProofFromTiles`), and the
  base64+bytes compare are independent paths → not a tautology.
- **Mutation / non-vacuity (REQUIRED).** Add a negative subtest: flip one byte of one
  `ev.InclusionProof` hash (or pass a wrong `LeafIndex` with the right proof) and assert
  `errors.Is(err, ErrInclusionMismatch)`. A green-but-wrong check that ignored the proof bytes must
  FAIL this — proving the byte-comparison is load-bearing.
- Note `proof.VerifyInclusion`'s arg order is `(hasher, index, size, leafHash, proof, root)` — but you
  do NOT call it here; you only byte-compare the proof node lists, which is exactly what M2's Verify
  bar asks ("computed inclusion proof matches the hub's `evidence.IsccLogInclusionProof`").

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -run TestVerifyInclusionEvidence -count=1 ./internal/logclient` passes.
- The golden subtest asserts `VerifyInclusionEvidence` returns `nil` for `{0, 5, 255, 256, 299}` on
  the 300-leaf `testonly.Tree`, with the evidence's `inclusionProof` built base64-Std from
  `tree.InclusionProof(index, 300)`.
- A mutation subtest (corrupted proof hash and/or wrong `LeafIndex`) returns a non-nil error with
  `errors.Is(err, ErrInclusionMismatch) == true`; a missing-tile fetcher returns a non-nil error with
  `errors.Is(err, os.ErrNotExist) == true`.
- `go test -run TestParseInclusionEvidence -count=1 ./internal/logclient` passes (round-trips a valid
  evidence JSON; rejects wrong `Type` and `TreeSize == 0`).
- `git diff --quiet HEAD -- go.mod go.sum internal/store/schema.sql` exits 0 (no dep/schema change).
- `GOOS=js GOARCH=wasm go build ./internal/didweb` exits 0 (WASM purity guard unaffected).

## Done When
`internal/logclient/inclusioncheck.go` exposes `ParseInclusionEvidence` + `VerifyInclusionEvidence`
(the first production-shaped caller of `InclusionProofFromTiles`), all Verification criteria pass, and
the golden + mutation tests prove the hub-vs-monitor inclusion-proof byte-comparison is correct and
non-vacuous — leaving only the `PollHub`/`iscc_index` wiring to complete M2's second Verify half.
