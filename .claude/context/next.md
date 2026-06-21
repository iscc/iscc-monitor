# Next Work Package

## Step: Wire CI — `mise run check` + the `notecheck` signature-parity oracle shell-out

## Goal
Stand up `.github/workflows/` so the green quality gate (`mise run check`) and the
fully-independent external signature oracle (`cmd/notecheck`) run on every push/PR — not just
locally. This closes the sole open `normal` issue and makes the trust-root conformance gate actually
gate, which `target.md` requires ("the fully-independent external oracle `notecheck` … is built and
**shelled out in CI**").

## Scope
- **Create**: `.github/workflows/ci.yml` — a single GitHub Actions workflow (CI infrastructure, not a
  Go source file; it does not count against the 3-non-test/doc-file budget).
- **Modify**: (none — no Go source or config changes are needed; `mise.toml` already defines `check`).
- **Reference**:
  - `/workspace/iscc-monitor/.claude/context/handoff.md` — the `**Next:**` block spells out the exact
    CI shape (build `./cmd/notecheck`, shell it against a captured checkpoint, assert `OK <name>` +
    exit 0, exit 1 on a corrupted one, plus a `mise run check` job).
  - `/workspace/iscc-monitor/cmd/notecheck/main.go` — confirms the CLI contract: `--vkey <string>`,
    checkpoint text on **stdin**, prints `OK <name>` + exit 0 / exit 1 verify-fail / exit 2 setup-fail.
  - `/workspace/iscc-monitor/testdata/live/sb0.iscc.id_checkpoint` — the captured checkpoint the CI
    shells the oracle against (signed by `sb0.iscc.id/log`, size 10183).
  - `/workspace/iscc-monitor/mise.toml` — the `[tasks.check]` gate the workflow invokes; `[tools] go =
    "1.24"`. Note `gofmt -l .` is NOT in `check` (the loop's `review` agent judges it); see Notes.
  - `/workspace/iscc-monitor/.gitignore` — confirms `cauldron/` is gitignored so it never reaches a
    fresh CI checkout (the `go build ./...` cauldron pitfall in `learnings.md` does not apply to CI).

## Not In Scope
- Do **not** add a `gofmt`/formatting gate step to CI. `mise run check` deliberately excludes `gofmt
  -l .` (it exits 0 even when listing files, so it cannot gate by exit code portably — see `mise.toml`
  header); formatting stays the `review` agent's job. Adding a hand-rolled `gofmt -l` exit-code check
  is a separate decision, not this step.
- Do **not** edit any Go source, `mise.toml`, `go.mod`/`go.sum`, `cmd/notecheck`, or the `low`-issue
  vestigial `out` param. CI is purely additive.
- Do **not** start the inclusion cross-check vs `IsccLogInclusionProof` (the next M2 slice) or the
  live tile-ingestion writer — those wait until CI is green.
- Do **not** add caching, matrix builds, multi-OS runners, release jobs, or `go mod tidy`/`go mod
  verify` steps beyond what is needed to gate. Keep the workflow minimal (KISS); a tidy-drift gate can
  be a later additive step.

## Implementation Notes
- Trigger on `push` and `pull_request`. Include both `develop` (the working branch) and `main` (the
  default), or use no branch filter — either is fine. Run on `ubuntu-latest`; the workflow `run:` steps
  execute in bash on the Linux runner, so shell scripting (incl. the corruption one-liner below) is
  fine — the CLAUDE.md cross-platform rule constrains **dev tooling developers run locally**, not the
  CI runner OS.
- Prefer one job with clearly named steps (simplest), `env: CGO_ENABLED: 0` at the job level per
  ADR-0003 / `target.md`:
  1. `actions/checkout@v4`.
  2. `actions/setup-go@v5` with `go-version: '1.24'`.
  3. **check gate**: `go build ./... && go vet ./... && go test ./...` (this IS `mise run check`,
     inlined). Inlining avoids needing `mise` on the runner. If you instead prefer to invoke `mise run
     check` for single-source-of-truth, add `jdx/mise-action@v2` before it — acceptable, but inlining
     the three commands is the lighter path. Pick one; do not do both.
  4. **notecheck oracle** (after the build): `go build -o notecheck ./cmd/notecheck`, then shell the
     assertions against `testdata/live/sb0.iscc.id_checkpoint` (use `set -euo pipefail` and a guarded
     non-zero-exit check so the binary's intentional exit 1 doesn't fail the step):
     - **accept**: `./notecheck --vkey "$VKEY" < testdata/live/sb0.iscc.id_checkpoint` prints exactly
       `OK sb0.iscc.id/log` and exits 0. Assert the output string, not just the exit code.
     - **reject (corrupted)**: corrupt the signature line and assert the binary exits **1**. Use the
       deterministic base64-char flip the handoff used: `sed 's/QLdEY/QLdEZ/'` (`QLdEY…` begins the sig
       line's base64). Guard it, e.g.
       `if sed 's/QLdEY/QLdEZ/' testdata/live/sb0.iscc.id_checkpoint | ./notecheck --vkey "$VKEY"; then
       echo "ERROR: oracle accepted a corrupted checkpoint"; exit 1; fi` so a *spurious accept* fails CI.
     - **bad vkey** (optional, cheap): `./notecheck --vkey not-a-valid-vkey < … ` exits **2**.
  - The golden vkey to pass: `sb0.iscc.id/log+40b74463+AaV+ivnly67hhzQSQfGqCBP3PlOV2NBcmfGyzGdE2ZE5`
    (the embedded `sb0VKey`, confirmed in `cmd/notecheck/main_test.go` and `derive_vkey.py`). Put it in
    a step-level `env: VKEY:` and always double-quote `"$VKEY"` — the `+` chars must not shell-mangle.
- `go build ./...` on a fresh CI checkout is **safe**: `cauldron/` is gitignored (verified in
  `.gitignore`), so the "cauldron breaks `go build ./...`" pitfall in `learnings.md` does not reach
  CI. The `./cmd/notecheck` build sidesteps it regardless.
- Pin action versions (`@v4`/`@v5`/`@v2`) rather than `@main` for reproducibility.
- **Correctness rule in play (target.md "Oracle / conformance gate"):** `notecheck` is the
  *fully-independent* external oracle — a green-but-wrong verify must not ship on `mise run check` + an
  LLM PASS alone. The **reject-corrupted assertion is the load-bearing half**: a workflow that only
  checks the accept path is a green-but-useless gate. Both the accept and the reject step must be
  present.

## Verification
- `.github/workflows/ci.yml` exists and is valid YAML:
  `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml'))"` exits 0 (PyYAML present
  with the repo's Python 3.11; if PyYAML is absent, `yamllint`/any YAML parser substitute is acceptable).
- The accept step reproduces locally:
  `go build -o /tmp/notecheck ./cmd/notecheck && /tmp/notecheck --vkey
  "sb0.iscc.id/log+40b74463+AaV+ivnly67hhzQSQfGqCBP3PlOV2NBcmfGyzGdE2ZE5" <
  testdata/live/sb0.iscc.id_checkpoint` prints `OK sb0.iscc.id/log` and exits 0.
- The reject step reproduces locally:
  `sed 's/QLdEY/QLdEZ/' testdata/live/sb0.iscc.id_checkpoint | /tmp/notecheck --vkey
  "sb0.iscc.id/log+40b74463+AaV+ivnly67hhzQSQfGqCBP3PlOV2NBcmfGyzGdE2ZE5"; test $? -eq 1` exits 0
  (i.e. the binary exited 1 on the corrupted checkpoint).
- `mise run check` is green locally (proves the gate the workflow runs still passes; CI did not
  regress it).
- After push to `develop`: `gh run list --branch develop --json status,conclusion,name` shows a
  completed run with `conclusion == "success"` — the real CI-is-live confirmation the issue's "Verify
  fixed" asks for. `advance` performs this push-and-check once the workflow is committed.

## Done When
`.github/workflows/ci.yml` is valid YAML running both a `mise run check`-equivalent (build+vet+test,
`CGO_ENABLED=0`) gate and the `notecheck` oracle shell-out (accept → `OK sb0.iscc.id/log` exit 0,
corrupted → exit 1), the local reproductions above all pass, and a pushed `develop` run reports
`success`.
