## 2026-06-22 — WASM verifier entrypoint — `cmd/wasm` exporting `verify.VerifyInclusion` to JS

**Done:** Laid the WASM verifier entrypoint: a `//go:build js && wasm` `cmd/wasm/main.go` that wraps the
pure adapter in a `js.FuncOf` and exposes it as `globalThis.isccVerifyInclusion`, plus the pure,
linux-testable marshaling adapter (`VerifyJSON`) that base64-Std-decodes the monitor's proof-bundle
fields into the shared `internal/proof/verify`.`VerifyInclusion` args and folds the three-way verdict
into a flat `(verified, errMsg)` JS result. The adapter is proven WASM-vs-server verdict-parity against
the SAME 4-leaf golden vector the core test pins.

**Files changed:**
- `cmd/wasm/main.go` (new): `js && wasm`-tagged entrypoint; thin `syscall/js` shim pulling `(record,
  root, proofArray, index, size)` off `[]js.Value` and calling the adapter, returning
  `{verified, error}`; `select{}` keeps the runtime alive; defensive arg-count guard → error result.
- `cmd/wasm/verifyadapter/verify_adapter.go` (new): the untagged, **non-main** pure adapter package.
  `VerifyJSON(record, root string, proofB64 []string, index, size uint64) (verified bool, errMsg
  string)` — base64-Std-decodes inputs, calls the shared core, maps base64/precondition errors to
  `errMsg!=""` and a negative Merkle verdict to `verified=false, errMsg==""`. Imports only
  `encoding/base64` + `fmt` + `internal/proof/verify` (stays WASM-pure).
- `cmd/wasm/verifyadapter/verify_adapter_test.go` (new): table-driven golden test reusing the verbatim
  `goldenRoot`/`goldenProof` literals + `leaf-1`/index 1/size 4 from `internal/proof/verify`. Five
  cases: positive → `verified=true,errMsg==""`; wrong record + tampered root → `verified=false,
  errMsg==""`; malformed base64 + `index>=size` → `errMsg!=""`.

**Verification:** `mise run check` → **green** (`go build ./...`, `go vet ./...`, `go test ./...` all
exit 0; `gofmt -l .` excl. `cauldron/` empty; `go mod tidy -diff` clean — no new dep, `syscall/js` is
stdlib). Per-criterion:
- [x] `GOOS=js GOARCH=wasm go build -o /tmp/iscc-verify.wasm ./cmd/wasm` → exit 0 (2.9 MB artifact, not
  committed).
- [x] `go build ./...` / `go vet ./...` (the wildcard forms the gate runs) → exit 0; `cmd/wasm` is
  silently skipped on linux (no Go files for the platform after the tag excludes `main.go`).
- [x] `go test -count=1 -run TestVerifyJSON ./cmd/wasm/verifyadapter` → PASS, all 5 cases.
- [x] Golden assertion: `VerifyJSON("bGVhZi0x"=b64("leaf-1"), goldenRoot, goldenProof, 1, 4)` →
  `verified=true, errMsg==""` — same verdict as the server core for the same vector (parity at the
  marshaling boundary).
- [x] Mutation-proven non-vacuous: forcing the adapter to always return `verified=true` fails exactly
  `wrong_record` + `tampered_root`; reverted → green.

**Next:** The WASM side is now callable. The natural next sub-step is the **tier-2 progressive
enhancement** wiring it into the certificate/dossier (embed `wasm_exec.js` from `$(go env
GOROOT)/lib/wasm/wasm_exec.js`, a `<script>` that loads the `.wasm` and calls `isccVerifyInclusion`
with the base64-Std fields the surface already emits), followed by the standalone `monitor.iscc.codes`
Independent Verification app (Surface C) and the reproducible-build / published-hash / SRI pin +
`mise run build:wasm` task. A `VerifyConsistency` sibling export waits for a caller and needs its own
arg-order wrapper (proof-verify learning).

**Notes:**
- **DESIGN DEVIATION from next.md's exact file layout (flagged, not silent):** next.md's Scope placed
  the untagged `verify_adapter.go` **inside `package main` alongside the tagged `main.go`**. That does
  NOT work on linux: an untagged file in `package main` makes `cmd/wasm` a buildable command with no
  `main` function on linux, and **`go build ./...` fails to link** it (`function main is undeclared in
  the main package`). next.md's scoping note (lines 106-110) claimed `go build ./...` "silently skips"
  such a package — that is **incorrect**; I reproduced the link failure in both this repo and a scratch
  module. `go vet`/`go test` pass, but `go build ./...` (part of `mise run check`) does not. The
  idiomatic, in-scope fix: move the pure adapter into its own **non-main subpackage**
  `cmd/wasm/verifyadapter` (a library — always builds on linux) and leave `cmd/wasm` with ONLY the
  tagged `main.go`, so `go build ./...` correctly skips it as "no Go files for the platform" while the
  WASM gate compiles `main.go` + the imported adapter. All other next.md intent is preserved verbatim
  (untagged-and-linux-tested marshaling, exported `VerifyInclusion` to JS, same golden vector, same
  three-way verdict mapping). 3 new files, all within `cmd/wasm/`. **Action for review:** consider
  correcting the scoping note in the proof-verify / a cmd-wasm learning so the next WASM step does not
  re-trip the "untagged file in `package main` breaks `go build ./...`" assumption.
- The exported Go function symbol is `VerifyJSON` (capitalized) because it now crosses a package
  boundary (main.go imports verifyadapter); next.md wrote it lowercase `verifyJSON` assuming same-package.
- `go vet ./cmd/wasm` (explicitly naming the package) returns exit 1 "build constraints exclude all Go
  files" — this is NOT a `syscall/js` compile error; it is the expected response to explicitly targeting
  a package with no files for the host platform. The gate uses `go vet ./...` (wildcard), which skips
  it; `mise run check` is green.
- No `.wasm` artifact committed; no `mise` WASM task; no HTML/DS/`internal/web` change — all correctly
  deferred to later sub-steps per Not In Scope.
- Standing `normal`/`low` hardening defects untouched (none on these new files), per next.md.
