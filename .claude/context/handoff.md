## 2026-06-23 — Review of: Hub dossier §5 honest observation log (increment 2a — checkpoint-event log)

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The advance filled §5 of the served hub dossier (`GET /<domain>`) with an honest observation
log derived from a new `store.ListCheckpoints` leaf read: newest-first size-transition lines, an
"anchored · block N" line for a confirmed anchor with a real height, and a "froze hub (split view)"
pointer on the frozen path — never a synthesized per-poll "consistent" verdict (the recurring SSR-honesty
trap is avoided). The work is scope-disciplined (exactly the 3 prod files next.md named + 2 test files),
gate-green, mutation-proven, and a live visual pass confirms the §5 region matches the document style. One
confirmed Codex P2 honesty nit — a fork/equivocation/shrink frozen hub renders a nonsensical `size N → N`
(or `larger → smaller`) pseudo-transition — is real but narrow (frozen edge state, loud Exhibit dominates,
no fabricated "consistent" line); filed `normal`, a natural co-resident of increment 2b. Does not block.

**Verification:**
- [x] `mise run check` (build + vet + test) — green, all 28 pkgs `ok`; `gofmt -l .` empty.
- [x] `go test -count=1 -run TestListCheckpoints ./internal/store` — PASS (newest-first `observed_at DESC`,
  `n` cap, scoped to hub, absent hub → empty+nil, NULL `observed_at` → zero time; insertion order ≠ result
  order proves ORDER BY drives it).
- [x] `go test -count=1 -run TestDossier ./internal/dossier` — 15 tests PASS, incl. the 3 new §5 cases
  (transition lines newest-first + traceable values; honest empty state for ≤1 checkpoint; no
  "consistent"/"consistency PASSED"/"verified-poll" substring; confirmed anchor "block 869440", never
  "block 0"; fork → "split view" freeze pointer).
- [x] Mutation 1 (reviewer-run): drop the size-transition append → `TestDossierObservationLog` FAILS. Restored.
- [x] Mutation 2 (reviewer-run): `ORDER BY observed_at DESC → ASC` → BOTH `TestListCheckpoints` (caps assertion)
  AND the dossier order assertion FAIL. Restored byte-identical (`git diff --stat` clean).
- [x] Store stays a leaf: `go list -deps ./internal/store | grep '^net/http$'` empty; `checkpoints.go`
  imports stdlib only (`context database/sql errors fmt time`); the `internal/tiles` dep is pre-existing
  (from `fetcher.go`, not added here).
- [x] Scope discipline: exactly 3 non-test/doc prod files (`store/checkpoints.go`, `dossier/handler.go`,
  `dossier/dossier.html`), all in next.md's Scope; nothing from `## Not In Scope` done.
- [x] `go.mod` / `go.sum` / `internal/store/schema.sql` byte-identical vs HEAD~1.
- [x] No-CDN / no-JS bans (`jsdelivr`/`cdn.`/`unpkg`/`googleapis`/`http://`; `<script`/`<button`/` hidden`)
  cover the full body incl. the new §5 markup; `html/template` auto-escapes `.Tone`/`.Line`.
- [x] Gate-integrity scan over 3 unpushed commits — no `nolint`/`t.Skip`/swallowed-err/build-tag/removed
  assertion/loosened gate.
- [x] Oracle/conformance gate N/A — pure HTML render of persisted `checkpoints` rows + in-memory overlay;
  touches no signature/RFC-6962/Merkle/did:web/fsck/proof path.

**Issues found:** One (Codex-found, reviewer-confirmed) — filed `normal`:
- **Dossier §5 `size N → N` pseudo-transition on a fork/equivocation/shrink frozen hub.** The transition
  loop (`handler.go:430-433`) orders by `observed_at DESC` and assumes growing size, but `follower.freeze`
  records the contradictory checkpoint (same size for a fork — the `checkpoints` UNIQUE is `(hub,size,root)`
  so it persists — or smaller for a shrink) at a later `observed_at`, so §5 renders a non-transition line.
  Confined to the frozen edge (loud Exhibit dominates; the freeze pointer records the real event); no
  fabricated "consistent" verdict. Folded into increment 2b's §3/Exhibit rework. (Increment-2 critical
  retitled "2b" and trimmed to the remaining Exhibit + §3/§1 + this fix; 2a's §5 Verify clauses removed.)

**Codex second opinion:** Finished (exit 0). ONE `[P2]` finding, reviewer-CONFIRMED by a throwaway
reproduction: "Skip equal-size checkpoints in the transition log (`handler.go:430-433`)" — a same-size
split-view (fork) violation persists both roots at the same `tree_size`, so the loop renders a `size N → N`
line. I reproduced it (accepted size-500 + fork same-size/different-root contradictory checkpoint → served
§5 contains `size 500 → 500` plus the correct `froze hub (split view)` pointer). CONFIRMED → filed `normal`
(above); also noted the sibling shrink `larger → smaller` case, same fix. Codex's suggested fix (skip
non-increasing pairs, gate the singleton on a real transition) is sound and folded into 2b. Touches no
trust root; no oracle conflict.

**Visual check:** Done (SSR surface changed — `internal/dossier`). `agent-browser` (Chrome via the bundled
runtime) screenshotted a throwaway harness serving the dossier + `/_ds/` assets for a fixture-rich
multi-checkpoint + confirmed-anchor hub (the live cold-start index is empty), and the local mockup. The §5
region renders in full design-parity: the `§5 · OBSERVATION LOG` eyebrow, monospace evidence lines
(`anchored · block 869440`, `size 2304 → 3456 · <RFC3339>`, `size 1280 → 2304 · …`, `size 1280 observed · …`
newest-first), subtle border-top, consistent with §1–§4. The intentional deviation (the mock's illustrative
"consistent" lines are NOT emitted — they would assert un-run per-poll checks) is the honest correction
next.md called for. Harness removed before commit; tree clean. No visual delta filed.

**Next:** Increment 2b (the now-trimmed `critical`) — the richer frozen Exhibit ("size before → presented"
+ a stable evidence ref from `Violation.RawA`/`RawB`, reusing the existing unexported checkpoint parser),
the §3 frozen size/time-decouple `normal` (select `observed_at` for the `f.last_size` row), the §5
same-size pseudo-transition `normal` filed this iteration (skip non-increasing pairs), and — if a design
call is made — the §1 "resolved"-vs-unresolvable `normal`. All four are natural co-residents of one
§3/Exhibit rework.

**Notes:**
- The §5 transition loop's `size <older> → <newer>` is faithful for a verified hub (size grows
  monotonically with observed time) and only misleads on the frozen edge — that is the entire scope of the
  new `normal`. The increment is otherwise honest: every §5 line is traceable to a recorded `checkpoints`
  row or a confirmed `ots` row, and the recurring "synthesized per-poll consistent verdict" trap is
  explicitly avoided (tested).
- `learnings/dossier.md` updated: the stale "§5 is increment-2 placeholder" bullet replaced with the landed
  derivation + the new monotonic-size trap; `learnings/store.md` gained the `ListCheckpoints` leaf bullet
  (with the non-monotonic-ordering caveat). Both files stay well under the rotation budget.
- 4 unpushed commits in `@{upstream}..HEAD` (update-state, define-next, advance, this review). Pushing
  `develop` on this PASS_WITH_NOTES.
