## 2026-06-22 — Ref-guard the `workflow_dispatch` publish/pages jobs (and bump the deprecated Node-20 actions)

**Done:** Added a job-level `if: github.ref == 'refs/heads/develop'` guard to `publish.yml`'s `publish`
job and `pages.yml`'s `build` job, so a maintainer `workflow_dispatch` from a non-develop ref can no
longer republish that branch's code as the floating `:develop` GHCR tag or push it to
`monitor.iscc.codes` (`pages.yml`'s `deploy` job inherits the guard via `needs: build`). Bumped every
Node-20-pinned `actions/*` in all three workflows off the deprecated runtime to their current Node-24
majors. No Go source touched — workflow/config only.

**Files changed:**
- `.github/workflows/publish.yml`: job-level ref guard on `publish` (with an explanatory comment);
  `actions/checkout@v4` → `@v7`. Immutable `:sha-<short>` tag, `:develop` tag, `VERSION=${{ github.sha }}`
  build-arg, GHCR login, and the Docker actions (`login-action@v3`, `build-push-action@v6` — container
  actions, not in the Node-20 list) all left intact.
- `.github/workflows/pages.yml`: job-level ref guard on `build` (with comment); `checkout@v4`→`@v7`,
  `setup-go@v5`→`@v6`, `configure-pages@v5`→`@v6`, `upload-pages-artifact@v3`→`@v5`,
  `deploy-pages@v4`→`@v5`. The `path: dist`, `go-version: "1.26"`, `id: deployment` + `page_url` output,
  and `cp CNAME` step are unchanged.
- `.github/workflows/ci.yml`: `checkout@v4`→`@v7` (both uses, lines 23 & 77), `setup-go@v5`→`@v6`. No ref
  guard added (its `push`/`pull_request` triggers must stay unguarded or PR CI stops).

**Verification:** `mise run check` → green (28 pkgs `ok`, all cached — no `.go` file changed, confirming
the edits broke no build/vet/test). `gofmt -l .` → empty. Per-criterion:
- YAML validity (all three files) — PASS. Validated via a throwaway `go run` using the cached
  `gopkg.in/yaml.v3` `yaml.Unmarshal` into `map[string]any` (PyYAML absent locally); all three printed
  `OK` and exit 0.
- `grep "if: github.ref == 'refs/heads/develop'" publish.yml` → line 34 (job-level, above `runs-on`) — PASS.
- `grep "if: github.ref == 'refs/heads/develop'" pages.yml` → line 35 (`build` job) — PASS.
- No Node-20 pins remain: `! grep -RqE "actions/checkout@v4|actions/setup-go@v5" .github/workflows/` — PASS.
- `publish.yml` immutable tag + build-arg unchanged: `grep -F 'sha-${{ steps.vars.outputs.short }}'` →
  line 69, `grep -F 'VERSION=${{ github.sha }}'` → line 66 — both PRESENT (use `grep -F`; under zsh the
  `${{ }}` braces brace-expand even inside single quotes, so the un-`-F` form in next.md prints 0 — a
  shell-quoting artifact, not a missing line).
- `ci.yml` ref-guard gate — see Notes: the literal `! grep -q "if:" ci.yml` gate FAILS on a pre-existing
  `if: always()` cleanup-step line, but the gate's *intent* (no `github.ref` guard in ci.yml) PASSES.

**Next:** This closes the one genuinely code-closeable `normal` (the publish/pages ref-guard) plus the
`pages.yml` Node-20 `low`. The DONE-blocker scan is now **0 critical, 4 normal** — and all four remaining
`normal`s are design/human-blocked, not loop-closeable: (1) the DB-migration mechanism design decision
(ADR-0007, wants a grilling pass), (2) the WASM signature-half browser did:web design pass, (3) the
per-hub-Anchor design-honesty question, (4) the `/` "recent declarers" footer needing store lookup
history the store does not track. Per the standing "loop stalls on human-blocked DONE" memory,
`define-next` should surface these as a **STOP / human-design edge** rather than spin on cosmetic chrome —
there is no code-closeable `normal` left to advance.

**Notes:**
- **Current action majors came from `gh api` (the authoritative method next.md specifies), not the
  fallback list.** `gh` reached the network, so I pinned to the live latest majors, which are NEWER than
  both next.md's "well-known as of this writing" fallback and the issue text: `checkout@v7` (next.md
  guessed v5), `setup-go@v6` (matches), `configure-pages@v6` (next.md said v5 still current — it is now
  v6), `upload-pages-artifact@v5` (next.md guessed v4), `deploy-pages@v5` (next.md said v4 still current —
  now v5). next.md is explicit: pin to the major of `gh api …/releases/latest`, fall back to the
  well-known list only "if gh cannot reach the network." It could, so the live majors win. I confirmed
  each bumped major's contract before pinning: `setup-go@v6` still takes `go-version`;
  `upload-pages-artifact@v5` still has `path` (required); `deploy-pages@v5` still emits `page_url`;
  `configure-pages@v6` exists. All input/output references in the workflows remain valid.
- **The `! grep -q "if:" ci.yml` Done-When gate is over-broad and trips a FALSE positive.** `ci.yml` line
  125 already carries `if: always()` on the docker-cleanup step — a pre-existing line, untouched by this
  step, unrelated to the ref guard. The gate's real intent is "ci.yml carries no `github.ref` ref guard,"
  which is satisfied: `! grep -q "github.ref" .github/workflows/ci.yml` is clean. I did NOT alter
  `ci.yml`'s `if: always()` (out of scope, and removing a cleanup guard would be wrong). Flagging so
  `review` reads the gate by intent, not literally.
- **`learnings/ci.md` is now stale in two spots** (review owns it, I cannot edit it): §publish "workflow_dispatch
  has no ref guard" and §pages's implied no-guard state are both false after this step; the action-pin
  versions quoted there (`checkout@v4`, `setup-go@v5`, `upload-pages-artifact@v3`, `deploy-pages@v4`,
  `configure-pages@v5`) are also superseded. The in-file workflow comments explaining the `if:` guards are
  the doc written here (per next.md); the learnings update is review's.
- No oracle/crypto path touched — trust-root conformance (`notecheck`, golden vectors) is N/A. The
  `notecheck` *step* in `ci.yml` is unchanged; only its `checkout`/`setup-go` pins moved.
- Workflow-runtime behavior (the guard actually skipping a non-develop dispatch, a green Node-24 Pages run
  with no deprecation annotation) can only be confirmed by an actual GitHub Actions run — not reproducible
  in this environment. Static verification (YAML validity + job-level placement + correct `github.ref`
  expression) is complete; the live-run confirmation is the post-merge CI check.
