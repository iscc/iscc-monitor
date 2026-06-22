## 2026-06-22 — Add a public-facing root `README.md`

**Done:** Created a tracked root `README.md` — the human-facing front door — with an honest
verifiable-cache project overview, the Go 1.26 / `CGO_ENABLED=0` stack, a copy-pasteable testnet
build/run snippet, the `mise run check` gate, a GHCR/deploy pointer, and links to the specs. The full
env-var table is LINKED (not duplicated) to `CLAUDE.md` so the two cannot drift. This closes the last
code/doc-closable `target.md` "Done When" gate.

**Files changed:**
- `README.md` (created, repo root): the only file this step creates. No Go source touched; the
  ≤3 non-test/doc budget is untouched.

**Verification:** `mise run check` → green (28 packages `ok`, cached — no `.go` file changed, gate
re-confirmed). Per `next.md` criteria, all PASS:
- `test -f README.md` → PASS (file exists at repo root).
- `gofmt -l .` → empty (exit 0; no Go file touched).
- `grep -q 'mise run check'` → PASS (names the gate).
- `grep -q 'cmd/iscc-monitor'` → PASS (shows build/run path).
- `grep -qi 'verifiable cache'` → PASS (honest framing present).
- No dead relative links: `CLAUDE.md`, `deploy/OPERATING.md`, `.claude/prd`, `.claude/adr`,
  `.claude/context/README.md`, `internal/registry/testdata/realm.txt`, and the two referenced ADR
  files (0011, 0003) all `test -e` 0. The `CLAUDE.md#running-a-local-dev-instance` anchor matches the
  live `## Running a local dev instance` heading's GitHub slug.

**Next:** With this `target.md` "Done When" gate closed, `update-state` should re-check the DONE
condition: the last code/doc-closable requirement is satisfied. Remaining open issues are the Compose
volume-prep `chown` `normal` (review's last finding), the iscc-infra `critical`s (persistence /
exposure / egress — infra-side, not code/doc-closable here), and assorted `normal`/`low`. If those
infra `critical`s are genuinely external to this repo, the loop may be at its IDLE/human-blocked edge
— flag that rather than spinning on cosmetic chrome.

**Notes:**
- Scope-clean: one created doc file, zero Go source, zero edits to the
  `CLAUDE.md`/`deploy/OPERATING.md`/`mise.toml` references (per Not-In-Scope).
- Kept the snippet honest to the config contract (`learnings/config.md`): both `ISCC_MONITOR_DB` and
  `ISCC_MONITOR_REALM` stated as required with NO default DB path; intervals/masthead noted as
  optional. The README's local-dev snippet passes `ISCC_MONITOR_REALM` explicitly (this is the
  source-tree dev path using `internal/registry/testdata/realm.txt`); the image-baked-realm nuance
  lives in `deploy/OPERATING.md`, which the README points to — the two are consistent, not
  contradictory.
- Did not duplicate the env-var table or restate OPERATING.md substance — linked both, as the issue
  mandates, so they cannot drift.
- No badges / CI-lint tooling added (per Not-In-Scope: GHCR pull badge waits on package visibility, an
  iscc-infra step). Plain prose + working relative links only.
