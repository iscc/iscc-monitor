# Next Work Package

## Step: Pin the mise Go toolchain to 1.26.4 and re-pin WasmVerifyHash so the committed verify.wasm is reproducible from `mise run build:wasm`

## Advances
Closes the lone open **critical** issue — *"Pinned verify.wasm is NOT reproducible from the documented
`mise run build:wasm` (bare-go 1.26.1 pin vs mise's 1.26.4) — fails TestWasmVerifyHashPinned on
regeneration"* (`issues.md`). It is the **NEEDS_WORK gate** at HEAD (review verdict 2026-06-22) and a
`critical`, so it preempts all milestone work. It also unblocks the front-of-queue **WASM verifier
upgrade** milestone Verify criterion — *"the verifier artifact hash matches the published value"* —
which cannot honestly stand while the published pin (`96b2a40d…`) is unreproducible from its own
documented build command (`mise run build:wasm` → `2c91e61f…`). With the toolchain pinned and the hash
re-pinned, artifact == pin == documented-command output, restoring the audited-artifact reproducibility
contract.

## Goal
Make the published `WasmVerifyHash` reproducible from the canonical, gate-runner command. Pin
`mise.toml` to the exact patch `go = "1.26.4"` (mise is the project's gate runner), rebuild the
verifier wasm with `mise run build:wasm`, and re-pin `web.WasmVerifyHash` to that toolchain's
deterministic output (`2c91e61f…`). After this, anyone (or CI) who runs the documented regeneration
command gets byte-identical bytes that pass `TestWasmVerifyHashPinned`.

## Scope
- **Create**: none
- **Modify** (2 non-test/doc source files, plus the regenerated binary artifact):
  - `mise.toml` — change `[tools] go = "1.26"` → `go = "1.26.4"` (exact patch, so the documented build
    is reproducible across machines, not floated to whatever patch mise has installed).
  - `internal/web/web.go` — re-pin `const WasmVerifyHash` (line 93) from
    `96b2a40d459c51817dd87911063ab14abf481080e30ecb17ef311090755852d3` to
    `2c91e61f20560fa98e0fbd6813746c40687861d4b4b3be604d4216557df0f48e`.
  - `internal/web/verify.wasm` — regenerate via `mise run build:wasm` (a generated binary artifact, not
    a hand-edited source file; the go1.26.4 build embeds the `go1.26.4` stamp). This is the artifact
    whose SHA-256 must equal the new const.
- **Reference**:
  - `.claude/context/learnings/cmd-wasm.md` — the TRAP at the bottom ("`mise run build:wasm` and a bare
    `go build` use DIFFERENT Go toolchains … pin the EXACT patch `go = "1.26.4"`") is exactly this fix;
    read it before editing.
  - `.claude/context/learnings/ci.md` — Pages copies the byte-pinned `verify.wasm`, never rebuilds it;
    `actions/setup-go go-version: "1.26"` in the workflows is a separate mechanism (do not touch).
  - `internal/web/web.go:84-93` (the `WasmVerifyHash` doc block) and `web.go:148-163` (the embedded
    `wasm_exec.js` + `verify.wasm` doc comments) — for the exact lines and the toolchain-version notes.

## Not In Scope
- **Do NOT touch `wasm_exec.js`.** Verified this iteration: `lib/wasm/wasm_exec.js` is BYTE-IDENTICAL
  across go1.26.1 and go1.26.4 (both `0c949f49…`) and equals the committed copy, so switching the build
  toolchain does not desync the runtime loader. The `web.go:149` comment ("Go 1.26.1 toolchain's
  lib/wasm/wasm_exec.js") stays factually correct (those bytes also ARE 1.26.4's). Leave it alone.
- **Do NOT edit `.github/workflows/ci.yml` / `pages.yml`** (`go-version: "1.26"`). They run `mise run
  check` (committed-bytes vs const) and copy the pinned wasm — they never rebuild it, so they do not
  affect pin reproducibility. Changing them is out of scope and risks unrelated CI churn.
- **Do NOT touch `CLAUDE.md` / ADR / PRD "Go 1.26" stack prose.** Those describe the locked minor
  series (1.26) and remain accurate; this step pins the build-toolchain patch in `mise.toml` only. No
  doc edit is needed because no documented behavior/usage changes (the regeneration command is
  unchanged).
- **Do NOT change any `cmd/wasm` / `cmd/wasm/verifyadapter` source.** The 6-arg/id-binding shim is
  already correct and behaviorally identical across both toolchains; this step is purely the
  reproducibility pin, not a behavior change.
- **Do NOT pursue the WASM signature-half gap, the Pages repo-Settings enablement, or the dossier WASM
  caller** — those are the next milestone sub-steps after this critical closes (see state.md "Next
  Milestone" 2 & 3), not this step.

## Implementation Notes
- **This is option (a)** from the issue / review / state ("Next Milestone" step 1) — preferred because
  mise is the gate runner, so the gate-runner toolchain becomes the single source of truth for the
  artifact bytes. Do NOT take option (b) (downgrade mise to 1.26.1 to keep `96b2a40d…`): the bare
  PATH go is 1.26.1 only on this devcontainer, while mise already has 1.26.4 installed and the
  workflows use `go-version: "1.26"` → newest patch; pinning the gate runner forward is the durable
  choice.
- **Exact sequence (verified reproducible this iteration):**
  1. Edit `mise.toml`: `go = "1.26"` → `go = "1.26.4"`.
  2. Run `mise run build:wasm`. The task is `GOOS=js GOARCH=wasm CGO_ENABLED=0 go build -trimpath
     -ldflags=-buildid= -buildvcs=false -o internal/web/verify.wasm ./cmd/wasm`. With go1.26.4 it
     deterministically writes a `verify.wasm` whose SHA-256 is
     `2c91e61f20560fa98e0fbd6813746c40687861d4b4b3be604d4216557df0f48e` and embeds the `go1.26.4` stamp.
  3. Edit `web.go:93` to that new hash.
  4. `git add internal/web/verify.wasm` so the regenerated binary is committed alongside the const.
- **The `-buildvcs=false` flag is load-bearing** (web.go:88-90): without it Go stamps VCS
  revision/dirty metadata into the wasm and the hash floats. The `build:wasm` task already sets it —
  do not change the task.
- **Verify the artifact embeds the right toolchain, not just the right length:** `strings
  internal/web/verify.wasm | grep -oE 'go1\.26\.[0-9]+'` must show `go1.26.4` (was `go1.26.1`). This is
  the cmd-wasm.md learning: confirm content, not just that the build exited 0.
- **The behavior content must survive the rebuild** (it is the same source, but assert it): `strings
  internal/web/verify.wasm | grep -c "expected 5 or 6 args"` → 1, `"expected 5 args"` → 0,
  `"decode record envelope"` → 1. (The two toolchains are behaviorally identical — review confirmed —
  but pin the check so a wrong rebuild can't slip through.)
- **Correctness rule applied (learnings.md):** the WASM build is the audited artifact; the pin is the
  client's only handle to confirm it loaded the audited verifier. The pin MUST be reproducible from the
  documented command (cmd-wasm.md TRAP: "always rebuild + pin via `mise run build:wasm`, … pin
  `mise.toml` to the EXACT patch `go = "1.26.4"`"). A pin produced by a different toolchain than the
  documented command is a broken contract even when `mise run check` is green (it checks committed-bytes
  vs const, not reproducibility-from-command).
- **Leave the working tree clean.** This iteration's feasibility probe already restored `verify.wasm`
  to the committed `96b2a40d…`; your edit deliberately replaces it with the `2c91e61f…` rebuild.

## Verification
- `mise run build:wasm && sha256sum internal/web/verify.wasm` prints
  `2c91e61f20560fa98e0fbd6813746c40687861d4b4b3be604d4216557df0f48e  internal/web/verify.wasm`, and that
  hash byte-equals the committed `internal/web/verify.wasm` (i.e. re-running the documented command
  leaves `git status` clean — the pin is now reproducible from its own command).
- That same hash string equals `web.WasmVerifyHash` in `internal/web/web.go`.
- `go test -run TestWasmVerifyHashPinned ./internal/web` passes (committed bytes == committed const).
- `strings internal/web/verify.wasm | grep -oE 'go1\.26\.[0-9]+'` → `go1.26.4` (single value);
  `grep -c "expected 5 or 6 args"` → 1, `grep -c "expected 5 args"` → 0, `grep -c "decode record envelope"` → 1.
- `mise exec -- go version` → `go1.26.4` (mise.toml now pins the exact patch).
- `mise run check` is green (all packages build + vet + test) and `gofmt -l .` is empty.

## Done When
`mise run build:wasm` regenerates `internal/web/verify.wasm` byte-for-byte to the committed artifact
whose SHA-256 equals `web.WasmVerifyHash`, `TestWasmVerifyHashPinned` passes, and `mise run check` is
green — i.e. artifact == pin == documented-command output, closing the reproducibility critical.
