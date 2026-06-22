## 2026-06-22 — Review of: Ref-guard the `workflow_dispatch` publish/pages jobs (+ Node-20 action bumps)

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The primary goal — job-level `if: github.ref == 'refs/heads/develop'` guards on `publish.yml`'s
`publish` job and `pages.yml`'s `build` job (the genuinely code-closable `normal`) — is correctly and
minimally implemented, with accurate evergreen comments and correct job-level placement; the `actions/*`
bumps are pinned to the live latest majors (reviewer-confirmed via `gh api`, not a stale guess). All gates
are green and the change touches zero Go source. One real residual that the advance/`next.md` got factually
wrong: `docker/login-action@v3` + `docker/build-push-action@v6` ARE node20 actions (not "container actions"),
so `publish.yml` is not fully off the deprecated runtime — filed as a `low` follow-up.

**Verification:**
- [x] `mise run check` green — 20 pkgs `ok` (all cached; zero `.go` files changed across the unpushed range, so the cache is honest).
- [x] `gofmt -l .` empty.
- [x] All three workflows parse as valid YAML — validated via a throwaway `go run` over the cached `gopkg.in/yaml.v3` (`yaml.Unmarshal` into `map[string]any`); all printed `OK`.
- [x] `grep "if: github.ref == 'refs/heads/develop'" publish.yml` → line 34, job-level (sibling to `name:`/`runs-on:`, above `steps:`).
- [x] `grep "if: github.ref == 'refs/heads/develop'" pages.yml` → line 35, on the `build` job; `deploy` inherits via `needs: build` (line 68).
- [x] `ci.yml` carries NO `github.ref` guard — `grep "github.ref" ci.yml` is empty. (The only `if:` is the pre-existing `if: always()` docker-cleanup step at line 125; the literal `! grep -q "if:"` Done-When gate is a false positive there, but its intent — no ref guard — holds. Advance flagged this correctly.)
- [x] No Node-20 `actions/*` pins remain — `grep "actions/checkout@v4|actions/setup-go@v5" .github/workflows/` empty.
- [x] Live action majors confirmed via `gh api .../releases/latest`: checkout@v7, setup-go@v6, configure-pages@v6, upload-pages-artifact@v5, deploy-pages@v5 — all exist and are the current majors (advance pinned each correctly; inputs `go-version`/`path`/`page_url` still valid).
- [x] `publish.yml` immutable `:sha-${{ steps.vars.outputs.short }}` tag (line 69) + `VERSION=${{ github.sha }}` build-arg (line 66) unchanged.
- [x] Quality-gate-integrity scan over all unpushed commits (`@{upstream}..HEAD`) — only 3 workflow + 3 context `.md` files; the `nolint`/`//go:build` grep hits are all in handoff/issues PROSE, not code. No `t.Skip`, no `continue-on-error`, no deleted assertions. Clean.

**Issues found:**
- (filed `low`) `publish.yml`'s `docker/login-action@v3` + `docker/build-push-action@v6` still target Node 20 — see Codex triage below. The advance handoff's claim that these are "container actions, not in the Node-20 list" is factually wrong (reviewer-confirmed `runs.using: 'node20'`).

**Codex second opinion:** Ran (`codex review --commit HEAD`); produced one finding after ~6 min.
- **[P2] "Bump the remaining Node-20 Docker actions" → CONFIRMED REAL, filed as `low`.** Codex claims
  `docker/login-action@v3` and `docker/build-push-action@v6` declare `runs.using: node20`, contradicting
  the handoff's "container actions, not in the Node-20 list." I verified independently:
  `gh api repos/docker/login-action/contents/action.yml?ref=v3` → `using: 'node20'`, and
  `...build-push-action/...?ref=v6` → `using: 'node20'`. Codex is right; the advance/`next.md` premise was
  wrong. Their current majors `@v4`/`@v7` use node24 with identical inputs (reviewer-confirmed). This is a
  `low` (deprecation warning today, future hard-fail; the publish job runs green now because GitHub
  force-runs node20 on node24), NOT a current breakage and NOT a progress blocker — so PASS_WITH_NOTES, not
  NEEDS_WORK. Filed as a `low` follow-up for the next time `publish.yml` is touched; corrected the false
  "container actions" claim in `learnings/ci.md`.

**Visual check:** n/a — no SSR surface changed (workflow/config-only change; no `internal/dashboard`, `dossier`, `web`, `certificate`, or template touched).

**Next:** This closes the ref-guard `normal` (deleted from `issues.md`) and the Pages-annotation Node-20
`low` (the `actions/*` half). The DONE-blocker scan is now **0 critical, 3 normal** — and all three remaining
`normal`s are design/human-blocked, not loop-closeable: (1) the DB-migration mechanism design decision
(ADR-0007, wants a grilling pass), (2) the WASM signature-half browser did:web design pass, (3) the
per-hub-Anchor design-honesty question. The `/` "recent declarers" footer normal needs store lookup history
the store does not track (also design-blocked). Per the standing "loop stalls on human-blocked DONE" memory,
`define-next` should surface these as a **STOP / human-design edge** rather than spin on cosmetic chrome —
there is NO code-closable `normal` left to advance. (The new docker-action `low` is skipped by the loop.)

**Notes:**
- **The factual lesson:** "is action X a Node-20 action?" must be answered by reading the action's
  `action.yml` `runs.using`, NOT by assuming (login/build-push @v3/@v6 are node20 JS actions despite being
  "Docker" actions). `next.md` asserted otherwise without checking and advance carried the assumption
  forward. Both Codex and the reviewer caught it against ground truth.
- **The action-major bumps are correct and well-justified:** advance used `gh api .../releases/latest` (the
  authoritative method `next.md` mandates) over the stale fallback list, so it pinned newer majors than
  `next.md` guessed (checkout@v7 vs guessed v5, configure-pages@v6 vs v5, etc.). Reviewer re-ran `gh api` —
  every pinned major matches the live latest, and the bumped inputs are unchanged. This is a correct,
  authoritative deviation from `next.md`'s guesses, not a defect.
- Workflow-RUNTIME behavior (the guard actually skipping a non-develop dispatch; a green Node-24 Pages run
  with no deprecation annotation) is only confirmable by a live GitHub Actions run — not reproducible here.
  Static verification (YAML validity + job-level placement + correct `github.ref` expr + live action majors)
  is complete; the live-run confirmation is the post-merge CI check.
- No oracle/crypto path touched — trust-root conformance (`notecheck`, golden vectors) is N/A. The
  `notecheck` *step* in `ci.yml` is unchanged; only its `checkout`/`setup-go` pins moved.
- `learnings/ci.md` updated: corrected the publish/pages "no ref guard" notes (now guarded), the stale
  action-pin versions (now the live majors), and added the docker-actions-ARE-node20 fact.
