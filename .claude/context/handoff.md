## 2026-06-22 — Close the CDN-free gate hole: narrow `stripLineComments` so protocol-relative `//cdn.` URLs still trip `noExternalCDN`

**Done:** Narrowed the load-bearing `stripLineComments` test helper in `internal/web/web_test.go` to
treat `//` as a comment only in an actual comment context (line-start OR preceded by whitespace
space/tab), replacing the over-broad `:`-only guard. A protocol-relative loadable CDN URL
(`src="//cdn.jsdelivr.net/x.js"`, `url("//cdn...")` — `//` preceded by `"`, not whitespace) now
survives the strip and trips the ban, while the vendored `wasm_exec.js`'s one genuine `// ` comment
URL (preceded by a tab) stays suppressed. Added a focused regression test and updated the docstring.

**Files changed:**
- `internal/web/web_test.go` (test-only): predicate at the former line 54 is now
  `(j == 0 || line[j-1] == ' ' || line[j-1] == '\t')`; `stripLineComments` docstring rewritten to
  describe the narrowed comment-context rule; added `TestNoExternalCDNProtocolRelative` driving
  `stripLineComments` directly (no HTTP round-trip) with three assertions (script-src host survives,
  jsdelivr survives, CSS url() host survives, genuine comment URL stays stripped).

**Verification:** `mise run check` → GREEN (build + vet + test, all 25 packages `ok`; `internal/web`
recompiled and passed). `gofmt -l .` empty. Per-criterion:
- [x] `go test -run TestWasmExec ./internal/web` → PASS (byte-verbatim `wasm_exec.js` still passes
  `noExternalCDN`; its `// ` comment URL stays suppressed).
- [x] `go test -run TestNoExternalCDN ./internal/web` → PASS (the new regression test).
- [x] Assertion: `stripLineComments(<script src="//cdn.jsdelivr.net/x.js">)` now contains `cdn.` and
  `jsdelivr` (host survives the strip, re-arming the ban). Confirmed by the test.
- [x] Assertion: `stripLineComments("// see https://example/x")` does NOT contain `https://` (genuine
  comment URL still stripped). Confirmed by the test.
- [x] Mutation check: reverted the predicate to the `:`-only guard → `TestNoExternalCDNProtocolRelative`
  FAILS with "cdn. host was stripped" / "jsdelivr was stripped" / "url() host was stripped"; restored
  the narrowed predicate (gofmt-verified). The gate hole is genuinely closed.

**Next:** The CDN-free gate hole is closed; this re-greens the WASM-loader sub-step for a PASS verdict.
THEN resume the WASM tier-2 progression review named: build + serve the verifier `.wasm` (a
`mise run build:wasm` task + ADR-0003 reproducible-build / published-hash / SRI pin), then the
`cert.html` `<script>` loader + record-bytes emission (where the open `normal` `js.Value.Int()`
truncation hardening belongs, at the first real caller).

**Notes:**
- Scope clean: exactly one test-only file touched (`internal/web/web_test.go`) plus `handoff.md`.
  Nothing from `## Not In Scope` was done — `web.go` and the byte-verbatim `wasm_exec.js` are
  untouched, the ban list (`jsdelivr`/`http://`/`https://`/`cdn.`) is unchanged (tightened the *strip*,
  not the *ban*), and the other open `normal` issues were left alone.
- Docstring gotcha for next editor: `gofmt`'s comment reformatter mangles a literal escaped single
  quote (`'\''`) inside a `//` comment into a curly quote. The docstring now spells out the URL-authority
  delimiters in words ("a colon … a double-quote / single-quote / open-paren") to avoid that artifact —
  do not reintroduce literal quote-char escapes in the comment.
- Same-origin `/_ds/` paths still pass (`TestFontsCSSServed`, `TestTokensCDNFree`, dashboard CDN tests
  all green) — the narrowed rule only changes whether a `//` after a non-whitespace byte is a comment,
  which does not affect any same-origin path form in the served assets.
- Oracle/conformance gate: N/A — static-asset test-helper change, no signature/RFC-6962/Merkle/did:web/
  fsck/proof code touched. Visual check: N/A — no SSR template or rendered HTML changed.
- The web.md detail file (lines 52–60) flags this exact over-strip as an open issue and prescribes the
  narrow-to-comment-context fix; that note can be updated/closed by review now that it is fixed.
