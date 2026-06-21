## 2026-06-21 — Fail-close certificate §3: verify the built proof rebuilds the accepted root (close the TOCTOU critical)

**Done:** Replaced the §3 status-flag gate (`data.HasClause2 && !hub.Frozen`) with a fail-closed
Merkle re-verification: `buildData` now reads the subject leaf's entry bundle from the mirror,
derives its RFC-6962 leaf hash, and sets `HasClause3` only when `proof.VerifyInclusion` proves the
built proof rebuilds the §2 accepted root. This subsumes the freeze gate (the `hub.Frozen` reference
is gone) and fails closed against ANY tile↔root divergence — the fork-poll TOCTOU race AND the
steady-state frozen case — making the rendered ✓ true by construction. Closes the open `critical`.

**Files changed:**
- `internal/certificate/handler.go`: rewrote the §3 branch in `buildData` — capture the raw `root`
  bytes from §2's `CheckpointAt`, build the proof, then `ReadEntryBundle` →
  `RecordBytesFromBundle` → `rfc6962.DefaultHasher.HashLeaf` → `proof.VerifyInclusion(hasher,
  Position, LastSize, leafHash, builtProof, root) == nil`. Added imports
  `merkle/proof`, `merkle/rfc6962`, `internal/tiles`; removed the `hub.Frozen` reference; rewrote the
  file / `buildData` / `certData` docstrings + the §3 block comment to describe the rebuild
  verification (the "rebuild gate") instead of the freeze flag.
- `internal/certificate/handler_test.go`: added an `encodeBundle` helper (copied from
  `logclient/fsck_test.go`) + `encoding/binary` import; `fixtureStoreTiled` now seeds byte-accurate
  entry bundles (the `leaf-i` preimages, framed) for every `BundleCoords(size)` so the §3
  verification can derive a leaf hash equal to `tree.LeafHash(seq)`. Converted
  `TestCertificateInclusionProofFrozen` → `TestCertificateInclusionProofContradictory`: same
  contradictory-tile fixture (mirror tree A, accept tree B's root) but `freeze=false`, asserting
  §1+§2 render while §3 is absent **even though the hub is not frozen**.

**Verification:** `mise run check` → green (all 21 packages `ok`, `go build`/`go vet`/`go test`).
Per-criterion:
- `go test -count=1 -run TestCertificate ./internal/certificate` → PASS uncached (clean §3 proof,
  tile-gap honest-decline, and the new non-frozen contradictory test).
- Oracle/conformance gate: `go test -count=1 ./internal/logclient ./internal/proofserve
  ./cmd/notecheck` → all `ok`.
- Mutation (non-vacuity): replacing the §3 `proof.VerifyInclusion(...) == nil` guard with
  `... == nil || true` (keeps `proof`/`leafHash` referenced so it still compiles) makes
  `TestCertificateInclusionProofContradictory` FAIL while the clean `TestCertificateInclusionProof`
  stays PASS; restored, tree clean. (The bare `if true` mutation in `next.md` won't compile — unused
  `proof` import + unused `leafHash` — so I used the `|| true` variant, which is the same logical
  mutation and keeps the build valid; review can reproduce it identically.)
- `gofmt -l .` empty; `git diff --stat go.mod go.sum` empty; `GOOS=js GOARCH=wasm go build
  ./internal/index ./internal/didweb` → exit 0; `git diff --stat internal/certificate/cert.html`
  empty; no `hub.Frozen`/`Frozen` reference left in `handler.go`.

**Next:** Resume the §3-plan continuation now that the critical is closed and this can push: §4
SIGNING KEY → §5 Bitcoin anchor → §6 record history, then the downloadable proof-bundle assembler
(which shares this same build+verify crypto path and must keep the oracle/conformance gate green).
The clean caps already proven here (`LastSize > 0`, `Position < LastSize`, entry-bundle seeding in
`fixtureStoreTiled`) carry forward.

**Notes:**
- Reused the `root []byte` returned by §2's `CheckpointAt` for `VerifyInclusion` rather than
  re-decoding `data.CheckpointRoot` (cleaner; `root` is only meaningful when `data.HasClause2`, which
  is exactly the branch guard). `root` was already in scope from the §2 block.
- `ErrLeafOutOfBundle` from `RecordBytesFromBundle` and `os.ErrNotExist` from `ReadEntryBundle` are
  honest gaps → §3 declined (NOT 500), mirroring §2's split and proofserve's `serveVerify`. A
  non-nil `VerifyInclusion` result is a silent §3 decline, never a 500.
- **Out-of-scope working-tree changes were present and NOT committed by me:** `.claude/agents/review.md`,
  `.claude/context/target.md`, `.devcontainer/Dockerfile`, and untracked
  `.claude/adr/0012-agent-browser-visual-verification.md` + `.claude/skills/agent-browser/`. These
  appeared in the tree independently of this work package (not mine to touch); I committed only
  `handler.go`, `handler_test.go`, and this handoff. Flagging for review/loop hygiene.
