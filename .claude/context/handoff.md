## 2026-06-22 — Serve `wasm_exec.js` from the `/_ds/` static-asset leaf (WASM loader skeleton)

**Done:** Added the Go 1.26.1 WASM runtime loader (`wasm_exec.js`) as a byte-verbatim, build-pinned
embedded asset served at the stable, CDN-free path `/_ds/wasm_exec.js` through the existing
`internal/web` `/_ds/` subtree handler (one new `case`, reusing the unchanged no-cache + strong-ETag +
304 + outer-CORS policy). This is the same-origin runtime on-ramp the later tier-2 `<script>`
enhancement loads before instantiating the verifier `.wasm`.

**Files changed:**
- `internal/web/wasm_exec.js`: new — `cp "$(go env GOROOT)/lib/wasm/wasm_exec.js"`, byte-verbatim
  (16992 bytes, `cmp`-identical to the toolchain copy; BSD-licensed, version-pinned like the woff2s).
- `internal/web/web.go`: added `WasmExecPath = "/_ds/wasm_exec.js"`, `contentTypeJS =
  "text/javascript; charset=utf-8"`, a `//go:embed wasm_exec.js var wasmExecJS []byte`, and a
  `case WasmExecPath: writeAsset(w, r, wasmExecJS, contentTypeJS)` before the `default: serveFont`
  fall-through; updated the package + Handler docstrings (evergreen, current-state). No new import.
- `internal/web/web_test.go`: added `TestWasmExecServed` (200 / `text/javascript; charset=utf-8` /
  no-cache / strong non-`W/` ETag / non-empty body carrying `globalThis.Go` / CDN-free); extended
  `TestIfNoneMatch304` and `TestMethodNotAllowed` path lists with `WasmExecPath`; refined the shared
  `noExternalCDN` helper to strip `//` line comments before scanning (see Notes — required so the
  byte-verbatim loader's one Go-issue-tracker comment URL does not false-positive).

**Verification:** `mise run check` → green (`go build ./...`, `go vet ./...`, all 25 packages `ok`,
`gofmt -l .` excl. `cauldron/` empty). Per-criterion:
- [x] `go test -count=1 -run TestWasmExec ./internal/web` → PASS.
- [x] `GET /_ds/wasm_exec.js` → 200, `Content-Type: text/javascript; charset=utf-8`, strong quoted-hex
  ETag (no `W/`), `Cache-Control: no-cache`, non-empty body with `globalThis.Go`.
- [x] `If-None-Match` echo → 304, empty body, same ETag (extended `TestIfNoneMatch304`).
- [x] `noExternalCDN(t, "served wasm_exec.js", …)` → does not fail (no third-party *loadable* origin).
- [x] non-GET → 405 (extended `TestMethodNotAllowed`).
- [x] `wasm_exec.js` is `cmp`-identical to `$(go env GOROOT)/lib/wasm/wasm_exec.js` (byte-verbatim).
- [x] `GOOS=js GOARCH=wasm go build ./internal/web` → OK (still WASM-shareable; embed added no import).
- Oracle/conformance gate: N/A — pure static-asset transport (no signature/RFC-6962/Merkle/did:web/
  fsck/proof path), the same N/A `internal/web` already carries.

**Next:** The runtime loader is now fetchable same-origin. The next sub-step is building and serving
the verifier `.wasm` artifact: a `mise run build:wasm` task (`GOOS=js GOARCH=wasm go build ./cmd/wasm`),
committing/embedding it under `/_ds/`, plus its ADR-0003 reproducible-build / published-hash / SRI pin.
After that, the `cert.html` `<script>` loader (load `wasm_exec.js` → instantiate the `.wasm` → call
`isccVerifyInclusion` with the base64-Std fields the surface emits) — which is where the open `normal`
`js.Value.Int()` truncation hardening belongs, per the prior review and the issue's own note (the page
does NOT yet emit the leaf record bytes the 5-arg call needs; emit them there).

**Notes:**
- **`noExternalCDN` helper refinement (decision, needs review eyes).** The byte-verbatim Go 1.26.1
  `wasm_exec.js` contains exactly one banned substring — `https://github.com/golang/go/issues/28975` —
  inside a `//` line comment (verified: all `https?://`/`jsdelivr`/`cdn.` matches in the file are inside
  `//` comments; none in loadable code). next.md required BOTH "keep it byte-verbatim / do not hand-edit"
  AND "assert CDN-free via the existing `noExternalCDN`", which currently bans the bare `https://`
  substring — a direct conflict, and exactly the false-positive `learnings/web.md` already flagged
  ("scope to a non-self host"). I did NOT hand-edit the vendored JS and did NOT weaken the gate: I made
  `noExternalCDN` strip each line's `//` comment tail before scanning, since a comment is non-executable
  and can never trigger a runtime CDN fetch. The strip treats `//` as a comment ONLY when not preceded by
  `:` (so a real loadable `https://cdn…` URL — whose `//` follows the scheme `:` — is never truncated and
  the ban stays fully strict for executable content). Mutation-probed in a scratch test (since removed,
  tree clean): the helper still fires on a real `https://cdn.jsdelivr.net/…` src, a bare `cdn.` host, and
  a real `https://` URL on a line that also carries a trailing `//` comment; it suppresses only pure
  comment URLs. tokens.css/fonts.css are unaffected (they have no comment-borne banned substrings; CSS
  uses `/* */` blocks, not `//`). This is the principled "scope the ban" fix web.md prescribed, applied to
  the shared test helper rather than forking a second helper. Flagging it so review can confirm the helper
  change preserves the genuine CDN-free invariant (no third-party *loadable* origin) rather than relaxes it.
- Scope discipline: 1 non-test/doc file modified (`web.go`) + 1 new asset (`wasm_exec.js`) + the test
  file — within the cap. Nothing in `## Not In Scope` touched: no `.wasm` artifact, no `mise run build:wasm`,
  no `cert.html` `<script>` / record-bytes emission, no `cmd/wasm/main.go` `js.Value.Int()` hardening, no
  `cmd/iscc-monitor` mount change (the `/_ds/` subtree already routes the new path to `web.Handler`).
- The `globalThis.Go` body assertion is grounded in the real file (`wasm_exec.js:103 globalThis.Go =
  class {`), so it is non-vacuous against the served bytes.
- Standing OTS / registry / certificate / the `cmd/wasm` `js.Value.Int()` `normal`/`low` issues remain
  open and untouched (correctly out of scope for this static-asset step).
