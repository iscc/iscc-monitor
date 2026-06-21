# Handoff

## 2026-06-21 — Port `notecheck` into the monitor module as `cmd/notecheck`

**Done:** Ported the fully-independent signature-parity oracle from gitignored
`cauldron/iscc-hub/conformance/notecheck/main.go` into `cmd/notecheck` inside the monitor module,
giving it a real in-repo compile path. The binary reads `--vkey` + a checkpoint on stdin, runs the Go
reference signed-note tooling (`formats/note.NewVerifier` + `x/mod/sumdb/note.Open`) with the strict
`len(n.Sigs)==0 || len(n.UnverifiedSigs)!=0` reject, prints `OK <name>` on success, and maps faults to
the documented exit codes (2 setup / 1 verify / 0 ok). `main` is thin; logic lives in a testable
`run(vkey, in, out)` helper exercised directly by the golden + negative tests.

**Files changed:**
- `cmd/notecheck/main.go` (new): port of the reference oracle. Idiomatic split — pure
  `run(vkey string, in io.Reader, out io.Writer) (string, error)` does NewVerifier → ReadAll →
  note.Open → strict-reject; `main` parses the flag and maps the error to `os.Exit` via an `*errKind`
  (`setup` bool) → `exitCode`. The strict reject mirrors `internal/logclient/verify.go` exactly (the
  load-bearing parity invariant).
- `cmd/notecheck/main_test.go` (new, test): drives `run` directly (no subprocess, no `os.Exit`) — golden
  accept of the live `sb0.iscc.id_checkpoint` (asserts `("sb0.iscc.id/log", nil)`), reject of a
  one-base64-char-corrupted signature line, and bad/empty-vkey → exit-2 setup error. Embeds the exact
  sb0 vkey reproduced by `derive_vkey.py`.
- `go.mod`: `go mod tidy` promoted `github.com/transparency-dev/formats` from `// indirect` to a direct
  require (additive; already in the module graph — downloaded nothing). `go.sum` unchanged. Directive
  stays `go 1.24.0`, no `toolchain` line.

**Verification:** `mise run check` → green (build + vet + test, all 12 packages `ok`, incl.
`cmd/notecheck`). Per-criterion:
- [x] `gofmt -l cmd/notecheck/` prints nothing.
- [x] `go test -count=1 -run TestNotecheck ./cmd/notecheck` PASS uncached (golden accept +
      corrupted-body reject + bad-vkey exit-2).
- [x] Built binary end-to-end: `/tmp/notecheck --vkey "sb0…E2ZE5" < testdata/live/sb0.iscc.id_checkpoint`
      prints `OK sb0.iscc.id/log`, exit 0.
- [x] Corrupted checkpoint rejected non-zero: `sed 's/QLdEY/QLdEZ/' … | /tmp/notecheck --vkey "sb0…"`
      → `note.Open failed: note has no verifiable signatures`, exit 1 (`test $? -eq 1` passes).
      Also confirmed missing-`--vkey` → exit 2 and `--vkey garbage` → exit 2.
- [x] `go mod tidy && git diff --exit-code -- go.sum` clean; re-tidy is idempotent on the working tree
      (go.mod + go.sum byte-stable across a second tidy). `grep -c '^go 1.24.0$'` == 1,
      `grep -c '^toolchain'` == 0.
- [x] Oracle reproducibility: `python3 .claude/derive_vkey.py` prints
      `sb0.iscc.id/log+40b74463+AaV+ivnly67hhzQSQfGqCBP3PlOV2NBcmfGyzGdE2ZE5` byte-for-byte — exactly the
      constant embedded in the test. Scratch dir (`.claude/.scratch/`) removed after running; tree clean.

**Next:** Add `.github/workflows/` with the `notecheck` signature-parity CI job (the sole open `normal`
issue) — now unblocked: it only needs to `go build ./cmd/notecheck` (or `go run`) and shell the binary
out against a captured checkpoint, asserting `OK <name>` + exit 0 and exit 1 on a corrupted one (all
paths now build in-repo). NOTE for that step: CI running `go build ./...` on a fresh checkout must
account for gitignored `cauldron/` (it never reaches CI from this tree, per learnings) — building
`./cmd/notecheck` specifically sidesteps that entirely. Then the inclusion cross-check vs the hub's
`IsccLogInclusionProof` (the second half of M2's Verify bar).

**Notes:**
- The reference's package doc comment said `OK <name> <size>` but its code prints `OK <name>` (single
  field); I matched the CODE (and what `next.md` specifies), and the doc comment in the port says
  `OK <name>`. No behavior surprise.
- Departure from the reference's flat `main`: split into `main` + pure `run` + `*errKind`/`exitCode` so
  the test is a normal `go test` unit (no subprocess, no `os.Exit`) — exactly the idiomatic split
  `next.md` Implementation Notes asked for. Exit-code semantics are identical to the reference
  (verified end-to-end: 2/1/0).
- The corrupted-body negative flips a byte on the SIGNATURE line (last line) rather than the root line,
  so the note framing stays intact and `note.Open` reaches the Ed25519 check → "no verifiable
  signatures" → strict reject. Confirmed non-vacuous: the test asserts the corruption actually changed
  the body before feeding it.
- Oracle/conformance gate APPLIES (this IS the external signature oracle) and is satisfied: the golden
  proves the ported oracle accepts the genuine live sb0 checkpoint, the negative proves it rejects a
  corruption, and the ground truth is independently reproducible via `derive_vkey.py`. WASM purity is
  N/A — this is a `cmd/` binary importing `os`/`flag`/`io`, never part of the `internal/proof`/`didweb`
  WASM-shared seam.
- Scope clean: 1 non-test source file (`main.go`) + 1 test + `go.mod` tidy. Nothing from `## Not In
  Scope` touched — no `.github/workflows/`, no `mise.toml` wiring, no `internal/` test change, no
  follower/`cmd/iscc-monitor`/inclusion/tile-writer change, no fixture/`derive_vkey.py`/did.json edit.
