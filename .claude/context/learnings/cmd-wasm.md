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
- **settled: the integer guard now lives in the untagged adapter as `verifyadapter.SafeIndex` and is
  linux table-tested** (`TestSafeIndex`, mutation-proven non-vacuous). It moved out of the tagged
  `main.go` (which now only call-sites it), closing the "trapped behind the build tag, ZERO executable
  coverage" issue. **Non-vacuity trap to keep:** a boolean `wantErr`-only table is VACUOUS for the
  NaN/Inf branch — `math.NaN() != math.Trunc(NaN)` is true (the fractional check also catches NaN) and
  `+Inf > maxSafeInteger` is true (the range check also catches Inf), so dropping the finite-number
  branch leaves a `wantErr`-only test green. The test must pin the specific `"not a finite number"`
  message (`wantMsg` substring) on the NaN/Inf cases to gate that branch; do not relax this when the
  table is next touched. The guard rejects `>= 2^53` (i.e. `> maxSafeInteger = 2^53-1`), slightly
  stricter than the issue's "`> 2^53`" framing — preserve verbatim, it is well outside any leaf domain.
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

- **SCOPE GAP (id-binding half CLOSED in source, signature half still open): `isccVerifyInclusion`
  proves inclusion math + (with a 6th `id` arg) id-binding, but NOT the checkpoint signature.** The
  shim now accepts 5 OR 6 args: a 6-arg call additionally gates `verified` on
  `verifyadapter.RecordCommitsID(record, id)` (canonical `"ISCC:"+TrimPrefix` byte-compare of the
  record's committed `iscc_id` vs the requested id), so a different-declaration bundle yields a negative
  verdict, not a false green. Still OPEN (filed `normal`): the did:web-key + checkpoint-note signature
  half — a `verified` cross-origin still trusts the monitor for the signature. A full re-verification is
  signature + did:web-key + id-binding + inclusion; only the last two run. Until the signature lands,
  do NOT loosen `verifier.html`'s "hub-signed root" success copy to claim a check it skips.
- **The arg-count guard is intentionally `5 OR 6`, NOT a hard `!= 6` (design deviation from a `5→6`
  bump).** The SAME committed `/_ds/verify.wasm` is shared by the cross-origin verifier (6-arg, with
  id-binding) AND the same-origin certificate (`cert.html:565`, still 5-arg). A hard `!= 6` would
  regress the certificate's live tier-2 verifier to an `error` on every certifiable id with no in-scope
  fix (the cert test only asserts markup, not WASM execution — it would NOT catch the runtime break).
  The optional 6th arg honors both. To go strict `!= 6` later, you MUST pair it with a `cert.html`
  edit (pass a 6th id from its data island) in the SAME increment.
- **TRAP — the committed `verify.wasm` is a PINNED BYTE ARTIFACT, not auto-rebuilt from source.** A
  `cmd/wasm`/`verifyadapter` source change does NOT update `internal/web/verify.wasm`; `mise run check`
  stays green because `GOOS=js GOARCH=wasm go build ./cmd/wasm` builds to `/tmp`, and
  `TestWasmVerifyHashPinned` only checks the COMMITTED bytes match `WasmVerifyHash` (both stale → still
  green). So a shim change that lands the source but not the rebuilt artifact ships a self-contradiction:
  `verifier.html` passing the new 6th arg against an old `expected 5 args` wasm makes EVERY live
  Surface-C call render `error`. ANY change to `cmd/wasm` or `cmd/wasm/verifyadapter` that alters runtime
  behavior MUST be followed by `mise run build:wasm` + re-pin `WasmVerifyHash` in the SAME increment —
  verify with `strings internal/web/verify.wasm | grep -c "<a new message you added>"` (must be 1), not
  just the source build exiting 0.
- **TRAP — `mise run build:wasm` and a bare `go build` use DIFFERENT Go toolchains, so they emit
  DIFFERENT artifact bytes (the wasm embeds the toolchain version string).** The two artifacts are
  behaviorally IDENTICAL (same shim); only the embedded version stamp differs, so each is independently
  deterministic — a re-pinned hash that differs from a bare build is NOT nondeterminism, it is the other
  toolchain's output. ALWAYS rebuild + pin via `mise run build:wasm` (NOT a bare `go build` on PATH),
  because mise is the gate runner and the only canonical source for the artifact bytes. Verify the pin is
  reproducible-from-command with `mise run build:wasm && git status` showing `verify.wasm` clean (NOT just
  `mise run check` green — that only checks committed-bytes-vs-const, never reproducibility from the
  documented command).
- **settled (`dc657f8`): `mise.toml` pins the EXACT patch `go = "1.26.4"` (not the floating minor
  `1.26`), and `WasmVerifyHash = 2c91e61f…` is that toolchain's deterministic output.** Reviewer-confirmed
  reproducible: two consecutive `mise run build:wasm` both emit `2c91e61f…` and leave the tree clean; the
  artifact embeds `go1.26.4`; `TestWasmVerifyHashPinned` green. Keep the patch pinned to an EXACT version
  when bumping — a floating minor reintroduces the divergence (whatever patch mise happens to have
  installed wins, drifting the pin off the documented build). On a Go bump: change `mise.toml` to the new
  exact patch, `mise run build:wasm`, re-pin `WasmVerifyHash` to the new SHA, in ONE increment.
