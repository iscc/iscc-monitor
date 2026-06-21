# Handoff

## 2026-06-21 — Commit the 22 `go mod tidy` go.sum lines so the tidy gate is idempotent

**Done:** Staged and committed the 22 already-present `go.sum` checksum lines (tessera's transitive
module-graph requires) as a deliberate go.sum-only change, making `go mod tidy` idempotent against
the committed tree. No code, no `go.mod`, no test touched.

**Files changed:**
- `go.sum`: +22 lines, 0 removed — the `h1:`/`/go.mod` checksum pairs for tessera's require-graph
  modules that never compile into any monitor package (`cenkalti/backoff/v5`, `cespare/xxhash/v2`,
  `go-logr/logr`, `go-logr/stdr`, `transparency-dev/formats`, `go.opentelemetry.io/auto/sdk`,
  `go.opentelemetry.io/otel{,/metric,/trace}`, `golang.org/x/crypto`, `k8s.io/klog/v2`).

**Verification:** `mise run check` → green (build + vet + test, all 10 packages pass). Per-criterion:
- [x] `go mod tidy` added **nothing** to the working-tree go.sum (byte-identical diff before/after
      tidy → tidy is idempotent). After this commit, `go mod tidy && git diff --exit-code -- go.sum`
      exits **0** against HEAD (re-verified post-commit, see below).
- [x] `git diff --exit-code -- go.mod` exits **0** (go.mod byte-untouched).
- [x] `go mod verify` → `all modules verified` (exit 0).
- [x] `mise run check` green — additive checksums change no compiled code path.
- [x] `gofmt -l .` empty (no source changed).
- [x] `git show --stat HEAD` lists **`go.sum` as the only tracked-source change** (handoff.md is the
      only other file; no `go.mod`, `internal/`, `cmd/`, or `testdata/` path touched).

**Next:** The `fsck` root-rebuild conformance slice (M2 Verify) — the first slice to face the
trust-root oracle in CI, needing real tile fixtures and `fsck.New(...).Check(...)` over the
`SQLiteFetcher`. The remaining open `normal` issue (wire `.github/workflows/` CI + the `notecheck`
oracle) is the natural companion to that slice, since the tree is now tidy-clean and a
`go mod tidy && git diff --exit-code` CI gate will pass on these lines.

**Notes:**
- **The pre-commit `git diff` showing the 22 lines (exit 1) is expected, not a failure.** The
  `next.md` criterion phrases it as "tidy is idempotent — the *committed* go.sum is exactly what tidy
  wants." Pre-commit the lines live only in the working tree, so `git diff` against HEAD shows them;
  the load-bearing fact is that `go mod tidy` itself added **nothing** (proving idempotence), and the
  exit-0 holds once HEAD carries them. Re-verified after committing.
- **This is the opposite of a gate dodge** — it records the checksums Go's module graph legitimately
  requires (Go errors only on *missing*/*mismatched* go.sum entries, never on extra valid ones), so a
  future `go mod tidy && git diff --exit-code` CI step passes cleanly. No `//nolint`, `t.Skip`, or
  build-tag involved.
- **Scope is exactly one source file.** Per the issues-file protocol I did **not** edit `issues.md`
  (`review` deletes the resolved entry). No CI workflow authored (separate open issue, out of scope).
