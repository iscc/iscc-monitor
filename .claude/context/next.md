# Next Work Package

## Step: WASM verifier entrypoint — `cmd/wasm` exporting `verify.VerifyInclusion` to JS

## Advances
Milestone **WASM verifier upgrade** (target.md) — the only un-started v1 milestone with offline-provable
Verify criteria, continuing the arc the prior iteration began (the pure `internal/proof/verify` core
landed and compiles under `GOOS=js GOARCH=wasm`). The handoff `**Next:**` from review names exactly this:

> the natural next WASM sub-step is the `GOOS=js`/`syscall/js` entrypoint (`cmd/wasm`) that exports
> `verify.VerifyInclusion` to JS.

Its Verify criterion:

> **Verify:** identical vectors yield identical verdicts (WASM vs server); the verifier artifact hash
> matches the published value; a `(size, root)` mismatch renders the guided split-view alert, not a dead
> error.

This step closes the **first half** of "identical verdicts (WASM vs server)": it makes the WASM side
*callable* — a `GOOS=js GOARCH=wasm` `package main` that exports the shared `verify.VerifyInclusion` core
to JavaScript, marshaling the same base64-Std proof-bundle inputs the server already emits. It is a
**verifiable skeleton**: the lazy tier-2 enhancement on the certificate/dossier and the standalone
`monitor.iscc.codes` app (published hash + SRI + split-view alert) are the remaining sub-steps, listed
under Not In Scope so later iterations continue this same arc.

## Goal
Lay the WASM verifier entrypoint: a thin `syscall/js` glue that registers a JS-callable function, plus a
**pure, linux-testable** decode adapter (`verifyJSON`) that turns the base64-Std proof-bundle fields the
monitor emits (record, proof hashes, root) into `verify.VerifyInclusion` args and folds the three-way
verdict into a JS-friendly result. This makes the WASM side callable so a later step can wire it as
progressive enhancement, and proves WASM-vs-server verdict parity at the marshaling boundary.

## Scope
- **Create**:
  - `cmd/wasm/verify_adapter.go` — the pure (no `syscall/js`) decode-and-verify adapter:
    `verifyJSON(record, root string, proofB64 []string, index, size uint64) (verified bool, errMsg string)`
    (or a small result struct). base64-Std-decodes the inputs, calls
    `github.com/iscc/iscc-monitor/internal/proof/verify`.`VerifyInclusion`, and maps the three-way verdict
    to a flat `(verified, errMsg)` JS-shaped result. **No build tag** — this file compiles + tests on
    linux. Start the file with a docstring naming it the pure WASM-marshaling adapter.
  - `cmd/wasm/main.go` — the `//go:build js && wasm` entrypoint: imports `syscall/js`, wraps `verifyJSON`
    in a `js.FuncOf` exposing it as a global JS function (e.g. `globalThis.isccVerifyInclusion`), and
    blocks (`select{}`) so the runtime stays alive. **This file is excluded from `go build ./...` on
    linux** by the build tag (verified during scoping). Start it with a docstring stating it is the
    WASM-only entrypoint and the marshaling lives in the untagged adapter.
  - `cmd/wasm/verify_adapter_test.go` — table-driven golden test for `verifyJSON` using the SAME 4-leaf
    base64-Std vector already pinned in `internal/proof/verify/verify_test.go`
    (`goldenRoot = "vdHF/1WxnLaw58dhv5psyqJ/u/wHt08fq7bpEaC9KrM="`, the two `goldenProof` hashes, record
    `"leaf-1"`, index 1, size 4 → `verified=true`). Cases: positive; wrong record → `verified=false,
    errMsg==""` (negative verdict, NOT an error); tampered root → same; malformed base64 → `errMsg!=""`;
    `index>=size` → `errMsg!=""`.
- **Modify**: (none — all new files)
- **Reference**:
  - `.claude/context/learnings/proof-verify.md` — the three-way verdict contract, the arg-order gotcha
    (already hidden inside `VerifyInclusion`), the purity rule, and the golden 4-leaf vector to reuse.
  - `internal/proof/verify/verify.go` + `internal/proof/verify/verify_test.go` — the function being
    exported and the exact golden literals (`goldenRoot`, `goldenProof`) to copy into the adapter test.
  - `.claude/adr/0003-client-verification-and-in-browser-verifier.md` — WASM is a progressive enhancement
    on the shared `internal/proof/verify`; one verifier codebase to native + WASM; reproducible build /
    published-hash / SRI are the LATER sub-steps (Not In Scope here).
  - `.claude/design/ISCC Monitor - Independent Verification.dc.html` — Surface C, the eventual consumer of
    this export (skim only; not built here).

## Not In Scope
- **No in-browser app, no HTML, no DS shell, no `?monitor=<url>` fetch.** Surface C
  (`monitor.iscc.codes` Independent Verification) is a later sub-step.
- **No tier-2 progressive enhancement** on the certificate/dossier (no JS `<script>`, no `wasm_exec.js`
  embedding, no `internal/web` change). That is the next sub-step after this entrypoint exists.
- **No reproducible-build / published-hash / SRI pin** and **no `.wasm` artifact committed** to the repo
  or served by the monitor. Those land with the hosting sub-step.
- **No `mise.toml` task for the WASM build.** Keep the WASM compile a documented manual gate this step;
  a `mise run build:wasm` task can be added when the artifact is actually served.
- **No `VerifyConsistency` export** — the WASM tier-2 result is inclusion-only; a consistency wrapper
  waits for a caller (and needs its own arg-order wrapper per the learning).
- **Do not add a build tag to `verify_adapter.go`** — keeping the marshaling logic untagged is what makes
  it linux-testable; only `main.go` (which imports `syscall/js`) carries `//go:build js && wasm`.
- The standing `normal` hardening defects (§5 digest-binding, `host:port` DID `%3A`-encode, `safeStamp`,
  `hubDomain` ForceQuery, §6 timestamp) — none are on the files this step touches; do not fold them in.

## Implementation Notes
- **Split the testable logic from the untestable glue.** `syscall/js` only compiles under
  `GOOS=js GOARCH=wasm`, and js/wasm tests need a browser/node harness this loop does not have. So put ALL
  the marshaling (base64 decode, verdict folding) in the untagged `verify_adapter.go` `verifyJSON`, and
  keep `main.go` a small `js.FuncOf` shim that pulls args off the `[]js.Value` (`.String()`, `.Int()`, and
  ranging a JS array of strings) and calls `verifyJSON`. The shim is exercised only by the WASM compile
  gate; the logic is exercised by the linux golden test.
- **Verdict mapping (preserve the three-way contract — proof-verify learning).** `verify.VerifyInclusion`
  returns `(true,nil)` verified / `(false,nil)` genuine negative / `(false,err)` precondition error. Map:
  a base64 decode error OR a non-nil verify error → `verified=false` + non-empty `errMsg`; otherwise
  return the boolean with `errMsg==""`. **A wrong record / tampered root must surface as `verified=false,
  errMsg==""` — a negative VERDICT, not an error** — or the eventual split-view alert can't distinguish
  "mismatch" from "broken input". This mirrors how the server call sites fold the verdict (both discard
  the `, _` err and route the boolean).
- **Inputs are base64-Std** (always-loaded + proof-verify learning): the monitor emits root, record, and
  every proof hash base64-Std (`internal/proofserve` + `internal/certificate` use `base64.StdEncoding`
  throughout). The adapter base64-Std-decodes each before calling the core — do NOT change the core's
  `[]byte` signature.
- **Reuse the golden vector verbatim.** Copy the `goldenRoot` / `goldenProof` literals + `leaf-1` / index
  1 / size 4 from `internal/proof/verify/verify_test.go` so the adapter test proves WASM-side parity
  against the SAME vector the core test pins — that IS the "identical vectors → identical verdicts" check
  at the marshaling boundary.
- **Keep the adapter pure** (always-loaded rule + ADR-0003): `verify_adapter.go` imports only
  `encoding/base64` + the `verify` package (+ `fmt` if a wrapped errMsg is built). No `net`/`os`/
  `syscall/js`. The whole point is that the SAME core+marshaling runs identically on server and WASM.
- **Verified build-tag behavior (scoping):** a `//go:build js && wasm` file whose import (`syscall/js`) is
  unavailable on linux is silently skipped by `go build ./...`, `go vet ./...`, and `go test ./...` on
  linux when the module has other packages (reproduced in a scratch module: all three exit 0, `cmd/wasm`'s
  untagged adapter + test still compile and run). So `mise run check` stays green; `cmd/wasm/main.go` is
  compiled only by the explicit `GOOS=js GOARCH=wasm` gate. Use the canonical `js && wasm` constraint
  (matches `wasm_exec.js` shipped at `$(go env GOROOT)/lib/wasm/wasm_exec.js`).
- **`js.FuncOf` signature:** `func(this js.Value, args []js.Value) any`. Return a JS object literal via
  `map[string]any{"verified": v, "error": msg}` (a `js.ValueOf`-able map), keeping the shim trivial. Guard
  `len(args)` defensively and treat a wrong arg count as an error result, not a panic.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` pass and `gofmt -l .` excl.
  `cauldron/` is empty) — the tagged `cmd/wasm/main.go` is excluded on linux, so the linux gate is
  unaffected.
- `GOOS=js GOARCH=wasm go build -o /tmp/iscc-verify.wasm ./cmd/wasm` exits 0 (the WASM entrypoint
  compiles, importing `syscall/js` + `internal/proof/verify`).
- `go vet ./cmd/wasm` on linux does NOT error on the `syscall/js` import (assert: the linux build of the
  package never tries to compile `main.go` — only the untagged adapter + test are built).
- `go test -count=1 -run TestVerifyJSON ./cmd/wasm` passes: positive 4-leaf vector → `verified=true,
  errMsg==""`; wrong record + tampered root → `verified=false, errMsg==""`; malformed base64 and
  `index>=size` → `errMsg!=""`.
- Assertion: `verifyJSON("leaf-1", "vdHF/1WxnLaw58dhv5psyqJ/u/wHt08fq7bpEaC9KrM=", goldenProof, 1, 4)`
  returns `verified=true, errMsg==""` (the same verdict the server core returns for the same vector —
  WASM-vs-server parity at the marshaling boundary).
- `go mod tidy -diff` clean (no new prod dependency — `syscall/js` is stdlib; the adapter reuses the
  existing `internal/proof/verify`).

## Done When
`mise run check` is green, `GOOS=js GOARCH=wasm go build ./cmd/wasm` succeeds, and
`go test -run TestVerifyJSON ./cmd/wasm` proves `verifyJSON` returns the same three-way verdict as the
server core for the shared golden vector (positive, both negatives, and both error cases).
