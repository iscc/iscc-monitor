# Handoff

## 2026-06-21 — Wire `LeafHashes` + `SQLiteFetcher` into `fsck.New(...).Check(...)` — the M2 root-rebuild conformance slice

**Done:** Landed `logclient.RunFsck(ctx, vkey, origin string, f fsck.Fetcher) error` — thin glue that
builds a `note.Verifier` from the vkey and runs tessera's `fsck.New(origin, v, f, LeafHashes,
fsck.Opts{N:1}).Check(ctx)`, wrapped with `%w`. Proved end-to-end by a conformance test that
synthesizes a 5-leaf tlog-tiles log in-process (`testonly.Tree` as the single leaf/node-hash source of
truth), seeds its signed checkpoint + partial entry bundle + partial level-0 hash tile into a real
`store.SQLiteFetcher`, and asserts `RunFsck` rebuilds the signed root and rejects a one-byte corruption
of either a mirrored tile OR an entry-bundle BLOB. `go mod tidy` pulled the `tessera/fsck`
require-graph (`klog/v2`, `otel*`, `x/sync`, `transparency-dev/formats`) into the `// indirect` block;
directive stays `go 1.24.0`, no `toolchain` line.

**Files changed:**
- `internal/logclient/fsck.go` (new): `RunFsck` — takes the `fsck.Fetcher` interface (NOT concrete
  `store.SQLiteFetcher`), so production `logclient` gains no `store` import edge. Imports are exactly
  `context` + `fmt` + `tessera/fsck` + `x/mod/sumdb/note` (+ same-package `LeafHashes`).
- `internal/logclient/fsck_test.go` (new): conformance test in external `package logclient_test` (may
  import both `logclient` and `internal/store`). `seedMirror` is the single source of truth; three
  subtests — `RebuildsSignedRoot` (nil), `RejectsCorruptedTile`, `RejectsCorruptedBundle` (both non-nil).
- `go.mod` / `go.sum`: `go mod tidy` added the `tessera/fsck` require-graph indirect entries.

**Verification:** `mise run check` → green (all 10 packages `ok`; build + vet + test). Per-criterion:
- [x] `mise run check` green; `gofmt -l .` empty.
- [x] `go test -count=1 -run TestRunFsck ./internal/logclient` PASS uncached — 3 subtests: 1 nil over
      correctly-seeded `SQLiteFetcher`, 2 non-nil after a one-byte corruption (tile + bundle).
- [x] `go mod tidy && git diff` idempotent — re-running tidy produces NO diff vs the committed state
      (confirmed by snapshot-diff: `go.mod` + `go.sum` both UNCHANGED by re-tidy).
- [x] `grep -c '^go 1.24.0$' go.mod` == 1; `grep -c '^toolchain' go.mod` == 0.
- [x] Trust-root goldens unchanged: `go test -run 'VerifierKey|Origin' ./internal/didweb/
      ./internal/logclient/` PASS.
- [x] **Oracle gate APPLIES (RFC-6962 root-rebuild crypto) and is satisfied + mutation-proven.** Forced
      `RunFsck` to always return `nil` (throwaway, reverted) → both corruption subtests FAIL, proving the
      rebuild genuinely compares against the checkpoint root. klog confirms the green case: "Successfully
      fsck'd log with size 5 and root 00d21829…". The prover (`testonly.Tree`) and verifier (`fsck` +
      `LeafHashes`) are independent of the fetcher under test; the bundle encoder (manual uint16 framing)
      is a third path distinct from `EntryBundle.UnmarshalText`. The fully-independent `notecheck` oracle
      is the deferred CI companion (documented in the test).

**Next:** The CI / `notecheck` workflow (the open `normal` issue) is now the natural companion — this
slice first exercises the mirror path against the trust-root oracle structurally, but `notecheck` runs
only in CI and `RunFsck` has no production caller yet. Two follow-on slices: (1) wire `.github/workflows/`
+ `notecheck` so the mirror faces the external oracle; (2) the inclusion cross-check against the hub's
own `evidence.IsccLogInclusionProof` (the SECOND half of M2's Verify bar, needs real captured
`IsccLogInclusionProof` fixtures + an inclusion `ProofBuilder`). A live tile-ingestion writer (making
`PollHub` mirror real tiles/bundles) is the prerequisite for a *production* fsck caller.

**Notes:**
- `RunFsck` is an intentional **unused-until-wired** export seam (like the consistency triggers,
  `IsFull`, `LeafHashes`, `LookupHubKey`) — `go vet` clean, NOT dead code. No production caller was
  added (explicitly out of scope per `next.md`; a periodic fsck loop is a later slice). Its first caller
  needs the live tile-ingestion writer, which doesn't exist yet.
- **WASM purity is file-level (per learnings).** `fsck.go` pulls `tessera/fsck` → `net/http`/`otel`/
  `klog`, so the `logclient` *package* no longer builds for WASM — but `fsck` is a server-side mirror
  check, never WASM, and the load-bearing invariant is the OTHER files' import sets (`verify.go`,
  `leafhasher.go`, `proofbuilder.go` unchanged). `fsck.go`'s imports are exactly `context`+`fmt`+
  `tessera/fsck`+`note`.
- The `go.mod`/`go.sum` indirect additions (`klog/v2`, `go-logr`, `otel`+`auto/sdk`+`metric`+`trace`,
  `x/sync`, `transparency-dev/formats`, `cespare/xxhash`) are the `tessera/fsck` require-graph the
  earlier go.sum-only commit (79e5e37) already recorded the checksums for — they now move from go.sum
  into go.mod's indirect block as genuine compile-graph requirements (fsck IS compiled by `fsck.go`).
  No existing entry was rewritten. tidy is idempotent and `go vet`/build/test all green.
- The synthetic keypair (`note.GenerateKey`) and within-one-tile log (5 leaves) avoid the 256-leaf
  boundary for the first green and avoid reusing the real testnet checkpoint fixtures (whose ~10183-leaf
  root has no mirrored tiles here). An optional 256-crossing case would strengthen but is not required
  for the Done bar; the existing `proofbuilder_test.go` already exercises the boundary for the proof path.
