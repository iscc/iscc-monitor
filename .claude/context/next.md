# Next Work Package

## Step: Commit the 22 `go mod tidy` go.sum lines so the tidy gate is idempotent

## Goal
Resolve the open `normal` issue "`go mod tidy` adds 22 unstaged go.sum lines" by committing those
checksums as a deliberate go.sum-only change, so `go mod tidy && git diff --exit-code` passes. This
clears the dirty working tree (it currently carries exactly these 22 lines uncommitted) and unblocks
the CI + `fsck`-conformance slices that follow — a future tidy-cleanliness gate would otherwise fail
on these lines.

## Scope
- **Create**: (none)
- **Modify**: `go.sum` (the only non-test/doc file — already modified in the working tree with exactly
  the 22 lines; this step stages and commits it). `go.mod` is **not** modified.
- **Reference**:
  - `/workspace/iscc-monitor/.claude/context/issues.md` — the issue "`go mod tidy` adds 22 unstaged
    go.sum lines" (the authoritative problem statement and its "verify fixed" criterion).
  - `/workspace/iscc-monitor/.claude/context/learnings.md` — the "Consistency-proof builder" section,
    bullet "`go mod tidy` is NOT a no-op here (adds 22 go.sum lines)…" (root cause: importing
    `tessera/api` in `internal/logclient/proofbuilder.go` widened the module-graph require footprint).
  - `/workspace/iscc-monitor/go.sum` — the file being committed; inspect the 22 added lines first.

## Not In Scope
- **Do NOT author any `.github/workflows/` CI** — wiring CI + the `notecheck` oracle is the *separate*
  open `normal` issue and a distinct later step. This step only makes the tree tidy-clean so that
  future CI step can pass; it does not create the CI itself.
- **Do NOT start the `fsck` root-rebuild conformance slice** (M2 Verify) or add any tile fixtures.
- **Do NOT touch `go.mod`**, any `internal/` source, or any test — no new import, no dependency bump,
  no code change. The require graph is unchanged; only the already-present go.sum checksums are
  committed.
- **Do NOT remove or rewrite existing go.sum entries** — this is an additive, checksum-only commit. Do
  not attempt option (b) "scope tidy to compiled deps"; option (a) (commit the 22 lines) is chosen
  because it is the standard Go convention (go.sum legitimately carries module-graph checksums) and is
  the minimal change.
- **Do NOT edit `issues.md`** — per the issues-file protocol, `review` deletes the resolved entry, not
  this step nor `advance`.

## Implementation Notes
- The 22 added lines are the `h1:`/`/go.mod` checksum pairs for tessera's transitive **require-graph**
  modules that never compile into any monitor package: `cenkalti/backoff/v5`, `cespare/xxhash/v2`,
  `go-logr/logr`, `go-logr/stdr`, `transparency-dev/formats`, `go.opentelemetry.io/auto/sdk`,
  `go.opentelemetry.io/otel{,/metric,/trace}`, `golang.org/x/crypto`, `k8s.io/klog/v2`. They are valid
  checksums Go's module graph expects; adding them is safe — Go errors only on *missing* or
  *mismatched* go.sum entries, never on extra valid ones — so the build that already works without them
  stays green.
- The working tree **already holds exactly these 22 lines** (verified: `git diff --numstat go.sum` →
  `22  0  go.sum`, i.e. 22 additions, 0 deletions). With them present `go mod tidy` is already a no-op
  (verified: re-running tidy adds nothing further). So the advance is mechanical: optionally re-run
  `go mod tidy` to confirm no further change, then `git add go.sum` and commit with a clear message
  (e.g. `chore: record tessera module-graph checksums in go.sum (tidy idempotent)`).
- Do NOT re-run `go mod tidy` expecting *new* content — the divergence is already materialized in the
  working tree; this step commits it, it does not regenerate it.
- Relevant learnings rule: the "go mod tidy is NOT a no-op here" bullet explicitly lists option (a)
  ("a deliberate go.sum-only commit adding the 22 entries") as a valid resolution — this step takes it.
- Gate-integrity rule (`target.md` / `learnings.md` "Never weaken a gate to pass"): this is the
  *opposite* of a dodge — it makes a future gate pass *legitimately* by recording the checksums Go's
  module graph requires. No `//nolint`, `t.Skip`, or build-tag is involved.

## Verification
- `go mod tidy` then `git diff --exit-code -- go.sum` exits **0** (tidy is idempotent — the committed
  go.sum is exactly what tidy wants; no further lines added).
- `git diff --exit-code -- go.mod` exits **0** (go.mod untouched by this step).
- `go mod verify` prints `all modules verified` (exit 0).
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all pass) — the additive
  checksums change no compiled code path.
- `gofmt -l .` is empty (no source changed, so trivially still empty).
- `git show --stat HEAD` (the advance commit) lists **`go.sum` as the only tracked-source file
  changed** — `go.mod` and every `internal/`, `cmd/`, `testdata/` path are byte-unchanged.

## Done When
`go mod tidy && git diff --exit-code -- go.sum` exits 0 on the built tree, `mise run check` is green,
and the advance commit's only source/dependency change is `go.sum` (go.mod and all packages
byte-unchanged).
