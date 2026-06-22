# Next Work Package

## Step: Close the CDN-free gate hole — narrow `stripLineComments` so protocol-relative `//cdn.` URLs still trip `noExternalCDN`

## Advances
This step does **not** advance a new milestone Verify criterion; it **closes the open `normal` issue
that is holding the loop at NEEDS_WORK** and re-greens the WASM-loader sub-step so its review can
return to PASS. HEAD is a `cid(review): NEEDS_WORK` (state.md "Quality gates: AMBER … the latest
review verdict is NEEDS_WORK"), and DONE requires a PASS verdict — so this preempts new milestone
work. The issue (issues.md "`noExternalCDN`'s `stripLineComments` over-strips protocol-relative CDN
URLs, holing the CDN-free gate") is a confirmed weakening of the **M-UI hard CDN-free constraint**:

> "every SSR surface … embeds the DS tokens + self-hosted fonts with **no external CDN URL in the
> body**" — target.md M-UI Verify.

The handoff `**Next:**` from review names this exactly: "Fix the confirmed CDN-free gate hole FIRST
(small, well-scoped test-helper change in `internal/web/web_test.go`): narrow `stripLineComments` to
real comment contexts and add a regression test."

## Goal
Restrict the `//`-comment strip in the load-bearing `noExternalCDN` test helper to *actual comment
contexts* so a protocol-relative loadable CDN URL (`src="//cdn.jsdelivr.net/x.js"`) still trips the
ban, while the vendored `wasm_exec.js`'s genuine `// `-prefixed comment URL stays suppressed. This
restores the M-UI no-CDN quality gate to full strictness and returns the WASM-loader sub-step to a
PASS-able state.

## Scope
- **Modify**: `internal/web/web_test.go` (test-only file — narrow `stripLineComments`, update the
  helper docstring to describe the current state, and add the regression test; does not count against
  the 3-file non-test budget).
- **Reference**:
  - `.claude/context/learnings/web.md` — the `internal/web` detail file; bullets at lines 30–60
    describe `noExternalCDN`'s ban list (third-party-origin only) and the current over-strip hole, and
    say to "narrow the strip to actual comment contexts so a protocol-relative `//cdn.` still trips the
    ban" when the helper is next touched.
  - `internal/web/web_test.go:28–61` — the `noExternalCDN` helper + the flawed `stripLineComments`
    (the `:`-only guard at line 54).
  - issues.md "`noExternalCDN`'s `stripLineComments` over-strips protocol-relative CDN URLs" — the
    confirmed defect, with the exact fix and verification it prescribes.

## Not In Scope
- **Do NOT build or serve the verifier `.wasm`, add a `mise run build:wasm` task, or wire any SSR
  `<script>` caller** — that is the *next* WASM step after this gate is re-greened (handoff "THEN
  resume the WASM tier-2 progression: build + serve the verifier `.wasm` … then the `cert.html`
  `<script>` loader").
- Do NOT touch `web.go` or the byte-verbatim `wasm_exec.js` — the asset-serving path is correct; only
  the test helper is wrong (review: "keep the byte-verbatim copy … tighten the comment detection, not
  revert").
- Do NOT widen or change the ban list itself (`jsdelivr`/`http://`/`https://`/`cdn.`) — same-origin
  `/_ds/` paths must still pass; you are tightening the *strip*, not the *ban*.
- Do NOT touch the other open `normal` issues (`js.Value.Int()` truncation, OTS `safeStamp`, §5
  digest-bind, `hubDomain` ForceQuery, §4 `host:port` DID, §6 timestamp) — each waits for a step that
  edits its own lines.

## Implementation Notes
The bug is the comment-detection predicate at `web_test.go:54`:
`if line[j] == '/' && line[j+1] == '/' && (j == 0 || line[j-1] != ':')`. It treats `//` as a comment
start unless the preceding byte is `:`. A protocol-relative loadable URL puts the `//` after a
**URL-authority delimiter** (`"`, `'`, or `(`), so it is wrongly treated as a comment and the line is
truncated before the ban can see `cdn.`/`jsdelivr`.

Fix the root cause as the issue prescribes: treat `//` as a comment **only in an actual comment
context** — i.e. NOT when it is a URL authority delimiter. The robust predicate is "`//` is a comment
only when it is at line-start OR the preceding byte is whitespace (space/tab)" — and never when the
preceding byte is `:` `"` `'` `(`. This both (a) keeps suppressing the vendored `wasm_exec.js`
comments (they sit after `// `, i.e. whitespace) and (b) re-arms the ban for a protocol-relative
`src="//cdn…"` / `url("//cdn…")` (preceded by `"` — not whitespace, so not stripped). The
whitespace-or-line-start rule is the simplest form that satisfies the issue's "require the byte before
`//` to be whitespace or line-start" wording; you do not also need the explicit `:"'(` exclusion list
once you require whitespace, but stating it in the docstring is fine.

Sanity-anchor on the real bytes before changing the predicate: the existing `TestWasmExecServed`
already proves `noExternalCDN` passes on the verbatim 1.26.1 `wasm_exec.js`, so keep that test green —
the file's comment URLs are all of the genuine `// …` whitespace form, which the narrowed rule still
strips.

Add a focused regression unit test (e.g. `TestNoExternalCDNProtocolRelative`) that drives
`stripLineComments` **directly** on small literal bodies (no HTTP round-trip; avoids `t.Errorf`-capture
gymnastics since `noExternalCDN` itself only calls `t.Errorf`). Assert in both directions:
1. `<script src="//cdn.jsdelivr.net/npm/x.js"></script>` →
   `bytes.Contains(stripLineComments(body), []byte("cdn."))` is **true** (the host now survives the
   strip, so `noExternalCDN`'s ban would fire). Also assert `bytes.Contains(..., []byte("jsdelivr"))`
   is true.
2. The CSS protocol-relative form `url("//cdn.example/x.woff2")` → `cdn.` survives the strip too.
3. A genuine comment line `// see https://github.com/golang/go/issues/12345` →
   `bytes.Contains(stripLineComments(line), []byte("https://"))` is **false** (still suppressed).

Relevant always-loaded rule: **"Never weaken a gate to pass" / fix the root cause** (target.md
Quality bar; CLAUDE.md). Per `learnings/web.md`, the ban list stays third-party-origin only and
same-origin `/_ds/` paths must still pass — preserve that; you are tightening the *strip*, not the
*ban*. Update the `stripLineComments` docstring to describe the current (narrowed) behavior — an
evergreen comment, not a changelog note.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all pass; `gofmt -l .`
  empty).
- `go test -count=1 -run TestWasmExec ./internal/web` passes (the byte-verbatim `wasm_exec.js` still
  passes `noExternalCDN` — its genuine `// ` comment URLs stay suppressed).
- `go test -count=1 -run TestNoExternalCDN ./internal/web` passes (the new regression test).
- Assertion: `bytes.Contains(stripLineComments([]byte(`<script src="//cdn.jsdelivr.net/x.js">`)),
  []byte("cdn."))` is **true** (the protocol-relative CDN host now survives the strip, re-arming the
  ban).
- Assertion: `bytes.Contains(stripLineComments([]byte("// see https://example/x")), []byte("https://"))`
  is **false** (a genuine comment URL is still stripped).
- Mutation check: reverting the narrowed predicate back to the `:`-only guard makes the new
  `TestNoExternalCDN…` test FAIL (proves the gate hole is genuinely closed by this change).

## Done When
`mise run check` is green, the new `TestNoExternalCDN…` regression test passes (and fails when the
narrowed strip is reverted), and `TestWasmExecServed` stays green — proving a protocol-relative
`//cdn.` URL now trips `noExternalCDN` while the vendored `wasm_exec.js` comment URL still does not,
re-greening the WASM-loader sub-step for a PASS verdict.
