## 2026-06-22 — Review of: Pin the Compose volume name so the OPERATING.md volume-prep `chown` matches what Compose mounts

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** Doc-only, one-file fix: `deploy/OPERATING.md`'s Compose `volumes:` block now pins
`name: monitor-data` so the documented `chown` prep (`docker run --rm -v monitor-data:/data alpine
chown -R 65532:65532 /data`) operates on the SAME engine volume `docker compose up` mounts, plus a
one-clause prose note explaining the `name:` pin is what aligns the literal-name prep with the mount.
The change is scope-clean (zero Go, no new Compose file, no other section touched), internally
consistent (Compose path and bare `docker run` path now both use the literal `monitor-data`), and
all gates are green. This was the last doc caveat on the persistence-contract `critical`, which I
verified is now answered end-to-end and closed.

**Verification:**
- [x] `grep -nA2 '^volumes:'` shows `monitor-data:` with child `name: monitor-data` (lines 204-206) — PASS.
- [x] Literal volume name matches: `chown` prep target (`-v monitor-data:/data`) == pinned Compose `name:` (both `monitor-data`); `grep -c monitor-data` = 10 — PASS.
- [x] `mise run check` green — 28 packages `ok` (all cached; no `.go` file touched) — PASS.
- [x] `gofmt -l .` empty — PASS.
- [x] No tracked Compose file added: `test ! -f docker-compose.yml && test ! -f compose.yaml` exits 0 — PASS.
- [x] Other persistence claims intact: `recreate the volume on a schema change`, `65532`, `/etc/iscc-monitor/realm.txt` all still match — PASS.
- [x] Persistence `critical` Verify bar met end-to-end: DB path + volume + "back up this one file" (§State, volume & backup, lines 40-59), non-root uid 65532, migration issue linked with "recreate volume on schema change" interim policy (§Migration policy, lines 71-84) — PASS, critical closed.
- [x] Gate-circumvention scan over unpushed commits (`@{upstream}..HEAD`): doc-only, no `nolint`/`t.Skip`/build-tag/deleted-assertion — N/A (clean).
- [x] Trust-root oracle gate (notecheck / golden vectors / hub receipt): N/A — no signature/Merkle/proof code touched.

**Issues found:** (none) — the fix is correct and complete. Closed the `normal` "Compose volume-prep
`chown` targets the wrong volume" (now fixed + verified) and the `critical` "Persistence contract for
the SQLite DB volume + acknowledge the in-place migration hazard" (Verify bar met end-to-end). The
migration-hazard half remains tracked separately by the standing `normal` "No on-disk DB migration
story" (which the doc now explicitly links).

**Codex second opinion:** Clean. Verdict: "The change only updates deployment documentation to pin the
Compose volume name and align the documented chown command with the mounted volume. The Compose syntax
and surrounding instructions are consistent with the stated deployment flow, and no blocking
correctness issues are introduced." No findings to triage; matches my independent assessment.

**Visual check:** n/a — no SSR surface changed (doc-only `deploy/OPERATING.md` edit).

**Next:** Confirm/prune the remaining two iscc-infra `critical`s, both answered in substance by
`deploy/OPERATING.md` and reviewer-spot-checked this iteration: (1) **route exposure / `/metrics`** —
§"Route exposure & the `/metrics` decision" (lines 98-130) carries an explicit allow/deny
recommendation (deny `/metrics` at Caddy, scrape internal); (2) **egress + footprint** — §Egress
(lines 132-147: hub `/log` + `did.json` + OTS calendar host) + §Footprint (lines 149-166: RAM/CPU
estimates + the disk-growth-of-mirror-BLOBs variable + a DO disk-usage-alert recommendation). Both have
doc-closeable Verify bars that look met; `define-next` should confirm each verbatim and prune, or — if
either is judged to need genuine human/infra acceptance rather than a doc — surface it as a STOP edge
rather than spinning on cosmetic chrome (per the standing "loop stalls on human-blocked DONE" memory).

**Notes:**
- After this commit: **2 critical, 5 normal** open (was 3 critical, 6 normal). DONE requires 0 critical
  AND 0 normal, so the loop stays CONTINUE.
- The `name:` pin is standard Compose syntax (a `name:` key nested under the volume-name key); I
  visually confirmed the indentation (2 spaces `monitor-data:`, 4 spaces `name:`). PyYAML was not
  available in the env to machine-validate, but the fragment is trivially well-formed.
- No `learnings/` detail file applies — a one-time doc-correctness fix with no recurring
  forward-looking pitfall (advance read `config.md` to confirm the `ENV`/no-default claims stay
  accurate; they do). Nothing promoted to the `learnings.md` index.
- The remaining `normal`s (DB-migration hazard, `/` "recent declarers" footer, WASM signature-half gap,
  per-hub-Anchor design-honesty question, `publish.yml` ref-guard) are each their own later step; none
  was touched here. Fold-in candidate when a workflow file is next touched: the
  `publish.yml`/`pages.yml` `workflow_dispatch` ref-guard + the `pages.yml` Node-20 action bumps.
