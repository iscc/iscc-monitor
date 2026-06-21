## 2026-06-21 — Review of: Single-record page at `GET /record?index=<seq>`

**Verdict:** NEEDS_WORK
**Loop:** CONTINUE

**Summary:** The single-record page lands with a clean handler (serveRecord faithfully copies
serveEntries' accepted-tree-capped bundle math, treats the mirrored bytes as the source of truth so a
missing projection renders rather than 404s, and mounts correctly in cmd/iscc-monitor), a well-dressed
DS-shell template, and thorough HTTP-seam tests; gates are green and store stays a leaf. BUT it ships a
real correctness defect: the kind-label constants are glossary short forms (`iscc-note-0.8.0`), while
production `note.$schema` is the full URI `http://purl.org/iscc/schema/iscc-note-0.8.0.json`, so every
real declaration/deletion renders "Unknown record type" — defeating the slice's primary M-UI Verify
criterion. The synthetic test masks it by inventing the short forms.

**Verification:**
- [x] `mise run check` green (build + vet + all 20 packages test; cached + spot re-run uncached) — PASS
- [x] `gofmt -l .` empty — PASS
- [x] `go test -run TestRecord ./internal/proofserve` (uncached) — PASS (but tests use short-form
  schemas that don't match production wire values; see defect)
- [x] `go test -run TestRecordsListsNewestFirst ./internal/proofserve` — PASS (rows now link `record?index=`)
- [x] `go test -run TestRecordAt ./internal/store` — PASS (existing row found; absent → found=false, nil)
- [x] `go list -deps ./internal/store | grep '^net/http$'` empty AND no `internal/logclient` — PASS (leaf)
- [x] `git diff --stat HEAD~1..HEAD -- schema.sql go.mod go.sum` empty (byte-identical) — PASS
- [x] Oracle/conformance gate correctly N/A and NOT skipped; inclusion/consistency conformance re-ran
  uncached green — PASS
- [x] Scope: exactly 3 production `.go` files (iscc_index.go, handler.go, main.go) — at budget; no gate
  circumvention in the 3 unpushed commits (only removed assertion is the stale `entries?index=4` link,
  replaced by `record?index=4`) — PASS
- [ ] **M-UI Verify: "single-record page renders declaration, deletion AND an unknown note.$schema
  without erroring"** — FAIL for declaration/deletion on real data: both render as "Unknown" because the
  kind-label constants miss the production schema URI (the "without erroring" half holds; the
  correct-label half does not).

**Issues found:**
- **[normal] Single-record kind-label constants miss the real `note.$schema` URIs** (filed in issues.md).
  `handler.go:108-109` uses `iscc-note-0.8.0` / `iscc-note-delete-0.8.0`, but `BundleProjections` stores
  the verbatim wire value, which is `http://purl.org/iscc/schema/iscc-note-0.8.0.json` (and `…delete…`).
  Ground truth: `internal/logclient/projection_test.go:19-20` (golden test decoding real envelopes),
  `internal/follower/fsck_test.go:119`, and the store's own `iscc_index_test.go` fixtures all use the
  `…-0.8.0.json` form. So in production `recordKind` falls through to `kindUnknown` for every real
  declaration/deletion. The new `record_test.go::schemaForSeq` invents the bare short forms to match the
  constants — a self-consistent fixture that masks the bug. Fix: set both constants to the full URIs and
  reseed the test with those URIs (also re-scope the no-CDN `http://` body ban so it doesn't false-fail
  on the now-realistic schema data — it should target the template/CDN region, not verbatim record fields).

**Codex second opinion:** Codex (exit 0) raised exactly one finding, **[P2]** "Use the full note schema
URIs for kind labels" at handler.go:108-109 — **CONFIRMED**. I verified it independently against the
trust-adjacent ground truth (the `BundleProjections` golden test and the follower/store fixtures all pin
the full `http://purl.org/iscc/schema/…json` URI), filed it as a `normal` issue, and it is the blocking
defect for this verdict. No other Codex findings. Good catch — this is precisely the class of bug a
second skeptic exists for (green synthetic tests over a wrong constant).

**Next:** Fix the kind-label constants + test in a small advance (it is a 2-line constant change plus a
fixture/no-CDN-assert correction, scoped to `internal/proofserve`). That closes the open `normal` issue
and completes the single-record M-UI Verify criterion. Then proceed to the originally-planned next
slice: the certificate of inclusion (`/inclusion/{iscc_id}`) + downloadable proof-bundle assembler,
which re-engages the oracle gate.

**Notes:**
- The handler flow itself is correct and should be kept: accepted-tree cap, partial-bundle `p`, bytes-as-
  source-of-truth (missing projection → 200 "no projection indexed", not 404), buffer-then-200, overlay
  status via the shared badge partial. Only the two kind-label constants (and the test that feeds them)
  are wrong — the fix is local and low-risk.
- Watch the no-CDN test interaction: once the constants/test use the real URI, the rendered page body
  will contain `http://purl.org/...` as legitimate record data. The `TestRecordLinksTokensNoCDN`
  `http://`/`https://` ban must be narrowed to the template/style region (or the projection in that test
  given a non-URL schema) so it bans CDN refs without false-failing on verbatim record fields.
- The 3 unpushed commits (update-state, define-next, advance + this review) are NOT pushed — verdict is
  NEEDS_WORK, so per protocol the next cycle fixes the defect first, then pushes.
- Pre-existing unstaged change to `.claude/context/target.md` was left untouched (not a review-owned file).
- Learnings rotated: store.md 149→142 lines (collapsed the iscc_index writer/reader detail into one
  `settled:` bullet + added the durable "note.$schema is the full wire URI, not a short name" trap);
  http-surface.md 158→157 (collapsed inclusion/consistency/entries/records settled bullets, added the
  single-record section + the schema-URI interpretation trap).
