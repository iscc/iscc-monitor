## 2026-06-22 — Review of: Confirm the two iscc-infra ops `critical`s are answered verbatim by `deploy/OPERATING.md`, then prune them

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** Context-only increment: advance deleted the two resolved iscc-infra `critical` entries
(route-exposure/`/metrics` and egress/footprint) from `issues.md` after confirming `deploy/OPERATING.md`
answers each clause. No Go source, no doc change — `OPERATING.md` is byte-identical (verified). The prune
is surgical and the gates are green; I independently confirmed each critical's Verify bar maps to the
deployed doc. Open-critical count is now 0. One PASS_WITH_NOTES caveat: Codex correctly observed the
egress/footprint bar's disk-growth clause is answered *qualitatively* (no concrete rate) — I refiled that
as a `low` because a real rate needs live testnet measurement the loop cannot perform; it does not block
the prune or DONE.

**Verification:**
- [x] `mise run check` green — 28 packages `ok` (all cached; no `.go` file touched — a no-op confirmation the prune broke nothing).
- [x] `gofmt -l .` empty — PASS.
- [x] Critical 1 absent — `! grep -q "Decide which routes are safe to publish at the public vhost"` → ABSENT (good).
- [x] Critical 2 absent — `! grep -q "Document egress + resource footprint for box sizing"` → ABSENT (good).
- [x] Open `critical` ENTRIES = 0 — `grep -cE "^- \*\*Priority:\*\* critical$"` → `0`. (The un-anchored `grep -c "Priority:\*\* critical"` next.md cited prints `1` — that single match is the permanent format-legend line `- **Priority:** critical | normal | low`, NOT an open entry. The advance's note flagged this correctly; the spirit of the gate, 0 open criticals, is met.)
- [x] Container-image tracking entry survives — `grep -q "Publish a deployable container image to GHCR"` → PRESENT (prune was surgical, not a bulk delete).
- [x] `deploy/OPERATING.md` unchanged — `git diff --stat HEAD~1..HEAD -- deploy/OPERATING.md` empty → PASS.
- [x] Clause-by-clause confirmation (the prune justification) — re-verified independently against the deployed doc I read in full:
  - **Critical 1 (routes/`/metrics`):** §"Route exposure & the `/metrics` decision" (lines 98-130) carries the **Public by design** allow-list (`/`, `/<domain>`, `/<domain>/log/…`, `/inclusion/…`, `/_ds/…`, `/healthz`, `/version`), the **"No route carries a secret … no signing key in v1 … no route needs authentication"** confirmation (lines 115-117), and the explicit **deny-`/metrics`-at-Caddy** recommendation (lines 119-130). Every clause maps.
  - **Critical 2 (egress + footprint):** §Egress (lines 132-147) lists all three egress endpoints (each hub's `/log`, each hub's `/.well-known/did.json`, the OTS calendar host); §Footprint (lines 149-166) gives RAM ("tens of MB") + CPU ("near-idle, per-poll bursts") ballparks + a disk-growth characterization + DO-alert recommendation. Egress + RAM + CPU clauses map fully; disk-growth maps qualitatively (see Codex note).
- [x] Gate-circumvention scan over unpushed commits (`@{upstream}..HEAD`) — **0 `.go` files changed in the unpushed range**; the only `nolint`/`//go:build` grep hits are in handoff/issues prose, not code. N/A (clean).
- [x] Trust-root oracle gate (notecheck / golden vectors / hub receipt) — N/A; no signature/Merkle/proof/did:web code touched (context + doc-prune only).

**Issues found:** One refiled (`low`, from the Codex triage below): "`deploy/OPERATING.md` §Footprint gives
a QUALITATIVE disk-growth answer, not a concrete per-hub/N-hub rate". No other problems — the prune is
correct, scope-clean, and honest (every deleted entry's Verify bar maps to a concrete doc line, not
deleted to shrink the count).

**Codex second opinion:** One finding, triaged. **[P2] "Keep sizing issue open until disk growth is
quantified"** (`handoff.md:34-37`) — Codex argued the egress/footprint critical was pruned before its
Verify bar is met because §Footprint says disk growth is "proportional to mirror BLOB activity" + an
alert recommendation, while the deleted bar asked for a disk-growth *rate*. **Triaged as a confirmed-real
but non-blocking weak spot → refiled `low`, not a re-block.** Reasoning: the observation is factually
correct (the disk-growth clause is qualitative-only; RAM and CPU DO carry ballparks), but a concrete rate
is **not derivable in-repo** — there are no benchmarks, no on-disk size fixtures, and the rate depends on
each testnet hub's real-world record volume + actual BLOB sizes, none of which is loop ground truth. The
§Footprint header itself says the estimates are "to be refined against live data — not measured
benchmarks", and fabricating a "~X MB/day" number would assert an un-run measurement (the exact
"asserts an un-run verification" anti-pattern this loop guards against) — *less* honest than the
qualitative answer + alert recommendation, which is the operationally-correct response to an unmeasurable
quantity. So the critical's bar is met as well as in-repo data allows; the missing number is a live-data
refinement (a human/infra observation), filed `low` so it is tracked without re-blocking the loop on a
number it cannot compute. I did NOT restore the critical: re-blocking on an in-repo-uncomputable rate is
the wrong outcome, and the doc already defers it explicitly.

**Visual check:** n/a — no SSR surface changed (context-only `issues.md`/`handoff.md` prune; zero `.go`
and zero template files touched).

**Next:** With both iscc-infra `critical`s closed and the disk-growth gap demoted to `low`, the
DONE-blocker scan is **0 critical, 5 normal** (the refiled disk-growth note is `low`, loop-skipped).
`define-next` should triage the five open `normal`s against the state→target gap. The two code/doc-closeable
ones are the `publish.yml`/`pages.yml` `workflow_dispatch` ref-guard `normal` (a small, self-contained
`if: github.ref == 'refs/heads/develop'` workflow edit — good fold-in when a workflow file is next touched,
can carry the `pages.yml` Node-20 action bumps `low` along) and the DB-migration mechanism `normal`
(ADR-0007 — a real first-migration *design* decision, wants a grilling pass, not a quick edit). The other
three `normal`s are design-honesty/scope questions (the `/` "recent declarers" footer needs lookup history
the store doesn't track; the WASM signature-half gap needs a browser did:web design pass; the per-hub-Anchor
honesty question wants a design decision) — surface them as STOP/design edges rather than spinning on chrome
if no code-closeable `normal` remains (per the standing "loop stalls on human-blocked DONE" memory).

**Notes:**
- Open issue counts after this review: **0 critical, 5 normal, 16 low** (was 0/5/15 — the disk-growth
  `low` is the one addition). DONE requires 0 critical AND 0 normal, so the loop stays CONTINUE.
- The PASS_WITH_NOTES (not plain PASS) is solely the Codex disk-growth note — the increment itself is
  flawless (correct, surgical, gate-green, honestly justified). I downgraded from PASS only to record the
  triaged caveat, not to flag a defect in the advance's work.
- No `learnings/` detail file needed an append — a doc-presence prune + a refiled `low` touches no
  recurring forward-looking CODE pitfall (read `config.md` per the Reference list to confirm the env-key
  claims `OPERATING.md` references stay accurate; they do — masthead keys + the baked-`ENV` realm story
  match). Nothing promoted to the `learnings.md` index.
- Pushing on PASS_WITH_NOTES per protocol (push allowed on PASS / PASS_WITH_NOTES); remote `origin/develop`
  is configured.
