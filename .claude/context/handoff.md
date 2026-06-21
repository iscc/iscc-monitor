# Handoff

## 2026-06-21 — Review of: Wire CI — `mise run check` + the `notecheck` signature-parity oracle shell-out

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added `.github/workflows/ci.yml`: one `ubuntu-latest` job (`CGO_ENABLED=0`) that
inlines the `mise run check` gate (build + vet + test) and shells out the fully-independent
`cmd/notecheck` signature oracle against the captured `sb0.iscc.id` checkpoint — asserting the exact
accept string (`OK sb0.iscc.id/log`, exit 0) and that a corrupted signature line is rejected. The diff
is purely additive (only `ci.yml` + this handoff; zero Go source, `go.mod`/`go.sum`, `mise.toml`
untouched), scoped exactly to `next.md`, and reproduced byte-for-byte locally. This closes the sole
open `normal` issue (no CI).

**Verification:**
- [x] `mise run check` (build + vet + test) — green, all 11 packages `ok` uncached (`-count=1`).
- [x] `gofmt -l .` (outside gitignored `cauldron/`) — clean.
- [x] `.github/workflows/ci.yml` is valid YAML — PyYAML/yamllint absent locally; validated via the
      cached `gopkg.in/yaml.v3@v3.0.1` in a throwaway module → "YAML valid".
- [x] Accept reproduces — `/tmp/notecheck --vkey "sb0…E2ZE5" < testdata/live/sb0.iscc.id_checkpoint`
      → `OK sb0.iscc.id/log`, exit 0.
- [x] Reject(corrupted) reproduces — `sed 's/QLdEY/QLdEZ/' … | /tmp/notecheck …` → exit 1; the `QLdEY`
      token occurs only once (start of the line-5 base64 sig), so the flip genuinely corrupts the
      *signature*, not the body. Bad vkey → exit 2.
- [x] The verbatim CI `notecheck` `run:` block under `bash -c 'set -euo pipefail …'` → "accept +
      reject-corrupted both held", overall exit 0 (the corrupted input's intentional exit-1 is caught
      by the guard, not propagated).
- [x] Inlined CI commands byte-identical to `mise.toml [tasks.check]`; GitHub Actions runs `run:` with
      `bash -e`, so a non-zero `go build`/`go vet` aborts the step like the `&&` short-circuit.
- [x] go.mod/go.sum byte-identical (`git diff HEAD~1..HEAD -- go.mod go.sum` empty); working tree clean.
- [x] Gate-integrity scan over all 3 unpushed commits — no `//nolint`/`t.Skip`/swallowed-error/
      build-tag/deleted-assertion patterns; no `.go` file changed at all (only context md + `ci.yml`).
- [x] **Oracle gate (APPLIES — this wires the external signature oracle into CI):** reviewer
      mutation-proved the reject guard with a stdin-draining always-accept stub → the CI block exits 1
      (a realistic green-but-wrong verify regression IS caught). No crypto/derivation code changed, so
      `derive_vkey.py`/`fsck`/golden re-derivation are N/A for the change set; the embedded golden vkey
      is the one already confirmed from ground truth.
- [ ] Pushed-`develop` `gh run list … conclusion == success` — confirmable only *after* push (CI runs
      post-push). `gh` + remote are present; the push below triggers it. Everything CI executes is
      reproduced locally, so failure risk is low (action-version availability aside).

**Issues found:**
- (none blocking) Robustness nuance recorded in learnings, not filed: the CI reject guard's correctness
  leans on the downstream process draining stdin before exiting. The real `notecheck` does
  (`io.ReadAll` before deciding), so reject → exit 1 and a wrongly-accepting regression → exit 0
  (caught). The one case the `sed | tool` + `pipefail` shape *misses* is a tool that exits 0 without
  reading stdin (then `sed` dies of SIGPIPE 141, `pipefail` masks the accept) — but a signature verifier
  cannot do that, so the trust-root gate is sound. Future guards piping into possibly-short-circuiting
  tools should use a temp-file + explicit `$?` check instead.
- (low, still open) `cmd/notecheck`'s `run` has a vestigial `out io.Writer` param — untouched this slice.

**Next:** Start the inclusion cross-check vs the hub's `IsccLogInclusionProof` — the second half of M2's
Verify bar and the first independent conformance check beyond signature parity. Needs a captured
`IsccLogInclusionProof` fixture + `proof.VerifyInclusion` over the mirrored tiles; pair it with the
`fsck` root-rebuild over `SQLiteFetcher` (`RunFsck` already landed, still unwired) once the live
tile-ingestion writer exists. A `go mod tidy` drift gate can be a later additive CI step.

**Notes:**
- **CI-is-live confirmation happens at push, not before.** This is inherent to CI (runs after push),
  not a gap in the work — the `[ ]` above is the only unmet `next.md` criterion and is unmeetable
  pre-push. The verdict is PASS on the strength of the full local reproduction + a sound, mutation-proven
  reject gate.
- **`notecheck` build artifact at repo root is NOT gitignored**, but CI builds it on an ephemeral
  runner so it never reaches the repo; no local dev step does `go build -o notecheck` either. Harmless,
  not filed. (If a future local script emits it, add `/notecheck` to `.gitignore` alongside the
  existing `/iscc-monitor` entries.)
- **Single-source-of-truth tradeoff:** the inlined CI commands duplicate `mise.toml [tasks.check]`. They
  are byte-identical today, but if `[tasks.check]` changes, `ci.yml` must be updated in lockstep. An
  acceptable KISS choice (avoids needing `mise` on the runner); worth a glance whenever `mise.toml`
  changes.
