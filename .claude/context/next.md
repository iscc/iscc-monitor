# Next Work Package

## Step: Ref-guard the `workflow_dispatch` publish/pages jobs (and bump the deprecated Node-20 actions)

## Advances
This step does **not** advance an unmet milestone Verify criterion — there are none reachable. Per
`state.md` ("M-Deploy is fully met (criticals pruned). DONE is now blocked only by the 5 open
`normal`s") and the latest `review` **Next** note, every v1 milestone Verify bar (M1–M3 / M-UI behavioral
/ M-Deploy in-repo) is already met or design/human-blocked. The DONE bar ("no open `critical` or `normal`
issue") therefore turns on the 5 open `normal`s, and this step **closes the one genuinely code-closable
`normal`**:

> **`publish.yml` `workflow_dispatch` can push the floating `:develop` tag from a non-develop ref**
> (`issues.md`) — *"guard the publish job (or just the `:develop` tag step) on
> `if: github.ref == 'refs/heads/develop'` … Apply the same guard to `pages.yml` for consistency."*

It also folds in the related touched-file `low`:

> **`pages.yml` actions target deprecated Node 20 — bump to current major versions** (`issues.md`) —
> *"Fix when `pages.yml` is next touched … Also re-check `ci.yml` for the same pinned actions while there."*

This preempts surfacing a STOP/IDLE decision because `state.md` §"Next Milestone" item 1 and the `review`
**Next** both name the ref-guard as the explicit *first* action to take before routing the remaining
design `normal`s to a human pass. The other four `normal`s (DB-migration design decision, WASM
signature-half browser did:web design pass, per-hub-Anchor design-honesty question, `/` "recent declarers"
footer needing store lookup history the store does not track) are **not loop-closeable** — they are the
human/design-blocked edge the "loop stalls on human-blocked DONE" memory warns about.

## Goal
Make the publish (`publish.yml`) and Pages (`pages.yml`) jobs that push floating, infra-pulled tags refuse
to run from any ref other than `refs/heads/develop`, so a maintainer `workflow_dispatch` from a feature
branch can never republish that branch's code as `:develop` / to `monitor.iscc.codes`. While both workflow
files are open, bump their (and `ci.yml`'s) pinned `actions/*` majors off the deprecated Node-20 runtime.

## Scope
- **Create**: (none)
- **Modify**:
  - `.github/workflows/publish.yml` — add a job-level ref guard to the `publish` job; bump pinned `actions/*`.
  - `.github/workflows/pages.yml` — add the same ref guard to the `build` job; bump pinned `actions/*`
    (incl. the `deploy` job's `deploy-pages`).
  - `.github/workflows/ci.yml` — bump the same pinned `actions/*` (`checkout`, `setup-go`) only; **no ref
    guard** (it triggers on `push`/`pull_request`, not `workflow_dispatch` — guarding it would break PR CI).
- **Reference**:
  - `/workspace/iscc-monitor/.claude/context/learnings/ci.md` — the CI/Pages/Publish workflow mechanics,
    the existing ref-guard notes (publish §"`workflow_dispatch` has no ref guard"; pages §CNAME), and the
    YAML-validation method (validate via the cached `gopkg.in/yaml.v3`, **not** PyYAML — PyYAML is absent
    locally).
  - `/workspace/iscc-monitor/.claude/context/issues.md` — the two entries quoted under **Advances** (their
    exact "Verify fixed" clauses).

## Not In Scope
- Do **not** touch any Go source, `Dockerfile`, `.dockerignore`, `mise.toml`, or `deploy/*` — this is a
  workflow-only change (zero non-doc/config files).
- Do **not** alter the `:sha-<short>` immutable tag, the `VERSION` build-arg, the GHCR login, the
  verifier-site render command, or the CNAME step — the guard gates *whether the job runs*, not what it does.
- Do **not** attempt to fix the other four `normal`s (DB-migration, WASM signature half, per-hub-Anchor
  honesty, `/` declarers footer) — they are design/human-blocked; this step does not chase them.
- Do **not** add the ref guard to `ci.yml` — its `push`/`pull_request` triggers must stay unguarded or PR
  CI stops running.
- Do **not** bump an action to a major version whose inputs differ from this workflow's usage without
  confirming the inputs still match (e.g. `setup-go`'s `go-version`, `upload-pages-artifact`'s `path`).

## Implementation Notes
- **The ref guard** is a single job-level key. On `publish.yml`'s `publish` job and `pages.yml`'s `build`
  job, add (sibling to `name:` / `runs-on:`):
  ```yaml
  if: github.ref == 'refs/heads/develop'
  ```
  Place it at the **job** level (not per-step) so the whole job is skipped from a non-develop ref. For
  `pages.yml`, guarding only `build` is sufficient — `deploy` has `needs: build`, so a skipped `build`
  skips `deploy` too (a skipped dependency does not run dependents). A `push: [develop]` event already has
  `github.ref == 'refs/heads/develop'`, so the guard is a no-op on the normal push path and only bites a
  `workflow_dispatch` from another ref — exactly the issue's intent. Add a short comment on the `if:`
  explaining it pins the floating `:develop` tag / Pages deploy to the develop ref so a manual dispatch
  from a feature branch cannot republish.
- **Node-20 action bumps** (the `low`, per the deprecation annotation in `issues.md`). Bump the pinned
  majors to their current Node-24 releases, the **same version across all three files** for consistency:
  - `actions/checkout@v4` → `@v5` (`publish.yml`, `pages.yml`, **and both** `ci.yml` uses — lines 23 & 77).
  - `actions/setup-go@v5` → `@v6` (`pages.yml` line 38, `ci.yml` line 26). Confirm `with: go-version: "1.26"`
    is still a valid `setup-go@v6` input (it is — `go-version` is unchanged across these majors).
  - `actions/configure-pages@v5`, `actions/upload-pages-artifact@v3`, `actions/deploy-pages@v4`
    (`pages.yml`): bump each to its current major; keep `upload-pages-artifact`'s `path: dist` input and
    `deploy-pages`'s `id: deployment` / `${{ steps.deployment.outputs.page_url }}` output reference intact.
  - `docker/login-action@v3` and `docker/build-push-action@v6` (`publish.yml`) are **not** in the Node-20
    deprecation list (they are container actions, not Node) — leave them as-is.
- **Determine the current majors** without guessing: `gh api` against each action's latest release gives
  the latest tag, e.g. `gh api repos/actions/setup-go/releases/latest --jq .tag_name`. Pin to the **major**
  of that latest tag (`@v6`, not `@v6.1.2`) — matches the repo's existing `@vN` convention. If `gh` cannot
  reach the network in this environment, the well-known current majors as of this writing are:
  `checkout@v5`, `setup-go@v6`, `configure-pages@v5` (still current), `upload-pages-artifact@v4`,
  `deploy-pages@v4` (still current); verify each input still matches before committing.
- **Keep the workflow comments accurate** (CLAUDE.md "evergreen comments describe the current state"):
  `learnings/ci.md` §publish and §pages document "`workflow_dispatch` has no ref guard" — that statement
  becomes false after this step. `learnings.md` files are owned by `review`, so do **not** edit them
  yourself; the in-file workflow comment that explains the `if:` is the doc that must be written here.
- **No oracle/crypto path is touched** — the trust-root conformance gate (`notecheck`, golden vectors) is
  N/A for this step. The `ci.yml` `notecheck` *step* is unchanged; only its `checkout`/`setup-go` action
  pins move.
- **YAML validation** (per `learnings/ci.md`): PyYAML is **absent** locally — validate each edited file by
  parsing it with the cached `gopkg.in/yaml.v3` (a tiny throwaway `go run` that `yaml.Unmarshal`s the file
  bytes into `map[string]any` and fails on error), not `python3 -c "import yaml"`.

## Verification
- `mise run check` is green (no Go source changed — confirms the edits broke no build/test/format).
- `gofmt -l .` is empty.
- Each edited workflow parses as valid YAML — e.g. a throwaway `go run` using the cached
  `gopkg.in/yaml.v3` `yaml.Unmarshal` over `.github/workflows/{publish,pages,ci}.yml` exits 0 for all three
  (PyYAML is absent — do not use it).
- `grep -n "if: github.ref == 'refs/heads/develop'" .github/workflows/publish.yml` matches the `publish`
  job (job-level, above `runs-on`/`steps`).
- `grep -n "if: github.ref == 'refs/heads/develop'" .github/workflows/pages.yml` matches the `build` job.
- `! grep -q "if:" .github/workflows/ci.yml` — `ci.yml` has **no** ref guard (its `push`/`pull_request`
  triggers stay unguarded).
- `! grep -RqE "actions/checkout@v4|actions/setup-go@v5" .github/workflows/` — no Node-20-pinned
  `checkout@v4` / `setup-go@v5` remains in any workflow.
- The `:sha-<short>` tag line and `VERSION=${{ github.sha }}` build-arg in `publish.yml` are unchanged:
  `grep -q 'sha-${{ steps.vars.outputs.short }}' .github/workflows/publish.yml` and
  `grep -q 'VERSION=${{ github.sha }}' .github/workflows/publish.yml` both match.

## Done When
`mise run check` + `gofmt -l .` are clean, both `publish.yml`'s `publish` job and `pages.yml`'s `build` job
carry the job-level `if: github.ref == 'refs/heads/develop'` guard, `ci.yml` carries no such guard, no
workflow still pins `actions/checkout@v4` or `actions/setup-go@v5`, and every edited workflow parses as
valid YAML — closing the `publish.yml`/`pages.yml` ref-guard `normal` and the Node-20 `low`.
