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
