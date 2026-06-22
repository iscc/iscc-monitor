## 2026-06-22 — Review of: Stamp pending OTS roots against the calendar in the off-path OTSTick loop

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The advance adds a `Stamper` func seam to `OTSTick` (stamp not-yet-stamped pending rows
off the poll path before the upgrade step), wires the real `otsclient.Stamp` closure via `stampFunc()`
in main.go, and adds a minimal status-untouching `store.MarkOTSStamped` mutator — closing the
`pending → Bitcoin-confirmed` transit the milestone Verify criterion names. The increment's stated goal
is met and all gates are green; the code is clean, scoped, and mutation-proven non-vacuous. Codex
surfaced three confirmed-real defects on the freshly-live stamp path (no panic-recover, no timeout, a
nil-Stamper guard-order bug) — none block this increment's goal, so they are filed as issues for a later
advance rather than failing the slice.

**Verification:**
- [x] `mise run check` green — build + vet + test, all 23 packages pass.
- [x] `gofmt -l .` (excl. `cauldron/`) — clean.
- [x] `go test -count=1 -run TestOTS ./internal/follower` — passes, and the new `TestOTSStampThenUpgrade`
  + `TestOTSStampBacksOff` DO match the `TestOTS` prefix (good; the new store tests `TestMarkOTSStamped*`
  do not, see Notes — they run under `mise run check`).
- [x] follower anchoring deps == 0 (`go list -deps ./internal/follower | grep -c` otsclient/ots/opentimestamps).
- [x] store coupling deps == 0 (`go list -deps ./internal/store | grep -c` follower/otsclient/`net/http`).
- [x] Mutation (reverted): disabling the `len(r.OTSBytes)==0` stamp branch → `TestOTSStampThenUpgrade`
  + `TestOTSStampBacksOff` FAIL; restored → green, file byte-identical.
- [x] Mutation (reverted): no-op `MarkOTSStamped` UPDATE (`WHERE … AND 1=0`) → store `TestMarkOTSStamped`
  AND follower `TestOTSStampThenUpgrade` FAIL; restored byte-identical.
- [x] `go.mod`/`go.sum` untouched; `go mod tidy -diff` clean.
- [x] `internal/follower/follower.go` byte-unchanged (deliberate rule-driven deviation: stamping off the
  poll path, not in `stampRoot` — correctly honored).
- [x] Gate-circumvention scan (`@{upstream}..HEAD`): no `//nolint`/`t.Skip`/swallow/build-tag/deleted
  assertion in code (only prose matches in handoff/docstrings).
- [x] Oracle gate correctly N/A this slice (opaque `pending → confirmed` over an already-fsck-verified
  root; no signature/RFC-6962/Merkle/did:web/proof code). Sanity: `logclient`/`didweb`/`certificate`
  suites green; `didweb` WASM build green.

**Issues found:** 3 (all from the Codex triage, all confirmed real, none block this increment):
- (normal) The production OTS stamp path has neither a panic-recover nor a per-request timeout — a
  malformed calendar response crashes the monitor, a stalled one hangs the OTS goroutine. Filed.
- (low) Nil-Stamper + an empty-OTSBytes row falls through to the Upgrader (bogus back-off) instead of
  being left untouched, contradicting the `OTSTick` docstring. Filed.

**Codex second opinion:** Available; produced a verdict (3 findings), each reviewer-triaged:
- **[P1] No panic guard on the stamp response → CONFIRMED REAL.** Verified against
  `opentimestamps@v0.4.0/stamp.go` + `parsers.go`: `Stamp` runs the calendar response through
  `parseCalendarServerResponse` → `parseTimestamp`/`readInstruction` — the IDENTICAL panic-prone parser
  `recoverRead` was created to guard — with no recover in `Stamp` or the loop, and this advance is the
  step that puts `Stamp` on the live goroutine. Filed (normal).
- **[P2] No deadline on the stamp request → CONFIRMED REAL.** `opentimestamps.Stamp` uses
  `http.DefaultClient.Do` (no deadline) and `stampFunc` passes the process ctx; the upgrade path already
  bounds this via `safeUpgrade`'s `WithTimeout`. Filed with P1 (shared fix site / root cause).
- **[P3] Nil-stamper falls through to the Upgrader on an empty row → CONFIRMED REAL (minor).** Reproduced
  with a throwaway probe: nil Stamper + empty-OTSBytes row → Upgrader invoked, Attempts=1 (bogus
  back-off), contradicting the docstring. Production always wires a non-nil Stamper, so test-only. Filed (low).
- All three confirmed; none block the increment's goal (the stamp→upgrade transit works and is
  mutation-proven), so the verdict is PASS_WITH_NOTES with the defects filed for a later advance. No
  trust-root finding (oracle gate N/A this slice).

**Visual check:** n/a — no SSR surface changed (diff is `internal/follower`, `internal/store`,
`cmd/iscc-monitor`; no `dashboard`/`dossier`/`web`/`certificate`/template touched).

**Next:** Add a `safeStamp` guard (mirror `safeUpgrade`: `context.WithTimeout(ctx, stampTimeout)` +
`recover()`-to-error around `opentimestamps.Stamp`/the response parse) and route `Stamp` through it —
this closes the open `normal` OTS issue (crash/hang on a malformed/stalled calendar) before the stamp
path is exercised against a real calendar, and is the natural sibling of the prior upgrade-path
hardening. Fold the cheap nil-Stamper guard-order fix (P3, low) into the same `otsloop.go` touch. After
that, the `.ots` HTTP route + certificate §5 BITCOIN ANCHOR (`HasClause5`) are the remaining
Verify-closers — both now have a real path to confirmed rows.

**Notes:**
- The deliberate rule-driven deviation (stamp off the poll path in `OTSTick`, not in `stampRoot`;
  `follower.go` byte-unchanged) is correctly executed and matches `next.md`'s Not-In-Scope.
- A 3rd store mutator (`MarkOTSStamped`) was genuinely required (not riding `RecordOTS` `DO NOTHING` /
  `MarkOTSUpgraded` status-flip / `MarkOTSAttempted` attempts-only) — the slice stays at the ≤3
  non-test/doc file budget (otsloop.go, main.go, ots.go), not over.
- Filter caveat reinforced (existing `low` issue #103 stands): the new store tests `TestMarkOTSStamped*`
  do NOT match the documented `-run TestOTS` shorthand (they run under `mise run check`); the follower
  stamp tests do. No new issue filed — folded into the existing one.
- learnings: `learnings/follower.md` net-reduced 259→192 lines (collapsed the settled freeze/fork-wiring
  bullets into one `settled:` summary; expanded the OTS section for the Stamper seam + the two durable
  stamp-path traps). `learnings/otsclient.md` updated to note `Stamp` is now live-but-unguarded.
- Pushed to `origin/develop` on PASS_WITH_NOTES.
