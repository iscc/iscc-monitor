# Next Work Package

## Step: Fix the certificate proof-bundle download link `#ZgotmplZ` for the `ISCC:`-prefixed id form

## Advances
Closes the open **critical** issue *"Certificate proof-bundle download link renders `#ZgotmplZ` for the
`ISCC:`-prefixed id form"* — which preempts all milestone work (a `critical` preempts everything). It
re-passes the **M-UI certificate Verify criterion**: the certificate *"offers a downloadable proof
bundle `{checkpoint, inclusion/consistency proof, record bytes, hub key, ots?}`"* — and the design-parity
bar's **Download proof bundle primary action**. The last review (`89641a4`) held the proof-bundle cycle
at NEEDS_WORK precisely because this headline affordance is a dead link (`href="#ZgotmplZ.bundle"`) for
the canonical, explicitly-supported `/inclusion/ISCC:MAIG…` form; the bare form works. Until this is
fixed, the criterion FAILS and the 4-commit cycle stays unpushed (no CI at HEAD).

## Goal
Make the enabled "Download proof bundle" link resolve to a working `.bundle` URL for **both** the
`ISCC:`-prefixed and the bare id form, so the certificate page links its proof bundle via a genuinely
enabled download action. After it re-passes, the cycle can be pushed and CI confirmed green at the new
HEAD.

## Scope
- **Create**: (none)
- **Modify** (2 non-test source files, within the ≤3 budget):
  - `/workspace/iscc-monitor/internal/certificate/handler.go` — add a `BundleHref string` field to
    `certData` (right after `IsccID`, ~line 192), and populate it in `buildData` with a **path-rooted,
    prefix-free** href so `html/template`'s URL escaper never sees a leading `ISCC:` scheme. Update the
    `certData` field docstring.
  - `/workspace/iscc-monitor/internal/certificate/cert.html` — line 397: change the enabled action's
    `href="{{.IsccID}}.bundle"` to `href="{{.BundleHref}}"`.
- **Modify (test, not counted toward the budget)**:
  - `/workspace/iscc-monitor/internal/certificate/bundle_test.go` — extend
    `TestCertificateProofBundleLinkRendered` (line 161) with an `ISCC:`-prefixed case
    (`"ISCC:" + goldenID`) asserting a working `.bundle` href AND no `#ZgotmplZ`. Keep the existing
    bare-form assertion.
- **Reference** (read before implementing):
  - `/workspace/iscc-monitor/.claude/context/learnings/certificate.md` — read FIRST: the **TRAP** bullet
    (lines 139–147) and the rule *"Any template emitting a user-supplied id into a `url`/`href` context
    must path-root it, never let `ISCC:` lead."* Also the §1 `lookupID` canonicalization bullet
    (`"ISCC:" + strings.TrimPrefix(rawID, "ISCC:")`) — the precedent for stripping the prefix.
  - `/workspace/iscc-monitor/internal/certificate/handler.go:538-578` — `buildData`; the `lookupID`
    canonicalization at 578 is the exact prefix-handling precedent; `certData{IsccID: rawID}` at 543.
  - `/workspace/iscc-monitor/internal/certificate/handler.go:482` — `Content-Disposition` filename also
    embeds raw `IsccID`, but that is a header value (not a URL context), so it is **not** the critical and
    is out of scope.
  - `/workspace/iscc-monitor/.claude/context/issues.md` — the critical entry (top) names both acceptable
    fixes and the exact verification.

## Not In Scope
- The `host:port` `did:web:` mis-render (`normal`, on both §4 and the bundle) — do **not** touch the DID
  builders this step; it is a separate issue fixed when handler.go's DID sites are next reworked.
- Certificate **§5 BITCOIN ANCHOR** / the OTS store seam — blocked; the next milestone step after this.
- The §6 `· at` timestamp (`normal`), the Hub-List `ForceQuery` guard (`normal`), and any `low` issue.
- Re-touching the `.bundle` request handler / `serveBundle` — the endpoint already accepts both id forms
  (it strips `.bundle` then decodes); only the rendered href is wrong.
- The `Content-Disposition` filename (header value, not a URL context — renders fine).

## Implementation Notes
- **Root cause:** `cert.html:397` builds the href directly from `.IsccID`, which carries the RAW request
  id (`certData{IsccID: rawID}`, handler.go:543). In `html/template`'s URL context the leading `ISCC:` of
  the prefixed form reads as an unknown URL scheme, so the escaper replaces the whole attribute with the
  filtered sentinel `#ZgotmplZ` → `href="#ZgotmplZ.bundle"`. The bare form has no leading scheme, so it
  escaped through fine and the existing test (which uses only the bare `goldenID`) missed it.
- **Preferred fix — a canonical `BundleHref` field** (the review's named option, cleaner than rooting the
  raw id): in `buildData`, set `data.BundleHref = "/inclusion/" + strings.TrimPrefix(rawID, "ISCC:") +
  ".bundle"`. The mount is the exact subtree `PathPrefix = "/inclusion/"`, and the `.bundle` handler
  decodes the bare form fine (decode is prefix-agnostic), so a prefix-free path id resolves identically.
  A leading `/` AND the stripped `ISCC:` both remove the scheme ambiguity — the escaper then emits the
  path verbatim. Set it inside the certifiable branch (alongside the other clause data) so a
  non-certifiable id leaves it empty and the disabled-button branch renders. (`BundleHref` is read only
  under `{{if .HasBundle}}`, but populating it only on the certifiable path keeps it honest.)
- **Why not just root the raw id** (`href="/inclusion/{{.IsccID}}.bundle"`): that also works (a leading
  `/` makes it a path, not a scheme), and the issue lists it as acceptable — but it emits the
  `ISCC:`-prefixed id into the URL path with a literal `:`. A prefix-free `BundleHref` is the single
  canonical form and matches the endpoint's own canonicalization. Prefer `BundleHref`.
- **Test (non-vacuous):** add a sub-case for `"ISCC:" + goldenID` that (a) asserts the body contains the
  literal working href (e.g. `href="/inclusion/` + goldenID + `.bundle"`), and (b) asserts the body does
  **NOT** contain `#ZgotmplZ`. Keep the existing bare-form assertion. Reverting the template/handler fix
  must make this case FAIL — confirm by reverting locally before committing the advance (review will
  re-run this mutation). The link-render path needs no real crypto (the existing test runs on
  `[]byte("raw")` via `fixtureStoreTiled` + `get(t, h, …)`).
- **Correctness rule (learnings index):** this is a pure rendering fix — it does **not** touch the §3
  re-verification gate, the proof/Merkle path, or any signature code, so the **oracle/conformance gate is
  N/A** for this step (the seeded "gate a ✓ on a re-VERIFICATION" rule is untouched; `HasBundle ==
  HasClause3` stays the single gate). Keep `mise run check` green; do not weaken any gate.
- **No new dependency**; `go.mod`/`go.sum` must stay byte-identical.

## Verification
- `mise run check` is green (build + vet + test across all 21 packages; `gofmt -l .` empty).
- `go test -count=1 -run TestCertificateProofBundleLinkRendered ./internal/certificate` passes for BOTH
  the bare and the `ISCC:`-prefixed id forms.
- `go test -count=1 -run TestCertificate ./internal/certificate` passes (no regression in the cert suite).
- Assertion: the rendered certificate body for an `ISCC:`-prefixed certifiable id contains a working
  `.bundle` href (e.g. `href="/inclusion/MAIGHFECJMOPMIAB.bundle"`) and does **NOT** contain `#ZgotmplZ`.
- Mutation check (advance confirms before commit): reverting the href fix in `cert.html` / `handler.go`
  makes the new prefixed-form assertion FAIL.
- No new dependency: `git diff --stat HEAD -- go.mod go.sum` is empty.

## Done When
`mise run check` is green and `TestCertificateProofBundleLinkRendered` passes for both id forms with a
working `.bundle` href and no `#ZgotmplZ` in the rendered body, closing the open critical.
