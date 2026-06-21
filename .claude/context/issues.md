# Issues

Lightweight backlog `define-next` can prioritize. Append entries; `review` deletes resolved ones.

**Format** — one entry per issue:

```
## <short title>
- **Priority:** critical | normal | low
- **Source:** [human] | [review] | [advance]
- **What / where / how to verify:** <the problem, its location, and the check that proves it fixed>
- **Spec:** <optional — target.md or an ADR section this is rooted in>
```

**Priority semantics:** `critical` preempts everything; `normal` is weighed against the state→target
gap; **`low` is skipped by the loop** (reserved for human-directed work). The `Source` tag records who
filed it and does **not** affect priority.

---

## `go mod tidy` adds 22 unstaged go.sum lines (would fail a tidy-cleanliness CI gate)
- **Priority:** normal
- **Source:** [review]
- **What / where / how to verify:** Importing `tessera/api` in `internal/logclient/proofbuilder.go`
  widened the module-graph *require* footprint, so `go mod tidy` now adds 22 go.sum checksums for
  tessera's transitive requires (otel/klog/x-crypto/formats/backoff) that **never compile** (`go list
  -deps tessera/api` is stdlib-only). The committed go.sum is byte-identical to HEAD and the build is
  reproducible under `-mod=readonly` (`go mod verify` passes), so this is harmless **today** (no CI
  exists). But a future `go mod tidy && git diff --exit-code` gate WOULD fail. Resolve before CI lands:
  either a deliberate go.sum-only commit adding the 22 entries, or a tidy step scoped to compiled deps.
  Verify fixed: after the decision, `go mod tidy` leaves go.sum clean OR CI documents the exception.
- **Spec:** CLAUDE.md "Quality gates run via mise — `mise run check` must stay green"; ADR-0003 (Go
  module hygiene).

## No CI / `notecheck` signature-parity oracle wired (`.github/workflows/` absent)
- **Priority:** normal
- **Source:** [review]
- **What / where / how to verify:** The repo has no `.github/workflows/`, so the external
  signature-parity oracle (`notecheck`) and any build/test/format/tidy gate run only locally, never in
  CI. The trust-root conformance gate (golden-vector parity, fsck root-rebuild, inclusion cross-check)
  has no CI coverage. Wire before the `fsck`-rebuild conformance slice so the mirror path faces the
  oracle in CI when it first needs to. Verify fixed: a green CI run on `develop` that builds + shells
  out to `notecheck` and runs `mise run check`.
- **Spec:** build plan conformance/oracle gate; CLAUDE.md "CID loop … Specs are the source of truth".
