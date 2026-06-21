# Next Work Package

## Step: Certificate §3 INCLUSION PROOF — render the real RFC-6962 proof from the mirror

## Advances
M-UI — Evidence Ledger frontend, the **certificate of inclusion** Verify criterion:

> "the **realm-wide certificate** (`/inclusion/{iscc_id}` …) for a known id renders the numbered
> evidence clauses (subject + position; checkpoint `(size, root)`; **inclusion proof**; signing key;
> anchor state; full per-id record history …)"

§1 SUBJECT and §2 CHECKPOINT are landed + PASS-verified. This step closes the **§3 inclusion proof**
clause — the point the state, handoff, and learnings all name as next, and where the
**oracle/conformance crypto gate RE-ENGAGES** (the served inclusion proof must be mutation-proven
non-vacuous against the hub's `IsccLogInclusionProof` / golden vectors and rebuilt over the
`SQLiteFetcher`). It is one clause of a coherent in-flight arc (§1 → §2 → §3 → …), not a switch to
unrelated work.

## Goal
Make the certificate's §3 INCLUSION PROOF clause real: for a certifiable id, recompute the RFC-6962
inclusion proof of `seqs[0]` against the accepted tree (`hub.LastSize`) from the hub's mirrored tiles
via the existing `logclient.InclusionProofFromTiles` + `store.SQLiteFetcher`, and render the
leaf→siblings→root chain in §3. This is the first certificate clause that touches the Merkle path, so
its test must prove the proof is genuine (not vacuous), the way the proofbuilder/inclusioncheck tests do.

## Scope
- **Create**: (none)
- **Modify** (≤3 non-test/doc files — exactly 2 source + 1 test):
  - `/workspace/iscc-monitor/internal/certificate/handler.go` — in `buildData`'s certifiable branch
    (after §2, where `hub.LastSize > 0` and `seqs[0] < hub.LastSize` are already proven), build the
    inclusion proof from the mirror and populate the §3 view-model fields + `HasClause3`.
  - `/workspace/iscc-monitor/internal/certificate/cert.html` — render the §3 clause inside the existing
    `{{if .HasClause3}}` block (cert.html:345-350, currently an empty `.clause-value`).
  - `/workspace/iscc-monitor/internal/certificate/handler_test.go` — add a **tile-backed** fixture + a
    non-vacuous §3 test (test file, not counted toward the ≤3 budget).
- **Reference** (read before implementing):
  - `/workspace/iscc-monitor/.claude/context/learnings/certificate.md` — §1/§2 mechanics, fail-closed
    buffer-then-200, canonicalized lookup key, the accepted-tree cap; §3 re-engages the oracle gate.
  - `/workspace/iscc-monitor/.claude/context/learnings/logclient.md` — `InclusionProofFromTiles` (the
    proof source, its oracle-gate section), `VerifyInclusionEvidence`, and the `TileFetcher` ≡
    `SQLiteFetcher.ReadTile` signature.
  - `/workspace/iscc-monitor/.claude/context/learnings/store.md` — `SQLiteFetcher` / `widthForP` /
    partial-tile read-back; `CheckpointAt` / `AdvanceAccepted` set `last_size`.
  - `/workspace/iscc-monitor/internal/proofserve/handler.go` `serveInclusion` (lines 218-280) +
    `writeEvidence` (1049-1068) — the exact precedent: `InclusionProofFromTiles(ctx, f.ReadTile,
    leafIndex, size)` → base64-Std hashes; the `errors.Is(err, os.ErrNotExist)` tile-miss branch.
  - `/workspace/iscc-monitor/internal/follower/fsck_test.go` (lines ~217-274) — the canonical way to
    build a `testonly.Tree`, enumerate `tiles.TileCoords(size)`, `api.HashTile{Nodes}.MarshalText`, and
    ingest via `store.RecordTile`; reuse this to make the §3 fixture have real mirrored tiles.
  - `/workspace/iscc-monitor/internal/logclient/proofbuilder_test.go` (lines ~32-105) — `buildTree` /
    `tileFetcherFor` and the three-independent-paths assertion (`tree.InclusionProof` vs builder vs
    `proof.VerifyInclusion`).
  - `/workspace/iscc-monitor/.claude/design/ISCC Monitor - Certificate.dc.html` (lines 60-64) — the §3
    mockup: `leaf · seq N` chip, the sibling-hash chip list, the `root … ✓` chip, note "Five sibling
    hashes rebuild the root from your record" (use the actual proof length, not the literal "Five").

## Not In Scope
- Clauses **§4 (signing key)**, **§5 (Bitcoin anchor)**, **§6 (record history)** — later sub-steps.
  Leave `HasClause4/5/6` false and their template placeholders empty.
- The **downloadable proof-bundle assembler** `{checkpoint, inclusion/consistency proof, record bytes,
  hub key, ots?}` — keep the Download button the existing disabled placeholder. §3 renders the proof on
  the page; bundling it for download is a distinct later step.
- The separate **Bitcoin-anchor vs comparison-anchor** panels.
- The **ADR-0011 Go 1.26 / iscc-lib bump** (`normal`, foundational) — a separate sequenced increment;
  do not touch `go.mod`/`mise.toml`/CI here. (`InclusionProofFromTiles` + `SQLiteFetcher` are already in
  the closure, so `go.mod`/`go.sum` must stay byte-unchanged.)
- The `hubDomain` `ForceQuery` registry fail-open (`normal`) — this step does **not** touch
  `internal/registry/registry.go`, so that fix waits for a step that does (per its issue).
- Pixel-matching the mockup's chip styling — reuse the existing `.clause-*` / `.subject-mono` CSS;
  named-region parity (the §3 marker + the rendered hash chain) is the bar, not exact spacing.

## Implementation Notes
- **Where to wire it (`handler.go`):** at the end of `buildData`'s certifiable branch, after the §2
  `CheckpointAt` read. The leaf index is `seqs[0]` (already `data.Position`) and the tree size is
  `hub.LastSize` (already proven `> 0` and `> seqs[0]` by the cap). Construct the fetcher exactly as
  proofserve does: `f := store.SQLiteFetcher{Store: st, HubID: hub.HubID}`, then
  `proof, err := logclient.InclusionProofFromTiles(r.Context(), f.ReadTile, data.Position, hub.LastSize)`.
  (This adds the first `internal/logclient` import to `internal/certificate`; that direction is fine —
  certificate is an HTTP-surface package, logclient is below it.)
- **Fail-closed, NOT 500 on a tile miss.** proofserve maps `errors.Is(err, os.ErrNotExist)` to a 404
  because it has committed to serving a proof. The certificate is a clause-by-clause page that can
  decline a clause honestly (exactly as §2 does on `found == false`): a **tile-not-mirrored**
  (`os.ErrNotExist`) miss should leave `HasClause3 = false` (no fabricated proof, no error), so the page
  still renders §1+§2. A non-`os.ErrNotExist` build error stays a **500** via the existing
  buffer-then-200 path (`return certData{}, http.StatusInternalServerError`) — match §2's split. Import
  `errors` + `os` in `handler.go` for the `errors.Is(err, os.ErrNotExist)` check (currently neither is
  imported; `encoding/base64` already is for §2).
- **View-model.** Add to `certData`: `ProofHashes []string` (each `base64.StdEncoding.EncodeToString(h)`,
  matching §2's root + proofserve's `writeEvidence`, so the strings are byte-identical across surfaces)
  and set `HasClause3 = true` only when the proof built. Reuse `data.Position` (leaf) and
  `data.CheckpointRoot` (root, already base64-Std from §2) in the template's leaf/root chips — do not
  re-encode them. Note: a valid inclusion proof can be **empty** (`len(proof) == 0`) when `size == 1`
  (single-leaf tree); that is a legitimate proof, so gate `HasClause3` on the build *succeeding*
  (`err == nil`), not on `len(proof) > 0`. (§3 also depends on §2's root, so realistically render §3
  only when `HasClause2` already holds — `CheckpointRoot` is the root chip.)
- **Template (`cert.html`).** Inside `{{if .HasClause3}}` replace the empty `.clause-value` with: the
  §3 marker already present, a `.clause-mono` chain rendering `leaf · seq {{.Position}}`, then
  `{{range .ProofHashes}}…{{end}}` siblings, then `root {{.CheckpointRoot}} ✓`, and a `.clause-note`
  like "{{len .ProofHashes}} sibling hashes rebuild the accepted root from this record." Reuse existing
  `.clause`, `.clause-marker`, `.clause-value`, `.clause-mono`, `.clause-note` CSS — confirm no new CSS
  is needed (the §2 review confirmed these classes already exist). `html/template` auto-escapes.
- **Oracle / conformance gate (APPLIES — this is RFC-6962 inclusion crypto).** The §3 test MUST be
  mutation-proven non-vacuous, the way `proofbuilder_test.go` / `inclusioncheck_test.go` are: build a
  real `testonly.Tree`, ingest its hash tiles into the fixture store via `store.RecordTile` (port the
  `fsck_test.go` enumeration over `tiles.TileCoords(size)` — write full tiles as p==0, partials as
  their width per store.md's `widthForP`), seed the accepted checkpoint with the tree's REAL root
  (`tree.Hash()` at that size via `AdvanceAccepted`), index a leaf for the golden id at the chosen seq,
  then assert the body's rendered hash chips equal `base64.StdEncoding.EncodeToString` of each hash in
  `tree.InclusionProof(seqs[0], size)` (the independent prover path). The proof must be substantive —
  pick a leaf/size that yields a multi-hash proof (e.g. a small tree of 5–8 leaves so the proof has ≥2
  hashes, not an empty proof). The advance author should confirm two mutations FAIL the new test (e.g.
  neuter `HasClause3`; corrupt one rendered hash) and that a tile-less fixture leaves §3 unrendered (the
  `os.ErrNotExist` honest-gap path) — record them for `review`.
- **Reuse, do not reimplement (target "Stack (locked)").** `InclusionProofFromTiles` already exists and
  is itself oracle-gated; do NOT hand-roll Merkle math in the certificate. The certificate only calls it
  and base64-encodes the result.
- **Correctness rules (learnings index):** `iscc_id → seq` is one-to-many / schema-agnostic — the proof
  is for `seqs[0]` (the deterministic default, matching §1/§2/`serveVerify`); do not interpret the id.
  Coverage honesty (ADR-0001) — the proof is against the **accepted** tree (`hub.LastSize`), already
  capped by §1; never against an unaccepted projection. `proof/verify` purity is unaffected (the
  certificate is not WASM-shared; the proof builder it calls is already net-free at file level).
- **Keep the existing fixture helpers working.** The new tile-backed fixture is **additive** (a new
  helper). The §1/§2 tests that use the synthetic `Root: []byte("root")` fixture WITHOUT mirrored tiles
  must keep passing — they will now hit the `os.ErrNotExist` honest-gap path and simply render no §3
  clause, which is correct. Verify those tests do not newly assert on §3; the §3-presence assertion
  lives only in the new tile-backed test.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -count=1 ./internal/certificate` passes uncached.
- `go test -count=1 -run TestCertificate ./internal/certificate` passes; the new tile-backed §3 test
  body contains `§3 INCLUSION PROOF`, `leaf · seq <N>`, and each base64-Std-encoded hash of
  `tree.InclusionProof(seqs[0], size)` (assert each is `strings.Contains`-present).
- Mutation (advance author reproduces, then reverts): forcing `HasClause3 = false` (or skipping the
  proof render) makes the new §3 test FAIL; corrupting one rendered hash makes it FAIL — proving the
  assertion is grounded in the real proof, not a literal.
- The §1/§2 synthetic-fixture tests (`TestCertificateKnownID`, etc.) still pass, and a tile-less fixture
  renders NO `§3 INCLUSION PROOF` (the `os.ErrNotExist` honest-gap path; assert `§3 INCLUSION PROOF`
  absent there).
- `GOOS=js GOARCH=wasm go build ./internal/index` still exits 0 (no-regression on the WASM leaf).
- `go.mod` / `go.sum` are byte-unchanged (`git diff --stat go.mod go.sum` empty).

## Done When
`buildData` recomputes the real RFC-6962 inclusion proof of `seqs[0]` against `hub.LastSize` from the
mirrored tiles via `InclusionProofFromTiles`, the §3 clause renders the leaf→siblings→root chain for a
tile-backed certifiable id (and honestly omits §3 when tiles are not mirrored), the §3 test is
mutation-proven non-vacuous against `testonly.Tree.InclusionProof`, and all Verification criteria pass
with `mise run check` green and `go.mod`/`go.sum` byte-unchanged.
