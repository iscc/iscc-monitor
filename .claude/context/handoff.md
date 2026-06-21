## 2026-06-21 — Review of: Certificate §6 RECORD HISTORY — render the per-id record list (declaration + any deletion)

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** §6 RECORD HISTORY now renders the full one-to-many list of accepted-tree seqs a hub
indexed under the subject id — the declaration plus any later deletion, each labelled by its verbatim
`note.$schema` kind, with the deletion note shown when any row is a deletion. It is a pure store read
(`seqs` already in hand from `SeqsForISCCID`, one `RecordAt` per row), accepted-tree-capped
(`seq >= LastSize` dropped, matching §1) and rendering unconditionally for a certifiable id. Scope is
tight (2 source files + the test file), all gates are green, and the mutation is independently
reproduced non-vacuous. One minor visual delta (the mockup's per-record `· at` timestamp, which the
projection has no column for) is filed `normal`; it does not block the increment.

**Verification:**
- [x] `mise run check` — green, all 21 packages `ok` (build + vet + test).
- [x] `gofmt -l .` — empty (no formatting failures).
- [x] `go test -count=1 -run TestCertificate ./internal/certificate` — PASS (all §1-§4 plus the two
  new §6 tests).
- [x] `go test -count=1 -v -run TestCertificateRecordHistory ./internal/certificate` — PASS:
  `TestCertificateRecordHistory` (declaration + deletion rows + note + §1 position) and
  `TestCertificateRecordHistoryDeclarationOnly` (single row, no note) both PASS.
- [x] Mutation (non-vacuity — reviewer reproduced independently, 2 mutations): `data.HasClause6 = true`
  → `= false` makes `TestCertificateRecordHistory` FAIL (no §6 marker / rows / note); `if isDeletion`
  → `if false` makes it FAIL (deletion note suppressed, deletion-row label wrong). Both reverted, tree
  clean, tests green.
- [x] Oracle gate unbroken: `go test -count=1 ./internal/logclient ./internal/follower ./cmd/notecheck`
  all `ok`. Oracle gate is N/A for this step (a store read + render; no signature / Merkle / proof code
  touched) — run only to prove no regression.
- [x] WASM/purity: `GOOS=js GOARCH=wasm go build ./internal/index ./internal/didweb` exit 0.
- [x] No new dependency: `git diff --stat HEAD~1..HEAD -- go.mod go.sum` empty.
- [x] Scope discipline: exactly 2 non-test source files (`handler.go`, `cert.html`) + the test file;
  no §5 / proof-bundle / §4-DID / registry work done.
- [x] No gate circumvention across the 3 unpushed commits (no `nolint`/`t.Skip`/build-tag/swallowed
  error; the `continue` cap and the `found == false` unknown-label fallthrough are legitimate
  fail-open-on-gap rendering, not dodges).

**Issues found:**
- **[review, visual pass → filed `normal`]** §6 rows omit the per-record `· at` timestamp the mockup
  (`.dc.html:68`) shows, because `store.RecordRow` carries no timestamp column. Cosmetic; the named
  region's primary affordance (kind + seq + deletion note) is complete. Surfacing it needs a store
  schema change (out of scope). Filed for a later advance.

**Codex second opinion:** Clean verdict (exit 0): "The new §6 record history rendering is consistent
with the existing store APIs and accepted-tree gating, and the added tests cover declaration/deletion
and declaration-only cases. I did not find any introduced correctness, security, or maintainability
issues that warrant blocking the patch." No findings to triage.

**Visual check:** SSR surface (`internal/certificate`) — rendered the rich §6 state (declaration +
deletion) via a throwaway fixture-seeded harness to `/tmp/cert-history.html`, screenshotted it and the
`.dc.html` mockup with agent-browser (bundled Chromium; no system Chrome), and `Read` both. §6 renders
correctly: `Declaration · seq 24815`, `Deletion · seq 31002`, and the deletion note, in the same
clause-marker + clause-value structure as §2 (the standalone render is unstyled — it links
`/_ds/tokens.css`, served only by the live instance — an offline-render artifact, not a regression).
One delta filed `normal`: the mockup row carries a `· at` timestamp the projection has no column for.
Throwaway harness + scratch deleted, tree clean.

**Next:** §5 BITCOIN ANCHOR is the last remaining clause but is BLOCKED on the OTS/anchor store seam,
which does not exist (no `anchor`/`ots` store method). The data-grounded clauses (§1-§4, §6) are now
complete; the next M-UI certificate work is the **downloadable proof-bundle assembler**
`{checkpoint, inclusion/consistency proof, record bytes, hub key, ots?}`, which re-engages the
oracle/conformance gate (reuses the §3 build+verify crypto path + the §4 key read-path). That is the
bigger criterion item; either it or the OTS-seam-then-§5 work lands next.

**Notes:**
- The 3 unpushed commits are this §6 advance + its define-next + the §4 update-state. This
  PASS_WITH_NOTES pushes all 3 to `develop` (upstream `origin/develop`); CI on `develop` is the gate.
- The kind mapping is duplicated from `proofserve.recordKind` (its constants are unexported);
  `next.md` explicitly directed defining them locally. Two pure 6-line switches — minor DRY debt, not
  worth a shared leaf until a third caller appears (YAGNI). Noted in `learnings/certificate.md`.
- The §6 accepted-tree cap (`seq >= LastSize` dropped) has no dedicated deletion-above-checkpoint test,
  but the boundary is identical to §1's (which IS mutation-tested) and §1 guarantees `seqs[0] <
  LastSize` so the list is always non-empty. The 500-on-`RecordAt`-fault branch is also untested
  (mirrors §2/§3/§4's identical buffer-then-200 pattern). Both acceptable; flag only if load-bearing.
- The §4 `did:web:` + raw-domain `host:port` mis-render (`normal`) and the `hubDomain` ForceQuery gap
  (`normal`) remain open — neither is on the §6 path; fold each in when its DID/registry code is next
  touched.
