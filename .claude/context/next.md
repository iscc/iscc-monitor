# Next Work Package

## Step: Build the verifier `.wasm` reproducibly (`mise run build:wasm`) and serve it byte-pinned at `/_ds/verify.wasm`

## Advances
WASM verifier milestone (`target.md` lines 189-197), the artifact-hash half of its Verify:

> reproducible build + published hash + SRI pin (ADR-0003, ADR-0010). **Verify:** identical vectors
> yield identical verdicts (WASM vs server); the verifier artifact hash matches the published value;
> a `(size, root)` mismatch renders the guided split-view alert, not a dead error.

This is the **skeleton-first** sub-step the `review` handoff and `state.md` "Next Milestone (1)" both
name: "build + serve the verifier `.wasm` (a `mise run build:wasm` task + ADR-0003 reproducible-build /
published-hash / SRI pin)". State confirms the gap by direct probe — "no `.wasm` artifact built or
committed, no `mise run build:wasm` task". The remaining halves (the `<script>` caller, the standalone
app, the split-view alert) are listed under `## Not In Scope` as later sub-steps so the WASM arc
continues coherently rather than switching to an unrelated refactor.

## Goal
Produce the verifier WebAssembly artifact with a **deterministic** build (`-trimpath -ldflags=-buildid=`
→ byte-identical across a clean cache, verified below) wired as `mise run build:wasm`, commit it, pin its
SHA-256 as the **published hash**, and serve it byte-verbatim at `/_ds/verify.wasm` over the existing
`internal/web` static leaf. This lays the build-pinned, hash-published artifact the tier-2 `<script>`
loader instantiates next, closing the "verifier artifact hash matches the published value" Verify clause.

## Scope
- **Create**: `internal/web/verify.wasm` — the committed reproducible build output (a generated asset,
  not hand-written source; produced by the new `mise run build:wasm` task: `GOOS=js GOARCH=wasm
  CGO_ENABLED=0 go build -trimpath -ldflags=-buildid= -o internal/web/verify.wasm ./cmd/wasm`).
- **Modify** (2 non-test/doc files):
  - `mise.toml` — add a `build:wasm` task running the reproducible build command above (so the artifact
    is regenerable and CI/a human can rebuild-and-compare). Mind the `&&`-is-portable convention already
    noted in the `check` task.
  - `internal/web/web.go` — add `WasmVerifyPath = "/_ds/verify.wasm"` const, a `WasmVerifyHash` const
    (the published lowercase-hex SHA-256 of the committed bytes), a `//go:embed verify.wasm` `var`, a
    `contentTypeWASM = "application/wasm"` const, and a `case WasmVerifyPath:` in `Handler()` — mirroring
    the `wasm_exec.js` wiring already there (consts 65-69, embed 112-118, case 143-144).
- **Modify (test/doc, not counted against the 3-file budget)**:
  - `internal/web/web_test.go` — add `TestWasmVerifyServed` + a hash-pin test, and extend the path lists
    in `TestIfNoneMatch304` / `TestMethodNotAllowed` to include `WasmVerifyPath` (see Verification).
  - `CLAUDE.md` — add the `GET /_ds/verify.wasm` route to the endpoint list beside the existing
    `/_ds/wasm_exec.js` mention (the verifier runtime entry), keeping docs in sync with the new surface.
- **Reference**:
  - `.claude/context/learnings/web.md` — the `/_ds/` subtree mount, the `no-cache`+strong-ETag+304
    `writeAsset` shape, the "byte-verbatim, never hand-edit" wasm_exec posture, the pure-stdlib-leaf rule.
  - `.claude/context/learnings/cmd-wasm.md` — why `cmd/wasm` has no linux Go files (so `go build ./...`
    skips it) and the `GOOS=js GOARCH=wasm go build ./cmd/wasm` gate that compiles it.
  - `internal/web/web.go` lines 65-69, 112-149, 169-194 — the exact const/embed/case/`writeAsset` idiom
    to mirror for `verify.wasm`.
  - `internal/web/web_test.go` `TestWasmExecServed` (225), `TestIfNoneMatch304` (259),
    `TestMethodNotAllowed` (278) — the test shapes to mirror/extend for the new path.
  - `cmd/wasm/main.go` — the entrypoint the artifact is built from (registers `isccVerifyInclusion`).

## Not In Scope
- The certificate/dossier `<script>` loader that `fetch`/`instantiateStreaming`s the `.wasm` and calls
  `isccVerifyInclusion` — the NEXT sub-step (the first real caller). Do not edit `cert.html` /
  `internal/certificate` / `internal/dossier` this step.
- The `js.Value.Int()` safe-integer hardening (open `normal` issue) — it belongs with the first real
  caller (no caller exists yet; `cmd/wasm/main.go` and the adapter are untouched here).
- The standalone `monitor.iscc.codes` Independent Verification app (Surface C) — a later WASM sub-step.
- The `(size, root)` mismatch guided split-view alert UI — lands with the caller, not the artifact.
- An HTML `integrity=` SRI attribute — `.wasm` is `fetch`ed, not `<script src>`-loaded; the integrity pin
  here is the committed-hash const + the pin test. Header/subresource SRI for the loader arrives with the
  caller step.
- Adding the wasm build to `mise run check` / CI as a rebuild-and-compare gate — keep `check` unchanged
  (it must stay fast + linux-only); the pin test guards the committed bytes against the const.

## Implementation Notes
- **Reproducible build is verified.** `GOOS=js GOARCH=wasm CGO_ENABLED=0 go build -trimpath
  -ldflags=-buildid= -o internal/web/verify.wasm ./cmd/wasm` yields a byte-identical artifact across a
  `go clean -cache` (confirmed this iteration: SHA-256 `6c29eef9706a43e7db67de7e0eca3752b3367a46e37f9bfb3ff1bb6d944cd3f8`,
  2,891,616 bytes). Plain `go build` (no `-trimpath`/`-buildid=`) embeds absolute GOROOT/module paths and
  is NOT reproducible — the flag set is load-bearing for "artifact hash matches the published value". Use
  the exact flags in the `mise run build:wasm` task. Record in `WasmVerifyHash` whatever the task actually
  emits (it changes if the toolchain or source changes).
- **Mirror the `wasm_exec.js` asset wiring verbatim** in `web.go`: the stable `/_ds/verify.wasm` const, a
  `//go:embed verify.wasm var wasmVerify []byte`, `contentTypeWASM = "application/wasm"` (the IANA media
  type browsers require for `WebAssembly.instantiateStreaming`; set it explicitly so a sniffer can't
  downgrade it), and a `case WasmVerifyPath: writeAsset(w, r, wasmVerify, contentTypeWASM)` in
  `Handler()`. `writeAsset` already gives no-cache + strong content-ETag + 304 + GET-only/405 + the
  404-default — reuse it unchanged. Do NOT add a competing `/_ds/...` mux pattern (web.md: extend the
  in-handler switch, never add a second pattern).
- **Published-hash discipline (the "a build output that silently rots vs source" guard).** Add
  `WasmVerifyHash` as a Go const holding the lowercase-hex SHA-256 of the committed `verify.wasm`, and a
  test asserting `fmt.Sprintf("%x", sha256.Sum256(wasmVerify)) == WasmVerifyHash`. This const IS the
  *published hash* the SRI/verify-artifact criterion names, and the test is the regression guard:
  re-`mise run build:wasm` without re-pinning the const fails the test. The reproducibility-vs-source
  rebuild (`mise run build:wasm` → diff the committed file) is a human/CI step, NOT a unit test (the linux
  `go test` gate cannot cross-compile inside a test).
- **`noExternalCDN` is N/A for a binary asset** — it scans text bodies (CSS/JS) for third-party origins;
  do not run it on the `.wasm` bytes (web.md: the ban targets loadable text content). The existing
  `TestWasmExecServed` already covers the runtime loader's CDN-freeness.
- **`internal/web` stays a pure stdlib leaf** (web.md "Pure stdlib leaf, WASM-green"): embedding a
  `[]byte` adds no import (`embed` is already in use). go.mod/go.sum stay byte-identical (no new dep).
- **`cmd/wasm` stays untouched** — the artifact is built from the existing `main.go`; this step does not
  edit the entrypoint or the adapter (the "identical vectors → identical verdicts" parity test in
  `cmd/wasm/verifyadapter` already covers the WASM-vs-server verdict, per `state.md` / `cmd-wasm.md`).
- **Relevant learnings rule:** web.md — "`wasm_exec.js` is served byte-verbatim … never hand-edit it;
  re-`cp` on a toolchain bump." The same posture applies to `verify.wasm`: a build-pinned generated
  asset, regenerated by `mise run build:wasm` on a source/toolchain change, never hand-edited, with its
  hash re-pinned in the const.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all pass; `gofmt -l .`
  empty).
- `mise run build:wasm` exits 0 and writes `internal/web/verify.wasm`; re-running it after
  `go clean -cache` produces a byte-identical file (same `sha256sum`) — reproducibility holds.
- `go test -count=1 -run TestWasmVerify ./internal/web` passes:
  - `GET /_ds/verify.wasm` → `200`, `Content-Type: application/wasm`, `Cache-Control: no-cache`, a
    quoted-hex strong `ETag`, and a non-empty body equal to the embedded bytes.
  - the hash-pin assertion `fmt.Sprintf("%x", sha256.Sum256(wasmVerify)) == WasmVerifyHash` holds
    (mutation: flipping one byte of the const fails this test).
- `go test -count=1 -run 'TestIfNoneMatch304|TestMethodNotAllowed' ./internal/web` passes with
  `WasmVerifyPath` added to both path lists (`If-None-Match` echoing the served ETag → `304`; a non-GET
  method → `405`).
- Assertion: the committed `internal/web/verify.wasm` is a valid Wasm module — its first four bytes are
  the `\0asm` magic `00 61 73 6d` (e.g. `head -c4 internal/web/verify.wasm | xxd`).

## Done When
`mise run build:wasm` deterministically builds `internal/web/verify.wasm`, the file is committed and its
SHA-256 is pinned in `WasmVerifyHash`, `GET /_ds/verify.wasm` serves it as `application/wasm` with the
revalidating-ETag / 304 / 405 policy, and `mise run check` + `go test -run TestWasmVerify ./internal/web`
both pass.
