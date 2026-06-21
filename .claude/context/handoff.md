## 2026-06-21 — Review of: Fix the certificate proof-bundle download link `#ZgotmplZ` for the `ISCC:`-prefixed id form

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance closes the open critical exactly as `next.md` specified: a canonical,
path-rooted, `ISCC:`-prefix-free `BundleHref` field on `certData` (built in `buildData` on the
certifiable path), with `cert.html:397` pointing the enabled download action at it. The headline
"Download proof bundle" link now resolves to a working `/inclusion/<bare-id>.bundle` URL for BOTH the
bare and the `ISCC:`-prefixed request forms. Tight, in-scope (2 source files + 1 test file), no
behavior change beyond the rendered href; the fix is mutation-proven non-vacuous and Codex agrees it
is clean.

**Verification:**
- [x] `mise run check` → green (build + vet + test, all 21 packages `ok`).
- [x] `go test -count=1 -run TestCertificateProofBundleLinkRendered ./internal/certificate` → PASS for
  BOTH the bare and the `ISCC:`-prefixed id forms.
- [x] `go test -count=1 -run TestCertificate ./internal/certificate` → PASS (no cert-suite regression).
- [x] Rendered body for an `ISCC:`-prefixed certifiable id contains `href="/inclusion/MAIGHFECJMOPMIAB.bundle"`
  and NOT `#ZgotmplZ` — confirmed via the passing test.
- [x] Mutation (reviewer-reproduced independently): reverting `href="{{.BundleHref}}"` →
  `href="{{.IsccID}}.bundle"` renders `href="#ZgotmplZ.bundle"` and FAILS the prefixed-form sub-case.
  Template restored; tree clean.
- [x] `gofmt -l .` clean (excluding gitignored `cauldron/`).
- [x] No new dependency: `git diff --stat HEAD~1..HEAD -- go.mod go.sum` empty.
- [x] Scope discipline: 2 non-test source files (`handler.go`, `cert.html`) + 1 test file; within budget.
  Nothing from `## Not In Scope` touched (DID `host:port`, §5/OTS, §6 timestamp, ForceQuery all left).
- [x] Oracle/conformance gate: N/A — pure rendering fix, no §3 re-verification / proof / Merkle /
  signature code touched. The `HasBundle == HasClause3` single gate is byte-untouched.
- [x] Gate-circumvention scan over the 7 unpushed commits: no `//nolint`, `t.Skip`, build-tag exclusion,
  or swallowed error in added lines. The "deleted assertions" in the diff are the old single-form test
  body refactored into a both-forms loop that asserts strictly MORE (both ids + `#ZgotmplZ` absence).

**Issues found:** (none new). Resolved + deleted the open critical
("Certificate proof-bundle download link renders `#ZgotmplZ` for the `ISCC:`-prefixed id form") —
fix verified and mutation-proven.

**Codex second opinion:** Clean — "The change correctly replaces the unsafe raw ISCC-prefixed href
with a canonical path-rooted bundle URL, and the added tests cover both bare and ISCC-prefixed request
forms. I did not identify any introduced correctness, security, or maintainability issues." No findings
to triage. (Codex finished ~1.5 min after kickoff; verdict captured from `/tmp/codex-review.txt`.)

**Visual check:** n/a — the changed code path (the enabled download href) renders ONLY in the rich
certifiable state, which needs the in-process Go fixture (mirrored tiles + signed checkpoint + cached
key); the cold-start testnet serves only the cannot-certify state, where this attribute never appears.
The rendered href is asserted byte-exact and mutation-proven in `TestCertificateProofBundleLinkRendered`
(both id forms). The change is a single href-attribute value with no layout/chrome impact. `agent-browser`
is present, but a screenshot of the cannot-certify state would not exercise the changed branch.

**Next:** The critical is closed and the criterion re-passes, so this PASS pushes the full proof-bundle
cycle to `origin/develop` and CI re-engages at the new HEAD. After that, the remaining unblocked M-UI
work is the OTS store seam (unblocking §5 BITCOIN ANCHOR + the `ots` bundle member) or the WASM
in-browser re-verifier (`monitor.iscc.codes`). The ADR-0011 iscc-lib/Go-1.26 toolchain bump remains a
queued `normal` foundational change to sequence before more feature work.

**Notes:**
- `BundleHref` is the cleaner of the two acceptable fixes (named field vs inline-rooted raw id): a
  single canonical prefix-free form that matches the `.bundle` endpoint's own prefix-agnostic decode, so
  a prefix-free path id resolves to exactly the same bundle. Populating it on the certifiable path keeps
  the disabled-button branch honest for non-certifiable ids.
- Still-open `normal` issues for a later advance (untouched, correctly out of scope this step): the
  `host:port` `did:web:` mis-render on §4 AND `serveBundle`'s `bundle.Hub.DID` (fix both DID sites
  together, `%3A`-encode the port); the §6 `· at` per-record timestamp (needs an `iscc_index` schema
  column); the Hub-List `ForceQuery` fail-open guard. Plus the ADR-0011 toolchain bump.
- `learnings/certificate.md` net-reduced: collapsed the resolved `#ZgotmplZ` TRAP bullet into the durable
  "path-root user ids in href contexts" rule + a `settled:` line (file stays within the ~150-line budget).
- 7 commits ahead of `origin/develop` before this review commit (the 4-commit proof-bundle cycle + this
  fix cycle's 3). On PASS, all push together.
