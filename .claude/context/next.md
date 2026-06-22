# Next Work Package

## Step: Serve `wasm_exec.js` from the `/_ds/` static-asset leaf (WASM loader skeleton)

## Advances
WASM verifier upgrade milestone (target.md): "`internal/proof/verify` → `GOOS=js GOARCH=wasm`,
lazy-loaded progressive enhancement that elevates the Evidence Ledger's **tier-2** result … plus the
standalone Independent Verification verifier app … reproducible build + published hash + SRI pin."
**Verify:** "identical vectors yield identical verdicts (WASM vs server)…"

This is the first **skeleton sub-step** of the tier-2 progressive enhancement the handoff names as
`**Next:**` (review verdict at HEAD): "embed `wasm_exec.js` from `$(go env GOROOT)/lib/wasm/wasm_exec.js`,
a `<script>` that loads the `.wasm` and calls `isccVerifyInclusion`…". The full enhancement is far larger
than 3 files (it touches `internal/web` to serve both the loader script AND the `.wasm` artifact,
`internal/certificate` handler + `cert.html` to emit record bytes and the `<script>`, the `cmd/wasm`
shim hardening, and a `mise run build:wasm` task). Per the skeleton-first rule, this step lands the one
foundational, fully offline-provable piece: the Go runtime loader script every `<script>` enhancement
must load, served from the existing static-asset leaf at a stable `/_ds/` path.

## Goal
Make the Go WASM runtime glue (`wasm_exec.js`) fetchable at a stable, CDN-free `/_ds/` path so the later
tier-2 `<script>` loader has a same-origin runtime to load before instantiating the verifier `.wasm`.
This is the first real on-ramp from the callable `cmd/wasm` entrypoint toward an in-browser caller.

## Scope
- **Create**: `internal/web/wasm_exec.js` — a verbatim copy of
  `$(go env GOROOT)/lib/wasm/wasm_exec.js` (the Go 1.26.1 toolchain's WASM runtime loader; plain JS, Go
  BSD-licensed, a committed build-pinned asset exactly like the woff2 fonts already are).
- **Modify**: `internal/web/web.go` — add a `WasmExecPath` const (`/_ds/wasm_exec.js`), a `//go:embed
  wasm_exec.js` var, a `contentTypeJS` const (`text/javascript; charset=utf-8`), and one `case` in
  `Handler`'s path switch that `writeAsset`s it (reusing the existing no-cache + strong-ETag + 304 +
  CORS-via-outer-wrap policy unchanged). (1 non-test/doc file.)
- **Modify**: `internal/web/web_test.go` — add a golden test that `GET /_ds/wasm_exec.js` returns
  `200`, `Content-Type: text/javascript…`, a strong ETag, `Cache-Control: no-cache`, a non-empty body,
  AND is CDN-free via the existing `noExternalCDN` helper; extend the existing `TestIfNoneMatch304` and
  `TestMethodNotAllowed` path lists to cover the new path. (test file — does not count toward the cap.)
- **Reference**:
  - `.claude/context/learnings/web.md` — the `internal/web` `/_ds/` subtree mechanics: the in-handler
    path switch (do NOT add a competing mux pattern), the `writeAsset` no-cache+strong-ETag+304 shape,
    the `serveFont` traversal guard, and `noExternalCDN` (it bans only third-party origins; a
    same-origin path is fine).
  - `.claude/context/learnings/cmd-wasm.md` — the WASM entrypoint this loader will eventually drive
    (`globalThis.isccVerifyInclusion`, the 5-arg JS contract, the `js.Value.Int()` truncation gotcha
    that the LATER caller step fixes — not this one).
  - `internal/web/web.go` (TokensPath/FontsCSSPath/serveFont/writeAsset pattern to mirror exactly).
  - `internal/web/web_test.go` (`TestTokensServedAsCSS` / `TestFontBinaryServed` to model the new test
    on, and the `get`/`noExternalCDN` helpers + the `TestIfNoneMatch304`/`TestMethodNotAllowed` lists).

## Not In Scope
- The verifier `.wasm` artifact itself — building it, committing/embedding it, serving it, the
  `mise run build:wasm` task, and the reproducible-build/published-hash/SRI pin (the next sub-step;
  the `.wasm` is a large build output with an ADR-0003 reproducibility question of its own).
- The `<script>` loader in `cert.html` and emitting the subject record bytes (base64) into `certData`
  (a later sub-step — the page does NOT yet emit the leaf record bytes the 5-arg call needs).
- The `cmd/wasm/main.go` `js.Value.Int()` integer/safe-integer hardening (the open `normal` issue) —
  fix it at the FIRST real caller (the `<script>` step), where the JS→Go arg contract is exercised, per
  the issue's own "fix when the first real caller is wired" note and the handoff.
- The standalone `monitor.iscc.codes` Independent Verification app (Surface C).
- Touching `cmd/iscc-monitor/main.go`'s mount — it already routes the whole `web.Prefix` (`/_ds/`)
  subtree to `web.Handler`, so a new path inside the handler's switch needs no binary change.

## Implementation Notes
- **Copy, don't transform.** `cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" internal/web/wasm_exec.js`
  — keep it byte-verbatim so it stays a faithful, version-pinned copy of the 1.26.1 runtime (the same
  posture as the committed woff2 binaries). Do not hand-edit it.
- **Mirror the existing leaf exactly.** Add `WasmExecPath = "/_ds/wasm_exec.js"` next to `TokensPath`
  / `FontsCSSPath`; add `//go:embed wasm_exec.js` next to the other embeds (a `var wasmExecJS []byte`);
  add `contentTypeJS = "text/javascript; charset=utf-8"`; add
  `case WasmExecPath: writeAsset(w, r, wasmExecJS, contentTypeJS)` in `Handler`'s switch BEFORE the
  `default: serveFont(...)` fall-through. Reuse `writeAsset` verbatim — it already sets the strong
  content-ETag, `Cache-Control: no-cache`, and the `If-None-Match`→304 short-circuit (web.md: stable,
  overwrite-in-place `/_ds/` assets must NOT carry `immutable`; revalidate via the strong ETag).
- **Do NOT route it through `serveFont`** — that guard rejects non-`.woff2` names. The explicit `case`
  is the right placement, exactly as `tokens.css`/`fonts.css` are explicit cases above the font
  fall-through (web.md: extend the in-handler path switch, never add a second `/_ds/...` mux pattern).
- **Update `web.go`'s package docstring** lightly so it states the loader is now also served (evergreen
  comment describing current state) — and keep the "pure stdlib leaf" claim accurate (the new embed adds
  no import; `embed` is already imported).
- **CDN-free invariant (target.md M-UI hard constraint, inherited here).** The Go loader script is
  same-origin and references no third-party origin; assert it with the existing `noExternalCDN` helper
  (web.md: it bans `jsdelivr`/`cdn.`/cross-origin `http(s)://`, not same-origin paths). This keeps the
  no-CDN baseline true for the asset the tier-2 enhancement will load.
- **Oracle/conformance gate: N/A** — this is pure static-asset transport (no signature, RFC-6962,
  Merkle, did:web, fsck, or proof path), the same N/A `internal/web` already carries.
- This is the only un-started-to-DONE v1 milestone with offline-provable Verify criteria, and serving
  the runtime loader is its smallest forward, fully testable on linux with no WASM toolchain in the
  test path.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -count=1 -run TestWasmExec ./internal/web` passes.
- `GET /_ds/wasm_exec.js` returns `200` with `Content-Type: text/javascript; charset=utf-8`, a strong
  (non-`W/`) quoted-hex ETag, `Cache-Control: no-cache`, and a non-empty body (assert in the new test).
- A second request echoing that exact ETag in `If-None-Match` returns `304` with no body (extend the
  existing `TestIfNoneMatch304` path list to include `/_ds/wasm_exec.js`).
- `noExternalCDN(t, "served wasm_exec.js", get(t, WasmExecPath).Body.Bytes())` does not fail (the
  served body references no third-party origin).
- `GET /_ds/wasm_exec.js` with a non-GET method returns `405` (extend the existing
  `TestMethodNotAllowed` path list).

## Done When
`mise run check` is green and `go test -run TestWasmExec ./internal/web` proves the Go WASM runtime
loader is served byte-for-byte at `/_ds/wasm_exec.js` with the leaf's no-cache+strong-ETag+304+CDN-free
policy — the on-ramp the tier-2 `<script>` enhancement will load before instantiating the verifier
`.wasm` in the next sub-step.
