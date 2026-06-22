## 2026-06-22 — Pin the Compose volume name so the OPERATING.md volume-prep `chown` matches what Compose mounts

**Done:** Pinned the illustrative Compose volume in `deploy/OPERATING.md` to the literal engine-volume
name `monitor-data` (`name: monitor-data` under the `volumes:` block) so the documented `chown` prep
(`docker run --rm -v monitor-data:/data alpine chown -R 65532:65532 /data`) chowns the SAME volume
`docker compose up` mounts — closing the persistence `critical`'s last doc caveat. Added a one-clause
prose note explaining the explicit `name:` is what makes the literal-name prep line up (without it Compose
mounts a project-prefixed `<project>_monitor-data` the `chown` never touched).

**Files changed:**
- `deploy/OPERATING.md`: (1) Compose `volumes:` block now declares `monitor-data:` with a child
  `name: monitor-data` + inline why-comment; (2) the "Volume ownership (do this first)" prose note now
  states that the explicit `name:` is what aligns the literal-name `chown` with the Compose mount.

**Verification:** `mise run check` → green (28 packages `ok`, all cached — no `.go` file touched).
Per-criterion:
- [x] `grep -nA2 '^volumes:'` shows `monitor-data:` with child `name: monitor-data` (lines 204-206) — PASS.
- [x] Literal volume name matches: `chown` prep target (`-v monitor-data:/data`) == pinned Compose `name:`
  (both the string `monitor-data`); `grep -c monitor-data` = 10 — PASS.
- [x] `mise run check` green — PASS.
- [x] `gofmt -l /workspace/iscc-monitor` empty (exit 0) — PASS.
- [x] No tracked Compose file added: `test ! -f docker-compose.yml && test ! -f compose.yaml` exits 0 — PASS.
- [x] Other persistence claims intact: `recreate the volume on a schema change`, `65532`,
  `/etc/iscc-monitor/realm.txt` all still match — PASS.

**Next:** This was the last residual doc caveat on the persistence-contract `critical`. `review` should
confirm `deploy/OPERATING.md` now answers that `critical` end-to-end (State/volume/backup contract +
non-root uid + migration policy + a Compose quick-start that is now runnable verbatim) and confirm/prune
it. After that, DONE turns on the two remaining iscc-infra `critical`s (public-route/`/metrics`-exposure
decision — already substantially in OPERATING.md §"Route exposure & the `/metrics` decision"; egress +
resource-footprint sizing — already in §Egress + §Footprint), so `define-next` should target whichever of
those still has a doc gap; the open `normal`s (publish.yml/pages.yml `workflow_dispatch` ref-guard, Node-20
action bumps, proofserve masthead slice, shared `Resolve` leaf) are each their own later step.

**Notes:**
- Scope-clean: one doc edit, zero Go source, no `Dockerfile`/`mise.toml`/`CLAUDE.md`/`README.md` touch, no
  new `docker-compose.yml`/`compose.yaml`, unaffected sections (State/backup, migration, egress, footprint,
  `/metrics`, graceful shutdown) left untouched — all per Not-In-Scope.
- The bare `docker run` snippet already used `-v monitor-data:/data` literally, so it stays correct
  unchanged — the `name:` pin makes BOTH the Compose path and the bare-run path consistent against the one
  prep `chown`.
- Config claims (`ISCC_MONITOR_REALM` baked via `ENV` so the snippets correctly omit it; `ISCC_MONITOR_DB`
  un-defaulted; container publishes no host port) were not weakened — only the volume-name + prep wording
  changed.
- No `learnings/` detail file applies — this is a one-time doc-correctness fix with no recurring
  forward-looking pitfall (the closest detail file, `config.md`, was read to confirm the `ENV`/no-default
  claims stay accurate, and they do).
