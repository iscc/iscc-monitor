# Handoff

## 2026-06-21 — Review of: Wire `LeafHashes` + `SQLiteFetcher` into `fsck.New(...).Check(...)` — the M2 root-rebuild conformance slice

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` landed `logclient.RunFsck(ctx, vkey, origin string, f fsck.Fetcher) error` — thin
glue over tessera's `fsck.New(origin, v, f, LeafHashes, fsck.Opts{N:1}).Check(ctx)` — plus a
conformance test that synthesizes a 5-leaf tlog-tiles log in-process (one `testonly.Tree` as the single
leaf/node-hash source of truth), seeds its signed checkpoint + partial entry bundle + partial level-0
hash tile into a real `store.SQLiteFetcher`, and asserts `RunFsck` rebuilds the signed root and rejects
a one-byte corruption of either a mirrored tile OR an entry-bundle BLOB. Scope is clean (1 non-test
source file + tidy's go.mod/go.sum), the trust root is intact, and the oracle gate is independently
mutation-proven.

**Verification:**
- [x] `mise run check` green — build + vet + test, all 10 packages `ok`.
- [x] `gofmt -l .` empty.
- [x] `go test -count=1 -run TestRunFsck ./internal/logclient` PASS uncached — 3 subtests
      (`RebuildsSignedRoot` nil; `RejectsCorruptedTile`, `RejectsCorruptedBundle` non-nil). klog confirms
      the green rebuild: root `00d21829…`.
- [x] `go mod tidy && git diff --exit-code -- go.mod go.sum` exits 0 — tidy idempotent.
- [x] `grep -c '^go 1.24.0$' go.mod` == 1; `grep -c '^toolchain' go.mod` == 0.
- [x] Trust-root goldens reproduce — `go test -run 'VerifierKey|Origin' ./internal/didweb/
      ./internal/logclient/` PASS; `derive_vkey.py` external oracle prints both vectors (`40b74463`,
      `22b08f3e`) byte-for-byte.
- [x] **Oracle gate APPLIES (RFC-6962 root-rebuild crypto) and is satisfied + INDEPENDENTLY
      mutation-proven by the reviewer.** Forced `RunFsck` to `return nil` (dropped the `Check` call,
      reverted) → both corruption subtests FAIL, proving the rebuild genuinely compares the re-derived
      root against the signed checkpoint root. Prover (`testonly.Tree`) and verifier (`fsck`+`LeafHashes`)
      are independent of the fetcher; the bundle encoder is a third path. `go mod verify` → all modules
      verified; new deps trace through `logclient → tessera/fsck` (`go mod why -m`), legitimate + additive.
- [x] Gate-circumvention scan over unpushed commits (`@{upstream}..HEAD`) clean — no `//nolint`,
      `t.Skip`, build-tag exclusion, swallowed error, or deleted assertion. Diff is purely additive.
- [x] `RunFsck` has no production caller (test-only references) — intentional unused-until-wired seam,
      `go vet` clean (not dead code). Nothing from `## Not In Scope` was done (no CI workflow, no
      inclusion cross-check, no follower/`cmd` change, no live tile-ingestion writer).

**Issues found:** (none blocking). One harmless documentation nuance: the advance handoff claimed the
`logclient` *package* no longer builds for WASM after `fsck.go`; in fact `GOOS=js GOARCH=wasm go build
./internal/logclient` still exits 0 (net/http/klog/otel are js/wasm-usable). The load-bearing purity
invariant rides on `internal/didweb` (clean), not on this package — corrected in learnings. No code
change needed.

**Next:** Wire `.github/workflows/` + the `notecheck` signature-parity job (the sole open `normal`
issue) — its natural companion now that the mirror path exercises the trust-root structurally but
`notecheck` (the truly-independent external oracle) runs only in CI. Then the inclusion cross-check
against the hub's own `evidence.IsccLogInclusionProof` (the SECOND half of M2's Verify bar — needs real
captured `IsccLogInclusionProof` fixtures + an inclusion `ProofBuilder`). A live tile-ingestion writer
(making `PollHub` mirror real tiles/bundles) is the prerequisite for a *production* `RunFsck` caller.

**Notes:**
- The synthetic within-one-tile log (5 leaves) avoids the 256-leaf boundary for the first green and the
  real testnet checkpoint fixtures (whose ~10183-leaf root has no mirrored tiles). The proof path already
  crosses 256 in `proofbuilder_test.go`; a 256-crossing fsck case would strengthen but is not required
  for the Done bar.
- The `tessera/fsck` require-graph entered cleanly: klog/otel/formats in go.mod indirect, testify-family
  (testify/go-spew/difflib/yaml.v3) in go.sum only as transitive test-deps. tidy idempotent, directive
  stays `go 1.24.0`, no `toolchain` line.
- Pushed to `origin/develop` (CI-gated PR `develop → main` is the human's; never push `main`).
