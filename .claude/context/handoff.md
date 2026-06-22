## 2026-06-22 — Review of: OTS store seam — `ots`-table CRUD (`RecordOTS` / `OTSForRoot` / `PendingOTS` / `MarkOTSUpgraded`)

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance adds the typed `ots`-table CRUD seam (`internal/store/ots.go` + `ots_test.go`)
on the already-present schema table, porting the checkpoint family's exact idioms (`DO NOTHING`+`RowsAffected`
dedupe, `ErrNoRows`→miss read, `ListViolations`-shaped pending read, `SetCoverage`-style ignore-`RowsAffected`
update). Scope is tight (1 non-test source file + its test), the store stays a `net/http`-free leaf with
`schema.sql`/`go.mod`/`go.sum` byte-unchanged, and all three load-bearing claims are independently
mutation-proven non-vacuous. Codex clean; no defects found.

**Verification:**
- [x] `mise run check` (build + vet + test) — green (all 21 packages `ok`).
- [x] `go test -count=1 ./internal/store` — green (uncached, existing suite unaffected, 1.96s).
- [x] `go test -count=1 -run TestOTS ./internal/store` — green, BUT note: this filter matches only
  `TestOTSForRootAbsent` (1 of 7 OTS tests). The round-trip/dedupe/zero-times/pending/upgrade tests are
  named `TestRecordOTS*`/`TestPendingOTS`/`TestMarkOTSUpgraded*` and are NOT caught by `-run TestOTS`.
  `next.md`'s criterion is an imprecise prefix; I re-ran the full set explicitly
  (`-run 'TestRecordOTS|TestOTSForRoot|TestPendingOTS|TestMarkOTSUpgraded'`) — all 7 PASS. Functionality
  is fully covered; the only gap is the literal filter string in `next.md` (definition imprecision, not a
  code defect).
- [x] `go list -deps ./internal/store | grep '^net/http'` — empty (leaf preserved).
- [x] `git diff --stat HEAD~1..HEAD -- internal/store/schema.sql go.mod go.sum` — empty (byte-unchanged).
- [x] `gofmt -l .` (excl. `cauldron/`) — clean.
- [x] Mutation-proven non-vacuous (sed, all reverted, tree restored clean): `DO NOTHING`→plain insert
  FAILS `TestRecordOTSDedupes`; `ASC`→`DESC` FAILS `TestPendingOTS` oldest-first order; dropping
  `WHERE status=?` FAILS `TestPendingOTS`+`TestMarkOTSUpgraded`.
- [x] Oracle/conformance gate correctly N/A — opaque-BLOB round-trip + plain CRUD, no
  signature/RFC-6962/Merkle/did:web/`fsck`-rebuild path (matches the store-leaf precedent).
- [x] Quality-gate-integrity scan over all unpushed commits (`origin/develop..HEAD`) — no `nolint`,
  `t.Skip`, build tags, swallowed errors, or deleted assertions.

**Issues found:** (none) — implementation matches the established idioms exactly and the spec's Not-In-Scope
boundary was respected (no dependency, no follower wiring, no HTTP route, no schema edit, no certificate
change). The `-run TestOTS` filter imprecision is recorded above for the next `define-next` but is not an
implementation defect and does not block.

**Codex second opinion:** Clean — "The new OTS store CRUD methods match the existing store patterns, are
covered by focused tests, and the full test suite passes. I did not identify any introduced correctness
issues that warrant a review finding." No findings to triage.

**Visual check:** n/a — no SSR surface changed (diff is `internal/store` pure-SQLite-leaf code only).

**Next:** Per the OTS milestone roadmap, the next sub-step is the **daily stamp pass** that writes through
`RecordOTS` for each distinct accepted root without blocking the follower poll ("OTS never blocks the
follower"). After that: the background **upgrade loop** (reads `PendingOTS`, marks via `MarkOTSUpgraded`
once Bitcoin-confirmed) — the first step to pull in `nbd-wtf/opentimestamps` + calendar HTTP, and where
the `Attempts`/`NextRetry` retry-policy columns finally get exercised. Then the `.ots` HTTP route and
certificate §5 BITCOIN ANCHOR (`HasClause5`), both reading `OTSForRoot`.

**Notes:**
- `Attempts` and `NextRetry` are persisted + round-tripped but no method increments `Attempts` or sets a
  back-off `NextRetry` yet — correctly deferred to the upgrade loop's retry policy; not a debt for this
  slice (the seam stores what callers give it).
- The prior handoff's reported mutation-testing `git checkout`/cwd hygiene incident did NOT recur: the
  committed `ots.go` is byte-clean (no `DESC`/`IS NOT NULL` leftovers) and the working tree is clean after
  my mutation runs (I used `cp` backup + restore and verified `git diff --quiet`).
- Caution for the next reviewer: a `perl -0pi` multi-line slurp silently no-op'd one of my mutation
  substitutions (gave a false "ok"); `sed -i` with a grep-confirmed before/after applied correctly. When
  mutation-testing, confirm the source actually changed before trusting a green/red result.
- `go list -deps ./internal/store` lists `internal/tiles` (pre-existing, for `SQLiteFetcher`'s p→width)
  and bare `net`/`net/url` (from `modernc.org/sqlite`) — neither is a leak; the load-bearing invariant
  "no `net/http` in the store closure" holds.
- Open issues (ForceQuery fail-open, `host:port` DID, §6 timestamp) are untouched per Not-In-Scope and
  remain in the backlog.
