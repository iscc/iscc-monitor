# Next Work Package

## Step: Confirm the two iscc-infra ops `critical`s are answered verbatim by `deploy/OPERATING.md`, then prune them

## Advances
This step closes the **two open `critical` issues** that are the SOLE remaining DONE blockers (state.md
"These are now the SOLE DONE blockers"; handoff `**Next:**` "Confirm/prune the remaining two iscc-infra
`critical`s"). They preempt all milestone/chrome work because **DONE requires 0 critical AND 0 normal**
(target.md "Done When"), and every other code/doc-closable M-Deploy Verify item has already landed.

Both criticals' Verify bars are **in-repo / doc-presence bars** that ADR-0013 explicitly assigns to the
loop (Consequences: "The CID loop owns and can verify in-repo / CI ... the deployment/operability docs
(volume/backup, **egress, exposure policy**, grace period)"). The enforcement half (Caddy deny-rule,
box sizing, the operator's per-instance choice) is the NON-loop-gating iscc-infra residual (ADR-0013:
"the per-instance `/metrics` exposure choice ... live in iscc-infra ... do not gate DONE here"). So the
doc-closeable Verify bars are met and the entries are now stale bookkeeping to prune — this is the
documented next action, NOT cosmetic chrome, and NOT a STOP edge (the work is closeable in-repo and the
doc already closes it).

The two `critical` entries and their Verify bars:
1. **"Decide which routes are safe to publish at the public vhost (especially /metrics)"** — Verify: "a
   documented allow/deny list of public paths for the vhost ... Confirm no route needs auth and none is
   unsafe to expose."
2. **"Document egress + resource footprint for box sizing"** — Verify: "a short 'deployment footprint'
   note lists the egress endpoints and ballpark RAM / CPU / disk-growth for an N-hub realm."

## Goal
Verify each critical's Verify bar is satisfied **clause-by-clause, verbatim** by the current
`deploy/OPERATING.md`, then delete the two resolved `critical` entries from `issues.md` so the
critical count drops to 0 — taking the loop one prune away from a clean DONE-blocker scan.

## Scope
- **Create**: (none)
- **Modify**: `.claude/context/issues.md` (delete the two resolved `critical` entries; this is the
  `issues.md` "review deletes resolved ones" prune, performed by advance because the confirmation IS
  the close). NO Go source, NO doc change — `deploy/OPERATING.md` already answers both.
- **Reference**:
  - `/workspace/iscc-monitor/deploy/OPERATING.md` — the doc that answers both criticals
    (§"Route exposure & the `/metrics` decision" lines 98-130; §Egress lines 132-147; §Footprint lines
    149-166).
  - `/workspace/iscc-monitor/.claude/adr/0013-server-packaging-and-deployment.md` — Decisions 6 (route
    exposure / `/metrics` is the operator's choice, not a code gate) + 8 (egress) and the Consequences
    block drawing the loop-owned vs iscc-infra line.
  - `/workspace/iscc-monitor/.claude/context/learnings/config.md` — confirms the env-key claims the doc
    references stay accurate (the doc points at CLAUDE.md's env table rather than duplicating it).

## Not In Scope
- **Do NOT edit `deploy/OPERATING.md`, the Dockerfile, the Compose snippet, or any Go source.** The doc
  already answers both criticals verbatim; touching it would be unprompted scope creep. If a clause is
  found genuinely UNanswered, do NOT prune that entry — instead leave it open and record the precise gap
  (then this step closes only the entry that IS answered).
- Do NOT delete or re-prioritize any `normal` or `low` entry — only the two iscc-infra `critical`s.
  Specifically leave open: the DB-migration `normal`, the `/` "recent declarers" footer `normal`, the
  WASM signature-half `normal`, the per-hub-Anchor `normal`, the `publish.yml` ref-guard `normal`, and
  the "Publish a deployable container image to GHCR" tracking entry (already demoted to `low`).
- Do NOT touch the `pages.yml`/`publish.yml` ref-guard or Node-20 bumps — those are separate `normal`/
  `low` fold-in candidates for when a workflow file is next edited, not this step.
- Do NOT enforce anything in Caddy / size any box — that is the explicitly non-loop-gating iscc-infra
  residual (ADR-0013 Consequences).

## Implementation Notes
This is a **verification + prune** step, not a code change. Procedure for `advance`:

1. **Re-read `deploy/OPERATING.md` and check each critical's Verify clauses against it.** Map each
   clause to a concrete line/section before pruning — a prune is only justified if EVERY clause is met:
   - Critical 1 ("routes / `/metrics`"):
     - "documented allow/deny list of public paths" → §"Route exposure", the bulleted **Public by
       design** list (lines 104-113: `/`, `/<domain>`, `/<domain>/log/…`, `/inclusion/…`, `/_ds/…`,
       `/healthz`, `/version`) PLUS the explicit **`/metrics` deny recommendation** for the public vhost
       (lines 119-130).
     - "Confirm no route needs auth and none is unsafe to expose" → "**No route carries a secret** ...
       the monitor holds **no signing key** in v1 ... no route needs authentication" (lines 115-117).
   - Critical 2 ("egress + footprint"):
     - "lists the egress endpoints" → §Egress (lines 134-147): each hub's `/log` tiles, each hub's
       `/.well-known/did.json` (did:web, ADR-0009), the OTS calendar
       `https://alice.btc.calendar.opentimestamps.org` (ADR-0004).
     - "ballpark RAM / CPU / disk-growth for an N-hub realm" → §Footprint (lines 149-166): resident
       memory (tens of MB), near-idle CPU with per-poll bursts, and the **disk-growth = mirror BLOBs**
       variable + the DO disk-usage-alert recommendation, scoped to the 2-hub testnet realm.
2. **Both bars are met** (verified at scope time, every clause maps). So **delete both `critical` entries
   in full** from `.claude/context/issues.md` — the two `## …` blocks under the
   "pre-deployment asks from the iscc-infra ops side" comment (the "Decide which routes are safe to
   publish at the public vhost" entry and the "Document egress + resource footprint for box sizing"
   entry). Leave the explanatory HTML comment block and the `## Publish a deployable container image to
   GHCR` `low` tracking entry intact.
3. **Edit hygiene:** delete only those two complete entries (title line through their `**Spec:**` line),
   leaving the surrounding entries and the `---` separators well-formed. Do not renumber or reword any
   other entry.

Correctness rule in play (from `learnings.md` always-loaded index): none of the seeded **Correctness
rules** govern a doc-prune (no `origin`/Merkle/freeze/OTS/did:web code path is touched) — the relevant
constraint is the issues.md format contract ("`review` deletes resolved ones") and the target.md "Done
When" gate (0 critical AND 0 normal). The single judgment call is honesty: prune ONLY because the
Verify bar is met in the deployed doc, never to make the count look smaller.

## Verification
- `mise run check` is green (no Go file touched — all packages cached `ok`; a no-op confirmation that
  the prune broke nothing): `mise run check` exits 0.
- The two iscc-infra critical entries are gone from `issues.md`:
  `! grep -q "Decide which routes are safe to publish at the public vhost" .claude/context/issues.md`
  exits 0 AND
  `! grep -q "Document egress + resource footprint for box sizing" .claude/context/issues.md` exits 0.
- The open `critical` count is now 0:
  `grep -c "Priority:\*\* critical" .claude/context/issues.md` prints `0`.
- The container-image tracking entry survives (proves the prune was surgical, not a bulk delete):
  `grep -q "Publish a deployable container image to GHCR" .claude/context/issues.md` exits 0.
- `deploy/OPERATING.md` is unchanged: `git diff --stat deploy/OPERATING.md` shows no change.
- `gofmt -l .` is empty.

## Done When
`mise run check` and `gofmt -l .` are green, both iscc-infra `critical` entries are deleted from
`issues.md` (the two greps above confirm absence, the critical count is 0, and the container-image
tracking entry plus `deploy/OPERATING.md` are untouched), having first confirmed each critical's Verify
bar is answered clause-by-clause in the deployed `deploy/OPERATING.md`.
