# Next Work Package

## Step: Port `notecheck` into the monitor module as `cmd/notecheck`

## Goal
Give the fully-independent signature-parity oracle (`notecheck`) a real in-repo
**compile path** by porting it from gitignored `cauldron/` into `cmd/notecheck` inside
the monitor module. This is the prerequisite the CI gap (the sole open `normal` issue +
the review handoff `Next:`) is blocked on: `notecheck` currently exists only as a
module-less `main.go` under `cauldron/` (gitignored, external deps, no local compile
path), so CI cannot `go build` or `go run` it on a fresh checkout. A real `cmd/notecheck`
package makes the oracle buildable and shell-out-able, after which the `.github/workflows/`
YAML becomes a trivial follow-up step that only invokes things that already build.

## Scope
- **Create**: `cmd/notecheck/main.go` — port of
  `cauldron/iscc-hub/conformance/notecheck/main.go`, verbatim-in-shape, into `package main`
  under the monitor module (`github.com/iscc/iscc-monitor/cmd/notecheck`). The only
  non-test source file.
- **Create**: `cmd/notecheck/main_test.go` — golden test driving the oracle against the
  live `testdata/live/sb0.iscc.id_checkpoint` fixture (test file, not counted toward the
  3-file limit).
- **Modify**: `go.mod` / `go.sum` — `go mod tidy` promotes
  `github.com/transparency-dev/formats` from `// indirect` to a direct require (it is
  ALREADY in the module graph; this is additive + tidy-idempotent, NOT a version bump).
- **Reference**:
  - `cauldron/iscc-hub/conformance/notecheck/main.go` — the source to port (45 lines:
    `f_note.NewVerifier(--vkey)` + `note.Open(stdin, VerifierList(v))` + the strict
    `len(n.Sigs)==0 || len(n.UnverifiedSigs)!=0` reject, prints `OK <name>`, exit codes 2/1/0).
  - `.claude/derive_vkey.py` — the trust-root oracle that prints the golden vkey
    (sb0: `sb0.iscc.id/log+40b74463+AaV+ivnly67hhzQSQfGqCBP3PlOV2NBcmfGyzGdE2ZE5`). It writes
    `.claude/.scratch/` (NOT gitignored) — `rm -rf .claude/.scratch` after running so the
    tree stays clean.
  - `internal/logclient/verify.go` + `learnings.md` "Checkpoint signed-note verification" —
    the monitor-side path this oracle is the external parity check for; the
    `len(n.Sigs)==0 || len(n.UnverifiedSigs)!=0` reject mirrors `note.Open` exactly.
  - `testdata/live/sb0.iscc.id_checkpoint` — the live signed checkpoint the test feeds on
    stdin (name `sb0.iscc.id/log`, size `10183`, signer keyhash `40b74463`).

## Not In Scope
- **Do NOT create `.github/workflows/` or any CI YAML.** That is the *next* step; it
  depends on this compile path existing first, and bundling it here would push the file
  count and the "runnable-without-GitHub-Actions" verification past one clean step.
- Do NOT wire `notecheck` into `mise.toml` tasks or into any `internal/` Go test — the
  monitor's own crypto path is already golden-tested; this step adds only the *external*
  oracle binary, it does not change how the monitor verifies.
- Do NOT touch the follower / `cmd/iscc-monitor` / inclusion cross-check / live
  tile-ingestion writer — all later M2 slices.
- Do NOT change `derive_vkey.py`, the checkpoint fixtures, or any did.json (the stale sb1
  fixture refresh is its own separate step).

## Implementation Notes
- **Port faithfully, do not redesign.** Keep the exact two-verifier composition from the
  reference: `f_note "github.com/transparency-dev/formats/note"` for `NewVerifier(*vkey)`,
  then `golang.org/x/mod/sumdb/note` for `note.Open(body, note.VerifierList(v))`. Both are
  already in `go.mod` (formats currently `// indirect`, x/mod direct), so `go mod tidy`
  downloads nothing new — it only re-classifies formats as a direct require.
- **Match the reference's exit codes** (CI relies on them): `--vkey` missing / bad vkey /
  stdin read error → `os.Exit(2)`; `note.Open` failure OR the
  `len(n.Sigs)==0 || len(n.UnverifiedSigs)!=0` reject → `os.Exit(1)`; success → print
  `OK <n.Sigs[0].Name>` and exit 0. This strict-signature reject is the same one
  `internal/logclient/verify.go` uses (learnings: "`note.Open` success check mirrors the
  notecheck oracle exactly") — it is the load-bearing parity invariant; keep it identical.
- **Keep `main` thin and testable.** Idiomatic split: a pure helper, e.g.
  `func run(vkey string, in io.Reader, out io.Writer) (string, error)` doing
  `NewVerifier → ReadAll → note.Open → strict-reject` and returning `(name, err)`; `main`
  parses the flag, calls `run`, maps errors to the documented exit codes via `os.Exit`.
  The test drives `run` directly (no subprocess, no `os.Exit`) so it stays a normal
  `go test` unit — that is what lets a single `mise run check` cover the oracle's logic
  while CI shells out the *binary* in the later step.
- **Correctness rule (learnings, "Checkpoint signed-note verification"):** the vkey's
  middle `+<hex>+` field is the signed-note keyhash; sb0's is `40b74463`. Embed the exact
  sb0 vkey string in the test and assert `run` returns `("sb0.iscc.id/log", nil)` for the
  live fixture. Add a NEGATIVE case: a one-byte-corrupted checkpoint body (flip a byte in
  the base64 signature line) must make `run` return a non-nil error — a green-but-wrong
  oracle that accepts anything is worthless. Also assert an empty/garbage vkey makes the
  `NewVerifier` leg error (the exit-2 path) so the flag-handling branch is covered.
- **WASM purity is N/A here** — this is a `cmd/` binary importing `os`/`flag`/`io`, never
  part of the `internal/proof`/`didweb` WASM-shared seam. No `GOOS=js` constraint applies.
- **Oracle/conformance gate APPLIES** (this IS the external signature oracle being made
  buildable) and is satisfied within this step by the golden + negative test proving the
  ported oracle accepts the genuine live sb0 checkpoint and rejects a corrupted one. The
  fully-independent ground truth is reproducible: `.claude/derive_vkey.py` prints the same
  sb0 vkey the test embeds.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all pass).
- `gofmt -l cmd/notecheck/` prints nothing.
- `go test -count=1 -run TestNotecheck ./cmd/notecheck` passes uncached (golden accept +
  corrupted-body reject + bad-vkey error).
- The built binary works end-to-end against the live fixture:
  `go build -o /tmp/notecheck ./cmd/notecheck && /tmp/notecheck --vkey "sb0.iscc.id/log+40b74463+AaV+ivnly67hhzQSQfGqCBP3PlOV2NBcmfGyzGdE2ZE5" < testdata/live/sb0.iscc.id_checkpoint`
  prints `OK sb0.iscc.id/log` and exits 0.
- A corrupted checkpoint is rejected non-zero:
  `sed 's/QLdEY/QLdEZ/' testdata/live/sb0.iscc.id_checkpoint | /tmp/notecheck --vkey "sb0.iscc.id/log+40b74463+AaV+ivnly67hhzQSQfGqCBP3PlOV2NBcmfGyzGdE2ZE5"; test $? -eq 1`.
- `go mod tidy && git diff --exit-code -- go.mod go.sum` exits 0 (idempotent after the
  formats direct-require promotion); `grep -c '^go 1.24.0$' go.mod` == 1 and
  `grep -c '^toolchain' go.mod` == 0.

## Done When
`cmd/notecheck` compiles inside the monitor module, its golden + negative test passes under
`mise run check`, and the built binary prints `OK sb0.iscc.id/log` for the live fixture
while exiting non-zero on a corrupted one — giving CI a real in-repo compile path to shell
the oracle out in the follow-up workflow step.
