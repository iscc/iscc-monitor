## 2026-06-21 — Land the pure inclusion cross-check `VerifyInclusionEvidence`

**Done:** Added the pure, golden-testable core of M2's second Verify half: `ParseInclusionEvidence`
decodes a hub's `IsccLogInclusionProof` VC evidence member, and `VerifyInclusionEvidence` recomputes
the inclusion proof from the monitor's mirrored tiles via `InclusionProofFromTiles` (its first
production-shaped caller) and byte-compares it against the hub-supplied, base64-Std-encoded proof.
Strictly additive — no existing production file changed; `PollHub`/`iscc_index` wiring is left for a
later slice.

**Files changed:**
- `internal/logclient/inclusioncheck.go` (new): `InclusionEvidence` struct mirroring the hub VC member;
  `ParseInclusionEvidence(raw)` (thin `json.Unmarshal`, rejects wrong `Type` and `TreeSize == 0`);
  `VerifyInclusionEvidence(ctx, fetch, ev)` (guards `LeafIndex < TreeSize` before fetching, computes
  from tiles, base64-Std-decodes each hub hash, length-then-`bytes.Equal` compares); package vars
  `ErrInclusionMismatch` (descriptive sentinel) + `inclusionEvidenceType` const. Imports exactly
  `bytes`/`context`/`encoding/base64`/`encoding/json`/`errors`/`fmt`.
- `internal/logclient/inclusioncheck_test.go` (new): golden + mutation + edge tests (see below).

**Verification:** `mise run check` → green (all 11 packages `ok`; `go build`/`go vet`/`go test` pass).
Per-criterion:
- [x] `go test -run TestVerifyInclusionEvidence -count=1 ./internal/logclient` — passes.
- [x] Golden: `VerifyInclusionEvidence` returns `nil` for `{0, 5, 255, 256, 299}` on the 300-leaf
  `testonly.Tree`, evidence's `inclusionProof` built base64-Std from `tree.InclusionProof(index, 300)`.
- [x] Mutation: corrupted proof hash AND wrong `LeafIndex` (valid proof for leaf 5 claimed as leaf 6)
  both return `errors.Is(err, ErrInclusionMismatch) == true`; missing-tile fetcher returns
  `errors.Is(err, os.ErrNotExist) == true` and NOT `ErrInclusionMismatch`.
- [x] `go test -run TestParseInclusionEvidence -count=1 ./internal/logclient` — passes (round-trips a
  valid evidence JSON; rejects wrong `Type`, `TreeSize == 0`, and bad JSON).
- [x] `git diff --quiet HEAD -- go.mod go.sum internal/store/schema.sql` — exit 0 (no dep/schema change).
- [x] `GOOS=js GOARCH=wasm go build ./internal/didweb` — exit 0 (WASM purity guard unaffected).
- [x] `gofmt -l .` — empty.
- [x] **Mutation, reproduced + reverted:** short-circuiting the length + `bytes.Equal` compares (kept
  `bytes` referenced via `if false && …`) makes BOTH `TestVerifyInclusionEvidenceCorruptedProof` and
  `TestVerifyInclusionEvidenceWrongLeafIndex` FAIL (function returns `nil`); revert → green. The
  byte-comparison is load-bearing; a green-but-wrong check that ignored proof bytes cannot ship.

**Next:** Wire this into `PollHub`/`follower.go`. That needs (a) the `iscc_index` projection writer
(schema-agnostic `iscc_id → seq` table) to resolve a sampled `iscc_id → leafIndex`, and (b) a real or
in-process `IsccLogInclusionProof` fixture + an entry-bundle to sample a leaf from. Pass
`store.SQLiteFetcher.ReadTile` straight into `VerifyInclusionEvidence` (signature already matches
`TileFetcher`), mirroring how `fsckMirror` sources tiles from the local mirror. Alternatively
`define-next` could take the still-open `CheckpointAt ORDER BY` fix or the `fsckMirror`
redundant-resolve efficiency item (both in `issues.md`).

**Notes:**
- **Oracle gate (APPLIES — RFC-6962 inclusion crypto):** satisfied by three independent merkle paths.
  The test plays the hub's role with `testonly.Tree.InclusionProof(index, 300)` (the prover),
  base64-Std-encodes it into `InclusionEvidence` exactly as `iscc_hub/log_tree.py inclusion_evidence`
  does (verified `base64.b64encode` = `StdEncoding`), serves tiles via the existing `tileFetcherFor`,
  and `VerifyInclusionEvidence` recomputes via `InclusionProofFromTiles` (the builder) + decode + bytes
  compare. Prover, tile-builder, and the base64/bytes compare are distinct → not a tautology. The
  external `notecheck`/`derive_vkey.py` oracles are correctly N/A here (no signature/did:web path — the
  embedded checkpoint sig is intentionally NOT re-verified, per `next.md` Not-In-Scope; that stays
  `AcceptCheckpoint`'s job).
- Per `next.md`, used the in-process `testonly.Tree` mirror, NOT live sb0/sb1 tiles or a captured real
  `IsccLogInclusionProof` (consistent with the fsck-slice precedent — live leaf preimages were never
  captured). `Checkpoint` is decoded into the struct but never re-parsed (`note.Open` not re-run).
- `VerifyInclusionEvidence` is an intentional unused-until-wired export seam (like the consistency
  triggers / `LeafHashes` / `RunFsck`) — `go vet` clean, not dead code. First caller is the M2
  `PollHub`/`iscc_index` slice above.
- Test helper `hubEvidenceFor(t, *testonly.Tree, index, leaves)` is the reusable "hub side" evidence
  builder for the future wiring test; it lives beside the cross-check in the same `package logclient`.
- No new imports → `go.mod`/`go.sum` byte-identical (`encoding/json`/`encoding/base64`/`bytes` stdlib;
  `InclusionProofFromTiles` + `testonly` already in the package closure). File stays net-free at the
  file level; the package's pre-existing `net/http` (via `didresolve.go`) is unaffected and the
  load-bearing WASM purity invariant rides on `internal/didweb`, which still builds (verified).
