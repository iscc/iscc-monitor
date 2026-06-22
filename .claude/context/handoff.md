## 2026-06-22 — Review of: OTS upgrade-loop core (`OTSTick` over an injected `Upgrader` seam, `Attempts`/`NextRetry` back-off)

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The OTS upgrade-loop control core (`OTSTick` + `backoff`) and the store back-off seam
(`MarkOTSAttempted` + `now`-filtered `PendingOTS`) land cleanly and exactly within scope — two production
files (`otsloop.go` new, `ots.go` modified), no third slot used, no `go.mod`/`go.sum` change, no
anchoring dependency, no `main.go`/§5 touch. Reviewed the full 6-commit unpushed range (OTS stamp-pass +
upgrade-loop core); both increments are well-documented, seam-tested through observable store outputs, and
independently mutation-proven non-vacuous. `mise run check` is green, gofmt clean, store stays a leaf,
WASM purity holds, and Codex found no actionable issues.

**Verification:**
- [x] `mise run check` green — all 21 packages `ok`; `go build`/`go vet`/`go test` pass.
- [x] `gofmt -l .` (excl. `cauldron/`) empty — clean.
- [x] `go test -run TestOTS ./internal/store ./internal/follower` passes (and the broader
  `TestOTS|TestMarkOTSAttempted|TestPendingOTS|TestPollHub*` union passes) — BUT `-run TestOTS` alone
  under-selects the store tests (see Issues / filed `low`).
- [x] Confirming `Upgrader` → row `OTSStatusConfirmed` with returned `OTSBytes`/`BTCHeight`,
  `UpgradedAt == now`, absent from `PendingOTS(ctx, now+1h)`.
- [x] Declining `Upgrader` → row stays `pending`, `Attempts==1`, future `NextRetry`; excluded at same
  `now` (re-tick is a no-op), re-surfaces at `now+backoff` with `Attempts==2`.
- [x] Erroring `Upgrader` → erroring row backed off (still pending, `Attempts==1`) AND the second seeded
  row still confirmed AND `OTSTick` returns the wrapped first error (`errors.Is`).
- [x] `go list -deps ./internal/store | grep '^net/http'` empty; store imports stay
  `{context, crypto/sha256, database/sql, embed, errors, fmt, internal/tiles, modernc/sqlite, os, time}`
  (the `crypto/sha256`/`os`/`tiles` lines predate this step via `iscc_index.go`). Follower prod imports
  add no anchoring package (`{context, fmt, logclient, metrics, store, tiles, log/slog}`).
- [x] `git diff --name-only origin/develop..HEAD -- go.mod go.sum` empty — no `opentimestamps` dep added.
- [x] Mutation checks reproduced independently (all reverted, tree restored): (1) no-op
  `MarkOTSAttempted` → FAILS `TestOTSTickDeclinesBacksOff`/`TestMarkOTSAttempted` (`Attempts`/`NextRetry`
  stay zero); (2) drop the `next_retry` WHERE leg → FAILS the back-off-exclusion assertion in BOTH the
  store and follower tests (row re-processed before back-off elapses). Non-vacuous.
- [x] Scope: HEAD advance touches exactly `otsloop.go` + `ots.go` (+2 test files); `main.go`,
  `certificate/handler.go` untouched; `PendingOTS` has exactly one production caller (`otsloop.go`).
- [x] Gate-circumvention scan over `origin/develop..HEAD`: no `//nolint`, `t.Skip`, build-tag, swallowed
  error, or removed assertion. The one `_, _, err :=` (stampRoot's `RecordOTS`) discards `(id, inserted)`
  but checks `err` — correct, dedupe is silent by design.
- [x] Oracle gate correctly N/A for this slice (opaque `pending`→`confirmed`/back-off over an
  already-fsck-verified accepted root; injected `Upgrader`, no `ots verify` crypto). Confirmed the
  trust-root suite stays green regardless: `logclient`+`certificate` conformance `ok`; `derive_vkey.py`
  reproduces `40b74463`/`22b08f3e`; WASM purity build (`didweb`/`index`) OK.

**Issues found:** One `low` filed — `-run TestOTS` does not catch the OTS *store* tests
(`TestMarkOTSAttempted*` + the `TestPendingOTS` back-off path), so the documented Verification shorthand
under-selects. The tests exist, are non-vacuous, and pass under `mise run check` + the union filter, so
this is a spec-literal / convenience gap, not a coverage hole or gate weakening — does not block PASS.

**Codex second opinion:** Clean — "The new OTS upgrade-loop core and store backoff changes are internally
consistent, covered by tests, and do not appear to break existing callers. No actionable correctness
issues were found in the HEAD diff." No findings to triage; agrees with my independent review.

**Visual check:** n/a — no SSR surface changed (only `internal/follower` + `internal/store`).

**Next:** The **real `Upgrader`** — pull in `github.com/nbd-wtf/opentimestamps` (the first
`go.mod`/`go.sum` change), implement the OpenTimestamps calendar-HTTP client as a closure of the
`Upgrader` type (stamp/query/upgrade against `OTSRecord.CalendarURLs`, return
`UpgradeResult{Confirmed, OTSBytes, BTCHeight}` once Bitcoin-attested), and wire the live `Run` goroutine
into `cmd/iscc-monitor/main.go` (own ticker, `defer Stop()`, log-and-continue, off the poll path). The
`ots verify` crypto oracle gate first applies at this step. Then the `.ots` HTTP route (reads
`OTSForRoot`) and certificate §5 BITCOIN ANCHOR (`HasClause5`).

**Notes:**
- Reviewed the full 6-commit unpushed range as the orchestrator directed (the prior review attempt died
  mid-response). The OTS stamp-pass commit (3b8db9f, `follower.go`/`stampRoot`) is also clean: stamps the
  accepted root once on the verified non-violation path via `RecordOTS` (dedupes on
  `UNIQUE(hub, tree_size, root)`), placed after `fsckMirror`, never inside freeze; the follower tests
  assert exactly one `ots` row on a verified poll, zero on fork/shrink/unverified, and no second row on a
  re-poll or clean frozen re-poll.
- Reconciled and finished the partial learnings edits from the prior attempt (`follower.md`/`store.md`):
  verified them accurate against the code, then completed the `follower.md` rotation — collapsed the
  fully-landed ingestion-writer, fsck-retirement, and inclusion-cross-check sections into compact
  `settled:` summaries keeping only the durable traps. `follower.md` 313 → 243 lines (31 bullets, under
  the 40-bullet cap); a 70-line net reduction this iteration. Still above the 150-line soft target but
  the remainder is genuinely durable forward-looking content; further collapse would drop load-bearing
  traps. `store.md` at 158 lines (CRUD-dense, acceptable).
- `backoff` policy: capped exponential, base 1h, shift cap 5 (32h pre-clamp), max 24h — pure,
  deterministic, golden-tabled (`TestOTSBackoff`). Exact cadence not safety-critical (OTS best-effort,
  ADR-0004).
- Three open `normal` issues (ForceQuery fail-open, `host:port` DID on two surfaces, §6 timestamp) remain
  untouched — none lie on this path. No `critical` open.
