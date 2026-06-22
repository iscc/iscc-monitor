## 2026-06-22 — Review of: Real OTS `Upgrader` + `main.go` wiring — first production caller for `OTSTick`/`ots.Confirmed`

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The advance added `internal/otsclient` (the real `follower.Upgrader` closure + `Stamp`
helper, wiring `nbd-wtf/opentimestamps`' calendar calls to `internal/ots`' classifier), wired a
background `runOTSLoop` ticker into `cmd/iscc-monitor/main.go` (the first production caller of
`follower.OTSTick`/`ots.Confirmed`, closing the drift watch-line), and folded in the `>math.MaxInt64`
height fail-closed guard in `internal/ots` (resolving the open `normal` issue). All gates are green, the
new tests are mutation-proven non-vacuous, and the scope/isolation invariants hold. Codex confirmed two
real `normal` defects on the new upgrade path (a library panic that would crash the process + a
timeout-less calendar request) — both filed; neither is reachable in production yet, so they do not block
progress but keep this off a clean PASS.

**Verification:**
- [x] `mise run check` — GREEN (build + vet + test, all 23 packages incl. new `internal/otsclient`).
- [x] `gofmt -l .` (excl. `cauldron/`) — clean.
- [x] `go mod verify` — all modules verified; `go mod tidy -diff` — clean (no `opentimestamps` re-add).
- [x] `go test -count=1 -run TestOTS ./internal/ots` — pass incl. `TestOTSConfirmedHeightOverflow`.
  **Mutation-proven (reverted):** removing the `>math.MaxInt64` guard FAILS the overflow test ("want
  overflow error, got nil"); the `hello-world`/`empty`/`merkle1` golden rows still pass; restored → green.
- [x] `go test -count=1 ./internal/otsclient` — pass (5 tests). **Mutation-proven (reverted):**
  `BTCHeight: height+1` FAILS both confirmed rows; inverting the pending branch to `Confirmed: true` FAILS
  `TestUpgradeStillPending`; restored → green. Both classify branches load-bearing.
- [x] `grep -rEl "follower\.OTSTick|otsclient\." cmd/` → `cmd/iscc-monitor/main.go` — drift line closed.
- [x] WASM purity: `GOOS=js GOARCH=wasm go build ./internal/{didweb,index,badge}` OK; each has 0 deps on
  `internal/ots`/`internal/otsclient`.
- [x] Store uncoupling: `internal/ots` `database/sql` count 0; `internal/otsclient` count 1 — verified
  CORRECT (the `store.OTSRecord` seam, identical to `internal/follower`'s baseline 1). Store stays a leaf
  (0 `otsclient`/`follower`/`net/http`); follower stays anchoring-free (0 `otsclient`/`ots`/`opentimestamps`).
- [x] Oracle / trust-root gate: no proof/verify/didweb file touched; `derive_vkey.py` reproduces
  `40b74463`/`22b08f3e` exactly (scratch cleaned); `internal/ots` golden fixtures byte-identical to the
  `nbd-wtf/opentimestamps` examples (external oracle; cmp-verified vs `internal/ots/testdata`).
- [x] Gate-circumvention scan over the 3 unpushed commits: no `//nolint`/`t.Skip`/build-tag/swallowed-error
  in added code (the one `nolint`/`t.Skip` grep hit is `next.md` prose forbidding gate weakening, not code);
  no deleted assertions/tests. `recoverRead` is a documented FFI-boundary panic guard that surfaces the
  panic as a returned error — not a gate-dodge.
- [x] Binary smoke: builds; starts with the OTS goroutine wired; clean SIGINT exit 0.
- [x] Scope: 3 non-test files (`internal/ots/ots.go`, `cmd/iscc-monitor/main.go`, new
  `internal/otsclient/client.go`) within the ≤3 budget; tests + testdata not counted. Nothing in
  `## Not In Scope` touched (`follower.go` byte-unchanged, no `.ots` route, no certificate §5, no `OTSRun`
  method). The `internal/otsclient` count-1 `database/sql` deviation from a next.md one-liner is documented
  and correct (the genuine load-bearing store-uncoupling invariants hold).

**Issues found:** Two Codex-confirmed `normal` defects (combined into one issue, same line/fix-step) — see
below. Resolved + deleted: the height-overflow `normal` issue (the guard landed, mutation-proven). The
remaining open issues (`hubDomain` ForceQuery, §4/bundle `host:port` DID, §6 timestamp + 6 low) are all on
surfaces this diff did not touch — none resolved or made stale.

**Codex second opinion:** Two `[P2]` findings, both at `internal/otsclient/client.go:86` (the `upgrade()`
call). **Both triaged CONFIRMED real** against the library source:
- *Recover panics from calendar upgrades* — CONFIRMED. `UpgradeSequence` → `seq.Compute` →
  `inst.Operation.Apply`, and `opentimestamps/ots.go:46-54` panics on unimplemented ops (`sha1`/`reverse`/
  `hexlify`/`keccak256`) + invalid-instruction panics. `recoverRead` guards only `ReadFromFile`, and
  `runOTSLoop` has no `recover`, so such a proof crashes the process (violates ADR-0004). Filed `normal`.
- *Bound calendar upgrade requests* — CONFIRMED. `UpgradeSequence` uses `http.DefaultClient` (no `Timeout`)
  with the deadline-free process context; a stalled GET hangs one `OTSTick` pass. Filed `normal` (same issue).
- **Severity / progress:** both are NOT reachable in production today — `stampRoot` writes pending rows with
  EMPTY `OTSBytes` (the calendar-submit is the deferred next sub-step), so the closure fails at `recoverRead`
  and `upgrade()` is never reached (reviewer-verified with a probe: `failUpgrade` never fires on
  `OTSBytes: nil`; the closure returns "invalid ots file header '': EOF"). So they do not block progress
  (`normal`, not critical) but keep this from a clean PASS. Both must land before/with the `stampRoot`
  calendar-submit step. The hard oracles (`notecheck`, `derive_vkey.py`, hub receipt) are untouched and
  green, so no trust-root concern.

**Visual check:** n/a — no SSR surface changed (`internal/otsclient` is a network adapter, `internal/ots`
a pure classifier, `main.go` a goroutine wire; no templates, no `internal/{dashboard,dossier,web,certificate}`).

**Next:** Wire `otsclient.Stamp` into `follower.stampRoot` (`follower.go:410-421`) so a stamped pending
row carries real `OTSBytes`/`CalendarURLs` — the deferred follow-up that finally lets a root transit
pending → Bitcoin-confirmed end-to-end. **Fold the two Codex-confirmed otsclient defects into that step**
(wrap the upgrade body in a panic-recover; add a per-request `context.WithTimeout`) since that step is what
first makes `upgrade()` reachable. After it: the `.ots` HTTP route (reads `OTSForRoot`) and certificate §5
BITCOIN ANCHOR (depend on confirmed rows existing).

**Notes:**
- The OTS milestone's drift watch-line is now closed: `OTSTick`/`ots.Confirmed` have a real production
  caller. But OTS is still NOT end-to-end functional in production — until `stampRoot` submits to a
  calendar and persists `OTSBytes`, `OTSTick`'s upgrade is a structural no-op (rows have empty bytes). The
  closure + ticker + guard are the deliverable here; the end-to-end transit is the very next step.
- The two filed otsclient defects are latent (unreachable today) but real and on the now-wired path; they
  are the highest-value items to fix alongside the `stampRoot` submit, before any real `OTSBytes` flows.
- Learnings: created `learnings/otsclient.md` + index pointer row (store-seam count-1, WASM keep-out,
  pending-only upgrade, the panic + timeout traps, fixtures-are-the-oracle); marked the height-guard
  settled in `learnings/ots.md`; added the `runOTSLoop` wiring note to `learnings/cmd-monitor.md`.
- Codex finished AFTER my own review + mutations were complete (slow run, ~8 min); its two findings were
  independently re-verified against the library source before filing.
