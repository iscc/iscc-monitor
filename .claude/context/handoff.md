## 2026-06-22 — Review of: Write the deployment/operability doc (`deploy/OPERATING.md`)

**Verdict:** NEEDS_WORK
**Loop:** CONTINUE

**Summary:** The advance added a single tracked operator doc (`deploy/OPERATING.md`, 219 lines,
doc-only — no Go/test/Dockerfile/workflow touched), scope-clean and well-sourced: every code fact I
spot-checked is correct (uid 65532, bind `:9464`/no host-publish, baked realm path, OTS calendar host,
WAL `-wal`/`-shm` siblings, single-writer `SetMaxOpenConns(1)`, `GET /version` JSON shape, SIGTERM
drain, `:develop`+`:sha-<short>` tags, the single-mux route list). But the doc's headline deliverable —
the "copy-pasteable" quick-start — **does not boot**: it omits the REQUIRED `ISCC_MONITOR_REALM` (Codex
P1, reviewer-confirmed) and mounts a fresh root-owned volume the non-root uid cannot write (Codex P2).
Both are reviewer-confirmed against the code, so the doc cannot yet close the persistence `critical`.

**Verification:**
- [x] `mise run check` green — 28 packages `ok` (doc-only; gate stayed green)
- [x] `gofmt -l .` clean (exit 0, no files listed)
- [x] `test -f deploy/OPERATING.md` — PASS
- [x] `grep monitor-test.iscc.io` — PASS (names the first-consumer instance)
- [x] `grep 65532` — PASS (and verified === Dockerfile uid)
- [x] `grep :9464` + `no host port|publishes no` — PASS (verified === `EXPOSE 9464`, no host-publish)
- [x] `grep -- -wal` + `backup` — PASS (verified === `PRAGMA journal_mode=WAL`)
- [x] `grep /metrics` — PASS (verified === single-mux `serveMetrics`; exposure decision recorded)
- [x] `grep alice.btc.calendar.opentimestamps.org` + `did.json` — PASS (verified === `DefaultCalendarURL`)
- [x] `grep 'recreate the volume|schema change'` — PASS (interim migration policy stated)
- [x] CLAUDE.md anchor `#running-a-local-dev-instance` — matches live `## Running a local dev instance`
- [x] Quality-gate integrity — scanned all unpushed commits (`@{upstream}..HEAD`, 3 commits); only
  non-`.claude` file is `deploy/OPERATING.md`. No `nolint`/`t.Skip`/skipped-or-deleted tests/loosened
  gates (the lone grep hits are handoff prose).
- [ ] **Manual content check — quick-start examples do not boot.** The doc's "valid `ISCC_MONITOR_REALM`
  out of the box" claim (l.65-68) and both quick-start snippets (l.192-193 Compose, l.209-215 `docker
  run`) are factually wrong: `config.Load` requires `ISCC_MONITOR_REALM` (`config.go:123`) and the
  Dockerfile sets NO `ENV` (only `COPY`s the file). So the persistence `critical` is not yet satisfied.

**Issues found:**
- **[critical]** `deploy/OPERATING.md` quick-start omits the required `ISCC_MONITOR_REALM` → snippets
  exit at startup with `config: required key "ISCC_MONITOR_REALM" is missing`. Filed (Codex P1).
- **[normal]** quick-start mounts a fresh `root:root` named volume that uid 65532 cannot write →
  `store.Open` fails permission-denied; contradicts the doc's own uid-65532 requirement. Filed (Codex P2).
- The 3 iscc-infra `critical`s (persistence, exposure, egress) stay OPEN — the doc answers the substance
  but the non-booting boot instructions mean the persistence ask's contract is not yet correctly stated.

**Codex second opinion:** Two findings, both reviewer-CONFIRMED → filed as issues (not dismissed):
- **[P1] CONFIRMED → critical issue.** Quick-start omits `ISCC_MONITOR_REALM`. Verified: `config.Load`
  calls `required(get, keyRealm)`; Dockerfile has ZERO `ENV` lines and no Go code defaults `RealmPath`,
  so the baked realm FILE does not make the VAR set — the snippets do not boot.
- **[P2] CONFIRMED → normal issue.** Fresh named volume is root-owned; image runs as uid 65532, so
  `store.Open` cannot create `/data/monitor.db`. The doc states the uid-65532 requirement but the runnable
  snippet doesn't satisfy it.
- (Note: Codex took ~4 min and explored far beyond the doc — for a doc-only change it still surfaced two
  real, actionable deployment blockers. The hard oracles are N/A here — no trust-root path touched.)

**Visual check:** n/a — no SSR surface changed (a Markdown doc, no template/handler/`.dc.html` touched).

**Next:** Fix the `deploy/OPERATING.md` quick-start (the active step's deliverable) — the smallest correct
change is to set `ISCC_MONITOR_REALM=/etc/iscc-monitor/realm.txt` explicitly in BOTH snippets and correct
the "valid out of the box" sentence, AND add a one-line volume-prep note (pre-`chown 65532:65532` or a
65532-writable bind mount) so the quick start boots; OR add `ENV ISCC_MONITOR_REALM=...` to the Dockerfile
(makes the "out of the box" claim true, but that is a Dockerfile change — weigh against the doc-only fix).
Then the 3 iscc-infra `critical`s can be cleared and the root `README.md` is the last DONE gate.

**Notes:**
- The doc's SUBSTANCE is strong and accurate — this is NOT a rewrite; it is two precise corrections to the
  quick-start + one overstated sentence. The persistence/exposure/egress content all maps correctly to the
  3 critical asks; only the runnable examples are wrong.
- Durable trap recorded in `learnings/config.md`: the baked realm FILE is not a set realm-config VAR — the
  Dockerfile `COPY`s the file but sets no `ENV`, so `ISCC_MONITOR_REALM` stays required in the image. Any
  future deploy doc/snippet must set it (or the Dockerfile must add the `ENV`).
- DONE blocked: 1 open `critical` (the new doc-boot fix) + the pre-existing 3 iscc-infra `critical`s, and
  the root `README.md` `normal` is still the last `target.md` "Done When" gate.
