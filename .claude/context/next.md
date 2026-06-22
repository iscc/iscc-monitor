# Next Work Package

## Step: Bake `ISCC_MONITOR_REALM` into the image so the deploy quick-start boots

## Advances
Closes the one NEW **`critical`** issue blocking M-Deploy — *"`deploy/OPERATING.md`
quick-start snippets omit the required `ISCC_MONITOR_REALM` — they do not boot"* (and folds in
the `normal` *"quick-start uses a fresh named volume that uid 65532 cannot write"*). This is the
active step's own deliverable per the `review` handoff (**Next:** "Fix the `deploy/OPERATING.md`
quick-start"). It moves the **M-Deploy** Verify item:

> a tracked **deployment/operability doc** states, for an operator: the SQLite volume path …, the
> reverse-proxy contract …, and the `/metrics` exposure decision

from "doc exists but does not boot" to "an operator can deploy correctly" — which is what lets the
three open iscc-infra `critical`s (persistence / exposure / egress) finally close. A `critical`
preempts everything (issues.md priority semantics), and this one IS the milestone work, not a detour
from it.

## Goal
Make the Dockerfile set `ENV ISCC_MONITOR_REALM=/etc/iscc-monitor/realm.txt` so a fresh container has
a valid realm var out of the box. Today the FILE is baked but the VAR is not — `config.Load` calls
`required(get, keyRealm)` (`config.go:123`), so any invocation without the var exits at startup with
`config: required key "ISCC_MONITOR_REALM" is missing`. Baking the `ENV` makes the Dockerfile's own
existing comment true, makes *every* invocation boot (not just the doc's two snippets), and lets the
operability doc's quick-start be copy-pasted and run. Prove the bake with the CI smoke job, and add
the missing volume-prep note so the snippet's named volume is uid-65532-writable.

## Scope
- **Modify** (non-doc, 2 files):
  - `Dockerfile` — add one `ENV ISCC_MONITOR_REALM=/etc/iscc-monitor/realm.txt` line, placed right
    after the existing `COPY deploy/realm-testnet.txt /etc/iscc-monitor/realm.txt` (`Dockerfile:46`),
    whose comment block (`:42-45`) already claims this. Keep `ISCC_MONITOR_DB` UN-defaulted — it has no
    sensible image default (it must point at the operator's mounted volume).
  - `.github/workflows/ci.yml` — in the `docker` smoke job's `docker run` (`ci.yml:94-98`), DROP the
    now-redundant `-e ISCC_MONITOR_REALM=/etc/iscc-monitor/realm.txt` line. The container must still
    reach `/healthz` → 200 purely from the baked `ENV` — this is the mechanical proof the bake works.
    Update the step comment (`ci.yml:87-90`) to say the realm comes from the image's `ENV`, not a `-e`.
- **Modify** (doc):
  - `deploy/OPERATING.md` — three surgical edits (review note: "this is NOT a rewrite; two precise
    corrections"):
    (a) "State, volume & backup" (`:65-68`) — say the image **sets** `ISCC_MONITOR_REALM` via `ENV`
        (not the false "defaults to the baked path"); a deploy overrides it by passing its own var.
    (b) the two quick-start snippets (`:186-205` Compose, `:209-216` `docker run`) may now legitimately
        OMIT the var — fix the inline comment to match (it is baked into the image, set it only to use a
        mounted realm document).
    (c) add a one-line volume-prep note so the fresh named volume is writable by uid 65532 (the `normal`
        fold-in): a bare `docker run -v monitor-data:/data` on a fresh root-owned volume needs a
        `chown 65532:65532` init (or a bind mount to a host dir already owned by 65532), consistent with
        the uid-65532 requirement the doc already states two sections earlier (`:56-59`).
- **Reference**:
  - `.claude/context/learnings/config.md` — the "baked realm FILE is not a set realm-config VAR" bullet
    (`config.md:29-35`) names BOTH fix paths and explicitly lists the Dockerfile `ENV` as the
    fix-on-touch alt that makes the "out of the box" claim true. READ before editing.
  - `internal/config/config.go:118-126` — `Load` → `required(get, keyRealm)` is why the var is needed;
    `optional(get, key, "")` is how the DISPLAY-only keys differ. Do NOT touch config.
  - `Dockerfile:42-46` — the existing `COPY` + its "valid ISCC_MONITOR_REALM out of the box" comment
    (currently FALSE; this step makes it true).
  - `.github/workflows/ci.yml:91-118` — the smoke `docker run` whose `-e ISCC_MONITOR_REALM` line is
    dropped to prove the bake; it currently passes the var explicitly (confirming the var is required
    even with the file baked).
  - `CLAUDE.md` "Running a local dev instance" env table — the authoritative env source the doc links;
    `ISCC_MONITOR_REALM` STAYS marked required there (it is a required *config* contract; the image now
    supplies a default *value* for it — not in conflict, so the table needs no edit).

## Not In Scope
- Do NOT change `internal/config` — `ISCC_MONITOR_REALM` must stay a `required` config key (config is a
  pure `{fmt time}` leaf that knows nothing about a baked default; the IMAGE supplies the value via
  `ENV`, not the loader). Adding a Go-side default would invert the contract and break the map-backed
  config tests.
- Do NOT default `ISCC_MONITOR_DB` in the Dockerfile — there is no safe image default (it must point at
  the operator's mounted volume); leaving it required is correct.
- Do NOT add a `USER`/init-container chown to the Dockerfile to "fix" the volume ownership — that is an
  operator/Compose concern (the doc's volume-prep note), not an image change; the image already
  correctly runs as the non-root uid 65532.
- Do NOT add the `publish.yml`/`pages.yml` `workflow_dispatch` ref-guard (open `normal`) — different
  workflow, not touched here; it waits for a step that touches those files.
- Do NOT add the on-disk DB migration mechanism (open `normal`) — the doc's interim
  "recreate-volume-on-schema-change" policy already covers it.
- Do NOT write the root `README.md` — that is the NEXT step once this `critical` clears (it is the last
  `target.md` "Done When" gate, but it is its own ≤3-file work package).

## Implementation Notes
- The Dockerfile change is ONE line: `ENV ISCC_MONITOR_REALM=/etc/iscc-monitor/realm.txt`, placed right
  after the `COPY deploy/realm-testnet.txt /etc/iscc-monitor/realm.txt` so the baked file and the var
  pointing at it sit together. The comment block above the `COPY` (`:42-45`) already says "so a fresh
  container has a valid ISCC_MONITOR_REALM out of the box" — that sentence becomes TRUE with this `ENV`;
  today it is false (the image has zero `ENV` lines, grep-confirmed).
- The CI proof is the load-bearing part. The `docker` smoke job (`ci.yml:91-118`) currently passes
  `-e ISCC_MONITOR_REALM=/etc/iscc-monitor/realm.txt` explicitly. DROP that line. If `/healthz` still
  returns 200, the baked `ENV` is doing its job; if the `ENV` were missing/misspelled, `config.Load`
  would exit non-zero, the container would never serve, and the smoke loop would time out and FAIL.
  This turns the fix into a mechanically-gated assertion, not a doc claim. Leave the
  `-e ISCC_MONITOR_DB` and `-e ISCC_MONITOR_ADDR` lines (those have no image default, correctly).
- Doc edits are minimal and surgical — the substance of `deploy/OPERATING.md` is already accurate; only
  the realm-var wording and the volume-prep note change. Keep the link to CLAUDE.md's env table as the
  authoritative source; do not duplicate the table.
- Relevant Correctness rule (learnings.md always-loaded): *"`-ldflags -X` with an empty value OVERRIDES
  the default"* — unrelated to this step but a reminder that the Dockerfile's `VERSION` build-arg
  fail-fast guard (`Dockerfile:28`) must stay untouched.
- Cross-platform: a `Dockerfile` `ENV` line and a workflow YAML edit are OS-neutral; introduce no
  shell-ism.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` pass, `gofmt -l .`
  empty). No Go source changes here, so this re-confirms the gate stays green.
- The Dockerfile sets the var: `grep -n 'ENV ISCC_MONITOR_REALM=/etc/iscc-monitor/realm.txt' Dockerfile`
  prints exactly one line.
- The CI smoke job no longer passes the var: `grep -c 'ISCC_MONITOR_REALM' .github/workflows/ci.yml`
  returns `0` (the `-e` reference is removed; the bake supplies it).
- The doc no longer claims a phantom default: `grep -n 'defaults to the baked' deploy/OPERATING.md`
  returns nothing, and `grep -n '65532' deploy/OPERATING.md` shows a volume-prep note in the quick-start
  region (not only the `:56-59` requirement).
- Local boot proof (the decisive end-to-end check, when Docker is available):
  `docker build --build-arg VERSION=dev -t iscc-monitor:nextcheck .` then
  `docker run -d --name nextcheck -p 41464:9464 -e ISCC_MONITOR_DB=/tmp/monitor.db -e ISCC_MONITOR_ADDR=0.0.0.0:9464 iscc-monitor:nextcheck`
  (NOTE: no `-e ISCC_MONITOR_REALM`) → `curl -fsS http://localhost:41464/healthz` returns 200 within
  ~15s. Reverting the `ENV` line makes the container exit with
  `config: required key "ISCC_MONITOR_REALM" is missing` and this curl FAILS. (If Docker is unavailable
  on the host, the pushed CI `docker` job is the authoritative gate — it runs the same realm-less boot.)

## Done When
`mise run check` is green, the Dockerfile bakes the `ENV ISCC_MONITOR_REALM`, the CI smoke job boots the
container to `/healthz` 200 WITHOUT passing the realm var, and `deploy/OPERATING.md`'s quick-start no
longer claims a phantom default and carries the uid-65532 volume-prep note — so the NEW `critical` is
closed and an operator can copy-paste a booting deploy.
