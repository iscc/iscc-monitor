# Next Work Package

## Step: Render the self-hosted ISCC logo in the remaining FOUR SSR mastheads

## Advances
Closes the front-of-queue human `critical` "Add the ISCC logo to the remaining FOUR mastheads (serve
route + `/` + dossier already landed)" (`issues.md`), which preempts everything (protocol step 3). It
also satisfies the last gap in the M-UI design-parity bar — **target.md:148-151** ("Document chrome +
instance identity. Every surface carries the shared handoff header: the ISCC logo + 'Trust &
Transparency Monitor' mark …") — for the four surfaces still rendering a text-only mark. The serve route
(`web.LogoPath` `/_ds/iscc-logo-black.png`) and the `/` + dossier renders already landed (review PASS at
`39d395f`); this is the named follow-up in the `review` handoff `**Next:**`. DONE is unreachable while any
`critical` is open, so this lands before the WASM milestone resumes.

## Goal
Add the identical one-line `<img class="chrome-logo" src="/_ds/iscc-logo-black.png" alt="ISCC">` + a
`.chrome-divider` (inside a `.chrome-brand` flex wrapper) to the four SSR mastheads that still show only
the text `.chrome-mark`, so the ISCC logo renders on **every** server-rendered surface and the human
`critical` fully closes.

## Scope
- **Modify** (4 template edits — `.html` templates, not Go source; no `.go` source change is required,
  the serve route already exists):
  - `internal/certificate/cert.html` — wrap the masthead `<div>` (cert.html:367) in `.chrome-brand`,
    prepend the `<img>` + `<span class="chrome-divider">`, add the three CSS rules after the `.chrome` block.
  - `internal/proofserve/browser.html` — same edit at the `<header class="chrome">` block (browser.html:220).
  - `internal/proofserve/record.html` — same edit (record.html:227).
  - `internal/proofserve/records.html` — same edit (records.html:226).
- **Tests (extend, not new source files):**
  - `internal/certificate/handler_test.go` — extend an existing rendering test (e.g.
    `TestCertificateKnownID` at handler_test.go:153) to assert the served HTML contains
    `src="/_ds/iscc-logo-black.png"`. This is the one MANDATORY assertion (the issue's verify bar:
    "a handler test asserts the `<img>` on at least the certificate surface").
  - Optionally add the same one-line assertion to a proofserve rendering test
    (`internal/proofserve/browser_test.go` / `records_test.go` / `record_test.go`) for symmetry.
- **Reference**:
  - `internal/dashboard/dashboard.html` — the canonical masthead to COPY VERBATIM: markup at lines
    328-336 (`.chrome-brand` > `<img class="chrome-logo">` + `<span class="chrome-divider">` + the
    existing `<div>`) and the CSS at lines 40-54 (`.chrome-brand`, `.chrome-logo` `height:19px`,
    `.chrome-divider` `width:1px;height:24px`).
  - `internal/web/web.go:78-82` — `const LogoPath = "/_ds/iscc-logo-black.png"`; the `<img src>` literal
    must stay byte-equal to this (templates cannot read the Go const).
  - `.claude/context/learnings/web.md` lines 76-87 — the logo serve contract + the two forward rules
    (src literal stays `== LogoPath`; never import `image/*`; same-origin `/_ds/` passes `noExternalCDN`).
  - `.claude/design/ISCC Monitor - {Certificate,Log Browser,Single Record}.dc.html` — the masthead
    mockups the ADR-0012 visual pass screenshots against.

## Not In Scope
- **Do NOT factor the now-six near-identical mastheads into a shared `html/template` chrome partial.**
  Review explicitly tagged that KISS factoring "a deferred `advance`-call, not required." Keep the
  copy-paste; consolidation is a separate later step.
- Do NOT touch `internal/web/web.go` or the serve route — it already serves the PNG (no `image/*` import,
  `contentTypePNG` is a string literal). No new `case` is needed.
- Do NOT add the full instance-identity block / `←` back-link parity / Checkpoint-Anchor columns /
  recent-declarers footer (the open `normal` "`/` realm-index sub-region deltas" + named-region parity
  work) — this step is ONLY the logo `<img>`, scoped to close the `critical`.
- Do NOT touch the unrelated open `normal` issues (certificate §5 digest binding, OTS `safeStamp`,
  `hubDomain` ForceQuery, §4/bundle `host:port` DID, §6 timestamp, `safeIndex` test gap, tier-2 honesty
  copy) — none of their lines are edited here.

## Implementation Notes
- All four files share the same structure: a `<header class="chrome">` containing a `<div>` that wraps
  `.chrome-mark` + `.chrome-sub`, and a `.chrome` CSS block (~lines 30-37) already using
  `display:flex;align-items:center`. The edit per file is mechanical:
  1. **CSS** — add the three rules from `dashboard.html` after the existing `.chrome` block:
     ```css
     .chrome-brand { display: flex; align-items: center; gap: var(--space-3); }
     .chrome-logo { height: 19px; width: auto; display: block; }
     .chrome-divider { width: 1px; height: 24px; background: var(--border-default); }
     ```
  2. **Markup** — wrap the existing masthead `<div>` in `.chrome-brand` and prepend the logo + divider:
     ```html
     <div class="chrome-brand">
       <img class="chrome-logo" src="/_ds/iscc-logo-black.png" alt="ISCC">
       <span class="chrome-divider"></span>
       <div>
         <div class="chrome-mark">Trust &amp; Transparency Monitor</div>
         <div class="chrome-sub">…existing sub copy — leave it verbatim…</div>
       </div>
     </div>
     ```
  Keep each file's existing `.chrome-sub` text exactly as it is — the cert/proofserve sub-copy may differ
  from dashboard's; do not normalize it.
- The `<img src>` literal MUST be exactly `/_ds/iscc-logo-black.png` (byte-equal to `web.LogoPath`) — the
  always-loaded learnings rule (templates can't read the const; the test asserting the literal is the
  regression guard). Do NOT introduce a Go-side template field for the path; the static literal is
  intentional and matches dashboard/dossier.
- **noExternalCDN must stay green.** The cert/proofserve tests already ban `http://`/`https://`/`cdn.`/
  `jsdelivr` in the rendered body (grep-confirmed in `handler_test.go`, `browser_test.go`,
  `record_test.go`, `records_test.go`). A same-origin `/_ds/iscc-logo-black.png` has none of those
  substrings, so it passes (learnings/web.md line 85) — but re-run those tests to confirm.
- Make the new assertion non-vacuous: host it on a test that already produces a 200 HTML body for a
  known/certifiable fixture (`TestCertificateKnownID`), and assert
  `strings.Contains(body, ` + "`" + `src="/_ds/iscc-logo-black.png"` + "`" + `)`. Mutation check: dropping
  the `<img>` line from `cert.html` makes the assertion FAIL.
- **Correctness rule (learnings.md):** none of the crypto/freeze/re-verify rules apply (pure static-asset
  rendering; oracle/conformance gate is N/A — no signature/RFC-6962/Merkle/proof path touched). The
  load-bearing rule here is the learnings/web.md logo contract: the src literal stays `== LogoPath` and the
  serve leaf never imports `image/*` (untouched in this step).

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`; `gofmt -l .` excl.
  `cauldron/` empty).
- `go test -run TestCertificate ./internal/certificate` passes, and the extended test asserts the served
  certificate HTML contains `src="/_ds/iscc-logo-black.png"`.
- `go test ./internal/proofserve` passes (the existing no-CDN body bans stay green with the new `/_ds/`
  img src).
- `grep -L 'iscc-logo-black.png' internal/certificate/cert.html internal/proofserve/browser.html internal/proofserve/record.html internal/proofserve/records.html`
  prints **nothing** (all four now reference the logo).
- Mutation check: removing the `<img>` line from `cert.html` makes the new certificate assertion FAIL;
  restoring it passes.

## Done When
All four remaining SSR mastheads (`internal/certificate/cert.html`,
`internal/proofserve/{browser,record,records}.html`) render `<img src="/_ds/iscc-logo-black.png">`, a
certificate handler test asserts that src, and `mise run check` is green — closing the human `critical`.
