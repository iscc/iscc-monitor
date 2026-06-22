## 2026-06-22 — Review of: Serve `wasm_exec.js` from the `/_ds/` static-asset leaf (WASM loader skeleton)

**Verdict:** NEEDS_WORK
**Loop:** CONTINUE

**Summary:** The advance correctly serves the Go 1.26.1 WASM runtime loader byte-verbatim at
`/_ds/wasm_exec.js` through the existing `internal/web` leaf (one `case` + `writeAsset`, reusing the
no-cache/strong-ETag/304/CORS policy unchanged; pure stdlib, WASM-green, scope-clean). The asset and
its named verification all pass. But the refinement it made to the load-bearing `noExternalCDN` test
helper — stripping `//` comment tails so the vendored loader's one comment URL stops false-positiving —
**over-strips and silently holes the CDN-free gate** for protocol-relative loadable CDN URLs
(`src="//cdn.jsdelivr.net/..."`). Codex flagged this [P2]; I confirmed it by probe. The hole is latent
(no current asset triggers it) but it weakens a quality gate, so the diff cannot be approved as-is — the
root-cause fix (narrow the strip to real comment contexts) is filed as a `normal` issue for the next cycle.

**Verification:**
- [x] `mise run check` green — `go build`/`go vet`/`go test` all 25 packages `ok`.
- [x] `gofmt -l .` empty (clean).
- [x] `go test -count=1 -run TestWasmExec ./internal/web` → PASS.
- [x] `GET /_ds/wasm_exec.js` → 200, `Content-Type: text/javascript; charset=utf-8`, strong (non-`W/`)
  quoted-hex ETag, `Cache-Control: no-cache`, non-empty body carrying `globalThis.Go`.
- [x] `If-None-Match` echo → 304 (extended `TestIfNoneMatch304` list includes `WasmExecPath`).
- [x] non-GET → 405 (extended `TestMethodNotAllowed` list includes `WasmExecPath`).
- [x] `wasm_exec.js` is `cmp`-identical to `$(go env GOROOT)/lib/wasm/wasm_exec.js` (16992 bytes, Go 1.26.1).
- [x] `GOOS=js GOARCH=wasm go build ./internal/web` → OK; dep closure still pure (no store/metrics/logclient).
- [x] Mutation probe: `stripLineComments` → identity makes `TestWasmExecServed` FAIL (the test is
  non-vacuous; it genuinely depends on the strip).
- [x] Mutation probe: the refined helper STILL fires on real loadable CDN refs (jsdelivr src, bare `cdn.`,
  plain `https://`/`http://`, a real URL on a line with a trailing comment) — relaxes nothing for those.
- [ ] **CDN-free invariant fully preserved** — FAILED. Probe-confirmed the strip ALSO truncates a
  protocol-relative loadable CDN URL (`src="//cdn.jsdelivr.net/x.js"` → stripped to `<script src="`), so
  the ban does not fire. The helper's stated guarantee ("ban stays exactly as strict for loadable
  content") does not hold for this common CDN URL form.
- [x] Gate-circumvention scan over unpushed commits — no `//nolint`, `t.Skip`, build-tag exclusions,
  swallowed errors, or deleted assertions in source (the grep hits are all handoff/doc prose).
- Oracle/conformance gate: N/A — pure static-asset transport (no signature/RFC-6962/Merkle/did:web/fsck/proof).

**Issues found:** ONE confirmed, blocking PASS (filed `normal`):
`noExternalCDN`'s `stripLineComments` over-strips protocol-relative CDN URLs, holing the CDN-free gate
(`internal/web/web_test.go`). Latent (no current asset triggers it), but it weakens the M-UI no-CDN
quality gate, so the verdict is NEEDS_WORK per the "never approve a gate weakening" rule. The fix is the
root cause: restrict the `//` strip to actual comment contexts (e.g. only when `//` is preceded by
whitespace/line-start, or not preceded by a URL authority delimiter `"`/`'`/`(`), so a protocol-relative
`//cdn.` URL still trips the ban while the pure-comment URL stays suppressed.

**Codex second opinion:** One [P2] — "Preserve protocol-relative URLs in CDN checks"
(`web_test.go:54-55`): the `//`-comment strip truncates a protocol-relative CDN URL like
`src="//cdn.jsdelivr.net/x.js"` because the byte before `//` is `"` not `:`, hiding `jsdelivr`/`cdn.`
from the ban. **CONFIRMED real** by reviewer probe (stripped output is literally `<script src="`; ban
`fires=false` for all three protocol-relative forms). Filed as the `normal` issue above and it sets the
verdict to NEEDS_WORK. No other findings; Codex's summary agreed the asset-serving path itself is
correct.

**Visual check:** n/a — no SSR surface changed. This is a `/_ds/` static-asset transport change
(`wasm_exec.js` is runtime JS glue, not a rendered surface); no `internal/dashboard|dossier|web|
certificate` template or rendered HTML was touched, so the agent-browser visual pass does not apply.

**Next:** Fix the confirmed CDN-free gate hole FIRST (small, well-scoped test-helper change in
`internal/web/web_test.go`): narrow `stripLineComments` to real comment contexts and add a regression
test asserting `src="//cdn.jsdelivr.net/x.js"` trips `noExternalCDN` while the `wasm_exec.js` comment URL
does not. That re-greens this skeleton sub-step. THEN resume the WASM tier-2 progression the advance
named: build + serve the verifier `.wasm` (a `mise run build:wasm` task + ADR-0003 reproducible-build /
published-hash / SRI pin), then the `cert.html` `<script>` loader + record-bytes emission (where the open
`normal` `js.Value.Int()` truncation hardening belongs, at the first real caller).

**Notes:**
- Scope was clean: advance commit (b62be88) touched exactly `web.go` (1 non-test/doc), `wasm_exec.js`
  (the in-scope asset), `web_test.go` (test), + handoff. Nothing from `## Not In Scope` was done (no
  `.wasm`, no `build:wasm`, no `cert.html` `<script>`, no `cmd/wasm` `Int()` fix, no mount change).
- The over-strip is a genuine dilemma the advance faced honestly: next.md required BOTH "keep
  `wasm_exec.js` byte-verbatim" AND "assert CDN-free via `noExternalCDN`", and the verbatim file carries
  one banned `https://` substring inside a `//` comment — a real conflict. The advance's instinct (scope
  the ban to loadable content, per web.md's own "scope to a non-self host" note) is correct; only the
  IMPLEMENTATION (a `:`-only guard) under-handles protocol-relative `//host` URLs. The fix is to keep the
  scoping idea but tighten the comment detection, not to revert the byte-verbatim copy.
- No remote push this cycle (verdict is NEEDS_WORK). Upstream is `origin/develop`; commits stay local
  until the gate hole is fixed and the verdict returns to PASS.
- Standing open issues (OTS §5 digest-bind, OTS stamp-path guards, `hubDomain` ForceQuery, §4 DID
  `host:port`, §6 timestamp, `js.Value.Int()` truncation, + the lows) remain untouched and out of scope
  for this static-asset step.
