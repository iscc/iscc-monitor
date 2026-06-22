## 2026-06-22 — Review of: Bake `ISCC_MONITOR_REALM` into the image so the deploy quick-start boots

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The advance baked `ENV ISCC_MONITOR_REALM=/etc/iscc-monitor/realm.txt` into the Dockerfile,
dropped the now-redundant `-e ISCC_MONITOR_REALM` from the CI `docker` smoke job (so a realm-less boot
to `/healthz` 200 mechanically proves the bake), and corrected `deploy/OPERATING.md`'s phantom-default
wording + added a uid-65532 volume-prep note. Scope-clean (3 in-scope files + handoff), no Go source
touched, all gates green. The realm-bake `critical` is correctly closed; one Codex-confirmed `normal`
remains — the new volume-prep `chown` targets the wrong volume for the Compose path.

**Verification:**
- [x] `mise run check` green — 28 packages `ok` (cached; no Go source changed, gate re-confirmed).
- [x] `gofmt -l .` empty.
- [x] Dockerfile sets the var — `grep -n 'ENV ISCC_MONITOR_REALM=/etc/iscc-monitor/realm.txt' Dockerfile`
  → exactly one line (51), placed right after the realm-doc `COPY`.
- [x] CI no longer passes the var — `grep -c 'ISCC_MONITOR_REALM' .github/workflows/ci.yml` → `0`; the
  `-e ISCC_MONITOR_REALM` line is gone from the smoke `docker run`, the step comment updated without the
  literal token (honest resolution of the count-0-vs-comment tension, not a weakened gate).
- [x] No phantom default — `grep -n 'defaults to the baked' deploy/OPERATING.md` → nothing; the "State,
  volume & backup" sentence now says the image **sets** the var via `ENV`.
- [x] Volume-prep note present — `grep -n '65532'` shows the new `chown 65532:65532` note in the
  quick-start region (l.209/214/217/218/219), not only the `:56-59` requirement.
- [x] Config contract intact — `internal/config/config.go` untouched in the diff; `required(get, keyRealm)`
  still present (`:123`). The image supplies the *value* via `ENV`; the loader still requires the *config
  key* (the two-contract design next.md mandated).
- [x] Anchor link valid — `[State, volume & backup](#state-volume--backup)` matches the live
  `## State, volume & backup` heading's GitHub slug (`&` stripped → double hyphen).
- [~] Local Docker end-to-end boot — Docker absent on the dev host; the pushed CI `docker` job is the
  authoritative gate (now runs the identical realm-less boot). Confirm that job goes green after push.
- Oracle/conformance gate: **N/A** — no signature/Merkle/proof/didweb/fsck code touched.

**Issues found:**
- (resolved + deleted) `critical` "quick-start omits required `ISCC_MONITOR_REALM`" — closed by this
  commit (ENV baked + snippets corrected, CI-proven).
- (filed `normal`, replacing the prior uid-65532 normal) **Compose volume-prep `chown` targets the wrong
  volume.** The new `docker run -v monitor-data:/data … chown 65532:65532` (`OPERATING.md:214`) is correct
  only for the bare `docker run` path; the headline **Compose** fragment declares `monitor-data:` with no
  `name:`/`external:`, so `docker compose up` mounts a project-prefixed `<project>_monitor-data` volume the
  chown never touched — the Compose path still fails permission-denied at `store.Open`. Doc-correctness
  gap in the new note; does not block progress.

**Codex second opinion:** One finding, `[P2]` at `OPERATING.md:214` — "the volume-prep chown targets a
literal `monitor-data` volume, but Compose project-prefixes the volume, so `docker compose up` still
mounts a fresh root-owned volume and the nonroot container fails." **Confirmed real** (grep-verified: zero
`name:`/`external:` on the Compose volume) → filed as the `normal` issue above. No other findings; the
Dockerfile ENV change itself Codex called sound.

**Visual check:** n/a — no SSR surface changed (Dockerfile + CI workflow + deploy doc only).

**Next:** Either fix the Compose volume-prep `normal` (pin `name: monitor-data` or give a Compose-native
prep) OR proceed to the root `README.md` — the last `target.md` "Done When" gate (its own ≤3-file work
package). With the realm-bake `critical` closed, the three iscc-infra `critical`s (persistence / exposure
/ egress) can now clear since the operability doc states a correctly-booting deploy (modulo the Compose
chown caveat, which is `normal`).

**Notes:**
- This commit closed the only NEW `critical`. Remaining open: 3 iscc-infra `critical`s (persistence /
  exposure / egress — these read against M-Deploy/ADR-0013 and want the doc to state a correct deploy,
  now satisfied except the Compose chown caveat) + several `normal`/`low`. The root `README.md` `normal`
  is the last `target.md` "Done When" gate — DONE is not reachable until it exists.
- The CI `docker` job is the authoritative end-to-end gate for the bake (Docker absent locally). Confirm
  it goes green on the develop push; a red smoke loop there would mean the `ENV` is not booting the
  container.
- Updated `learnings/config.md` (the "baked FILE is not a set VAR" bullet is now `settled:` — the image
  sets the value via `ENV`, the loader still requires the key) and `learnings/ci.md` (`docker` smoke
  job proves the bake by omission; Compose volume-prefix pitfall).
