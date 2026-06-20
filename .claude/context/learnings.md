# Learnings

High-signal pitfalls, patterns, and verified conventions accumulated by the `review` role. Keep each
entry specific and actionable. The entries below are **seeded from the build plan's "Correctness
rules"** — the load-bearing gotchas — so the loop knows them from iteration 1.

## Correctness rules (seeded from `.claude/plans/cosmic-baking-octopus.md`)

- **Origin = `<domain>/log`**, never the bare domain. Used for the signed-note `name` and the
  verifier key. One `origin()` helper, golden-tested against both live hubs. (Highest-probability bug.)
- **`iscc_id → seq` is one-to-many, and verification is schema-agnostic (ADR-0008).** Index by `seq`;
  store the raw `note.$schema`; lookups return a list. Unknown schemas are indexed + proof-able but
  never interpreted and never gate checkpoint acceptance.
- **Partial-tile discipline (ADR-0005).** Mark a tile/bundle BLOB `is_full` (immutable) only at
  `width == 256`; re-fetch partials (`.p/<W>`) every poll and overwrite. Never promote a partial.
- **A self-consistency violation freezes, never crashes (ADR-0006).** Three triggers —
  fork / shrink / equivocation. Persist both raw checkpoints + proof permanently, set `frozen=1`,
  alert once, keep polling evidence-only at a backed-off cadence, no auto-unfreeze, survive restart,
  other hubs unaffected.
- **Coverage honesty (ADR-0001).** Record `monitored_since`; state guarantees *from coverage start*.
  A late start backfills tiles for full data + forward guarantees but cannot retroactively detect
  pre-coverage equivocation.
- **SQLite single writer per DB (ADR-0005/0007).** WAL + `busy_timeout`; one goroutine owns all
  writes per network file; network fetch happens *outside* the write transaction.
- **OTS never blocks** the follower loop; calendars are best-effort with backoff + redundancy.
- **did:web is the only key source (ADR-0009).** Resolution failure → `unresolvable` (keep
  mirroring). A signature matching no listed key → `unverified`. Domain-compromise is out of scope.
- **`proof/verify` is pure** (no `net`/`os`/`sqlite` imports) — it is shared by the server,
  `verify-for-me`, and the WASM build. Keep it import-clean or the WASM build breaks.

## Go / tooling conventions

- `CGO_ENABLED=0` everywhere → **do not use `go test -race`** (the race detector needs cgo) and do
  not pull cgo-dependent deps. `modernc.org/sqlite` is the pure-Go SQLite driver for this reason.
- Port from `cauldron/` reference copies; never `import` them (they are gitignored, not a module dep).
