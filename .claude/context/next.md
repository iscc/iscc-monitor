# Next Work Package

## Step: Pin the Compose volume name so the OPERATING.md volume-prep `chown` matches what Compose mounts

## Advances
This step settles the open `normal` issue **"`deploy/OPERATING.md` Compose volume-prep `chown` targets the
wrong volume (project-prefix mismatch)"** (`issues.md`), which is the last doc-correctness caveat on the
`critical` iscc-infra issue **"Persistence contract for the SQLite DB volume + acknowledge the in-place
migration hazard"**.

That `critical`'s own "Verify fixed" bar is:

> docs state the DB path + volume + "back up this one file" contract and the non-root uid, and link the
> migration issue as the known constraint with "recreate volume on schema change" as the interim policy.

and the accompanying `normal`'s bar is:

> following the Compose quick-start verbatim (prep + `docker compose up`) boots to `/healthz` 200 without
> a permission-denied at `store.Open`; the chowned volume is the SAME one Compose mounts.

`deploy/OPERATING.md` already answers the persistence `critical` in substance (State/volume/backup,
migration policy, non-root uid) — but the headline Compose quick-start is **not runnable as written**: the
`chown` prep on line 214 names the volume `monitor-data` literally, while the Compose `volumes:` block
(lines 204-205) declares `monitor-data:` with no `name:`, so `docker compose up` mounts a project-prefixed
volume (`<project>_monitor-data`) the `chown` never touched → the non-root container fails permission-denied
at `store.Open`. Fixing this makes the persistence-contract docs correct end-to-end, which is the precondition
for `review` to confirm/close that `critical` (per `state.md`: the three iscc-infra `critical`s are now the
SOLE DONE blockers, and this is the one residual doc gap on the persistence ask).

This is a doc-only step (zero Go source). It is justified over milestone work because every code/doc-closable
`target.md` Verify criterion is already met and DONE turns entirely on confirming/pruning these three
`critical`s — `state.md` and the latest `review` handoff both name the persistence-contract doc as the
immediate next work.

## Goal
Make the headline Compose quick-start in `deploy/OPERATING.md` runnable verbatim by pinning the Compose
volume to the literal name `monitor-data`, so the documented `chown` prep operates on the SAME volume
`docker compose up` mounts — closing the persistence `critical`'s last doc caveat.

## Scope
- **Create**: none.
- **Modify**: `deploy/OPERATING.md` (doc only; the ≤3 non-test/doc Go-source budget is untouched).
- **Reference**:
  - `/workspace/iscc-monitor/deploy/OPERATING.md` — the doc to fix; the Compose fragment (lines ~187-206),
    the volume-ownership note (~208-220), and the bare `docker run` snippet (~226-231).
  - `issues.md` entry **"`deploy/OPERATING.md` Compose volume-prep `chown` targets the wrong volume"** —
    the precise diagnosis + the two suggested fixes (pin `name: monitor-data`, OR a Compose-native prep).
  - `issues.md` `critical` **"Persistence contract for the SQLite DB volume …"** — the ask this unblocks.
  - `.claude/context/learnings/config.md` — confirms `ISCC_MONITOR_REALM` is baked via `ENV` (the
    snippet correctly omits it) and `ISCC_MONITOR_DB` has no default — keep these claims intact when editing.

## Not In Scope
- Do **not** touch any Go source, `Dockerfile`, `mise.toml`, `CLAUDE.md`, or `README.md` — `OPERATING.md`
  only. (No code defect exists here; this is a doc-correctness fix.)
- Do **not** introduce a `docker-compose.yml` / `compose.yaml` file — the doc carries an illustrative
  fragment, not a tracked stack (the real Compose stack lives in iscc-infra, out of the loop's scope per
  `target.md` M-Deploy). Fix the fragment in place.
- Do **not** rewrite unaffected sections (State/volume/backup, migration policy, egress, footprint,
  `/metrics` decision, graceful shutdown) — they already satisfy their asks; touch only the volume-name +
  prep wording.
- Do **not** also fix the other open `normal`s in passing (the `publish.yml`/`pages.yml` `workflow_dispatch`
  ref-guard, the Node-20 action bumps, the proofserve masthead slice, the shared `Resolve` leaf) — each is
  its own later step.

## Implementation Notes
The fix the issue prefers (smallest, makes the literal-name `chown` correct for BOTH the Compose path and
the bare `docker run` path) is to **pin the Compose volume name**:

```yaml
volumes:
  monitor-data:
    name: monitor-data
```

With an explicit `name:`, Compose uses the literal engine-volume name `monitor-data` instead of the
project-prefixed default, so the documented prep `docker run --rm -v monitor-data:/data alpine chown -R
65532:65532 /data` chowns the exact volume `docker compose up` then mounts. The bare `docker run` snippet
already uses `-v monitor-data:/data` literally, so it stays correct unchanged.

Edge to handle in the prose so the doc stays internally honest:
- The volume-ownership note (~208-220) currently says "A fresh named Docker volume is created `root:root`
  … a bare `-v monitor-data:/data` makes `store.Open` fail permission-denied". That stays true; just make
  sure the note's "do this first" prep and the Compose fragment now refer to the SAME literal volume name
  (`monitor-data`), with no project-prefix ambiguity. A one-clause mention that the explicit `name:` is what
  makes the literal-name `chown` line up with the Compose mount keeps the WHY evident (CLAUDE.md: evergreen
  comments describe the current state).
- Keep the existing alternative (bind-mount a host dir you `chown 65532:65532`) — it is correct and is the
  more robust real-world path; this fix only corrects the named-volume path.
- Do not weaken any other claim: `ISCC_MONITOR_REALM` is baked via `ENV` (so the snippets correctly omit
  it), `ISCC_MONITOR_DB` has no default, the container publishes no host port. All must remain accurate.

Relevant correctness rule (`learnings.md` Go/tooling + CLAUDE.md): "the quick-start must be runnable as
written" / "smallest reasonable changes" — pin the name, do not restructure the doc.

## Verification
- `grep -nA2 '^volumes:' /workspace/iscc-monitor/deploy/OPERATING.md` shows `monitor-data:` with a child
  `name: monitor-data` (the Compose volume is pinned to the literal name).
- The literal volume name in the `chown` prep line and in the Compose `volumes:` block is the SAME string
  `monitor-data` — assert with:
  `grep -c 'monitor-data' /workspace/iscc-monitor/deploy/OPERATING.md` returns the count, and a manual read
  confirms the `chown` target (`-v monitor-data:/data`) equals the pinned Compose `name:`.
- `mise run check` is green (a doc-only change must not regress the gate; no `.go` file changed).
- `gofmt -l /workspace/iscc-monitor` is empty (no Go file touched).
- No new tracked Compose file was added: `test ! -f /workspace/iscc-monitor/docker-compose.yml && test ! -f
  /workspace/iscc-monitor/compose.yaml` exits 0 (the fix stays in the doc fragment).
- The doc's other persistence claims are intact: `grep -q 'recreate the volume on a schema change'`,
  `grep -q '65532'`, and `grep -q '/etc/iscc-monitor/realm.txt'` all still match in `OPERATING.md`.

## Done When
`deploy/OPERATING.md`'s Compose `volumes:` block pins `name: monitor-data` so the documented `chown` prep
chowns the same volume `docker compose up` mounts, all Verification checks pass, and `mise run check` stays
green — closing the persistence `critical`'s last doc caveat and clearing `review` to confirm/prune it.
