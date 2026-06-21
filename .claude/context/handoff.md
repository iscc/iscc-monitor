## 2026-06-21 — Review of: Fix single-record kind-label constants to the full `note.$schema` URIs

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The advance fixed the blocking defect: `schemaDeclaration`/`schemaDeletion`
(`handler.go:111-112`) now hold the full wire URIs (`http://purl.org/iscc/schema/iscc-note-0.8.0.json`
+ `…delete…`), byte-equal to the golden ground truth `internal/logclient/projection_test.go:19-20`, so
real declarations/deletions render "Declaration"/"Deletion" instead of "Unknown record type" — the
M-UI single-record Verify criterion is now genuinely met. The no-CDN ban was correctly scoped to the
document head (up to `</style>`). Production code is correct; the one residual is that the guarding test
is tied to the constant under test rather than to a literal, so it cannot catch a future regression of
the constant value (filed `low`).

**Verification:**
- [x] `mise run check` — green (build + vet + all 20 packages; proofserve re-run uncached 0.668s)
- [x] `gofmt -l .` — empty (PASS)
- [x] `go test -count=1 -run TestRecord ./internal/proofserve` — PASS uncached (`TestRecordKindLabels`
  decl/del/unknown/empty all 200 with correct labels; `TestRecordLinksTokensNoCDN` no longer
  false-fails on the verbatim `http://purl.org/...` schema)
- [x] Declaration URI → "Declaration", deletion URI → "Deletion", non-URL/empty → "Unknown record
  type" (200) — confirmed at runtime via the green suite
- [x] `grep '"iscc-note-0.8.0"\|"iscc-note-delete-0.8.0"' internal/proofserve/handler.go` — no match
  (short forms gone) PASS
- [x] `git diff --stat HEAD -- go.mod go.sum internal/store/schema.sql` — empty (byte-identical) PASS
- [x] Constants byte-match the golden `projection_test.go:19-20` ground truth — confirmed by direct
  string comparison (the literal ground-truth source, not the code under review)
- [x] Scope: exactly 2 files (1 production + 1 test), both in `internal/proofserve` — within budget
- [x] Oracle/conformance gate correctly N/A — the diff touches only the `proofserve` render surface;
  no `internal/proof`/`logclient/verify`/`didweb`/fork-shrink-equivocation path. `internal/proof` is
  not even a package (verify lives in `logclient`); the earlier grep hit was the `internal/proof*`
  prefix matching `proofserve` only.
- [x] Gate-integrity scan over all unpushed commits (`origin/develop..HEAD`, 8 commits) —
  no `//nolint`/`t.Skip`/build-tag/swallowed-error pattern; the one removed `t.Errorf`
  (`entries?index=4` link) was replaced by the `record?index=4` assertion in a prior reviewed PASS,
  not silently dropped.

**Issues found:**
- **[low] Single-record label test is vacuous on the kind-label constant value** (filed in issues.md).
  Mutation-found: `schemaForSeq` returns the `schemaDeclaration`/`schemaDeletion` constants and
  `recordKind` switches on the same constants, so reverting BOTH constants back to the old short forms
  leaves the entire `go test -run TestRecord ./internal/proofserve` suite GREEN (reviewer-verified).
  The test is written to the symbol under test, not to ground truth. Does NOT block PASS: the
  production constants are already correct (byte-equal to the golden `projection_test.go`), so the
  feature works and the M-UI criterion holds; this only hardens the regression gate. The real
  protection today is that `projection_test.go` is itself a non-vacuous golden test, and the constants
  match it.

**Codex second opinion:** Codex (exit 0) reviewed the advance commit: "The patch aligns the
single-record kind labels with the full wire schema URIs and adjusts the CDN assertion scope without
introducing a functional regression. The relevant Go tests pass." — no findings. Concurs with the
PASS direction. It did not surface the test-vacuity gap (a mutation-only observation), which I found
independently and filed `low`.

**Next:** Proceed to the originally-planned slice deferred out of this fix: the certificate of
inclusion (`/inclusion/{iscc_id}` HTML) + the downloadable proof-bundle assembler. That slice
re-engages the oracle/conformance gate (signature / RFC-6962 / Merkle path), so the reviewer must
mutation-prove the served bundle's inclusion proof non-vacuous and confirm `notecheck`/golden-vector
parity where the bundle carries hub-signed material. (Optionally fold in the `low` test-hardening above
if `record_test.go` is touched.)

**Notes:**
- The mutation finding is the durable lesson, now recorded in `learnings/http-surface.md`: tie a
  `note.$schema`→label test to a HARDCODED literal URI (or compare the constant against the
  `projection_test.go` literal), never to the constant under test — otherwise both sides move together
  and the gate is vacuous. This is the same class of "tests written to the code" the prior NEEDS_WORK
  flagged; the constant is now correct, but the test still has the structural blind spot.
- Learnings rotated: `http-surface.md` 157→155 lines (collapsed the now-landed single-record "defective"
  section to `settled:` + the durable schema-URI trap + the new test-vacuity lesson; deduped the
  `iscc_index above LastSize` paragraph that appeared twice; tightened the tilesserve + inclusion
  settled bullets). Index unchanged at 83 lines.
- Pre-existing unstaged/committed `.claude/context/target.md` edit (`155c8ed target(M-UI): named-region
  design-parity bar`) in the unpushed history is a target-owned file, left untouched (not review-owned).
- Branch is 8 commits ahead of `origin/develop` (`83588b1`); this PASS pushes them all.
