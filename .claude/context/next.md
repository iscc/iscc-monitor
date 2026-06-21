# Next Work Package

## Step: Fail-close certificate §3 — verify the built proof rebuilds the accepted root (close the TOCTOU critical)

## Advances
Closes the one open **`critical`** issue ("Certificate §3 can still render a self-contradictory
proof in the fork-poll TOCTOU window — the `!hub.Frozen` gate is incomplete"), which **preempts all
milestone-feature work** (issue priority `critical`). It is also the explicit `review` handoff
`**Next:**`. The critical blocks the active **M-UI — Evidence Ledger frontend** milestone Verify
criterion:

> "the **realm-wide certificate** (`/inclusion/{iscc_id}` …) for a known id renders the numbered
> evidence clauses (subject + position; checkpoint `(size, root)`; **inclusion proof**; …)"

— a rendered inclusion proof under a `✓` it does not rebuild violates the M-UI oracle/conformance
gate ("the proof-bundle assembler shares the crypto path and MUST keep the conformance/oracle gate
green") and the glossary **Proof bundle** / **Verifiable cache** contract (a client verifies the
artifact itself). Until §3 is honest the certificate Verify criterion cannot pass, no `critical` may
remain open at DONE, and the §3 cycle cannot push (CI is green only at `17c4957`, not at HEAD).

## Goal
Replace the §3 status-flag gate (`data.HasClause2 && !hub.Frozen`) with a fail-closed Merkle
re-verification: set `HasClause3` only when the built inclusion proof actually rebuilds the accepted
checkpoint root. This fails closed against ANY tile↔root divergence — the fork-poll race AND the
steady-state frozen case — so it subsumes and removes the `!hub.Frozen` gate, making the rendered `✓`
true by construction rather than by a racily-read flag.

## Scope
- **Create**: (none)
- **Modify**:
  - `/workspace/iscc-monitor/internal/certificate/handler.go` — rewrite the §3 branch in `buildData`
    (lines 325-381): keep building the proof, then read the subject leaf's entry bundle → leaf hash →
    `proof.VerifyInclusion` against the accepted root; set `HasClause3` only on a nil verdict. Add the
    `merkle/proof` + `merkle/rfc6962` + `internal/tiles` imports; remove the `hub.Frozen` reference;
    update the file / `buildData` / `certData` docstrings to describe the rebuild verification (not the
    freeze flag). Only non-test source file (≤3 budget: 1).
  - `/workspace/iscc-monitor/internal/certificate/handler_test.go` — seed a byte-accurate entry bundle
    for the subject leaf in `fixtureStoreTiled`; convert `TestCertificateInclusionProofFrozen` into a
    NON-frozen contradictory-tile assertion (mirror tree A, accept tree B's root, `freeze=false`) →
    §1+§2 render, §3 absent; mutation-proven. Test file, not counted toward the ≤3 budget.
- **Reference** (read before implementing):
  - `/workspace/iscc-monitor/internal/proofserve/handler.go` lines 535-575 — the EXACT fail-closed
    pattern to port: `bundleIndex := leafIndex/tiles.TileWidth`, `offset := leafIndex%tiles.TileWidth`,
    `p := tiles.PartialTileSize(0, bundleIndex, size)`, `f.ReadEntryBundle(ctx, bundleIndex, p)`,
    `logclient.RecordBytesFromBundle(bundle, offset)`, `rfc6962.DefaultHasher.HashLeaf(record)`,
    `proof.VerifyInclusion(rfc6962.DefaultHasher, leafIndex, size, leafHash, builtProof, root) == nil`.
  - `/workspace/iscc-monitor/internal/logclient/entries.go` — `RecordBytesFromBundle(bundle, offset)`
    + the `ErrLeafOutOfBundle` sentinel (checkable via `errors.Is`).
  - `/workspace/iscc-monitor/.claude/context/learnings/certificate.md` — the §3 / "built ≠ verified" /
    base64-Std / `html/template` escape (`+`→`&#43;`) notes.
  - `/workspace/iscc-monitor/.claude/context/learnings/logclient.md` — `InclusionProofFromTiles`; the
    `VerifyInclusion(hasher, index, size, leafHash, proof, root)` arg-order gotcha (leafHash precedes
    proof, unlike `VerifyConsistency`).
  - `/workspace/iscc-monitor/.claude/context/learnings/store.md` — `SQLiteFetcher.ReadEntryBundle`
    p→width + partial→full fallback; `os.ErrNotExist` survives `errors.Is`.
  - `/workspace/iscc-monitor/internal/logclient/fsck_test.go` lines 62-75 — the `encodeBundle` helper
    (manual `binary.BigEndian.PutUint16` framing) to copy into the cert test fixture.

## Not In Scope
- Clauses **§4–§6** (signing key, Bitcoin anchor, record history) and the downloadable proof-bundle
  assembler — the next step once this critical is closed and pushed.
- The deferred `internal/registry` `hubDomain` `ForceQuery` fix (`normal`) — this step does not touch
  `registry.go`; fold it in when `hubDomain` is next edited.
- The ADR-0011 Go 1.26 / iscc-lib bump (`normal`) — separate foundational increment.
- Any change to `cert.html` — the template already gates §3 on `{{if .HasClause3}}`; only the
  view-model condition changes. Keep `cert.html` byte-unchanged.
- Any change to `proofserve`'s `serveVerify` — it already does this verification and returns a
  client-verifiable JSON verdict (no asserted `✓`), so it is not an honesty defect; do not refactor it.

## Implementation Notes
- **Port, do not invent.** `proofserve.serveVerify` (handler.go:535-575) already performs exactly the
  fail-closed verification this step needs. Lift that logic into the §3 branch of
  `certificate.buildData` (handler.go:325-381). The handler already has `f := store.SQLiteFetcher{Store:
  st, HubID: hub.HubID}`, `data.Position` (== `seqs[0]`), `hub.LastSize`, and `data.CheckpointRoot`
  (base64-Std). Capture the raw `root []byte` from the §2 `CheckpointAt` call (currently discarded into
  `data.CheckpointRoot`) and reuse those bytes for `VerifyInclusion` — cleaner than re-decoding
  `data.CheckpointRoot`. The accepted-tree cap above already proved `hub.LastSize > 0` and
  `data.Position < hub.LastSize`, so the leaf is in range.
- **Sequence inside the §3 branch** (after `InclusionProofFromTiles` builds `proof`):
  `bundleIndex := data.Position / tiles.TileWidth`, `offset := data.Position % tiles.TileWidth`,
  `p := tiles.PartialTileSize(0, bundleIndex, hub.LastSize)`,
  `bundle, err := f.ReadEntryBundle(r.Context(), bundleIndex, p)`,
  `record, err := logclient.RecordBytesFromBundle(bundle, offset)`,
  `leafHash := rfc6962.DefaultHasher.HashLeaf(record)`. Set `HasClause3 = true` ONLY when
  `data.HasClause2 && proof.VerifyInclusion(rfc6962.DefaultHasher, data.Position, hub.LastSize,
  leafHash, builtProof, root) == nil` (keep gating on `HasClause2` because §3 renders the §2 root
  chip). Otherwise leave §3 unrendered — §1/§2 still show.
- **Arg-order gotcha (learnings):** `VerifyInclusion(hasher, index, size, leafHash, proof, root)` —
  `leafHash` precedes `proof`. Do NOT transpose with `VerifyConsistency`'s `(…, proof, root1, root2)`.
- **Fail-closed error mapping (ADR-0001), mirror §2's split:** an `os.ErrNotExist` from
  `InclusionProofFromTiles`/`ReadEntryBundle`, or `logclient.ErrLeafOutOfBundle` from
  `RecordBytesFromBundle`, is an honest tile/bundle gap → leave §3 unrendered (NOT a 500, NOT a
  fabricated proof). Any OTHER error (a genuine store/decode fault) → `return certData{},
  http.StatusInternalServerError` (buffered before any 200). A `VerifyInclusion` non-nil result is a
  silent decline of §3 (the proof did not rebuild the root), never a 500 — the certificate can decline
  a clause.
- **Remove the `!hub.Frozen` gate.** The verification subsumes it: a frozen-after-fork hub whose mirror
  diverges from the accepted root fails `VerifyInclusion`, so §3 is declined without reading
  `hub.Frozen`. Drop `&& !hub.Frozen` and the `hub.Frozen` reference. Rewrite the §3 block comment +
  the file / `buildData` / `certData` docstrings (they currently describe the freeze-gate mechanism) to
  describe the rebuild verification — evergreen wording, no "now"/"changed from".
- **Imports:** add `"github.com/transparency-dev/merkle/proof"`,
  `"github.com/transparency-dev/merkle/rfc6962"`, and `"github.com/iscc/iscc-monitor/internal/tiles"`.
  All are already in the build closure via `proofserve`/`logclient` — `go.mod`/`go.sum` stay
  byte-unchanged. Confirmed: `certificate` currently imports neither merkle package nor `tiles`.
- **Correctness rule (learnings index):** `proof/verify` is the shared crypto path — reuse the
  oracle-gated builder + `merkle/proof` verifier; never hand-roll Merkle math. base64 is **Std** (`+/`,
  `=` padding), byte-identical across the log browser / verify-for-me / §2. Also: "A self-consistency
  violation freezes, never crashes (ADR-0006)" + "Coverage honesty (ADR-0001) — never render a
  guarantee the accepted state does not support."
- **Test fixture (`fixtureStoreTiled`, handler_test.go:479):** today it seeds only HASH tiles, so
  `RecordBytesFromBundle` would miss → the new verification would never get a leaf hash. Add a
  byte-accurate ENTRY BUNDLE for the subject leaf so the derived `HashLeaf(record)` equals
  `tree.LeafHash(seq)` (that equality is what makes the clean `TestCertificateInclusionProof` pass
  THROUGH the new verification). The tree is built from `[]byte(fmt.Sprintf("leaf-%d", i))`; copy
  `encodeBundle` (manual `binary.BigEndian.PutUint16` length-prefix per record) from
  `internal/logclient/fsck_test.go:62-75`, and for each `BundleCoord` in `tiles.BundleCoords(size)`
  write `st.RecordEntryBundle(ctx, target, c.Index, c.Partial, encodeBundle(records), time.Unix(0,0))`
  with that bundle's slice of preimages (a 5-leaf tree is one partial bundle at index 0, partial 5).
- **Convert the test to a NON-frozen contradictory-tile assertion.** Change
  `TestCertificateInclusionProofFrozen`'s fixture call to `fixtureStoreTiled(..., treeB.Hash(),
  false)` (mirror tree A's tiles, accept tree B's root, `freeze=false`). Assert §1+§2 render but §3 is
  ABSENT **even though the hub is not frozen** (the proof builds from tree-A tiles but does not rebuild
  tree-B's accepted root, so `VerifyInclusion` rejects it). Rename to reflect the verification (e.g.
  `TestCertificateInclusionProofContradictory`). The clean `TestCertificateInclusionProof` (mirror and
  accepted root agree) MUST stay green — that proves the verification ACCEPTS a real proof, not just
  rejects everything. Leave `TestCertificateInclusionProofTileGap` untouched and green.
- **Non-vacuity (mandatory — review reproduces it).** Replacing the §3 `proof.VerifyInclusion(...) ==
  nil` guard with `true` (so §3 renders whenever the proof builds) must make the new contradictory-tile
  test FAIL; restoring it passes. State this in the test docstring and confirm locally before handoff.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -count=1 -run TestCertificate ./internal/certificate` passes uncached (full cert suite,
  incl. the clean §3 proof, the tile-gap honest-decline, and the new non-frozen contradictory test).
- Oracle/conformance gate stays green (the crypto path is touched):
  `go test -count=1 ./internal/logclient ./internal/proofserve ./cmd/notecheck` all `ok`.
- **Mutation (non-vacuity):** replacing the §3 `proof.VerifyInclusion(...) == nil` guard with `true`
  makes `go test -run TestCertificateInclusionProofContradictory ./internal/certificate` FAIL;
  restoring it passes. (Run, confirm, revert — leave the tree clean.)
- `go.mod` / `go.sum` byte-unchanged: `git diff --stat go.mod go.sum` is empty.
- WASM/purity unaffected: `GOOS=js GOARCH=wasm go build ./internal/index ./internal/didweb` exits 0.
- `cert.html` untouched: `git diff --stat internal/certificate/cert.html` empty.

## Done When
`mise run check` is green, the new non-frozen contradictory-tile test asserts §1+§2-render-but-§3-absent
(mutation-proven) while the clean §3 test stays green, the §3 render is gated on `proof.VerifyInclusion`
against the accepted root rather than on `hub.Frozen`, and `go.mod`/`go.sum` are byte-unchanged —
closing the open `critical`.
