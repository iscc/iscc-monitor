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

## The `syscall/js` shim (`main.go`) — JS-call-boundary gotchas (open hardening, no caller yet)

- **`js.Value.Int()` TRUNCATES** — it is `int(v.Float())`, so a JS Number `1.9` becomes `1` and a
  proof bundle with a non-integer `index`/`size` can verify against the wrong-but-truncated leaf
  (Codex P2, reviewer-confirmed: `int(1.9)==1`). Values beyond JS's 2^53 safe-integer range round
  similarly. The shim does NOT yet validate integer/safe-integer inputs (open `normal` issue) — the
  natural place to harden is the sub-step that wires the FIRST real caller, where the JS→Go arg
  contract is exercised. Until then there is no caller, and the real callers (certificate/dossier,
  the standalone verifier app) emit server-computed integer indexes, so it is latent, not exploitable.
- The shim guards `len(args) != 5` → an error result (never a panic) and returns a `map[string]any`
  JS object `{verified, error}`; keep that defensive arg-count guard.
