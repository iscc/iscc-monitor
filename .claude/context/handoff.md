## 2026-06-22 — Review of: Certificate §6 render — thread `RecordAt.NoteTimestamp` into `HistoryRow.At` and render `seq N · <at>`

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance closes the §6 render-only `normal` exactly as `next.md` scoped it: an `At string`
field on `HistoryRow` (with docstring), the verbatim `row.NoteTimestamp` captured under the existing
`if found` gate in the §6 loop, and a CONDITIONAL `{{if .At}} · {{.At}}{{end}}` render in `cert.html`.
Scope is exemplary (1 production Go file + 1 template + 1 test, nothing from `## Not In Scope`), all gates
green, the mutation is non-vacuous (I reproduced it independently), and a visual pass confirms the §6 row
now renders `Declaration · seq 24815 · 2026-02-14T18:40:00Z` with no trailing `· ` on the timestamp-less
deletion row. Codex returned a clean no-issues verdict.

**Verification:**
- [x] `mise run check` — green (build + vet + test, all 28 packages ok; certificate re-ran uncached PASS).
- [x] `gofmt -l .` — empty outside `cauldron/` (no formatting failure).
- [x] `go test -count=1 -run TestCertificateRecordHistory ./internal/certificate` — PASS (uncached): §6
      declaration row renders `Declaration · seq 24815 · 2026-02-14T18:40:00Z`; deletion row renders;
      no trailing `· ` artifact for the absent-timestamp deletion.
- [x] `go test -count=1 -run TestCertificateRecordHistoryDeclarationOnly ./internal/certificate` — PASS
      (uncached): a record with no timestamp renders `… · seq 24815` with no trailing `· `.
- [x] Mutation (independent, reverted byte-clean) — dropping `{{if .At}}…{{end}}` from `cert.html` makes
      `TestCertificateRecordHistory` FAIL on the exact marker
      `body missing §6 marker "Declaration · seq 24815 · 2026-02-14T18:40:00Z"` while
      `TestCertificateRecordHistoryDeclarationOnly` still PASSES. Tree confirmed `git diff`-clean after revert.
- [x] Oracle/conformance gate — N/A. Name-only diff over the trust-root globs (`internal/proof/`,
      `logclient/verify`, `didweb`, fork/shrink/equivocation/consistency, `derive_vkey`) → empty. Pure
      store-read into an `html/template` text node; no signature/RFC-6962/Merkle/did:web/proof/fsck path.
- [x] Gate-circumvention scan over the unpushed code range (`origin/develop..HEAD`, explicit paths) — no
      `//nolint`, `t.Skip`, build-tag exclusion, swallowed error, or deleted assertion. The lone `-` line is
      the §6 marker being STRENGTHENED (`Declaration · seq %d` → `Declaration · seq %d · %s`), not weakened.

**Issues found:** (none reviewer-originated). The §6 render-only `normal` is RESOLVED and removed from
`issues.md` (verified by the rendered screenshot + the non-vacuous mutation). The remaining humanization
(`2026-02-14 18:40 UTC`) is intentional ADR-0008-deferred visual polish, not a filed delta — it lives in
the handoff Next + `certificate.md`.

**Codex second opinion:** Clean — no findings. Verdict (verbatim): "The change cleanly threads the stored
note timestamp into the certificate history view and conditionally renders it without introducing new error
paths. The updated tests cover both present and absent timestamp rendering, and the package/full test suites
pass." No issue to triage; agrees with my own read.

**Visual check:** Performed (ADR-0012) — `internal/certificate/cert.html` is an SSR surface. agent-browser
0.29.0 launched headless (bundles its own Chromium; no system Chrome needed). I rendered the §6 fixture
through the REAL handler + `html/template` (a throwaway dump test, removed after), screenshotted the served
HTML, and read it: the §6 region renders `Declaration · seq 24815 · 2026-02-14T18:40:00Z` and
`Deletion · seq 31002` (no trailing `· `), matching the mockup §6 form `seq {{ rec.seq }} · {{ rec.at }}`
(`.claude/design/ISCC Monitor - Certificate.dc.html:68`). The dump renders unstyled over `file://` (the
root-absolute `/_ds/tokens.css` does not resolve there) — that is a harness artifact, not a surface delta;
content/structure is faithful since it is the real rendered bytes. No new visual delta filed.

**Next:** Two clean self-contained picks remain, both unblocked:
1. Wire the same `RecordRow.NoteTimestamp` into the log-browser record-list `Logged` column
   (`internal/proofserve`) — a separate SSR surface, reuses the landed store field, no new store read.
2. The `/` realm-index sub-region deltas (`normal`): config-driven instance identity + realm name, and the
   honest Checkpoint/Anchor data columns (a `ListHubs`/`HubSummary` store-projection change).
The no-migration `normal` and the WASM-verifier-signature `normal` both still want a deliberate design pass
(STOP/design candidates), not a code-only slice. The §6 humanization is ADR-0008-deferred polish.

**Notes:**
- Open count after this review: 0 critical / 4 normal / 10 low. DONE still requires 0 normal, so the loop
  continues. The §6 close took net normal 5→4 (no new issue filed; Codex clean).
- Learnings: `certificate.md` §6 bullet updated from KNOWN-GAP to LANDED (the `{{if .At}}` conditional
  render + the Z-suffixed-UTC fixture rule + the remaining humanization deferral). Nothing promoted to the
  always-loaded index (package-local render mechanics). Detail file within rotation budget.
- Push: 4 commits ahead of `origin/develop` after this review commit (update-state + define-next + advance +
  review). Pushing on PASS. The known `Pages` workflow failure on develop is the human-blocked custom-domain
  repo-settings step (a documented `normal`), not a code regression from this slice.
