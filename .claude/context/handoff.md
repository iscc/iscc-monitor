## 2026-06-22 — Build the verifier `.wasm` reproducibly (`mise run build:wasm`) and serve it byte-pinned at `/_ds/verify.wasm`

**Done:** Added a deterministic `mise run build:wasm` task (`GOOS=js GOARCH=wasm CGO_ENABLED=0 go build
-trimpath -ldflags=-buildid= -o internal/web/verify.wasm ./cmd/wasm`), committed its byte-identical
output (`internal/web/verify.wasm`), and wired it into the `/_ds/` static leaf as
`GET /_ds/verify.wasm` (`application/wasm`, the same no-cache + strong-ETag + 304 + GET-only policy as
`wasm_exec.js`). Pinned the published SHA-256 as `web.WasmVerifyHash` and added a regression-guard pin
test plus a served-asset test, closing the "verifier artifact hash matches the published value" half of
the WASM Verify clause.

**Files changed:**
- `mise.toml`: added `[tasks."build:wasm"]` (env `GOOS=js`/`GOARCH=wasm`/`CGO_ENABLED=0`, the reproducible
  `-trimpath -ldflags=-buildid=` build) so the artifact is regenerable; `check` left unchanged (fast,
  linux-only) per scope.
- `internal/web/web.go`: added `WasmVerifyPath = "/_ds/verify.wasm"`, `WasmVerifyHash` (the published
  lowercase-hex SHA-256), `contentTypeWASM = "application/wasm"`, `//go:embed verify.wasm var wasmVerify`,
  and a `case WasmVerifyPath: writeAsset(w, r, wasmVerify, contentTypeWASM)` in `Handler()` (mirroring the
  `wasm_exec.js` wiring; reused `writeAsset` unchanged); updated the package + `Handler` doc comments.
- `internal/web/verify.wasm`: NEW committed reproducible build output (2,870,559 bytes, magic `00 61 73 6d`,
  SHA-256 `17b0f4f81a0952c3bb8df1f85e300b90ea2d1f041a636d338cc00e38554445dc`).
- `internal/web/web_test.go` (test): added `TestWasmVerifyServed` (200 / `application/wasm` / no-cache /
  strong ETag / body == embedded bytes / `\0asm` magic) + `TestWasmVerifyHashPinned` (SHA-256 == const,
  mutation-proven); extended the `TestIfNoneMatch304` and `TestMethodNotAllowed` path lists with
  `WasmVerifyPath`; added `crypto/sha256` + `fmt` imports; updated the file doc.
- `CLAUDE.md` (doc): added the `GET /_ds/verify.wasm` route beside `/_ds/tokens.css` (it had no prior
  `/_ds/wasm_exec.js` entry to place it next to).

**Verification:** `mise run check` → green, all 25 packages `ok`; `gofmt -l .` empty (outside gitignored
`cauldron/`). Per criterion:
- [x] `mise run build:wasm` exits 0, writes `internal/web/verify.wasm`; re-run after `go clean -cache`
  byte-identical (cmp + same sha256) — reproducibility holds under the project toolchain.
- [x] `go test -run TestWasmVerify ./internal/web` → PASS: served route is 200 / `application/wasm` /
  `Cache-Control: no-cache` / quoted-hex strong ETag / non-empty body equal to the embedded bytes; hash-pin
  holds, and flipping one const byte FAILS it (mutation-proven, then restored byte-identical).
- [x] `go test -run 'TestIfNoneMatch304|TestMethodNotAllowed' ./internal/web` → PASS with `WasmVerifyPath`
  in both lists (If-None-Match echo → 304; POST → 405).
- [x] Assertion: committed `verify.wasm` first 4 bytes are the Wasm magic `00 61 73 6d` (`od -tx1 -N4`).
- [x] go.mod/go.sum byte-identical (no new dep); `cmd/wasm` untouched; WASM gate
  `GOOS=js GOARCH=wasm go build ./cmd/wasm` still compiles.

**Next:** The cert/dossier `<script>` loader — the first real caller that `fetch`/`instantiateStreaming`s
`/_ds/verify.wasm` (after `/_ds/wasm_exec.js`) and calls `isccVerifyInclusion`. That step should compare
the fetched bytes' hash against `web.WasmVerifyHash` (the SRI pin, since `.wasm` is `fetch`ed not
`<script src>`-loaded), emit the record/root/proof/index/size the adapter needs, and is the natural home
for the open `js.Value.Int()` safe-integer hardening (`cmd/wasm/main.go`) now that a caller exists. The
`(size, root)` mismatch guided split-view alert UI lands with it too.

**Notes:**
- **HASH IS TOOLCHAIN-DEPENDENT — pinned to the project toolchain (`mise run build:wasm`), as the task
  requires.** next.md predicted `6c29eef9…` / 2,891,616 bytes, but the committed artifact is
  `17b0f4f8…` / 2,870,559 bytes. Root cause: the build is reproducible *given a fixed Go toolchain*, and
  the toolchains differ. `mise run` uses mise's pinned `go = "1.26"` → **Go 1.26.4** (`GOROOT=…/go/1.26.4`),
  while a bare `go` in this shell resolves to a 1.24.13-path binary that *reports* 1.26.1 (toolchain
  auto-switch) and produces a third, different hash (`57973f3f…`). I pinned the hash to what
  `mise run build:wasm` emits — the documented, canonical task — and confirmed it is byte-identical across
  two `go clean -cache` cycles under mise. **CI/human rebuild-and-compare must run via `mise run build:wasm`
  (same pinned toolchain), not a bare `go build`**, or the bytes (and hash) will legitimately differ. If CI
  pins a different 1.26.x patch, the artifact + `WasmVerifyHash` must be regenerated there. The pin test
  guards the committed bytes against the const regardless of toolchain.
- `-trimpath -ldflags=-buildid=` is genuinely load-bearing: a plain `go build` (no flags) embeds absolute
  GOROOT/module paths and is not reproducible. The flag set is in the task verbatim.
- `noExternalCDN` is correctly NOT run on `verify.wasm` (binary, not loadable text) per web.md and next.md;
  `cmd/wasm` and the adapter are untouched (the WASM-vs-server parity test already lives in
  `cmd/wasm/verifyadapter`). Nothing from `## Not In Scope` was done.
- Scope: 3 non-test/doc files (`mise.toml`, `web.go`, the generated `verify.wasm` asset) + test/doc
  (`web_test.go`, `CLAUDE.md`). go.mod/go.sum stayed byte-identical (`embed` already imported); no new dep.
