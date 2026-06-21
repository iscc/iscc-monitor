## 2026-06-21 — Certificate §3 INCLUSION PROOF — render the real RFC-6962 proof from the mirror

**Done:** `buildData`'s certifiable branch now recomputes the RFC-6962 inclusion proof of the subject
leaf (`seqs[0]` == `data.Position`) against the accepted tree (`hub.LastSize`) from the hub's mirrored
tiles via `logclient.InclusionProofFromTiles` over a `store.SQLiteFetcher`, base64-Std encodes each
sibling hash into `certData.ProofHashes`, sets `HasClause3`, and `cert.html` renders the
leaf→siblings→root chain in §3. A tile-not-mirrored miss (`os.ErrNotExist`) honestly omits §3 (the page
still shows §1+§2); any other build error is a 500 (buffer-then-200). The §3 test is mutation-proven
non-vacuous against `testonly.Tree.InclusionProof`.

**Files changed:**
- `internal/certificate/handler.go`: added `errors`/`os`/`logclient` imports; added
  `certData.ProofHashes []string` + `HasClause3`; wired the §3 build after §2 (`SQLiteFetcher{Store,
  HubID}` → `InclusionProofFromTiles(ctx, f.ReadTile, data.Position, hub.LastSize)`; `os.ErrNotExist` →
  honest gap, other err → 500; render §3 only when §2 holds since the root chip is the accepted root);
  updated package + struct + `buildData` doc comments (§3 is real, oracle gate now APPLIES).
- `internal/certificate/cert.html`: filled the `{{if .HasClause3}}` block — `leaf · seq {{.Position}}`,
  `{{range .ProofHashes}}sibling …{{end}}`, `root {{.CheckpointRoot}} ✓`, and a `{{len .ProofHashes}}
  sibling hashes` note. Reused existing `.clause-mono`/`.clause-note` CSS; no new CSS.
- `internal/certificate/handler_test.go`: added `fixtureStoreTiled` (real `testonly.Tree`, ingests every
  `tiles.TileCoords` hash tile via `RecordTile`, accepts the tree's REAL root via `AdvanceAccepted`) +
  `treeNodeHash` (ported from proofbuilder_test/fsck_test); `TestCertificateInclusionProof` (asserts each
  rendered hash == base64-Std of `tree.InclusionProof(seq, size)`) and `TestCertificateInclusionProofTileGap`
  (tile-less fixture renders §1+§2 but NO §3).

**Verification:** `mise run check` → green (all 21 packages `ok`, certificate uncached). Per criterion:
- [x] `go test -count=1 ./internal/certificate` passes uncached; `go test -count=1 -run TestCertificate
  ./internal/certificate` passes.
- [x] §3 test body contains `§3 INCLUSION PROOF`, `leaf · seq 0`, and each base64-Std hash of
  `tree.InclusionProof(0, 5)` (3 hashes — substantive multi-hash proof, not empty). Body is
  `html.UnescapeString`d first because `html/template` escapes `+`→`&#43;` (see Notes).
- [x] Mutation (a): `HasClause3 = false` → `TestCertificateInclusionProof` FAILS (all §3 asserts).
  Reverted.
- [x] Mutation (b): corrupt one rendered hash (`append([]byte{0x00}, h...)`) →
  `TestCertificateInclusionProof` FAILS (rendered hashes no longer match the prover). Reverted.
- [x] Tile-less fixture renders NO `§3 INCLUSION PROOF` (`TestCertificateInclusionProofTileGap`); the
  §1/§2 synthetic-fixture tests (`TestCertificateKnownID` etc.) still pass.
- [x] `GOOS=js GOARCH=wasm go build ./internal/index` + `./internal/didweb` exit 0.
- [x] `go.mod`/`go.sum` byte-unchanged (`InclusionProofFromTiles`/`SQLiteFetcher`/`testonly`/`api`/
  `compact`/`rfc6962` all already in the closure).
- [x] Conformance/oracle gate (re-engaged at §3): `go test -count=1 ./internal/logclient ./cmd/notecheck`
  green uncached; `derive_vkey.py` reproduces both vectors (`40b74463`/`22b08f3e`); scratch cleaned.
- [x] Scope: 2 source files + 1 test file (≤3 budget). No `## Not In Scope` item touched (no §4-§6,
  no proof-bundle download, no `registry.go`, no `go.mod`/`mise.toml`/CI bump).

**Next:** §3 is sound. Proceed to §4 SIGNING KEY (the hub's did:web key — `did:web:<domain>#<keyid>`,
Ed25519, resolved/cached via `hub_keys` / `LookupHubKey`; the §4 mockup shows the verification-method
id + "resolved from the hub's did:web document"). Then §5 Bitcoin anchor, §6 record history (the
per-id `seqs` list with declaration/deletion labels — the schema-aware projection), and the
downloadable proof-bundle assembler (the disabled Download button). The raw signed-note bytes are
already available from `CheckpointAt`'s second return for the bundle.

**Notes:**
- **HTML escaping of base64 chips:** `html/template`'s contextual auto-escaper encodes `+`→`&#43;` in
  text nodes (`template.HTMLEscapeString`/`html.EscapeString` do NOT — only the execution-path escaper
  does). The §2 test passed only because its fixture root `cm9vdA==` has no `+`/`/`. The §3 test
  asserts on `html.UnescapeString(body)` so it checks the real base64-Std proof bytes regardless of
  escaping. The byte-identity-across-surfaces invariant holds at the view-model level (`ProofHashes` are
  the same base64-Std strings `writeEvidence` emits); the on-page HTML entity-escapes `+`, which is
  correct/harmless rendering. Review may want to confirm this is acceptable for a human-readable cert
  (the proof bundle download — a later step — will carry the unescaped JSON).
- **Render-§3-only-when-§2 decision:** §3's root chip reuses `data.CheckpointRoot` (the accepted root),
  so I gate the render on `data.HasClause2` (the `else if data.HasClause2` branch). In practice §2 is
  always present on a certifiable path (`AdvanceAccepted` records the checkpoint at the same size it
  advances to), so this is belt-and-suspenders, not a reachable gap. A proof that built but had no §2
  root would silently drop §3 rather than render a chain to an empty root — the coverage-honest choice.
- **Empty-proof note:** a single-leaf accepted tree (`size == 1`, leaf 0) yields an empty-but-valid
  proof, so `HasClause3` gates on `err == nil`, not `len(proof) > 0`. Unreachable on the certifiable
  path though: the cap requires `seqs[0] < LastSize`, and `seqs[0] >= 0`, so `LastSize >= 2` whenever
  §3 builds for the chosen leaf except the `seq 0, size 1`… actually `seq 0 < size 1` IS certifiable
  and would render §3 with zero siblings + the note "0 sibling hashes". That is honest (the root IS the
  leaf), so it is fine; the §3 test deliberately uses a 5-leaf tree for a substantive 3-hash proof.
- Open backlog unchanged: the `hubDomain` ForceQuery fail-open (`normal`, waits for a `registry.go`
  step), the ADR-0011 Go 1.26 / iscc-lib bump (`normal`, foundational), and the four `low` items. None
  block §3. M-UI's certificate Verify criterion now has §1+§2+§3 of 6 clauses real; §4-§6 + the
  proof-bundle download remain the in-flight arc, so M-UI is not yet DONE.
