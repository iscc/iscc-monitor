# Handoff

## 2026-06-21 — Wire CI — `mise run check` + the `notecheck` signature-parity oracle shell-out

**Done:** Added `.github/workflows/ci.yml`: a single `ubuntu-latest` job (`CGO_ENABLED=0`) that runs
the `mise run check` gate inlined (build + vet + test) and then shells out the fully-independent
`cmd/notecheck` oracle against the captured `sb0.iscc.id` checkpoint — asserting the exact accept
output (`OK sb0.iscc.id/log`, exit 0) and that a corrupted signature line is rejected (exit 1). This
closes the sole open `normal` issue and makes the trust-root conformance gate actually gate in CI.

**Files changed:**
- `.github/workflows/ci.yml` (new): CI workflow. Triggers on push/PR to `develop`+`main`;
  `actions/checkout@v4` + `actions/setup-go@v5` (go 1.24); one `check` step (`go build/vet/test`); one
  `notecheck` step under `set -euo pipefail` that string-asserts the accept and guards the
  reject-corrupted (`sed 's/QLdEY/QLdEZ/'`) so a spurious accept fails the build.

**Verification:** `mise run check` → green (build + vet + test, all 11 packages `ok`). Per criterion:
- [x] `.github/workflows/ci.yml` is valid YAML — PyYAML/yamllint absent locally, so validated via a
      throwaway Go program in a temp module using the cached `gopkg.in/yaml.v3` → "YAML valid", exit 0.
      Project `go.mod`/`go.sum` untouched (`git diff --exit-code` clean).
- [x] Accept reproduces: `go build -o /tmp/notecheck ./cmd/notecheck` then
      `/tmp/notecheck --vkey "sb0…E2ZE5" < testdata/live/sb0.iscc.id_checkpoint` → `OK sb0.iscc.id/log`, exit 0.
- [x] Reject reproduces: `sed 's/QLdEY/QLdEZ/' … | /tmp/notecheck --vkey …` exits 1 (`note has no
      verifiable signatures` on stderr). Bad vkey (`not-a-valid-vkey`) → exit 2.
- [x] Ran the workflow's full `notecheck` `run:` block verbatim under `bash -c 'set -euo pipefail …'`
      → "accept held" + "reject-corrupted held", overall exit 0 (the binary's intentional exit-1 on the
      corrupted input is caught by the guard, not propagated).
- [ ] Pushed-`develop` `gh run list … conclusion == success` — NOT performed (see Notes; this sandbox
      has no push/`gh` path). `review` or the operator must confirm the live run.

**Next:** Start the inclusion cross-check vs the hub's `IsccLogInclusionProof` — the second half of
M2's Verify bar (the first independent conformance check beyond signature parity). It needs a captured
`IsccLogInclusionProof` fixture and `proof.VerifyInclusion` over the mirrored tiles; pair it with the
`fsck` root-rebuild over the `SQLiteFetcher` (already landed as `RunFsck`, still unwired) once the live
tile-ingestion writer exists. A later additive CI step could add a `go mod tidy` drift gate (kept out
here per `## Not In Scope`).

**Notes:**
- **CI-is-live confirmation is deferred.** This environment cannot push to `origin/develop` or run
  `gh`, so the "pushed run reports success" criterion is unverified here. Everything CI executes was
  reproduced locally (the gate is `mise run check`, already green; the notecheck block ran verbatim).
  The commit is on `develop`; the operator/`review` should push and confirm
  `gh run list --branch develop --json status,conclusion,name` shows `success`.
- **Scope:** purely additive — only `.github/workflows/ci.yml` created. No Go source, `mise.toml`,
  `go.mod`/`go.sum`, `cmd/notecheck`, or the `low`-issue vestigial `out` param touched. No `gofmt` gate
  added to CI (deliberately, per `## Not In Scope` — `mise run check` excludes it and `gofmt -l` can't
  gate by exit code portably). No caching/matrix/multi-OS/release/tidy steps (KISS).
- **Inlined vs `mise run check`:** chose the lighter inline of the three commands (build/vet/test) so
  the runner needs no `mise`; did not also add `jdx/mise-action` (the notes said pick one). The inlined
  commands are byte-identical to `[tasks.check]` in `mise.toml`, so they cannot silently diverge in
  behavior — though if `mise.toml`'s `check` later changes, this workflow must be updated in lockstep
  (a single-source-of-truth tradeoff worth flagging).
- The reject step relies on `note.Open` itself failing on a flipped sig byte (exit 1), NOT the strict
  `len(n.UnverifiedSigs)!=0` branch — same path the unit test exercises; consistent with the prior
  handoff's note that the strict branch first becomes independently exercisable at M7 (witness cosig).
- Oracle gate APPLIES and is satisfied by *running the oracle in CI itself* — the whole point of this
  slice. No new crypto/derivation code, so `derive_vkey.py`/`fsck`/golden-vector re-derivation are N/A
  for the change set; the embedded golden vkey is the one `review` already confirmed from ground truth.
