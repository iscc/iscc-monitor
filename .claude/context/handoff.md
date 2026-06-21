## 2026-06-21 — Fix the certificate proof-bundle download link `#ZgotmplZ` for the `ISCC:`-prefixed id form

**Done:** Added a canonical, path-rooted `BundleHref` field to `certData` (populated on the certifiable
path as `/inclusion/<bare-id>.bundle`) and pointed `cert.html`'s enabled "Download proof bundle" action
at it, so the headline download link resolves to a working `.bundle` URL for BOTH the bare and the
`ISCC:`-prefixed request id forms. This closes the open critical — the prefixed form previously rendered
`href="#ZgotmplZ.bundle"` because `html/template`'s URL escaper read the leading `ISCC:` as an unknown
scheme.

**Files changed:**
- `internal/certificate/handler.go`: added `BundleHref string` field to `certData` (right after `IsccID`,
  with docstring) and populated it in `buildData` on the certifiable path as
  `PathPrefix + strings.TrimPrefix(rawID, "ISCC:") + bundleSuffix` (path-rooted + prefix-free, matching
  the `.bundle` handler's own prefix-agnostic decode).
- `internal/certificate/cert.html`: line 397 enabled action `href="{{.IsccID}}.bundle"` →
  `href="{{.BundleHref}}"`.
- `internal/certificate/bundle_test.go` (test): extended `TestCertificateProofBundleLinkRendered` to
  loop over both `goldenID` and `"ISCC:"+goldenID`, asserting each renders the same working path-rooted
  href (`href="/inclusion/MAIGHFECJMOPMIAB.bundle"`) AND contains no `#ZgotmplZ`. Bare-form assertion
  retained (updated to the new canonical path-rooted href the template now emits).

**Verification:** `mise run check` → green (all 21 packages `ok`; build + vet + test). Per criterion:
- `go test -run TestCertificateProofBundleLinkRendered ./internal/certificate` → PASS (both id forms).
- `go test -run TestCertificate ./internal/certificate` → PASS (no cert-suite regression).
- Rendered body for `ISCC:`-prefixed certifiable id contains `href="/inclusion/MAIGHFECJMOPMIAB.bundle"`
  and NO `#ZgotmplZ` — confirmed.
- Mutation check (reverting `href="{{.BundleHref}}"` → `href="{{.IsccID}}.bundle"`): the prefixed-form
  sub-case FAILS, reproducing the `href="#ZgotmplZ.bundle"` render. Template restored; tree clean.
- `gofmt -l .` empty; `git diff --stat HEAD -- go.mod go.sum` empty (no new dependency).
- Oracle/conformance gate N/A (pure rendering fix — no §3 re-verification, proof/Merkle, or signature
  code touched; `HasBundle == HasClause3` stays the single gate, untouched).

**Next:** With the critical closed and the criterion re-passing, the 4-commit proof-bundle cycle can be
pushed and CI confirmed green at the new HEAD. After that, the remaining unblocked M-UI work is the OTS
store seam (unblocking §5 BITCOIN ANCHOR + the `ots` bundle member) or the WASM in-browser re-verifier.

**Notes:**
- Chose the named `BundleHref` field (the review's preferred option) over rooting the raw id inline —
  it is the single canonical form and matches the `.bundle` endpoint's own prefix-agnostic decode, so a
  prefix-free path id resolves to exactly the same bundle. `BundleHref` is read only under
  `{{if .HasBundle}}`, but I populate it on the certifiable path (alongside the other clause data) so a
  non-certifiable id leaves it empty and the disabled-button branch renders honestly.
- Out of scope, untouched (per `next.md` / still-open issues for `review`): the `host:port` `did:web:`
  mis-render on §4 and the bundle's `Hub.DID` (`normal`); §5 BITCOIN ANCHOR / OTS store seam (blocked);
  the §6 `· at` timestamp (`normal`); the Hub-List `ForceQuery` guard (`normal`); the
  `Content-Disposition` filename still embeds raw `IsccID` but that is a header value (not a URL context)
  and renders fine — explicitly out of scope.
