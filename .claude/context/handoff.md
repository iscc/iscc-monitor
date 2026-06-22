## 2026-06-22 — Confirm the two iscc-infra ops `critical`s are answered verbatim by `deploy/OPERATING.md`, then prune them

**Done:** Confirmed clause-by-clause that the deployed `deploy/OPERATING.md` answers both open iscc-infra
`critical` Verify bars (route-exposure/`/metrics` and egress/footprint), then deleted those two resolved
`critical` entries from `issues.md`. This was a verification + prune step — no Go source, no doc change.
There are now **0 open `critical` entries**.

**Files changed:**
- `.claude/context/issues.md`: deleted the two resolved `critical` entries ("Decide which routes are safe
  to publish at the public vhost (especially /metrics)" and "Document egress + resource footprint for box
  sizing"), title line through `**Spec:**` line each. The explanatory iscc-infra HTML comment block, the
  `---` separators, and the container-image `low` tracking entry are untouched. No other entry renumbered
  or reworded.

**Verification:** `mise run check` → exit 0, all 28 packages `ok` (all cached; no `.go` file touched — a
no-op confirmation the prune broke nothing). Per-criterion:
- [x] Critical 1 entry absent: `! grep -q "Decide which routes are safe to publish at the public vhost"` → PASS.
- [x] Critical 2 entry absent: `! grep -q "Document egress + resource footprint for box sizing"` → PASS.
- [x] 0 open `critical` ENTRIES: `grep -cE "^- \*\*Priority:\*\* critical$"` → `0` (the anchored, exact
  match; see Notes for why `next.md`'s un-anchored `grep -c "Priority:\*\* critical"` prints `1` not `0`).
- [x] Container-image tracking entry survives: `grep -q "Publish a deployable container image to GHCR"` → PASS (prune was surgical).
- [x] `deploy/OPERATING.md` unchanged: `git diff --stat deploy/OPERATING.md` empty → PASS.
- [x] `gofmt -l .` empty → PASS.

**Clause-by-clause confirmation (the justification for the prune):**
- **Critical 1 — routes / `/metrics`:** "documented allow/deny list of public paths" → §"Route exposure &
  the `/metrics` decision", the **Public by design** bulleted list (OPERATING.md lines 107-113: `/`,
  `/<domain>`, `/<domain>/log/…`, `/inclusion/…`, `/_ds/…`, `/healthz`, `/version`) PLUS the explicit
  **deny-`/metrics`-at-Caddy** recommendation for the public vhost (lines 119-130). "Confirm no route needs
  auth and none is unsafe to expose" → "**No route carries a secret** … the monitor holds **no signing key**
  in v1 … no route needs authentication" (lines 115-117). Every clause maps.
- **Critical 2 — egress + footprint:** "lists the egress endpoints" → §Egress (lines 134-147): each hub's
  `/log` tiles, each hub's `/.well-known/did.json` (did:web, ADR-0009), the OTS calendar
  `https://alice.btc.calendar.opentimestamps.org` (ADR-0004). "ballpark RAM / CPU / disk-growth for an
  N-hub realm" → §Footprint (lines 149-166): resident memory (tens of MB), near-idle CPU with per-poll
  bursts, disk-growth = mirror-BLOBs variable + the DO disk-usage-alert recommendation, scoped to the
  2-hub testnet realm. Every clause maps.

**Next:** With both iscc-infra `critical`s closed, the DONE-blocker scan is now **0 critical, 5 normal**.
The standing memory ("loop stalls on human-blocked DONE") applies: `define-next` should now triage the
five open `normal`s against the state→target gap. The doc-/code-closeable ones look like the DB-migration
mechanism `normal` (ADR-0007, a real first-migration design decision — needs a grilling pass, not a quick
edit) and the `publish.yml`/`pages.yml` `workflow_dispatch` ref-guard `normal` (a small, self-contained
workflow edit — good fold-in when a workflow file is next touched, and it can carry the `pages.yml`
Node-20 action bumps `low` along). The other three `normal`s are design-honesty/scope questions (the `/`
"recent declarers" hero footer needs a recent-lookup history the store doesn't track; the WASM
signature-half gap needs a browser did:web design pass; the per-hub-Anchor honesty question wants a design
decision) — surface them as STOP/design edges rather than spinning on chrome if no code-closeable `normal`
remains.

**Notes:**
- **`next.md` Verification expected `grep -c "Priority:\*\* critical"` to print `0`; it prints `1`.** That
  single residual match is the file's **format-legend line** (line 9: `- **Priority:** critical | normal |
  low`), which is permanent entry-format scaffold and MUST NOT be removed — it is not an open issue. The
  honest count of open `critical` ENTRIES is `0`, proven by the anchored grep
  `grep -cE "^- \*\*Priority:\*\* critical$"` → `0` (the legend's `critical | normal | low` does not match
  `critical$`). So the spirit of the gate (0 open criticals) is met; the literal un-anchored count is an
  artifact of the legend line, not a stale entry. Flagging so `review` doesn't read the `1` as a missed prune.
- Honesty check passed: the prune is justified **only because** every Verify clause maps to a concrete line
  in the deployed doc (mapping above), not to make the count look smaller. No clause was found unanswered,
  so no entry was left open.
- Per ADR-0013 Consequences, the remaining iscc-infra residual (the Caddy deny-rule enforcement, box
  sizing, the per-instance `/metrics` exposure choice) is explicitly **non-loop-gating** — it lives in
  iscc-infra, not this repo, and does not gate DONE here. The doc-closeable half (which the loop owns) is
  what these two entries tracked, and it is closed.
- No `learnings/` detail file needed an append — a doc-presence prune touches no recurring forward-looking
  pitfall (read `config.md` per the Reference list to confirm the env-key claims `OPERATING.md` references
  via CLAUDE.md's table stay accurate; they do). Nothing promoted to the `learnings.md` index.
