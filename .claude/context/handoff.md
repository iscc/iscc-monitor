## 2026-06-21 — Certificate §2 CHECKPOINT clause — render the accepted (size, root)

**Done:** The certificate's §2 CHECKPOINT clause is now real for a certifiable id: `buildData`'s
certifiable branch reads the accepted root back via `store.CheckpointAt(ctx, hub.HubID, hub.LastSize)`
and populates `CheckpointSize = hub.LastSize` + `CheckpointRoot` (base64-Std, matching the log browser
and verify-for-me) + `HasClause2 = true`; the `cert.html` `{{if .HasClause2}}` block now renders
`size N · root <b64>` with an honest coverage note. A `CheckpointAt` DB error is a 500 (buffer-then-200
already in place); a `found == false` leaves `HasClause2 = false` (no fabricated checkpoint).

**Files changed:**
- `internal/certificate/handler.go`: added `encoding/base64` import; extended `certData` with
  `CheckpointSize uint64` + `CheckpointRoot string`; in `buildData`'s `Certifiable = true` branch,
  read `CheckpointAt(hub.LastSize)` and populate §2 fields + `HasClause2` (DB err → 500, `!found` →
  leave §2 unrendered); updated the package / `certData` / `buildData` doc comments to reflect §2 being
  real (§3-§6 still placeholders).
- `internal/certificate/cert.html`: filled the empty `{{if .HasClause2}}` §2 `.clause-value` with a
  `.clause-mono` (`size {{.CheckpointSize}} · root {{.CheckpointRoot}}`) + `.clause-note` (coverage-honest
  copy referencing `{{.Position}}`); reused existing `.clause-*` CSS, no new CSS.
- `internal/certificate/handler_test.go` (test): extended `TestCertificateKnownID` to assert `§2
  CHECKPOINT`, `size 24816` (hub.LastSize = seq+1), and `base64.StdEncoding.EncodeToString([]byte("root"))`;
  added "NO §2 clause" assertions to `TestCertificateUnacceptedLeaf` (both subtests) and
  `TestCertificateNotInLog`; added `encoding/base64` import.

**Verification:** `mise run check` → green (build + vet + all 21 packages `ok`, certificate uncached).
- `go test -count=1 ./internal/certificate` → ok (uncached).
- `TestCertificateKnownID` → PASS: body contains `§2 CHECKPOINT`, `size 24816`, and the real base64-Std
  accepted root `cm9vdA==` (`EncodeToString([]byte("root"))`).
- `TestCertificateUnacceptedLeaf` (both subtests) + `TestCertificateNotInLog` → PASS: NO `§2 CHECKPOINT`
  rendered for a non-certifiable id.
- `gofmt -l .` → empty. `GOOS=js GOARCH=wasm go build ./internal/index` → OK (no-regression).
- `go.mod`/`go.sum` byte-unchanged (`encoding/base64` is stdlib).

**Mutation checks (reviewer-reproducible):**
- Neuter `data.HasClause2 = true` → `false` → `TestCertificateKnownID` FAILS (no `§2 CHECKPOINT`).
- Render a corrupted root (`EncodeToString(append(root, 0x00))`, still uses `root`) → the base64-root
  assertion in `TestCertificateKnownID` FAILS. (Note: substituting a literal `[]byte("WRONG")` instead
  leaves `root` unused → build failure; the `append` form is the clean root-only mutation.)

**Next:** §2 is sound. Proceed to the §3 INCLUSION PROOF clause + the downloadable proof-bundle
assembler — this is where the oracle/conformance gate RE-ENGAGES: the served inclusion proof must be
mutation-proven non-vacuous against the hub's `IsccLogInclusionProof` / `notecheck` (and rebuilt over
the `SQLiteFetcher`). The raw signed-note bytes for the bundle are already available from
`CheckpointAt`'s second return (`raw`, ignored here) — wire it in at §3. After §3: §4 signing key
(read via the deferred `hub_keys` reader), §5 Bitcoin anchor, §6 record history.

**Notes:**
- Oracle/conformance gate is N/A for this slice (per `next.md`): pure store read (`CheckpointAt`) + HTML
  render; no signature/RFC-6962/Merkle/did:web/fsck/proof path. The gate APPLIES starting at §3.
- No new store method, no second store round-trip beyond the single `CheckpointAt` call — `CheckpointSize`
  reuses the `hub.LastSize` carry the §1 cap already proved `> 0`, so only the root is fetched.
- `found == false` is an unreachable-in-practice honesty branch (AdvanceAccepted records the checkpoint
  at the same `tree_size` it advances `last_size` to), but it keeps the render honest (no fabricated
  root) and is fail-closed by leaving `HasClause2 = false`. It is intentionally NOT a 500 (the row's
  absence at the accepted size is not the same as a DB fault — distinct from proofserve's verify-for-me,
  which DOES 500 on `!found` because that path has already committed to serving a proof). If review
  prefers strict parity with proofserve here, that is a one-line judgment call — flagging it but I
  believe the certificate's honest-absence stance (it can decline to assert §2) is the more
  coverage-honest choice for a clause-by-clause page.
- Scope: 2 non-test source files (`handler.go`, `cert.html`) + 1 test file — within the ≤3 budget.
- Deferred items untouched as required: `internal/registry/registry.go` (the `hubDomain` `ForceQuery`
  fail-open fix), the ADR-0011 Go 1.26 / iscc-lib bump, and §3-§6.
- Design reference (`Certificate.dc.html`) uses note copy "The hub-signed (tree size, root) this proof
  is checked against." — I used the `next.md`-specified coverage-honest variant instead (it references
  `{{.Position}}` and avoids implying the proof is checked, since §3 isn't built yet). No ADR/PRD
  conflict; the design copy is subordinate per `next.md`.
