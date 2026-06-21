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
  has no CI coverage. **Now UNBLOCKED** — `cmd/notecheck` compiles in-repo, so CI need only
  `go build ./cmd/notecheck` and shell the binary out against a captured checkpoint (assert `OK <name>`
  + exit 0, and exit 1 on a corrupted one). NOTE: CI doing `go build ./...` on a fresh checkout must
  account for gitignored `cauldron/` (it never reaches CI, but `./cmd/notecheck` sidesteps it cleanly).
  Wire before the inclusion-cross-check conformance slice. Verify fixed: a green CI run on `develop`
  that builds + shells out to `notecheck` and runs `mise run check`.
- **Spec:** build plan conformance/oracle gate; CLAUDE.md "CID loop … Specs are the source of truth".

## `cmd/notecheck`'s `run` has a vestigial `out io.Writer` parameter
- **Priority:** low
- **Source:** [review]
- **What / where / how to verify:** `cmd/notecheck/main.go` `run(vkey string, in io.Reader, out
  io.Writer) (string, error)` never writes to `out` — it returns the signer name and `main` prints
  `OK %s` to `os.Stdout` itself. The param matches the literal signature `next.md` specified and is
  harmless (tests pass a throwaway buffer; `go vet` does not flag unused params), but the signature
  is misleading. Fix when `run` is next touched: drop `out`, OR have `run` print `OK %s` to `out` and
  let the test assert on it. Verify fixed: `out` is either gone or written to. Low — skipped by the loop.
- **Spec:** KISS / YAGNI (CLAUDE.md code standards); no spec contract.
