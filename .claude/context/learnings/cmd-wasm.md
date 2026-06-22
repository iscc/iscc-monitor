<!-- area: cmd/wasm (+ cmd/wasm/verifyadapter) — the GOOS=js WASM verifier entrypoint -->
<!-- indexed-as: cmd-wasm.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# `cmd/wasm` — the WASM verifier entrypoint

Read this when a step touches the WASM verifier (the `GOOS=js GOARCH=wasm` entrypoint or its
marshaling adapter). Durable cross-cutting rules live in the index
(`.claude/context/learnings.md`); the package-local mechanics are here.

## Layout: split the tagged glue from the untagged adapter (and WHY the obvious layout breaks the gate)

- **An untagged `.go` file inside `package main` whose only `func main()` sits in a `//go:build js &&
  wasm` file BREAKS `go build ./...` on linux** with `runtime.main_main·f: function main is undeclared
  in the main package`. The tag drops `main.go` on linux, leaving a buildable `package main` command
  with no `main` — the linker fails. (Reviewer-reproduced in a scratch module AND this repo; the claim
  in an earlier next.md that `go build ./...` "silently skips" such a package is WRONG — only a package
  with *no* Go files at all for the platform is skipped.) `go vet`/`go test` happen to pass; `go build
  ./...` (part of `mise run check`) does not.
- **The working layout:** keep `cmd/wasm` holding ONLY the `//go:build js && wasm` `main.go`, and put
  the pure marshaling in a separate NON-main subpackage `cmd/wasm/verifyadapter`. A library package
  always builds on linux (so its golden test runs the parity check), and `cmd/wasm` now genuinely has
  *no* Go files for linux → `go build ./...` correctly skips it. The WASM gate (`GOOS=js GOARCH=wasm go
  build ./cmd/wasm`) compiles `main.go` + the imported adapter together.
- Because the adapter crosses a package boundary, its exported fn is `VerifyJSON` (capitalized), not the
  same-package lowercase a single-package layout would use.
- **Linux gate nuances (not failures):** `go vet ./cmd/wasm` (naming the package explicitly) exits 1
  with `build constraints exclude all Go files` — that is the EXPECTED response to targeting a
  platform-empty package, NOT a `syscall/js` compile error. The gate uses `go vet ./...` (wildcard),
  which skips it; `mise run check` stays green. The adapter test must be run via
  `./cmd/wasm/verifyadapter` (not `./cmd/wasm`).

## The marshaling adapter (`verifyadapter.VerifyJSON`)

- **Stays WASM-pure:** imports only `encoding/base64` + `fmt` + `internal/proof/verify`. The
  load-bearing proof is `GOOS=js GOARCH=wasm go build ./cmd/wasm/verifyadapter` succeeding, NOT grepping
  the dep list (`os` appears transitively via `fmt`). NO `net`/`os`/`syscall/js` here — the whole point
  is the SAME core+marshaling runs identically on server and WASM.
- **Preserves the proof/verify three-way verdict contract at the marshaling boundary** (see
  `proof-verify.md`): a base64-decode error OR a non-nil verify error (the `index >= size` precondition)
  → `verified=false, errMsg!=""`; otherwise the core boolean with `errMsg==""`. A wrong record / tampered
  root MUST be `verified=false, errMsg==""` — a negative VERDICT, not an error — or the eventual
  split-view alert can't tell "mismatch" from "broken input". Mutation-proven non-vacuous (review):
  forcing `verified=true` fails wrong-record/tampered-root; collapsing the error channel fails
  `index >= size`.
- **Inputs are base64-Std** (the monitor emits root/record/proof-hashes base64-Std throughout
  `internal/proofserve`/`internal/certificate`); the adapter decodes each before calling the core. Do
  NOT change the core's `[]byte` signature. Reuse the verbatim 4-leaf golden vector from
  `internal/proof/verify/verify_test.go` so the adapter test IS the "identical vectors → identical
  verdicts, WASM vs server" check.

## The `syscall/js` shim (`main.go`) — JS-call-boundary gotchas

- **`js.Value.Int()` TRUNCATES** — it is `int(v.Float())`, so a JS Number `1.9` becomes `1` and a
  proof bundle with a non-integer `index`/`size` would verify against the wrong-but-truncated leaf.
  Values beyond JS's 2^53 safe-integer range round similarly. CLOSED in code: the shim now reads each
  via `args[i].Float()` and routes it through `safeIndex(v, name) (uint64, string)`, failing closed
  (NaN/Inf/fractional/negative/`>= 2^53` → `{verified:false, error:…}`) BEFORE the `uint64` narrowing.
  `maxSafeInteger = 2^53 - 1`; the guard rejects `> maxSafeInteger` (i.e. `>= 2^53`), slightly stricter
  than the issue's "`> 2^53`" — fine, well outside any real leaf-index domain.
- **`safeIndex` is pure `float64 → (uint64, string)` with NO `syscall/js` dep — but it lives in
  `main.go` (`//go:build js && wasm`), so it is NOT unit-tested on linux** and the build gate only
  proves it compiles, not its branch behavior. The certificate caller's test feeds only valid
  integers, so the truncation/NaN/range branches have ZERO executable coverage. Forward rule: when the
  guard is next touched, MOVE `safeIndex`+`maxSafeInteger` into the untagged `verifyadapter` (or a new
  untagged helper) and table-test it (`1.9`/`NaN`/`-1`/`2^53` → error; `0`/`5` → ok) so the JS→Go
  integer contract is regression-gated. Filed `normal` (open issue).
- The shim guards `len(args) != 5` → an error result (never a panic) and returns a `map[string]any`
  JS object `{verified, error}`; keep that defensive arg-count guard.

## The first SSR caller (certificate tier-2) — progressive-enhancement loader

- **The certificate (`internal/certificate/cert.html`) is the first real SSR caller.** Under
  `{{if .HasBundle}}` it embeds a `<script type="application/json">` data island
  (`record`/`root`/`proof[]`/`index`/`size`) — `html/template` JSON-context-escapes it, so base64
  `+`/`/` survive (do NOT string-interpolate into executable JS). An end-of-body `<script
  src="/_ds/wasm_exec.js">` + inline loader does `instantiateStreaming(fetch("/_ds/verify.wasm"))`
  with an `arrayBuffer()` fallback, `go.run`, then calls `globalThis.isccVerifyInclusion(record, root,
  proof, index, size)`. Verified live end-to-end on the testnet (`ISCC:MAIGKSETI7MJ4EAB`): the WASM
  ran headlessly and rendered the `verified` verdict matching server §3.
- **Keep the three render states distinct** for the deferred split-view alert: `error` (broken input)
  vs `failed` (negative VERDICT — proof did not rebuild the root, the alert's key) vs `verified`. The
  data island + loader render ONLY under `HasBundle`, so an uncertifiable id wires no verifier (the
  negative test asserts none of the markers appear). Do not collapse `error`/`failed`.
