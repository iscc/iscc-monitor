## 2026-06-22 — Review of: GHCR publish workflow + `.dockerignore` secret/sidecar fold-in + `build:monitor` empty-SHA fail-fast

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** Added `.github/workflows/publish.yml` (push-to-`develop` + `workflow_dispatch` → GHCR
build-push tagging `:develop` + `:sha-<short>` with a non-empty `VERSION` build-arg), hardened
`.dockerignore` to list the gitignore's secret + WAL/SHM-sidecar patterns, and split `mise.toml`'s
`build:monitor` SHA lookup behind a `[ -n "$sha" ]` fail-fast gate. The diff matches `next.md` exactly
(3 files: 1 new workflow + 2 edits, no Go source), all gates are green, and the work is scope-clean and
mirrors the in-repo `pages.yml` precedent. Two Codex-confirmed defense-in-depth gaps remain (non-blocking,
both filed), so PASS_WITH_NOTES rather than PASS.

**Verification:**
- [x] `mise run check` green — all 28 packages `ok`, build + vet clean (no Go source touched; confirms the
  `mise.toml` task-table edit did not break the gate). Cert package passed here (this host is UTC; the
  earlier-noted local-TZ cert failure does not reproduce).
- [x] `gofmt -l .` empty outside `cauldron/`.
- [x] `publish.yml` valid YAML + contract holds (checked via cached `gopkg.in/yaml.v3`): parses;
  `on.push.branches` contains `develop`; `workflow_dispatch` present; `permissions.packages == write`,
  `contents == read`; tag set contains BOTH literal `ghcr.io/iscc/iscc-monitor:develop` AND a
  `…:sha-`-prefixed immutable tag; non-empty `VERSION=${{ github.sha }}` build-arg passed. All PASS.
- [x] `.dockerignore` lists `.env`, `.env.*`, `**/auth.json`, `*.db-wal`, `*.db-shm` — each `grep -qxF`
  PRESENT; now a complete superset of the gitignore never-commit set for root-level files
  (`.claude/settings.local.json` covered by the existing `.claude/` line).
- [x] `build:monitor` fail-fast holds: empty `sha` → `aborted: empty sha` (no build); non-empty → `built`.
  The real task on this host (git present) builds the 26 MB binary, exit 0, stamps the live short SHA;
  the binary runs and reports the required-env error — happy path intact.
- [x] Quality-gate integrity: scanned all unpushed commits (`@{upstream}..HEAD`); the two `t.Skip`/
  `//go:build` grep hits are both in prose (handoff/learnings), not code. No gate circumvention.
- [x] Scope discipline: exactly the 3 declared files; no Go source, `go.mod`/`go.sum`/`schema.sql`,
  Dockerfile, or `ci.yml` touched. `ci.yml`'s `docker` job correctly retains NO registry login (Not-In-
  Scope honored — publish lives in its own file). Oracle/conformance gate N/A (packaging + task-config only).

**Issues found:** (none reviewer-original) — two Codex-confirmed issues filed (below). Resolved this
iteration: the two carried `normal` issues (`build:monitor` empty-SHA; `.dockerignore` root-level
secret/sidecar gap) — both verified fixed and deleted/folded. The `critical` GHCR issue demoted to `low`:
its code half (Dockerfile + publish workflow) is now complete; only iscc-infra repo-settings (make package
public / issue `read:packages` token) remain, which are explicitly out of the loop's scope.

**Codex second opinion:** Two `[P2]` findings, both reviewer-confirmed REAL but non-blocking, both filed:
- **`.dockerignore` secret globs not recursive** (P2) → CONFIRMED. Slashless patterns (`.env`, `*.db-wal`)
  match only the context ROOT under Docker's `filepath.Match`, whereas `.gitignore` matches the basename at
  any depth (reviewer-verified: `git check-ignore` IGNORES `deploy/.env`/`data/monitor.db-wal`, Docker would
  not exclude them). So the "superset of gitignore" intent holds only for root-level files. Same defense-in-
  depth class as the gap this slice closed, strictly narrower; never reaches the shipped image (final stage
  only `COPY --from=build`s the binary) or CI. Filed `low` — fix: use `**/.env`, `**/*.db-wal`, etc.
- **`workflow_dispatch` can publish `:develop` from a non-develop ref** (P2) → CONFIRMED. The publish job
  has no ref guard, so a maintainer dispatching from a feature branch would push that code as the floating
  `:develop` tag (immutable `:sha-<short>` unaffected). Mirrors the in-repo `pages.yml` precedent
  (intentionally — `next.md` told advance to mirror it), so a pre-existing convention, not a regression;
  `workflow_dispatch` is maintainer-only. Filed `normal` — fix: gate on `github.ref ==
  'refs/heads/develop'` (apply to `pages.yml` too).

**Visual check:** n/a — no SSR surface changed (CI/packaging config only: `publish.yml`, `.dockerignore`,
`mise.toml`).

**Next:** The remaining code-closable M-Deploy slices are still open (none code-blocked): the canonical
`deploy/realm-testnet.txt` (smallest), the operability/deployment doc, and the root `README.md`
(`target.md` "Done When" requires it before DONE). Suggest `deploy/realm-testnet.txt` next, then the
deployment doc (which can fold in the persistence/egress/exposure `critical` infra asks), then `README.md`.
The new `normal` `workflow_dispatch` ref-guard issue is a small fold-in candidate whenever a workflow file
is next touched.

**Notes:**
- **Docker is absent on this host (CI-only).** The `docker push` itself cannot run locally — verified
  `publish.yml` exactly as the Verify bar asks (YAML validity + static inspection of trigger/tags/
  permissions/build-arg). The `.dockerignore` and `build:monitor` fixes ARE fully tested on this host.
- `github.sha` is always populated in Actions and cannot empty-expand, so the workflow's `VERSION` build-arg
  always satisfies the Dockerfile's fail-fast guard — the CI image path sidesteps the `-X` empty-stamp class
  entirely (the `build:monitor` host fix covers the host path; both stamped consumers now guarded).
- Action pins match siblings (`checkout@v4`; `docker/login-action@v3` + `docker/build-push-action@v6`);
  concurrency shape mirrors `pages.yml`. The deprecated Node-20 action bump stays a separate `low` issue.
- Pushed to `origin/develop` on this PASS_WITH_NOTES (CI is the real Docker oracle — the publish workflow
  itself runs only after this push lands on develop).
