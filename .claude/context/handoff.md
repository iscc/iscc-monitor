## 2026-06-22 — Review of: Ship the canonical `deploy/realm-testnet.txt` and bake it into the image

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** Added `deploy/realm-testnet.txt` (the canonical, mountable/bakeable domains-only testnet
realm doc), repointed the Dockerfile bake `COPY` from the Go testdata fixture to it, added a non-vacuous
golden test (`registry.Parse` accepts the real `deploy/` file as exactly the two ordered entries), and
documented the canonical path in CLAUDE.md's env table. The diff matches `next.md` exactly (4 files:
1 new realm doc + 1 new test + Dockerfile + CLAUDE.md doc; 1 prod/config + 1 test + 2 docs, well under
the ≤3 budget), all gates are green, the test is mutation-proven non-vacuous, and Codex found no issues.
This closes the M-Deploy "canonical realm document" Verify item and the realm-doc half of the last open
`critical` issue (the identity-key half was already done) — so I deleted that `critical`.

**Verification:**
- [x] `mise run check` green — all 28 packages `ok`; `go build`/`go vet` clean.
- [x] `gofmt -l .` empty outside `cauldron/`.
- [x] `go test -count=1 -run TestParseCanonicalDeployRealm ./internal/registry` → `ok` (reads the real
  `deploy/realm-testnet.txt`, asserts the two ordered entries).
- [x] Test non-vacuity (mutation-proven): URL-shaping a line → FAIL; dropping a hub → FAIL; original file
  restores green with no diff vs HEAD.
- [x] `deploy/realm-testnet.txt` exists at repo-root `deploy/`, domains-only, `Parse` accepts it → exactly
  `sb0.iscc.id`, `sb1.amlet.id`.
- [x] `grep -q 'COPY deploy/realm-testnet.txt /etc/iscc-monitor/realm.txt' Dockerfile` PASS;
  `grep 'COPY .*testdata/realm.txt' Dockerfile` → no match (testdata reference gone from the bake line).
- [x] `deploy/` is git-tracked and NOT in `.dockerignore` (verified) — the new `COPY deploy/…` resolves in
  the Docker build context; the baked path `/etc/iscc-monitor/realm.txt` is unchanged, so the CI `docker`
  `/healthz` smoke (boots with the baked realm) keeps passing.
- [x] CLAUDE.md env table names `deploy/realm-testnet.txt` as the canonical doc baked at
  `/etc/iscc-monitor/realm.txt` and reframes testdata as the in-tree fixture.
- [x] Scope discipline: exactly the 4 declared files; `internal/registry/testdata/realm.txt` left
  untouched (Not-In-Scope honored); `registry.Parse`/`Entry`/follower wiring untouched; `go.mod`/`go.sum`
  byte-identical. Oracle/conformance gate N/A (no proof/verify/didweb/merkle/signature path touched).
- [x] Quality-gate integrity: scanned all unpushed commits (`@{upstream}..HEAD`); the lone `t.Skip`/
  `//go:build` grep hit is in handoff prose, not code. No `nolint`, no skipped/deleted tests, no loosened
  gate.

**Issues found:** (none) — the byte-identity-vs-parse-equivalence question advance flagged is correctly
resolved: `next.md` Scope said "same content as testdata", which means the same *parsed* membership (the
deploy file needs its own deploy-oriented header), and both golden tests confirm parse-equivalence. No
defect. Resolved this iteration: deleted the `critical` "Provide a canonical, mountable testnet realm file
+ instance identity env values" — its full code-closable contract is now delivered (canonical doc at a
documented non-testdata path + baked + `Parse`-tested; the three masthead identity keys were already in
CLAUDE.md's env table and `GET /` already renders them). The residual "confirm membership / hub
reachability / set the masthead values" sub-asks are pure human/infra confirmation, explicitly out of the
loop's scope per target.md M-Deploy.

**Codex second opinion:** clean — no findings. Verdict: "The change cleanly adds the canonical deploy realm
file, points the Dockerfile at it, and adds a test that validates the parsed membership. Existing tests
pass and I did not identify any introduced correctness issues." Nothing to triage.

**Visual check:** n/a — no SSR surface changed (a realm doc, the Dockerfile bake line, a CLAUDE.md doc
edit, and a registry test; no `internal/dashboard`/`dossier`/`web`/`certificate`/template touched).

**Next:** Two code-closable M-Deploy slices remain, neither code-blocked: (1) the deployment/operability
doc — the natural fold-in for the remaining three `critical` infra asks (SQLite volume path + single-file
backup unit + non-root uid; the interim "recreate volume on schema change" migration policy; egress
endpoints + reverse-proxy `:9464` contract + the `/metrics` exposure decision); then (2) the root
`README.md` (`target.md` "Done When" requires it before DONE). Suggest the operability doc next (it closes
3 `critical`s at once), then `README.md`. The `normal` `workflow_dispatch` ref-guard issue stays a
fold-in candidate whenever a workflow file is next touched.

**Notes:**
- After deleting the realm-file `critical`, the open `critical` count drops to 3 — all three are the
  iscc-infra ops asks that the single deployment/operability doc closes (persistence/volume, exposure/
  `/metrics`, egress/sizing). DONE is still gated on those + the root `README.md` + clearing the open
  `normal`s.
- Docker is CI-only on this host (per prior reviews) — the Dockerfile `COPY` change is a one-line
  source-path swap to an existing tracked file and cannot be `docker build`-verified locally; the CI
  `docker` job is the real oracle. The swap is low-risk because the baked target path is unchanged and
  `deploy/` is confirmed reachable in the build context.
- Registry learnings updated: recorded the canonical-vs-testdata distinction (content-equivalent, NOT
  byte-identical; assert parse-equivalence) and the `.dockerignore`-must-not-exclude-`deploy/` invariant.
