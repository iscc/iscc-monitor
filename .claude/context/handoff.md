## 2026-06-21 — Review of: Pure entry-bundle → iscc_index projection decoder (`BundleProjections`)

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added the pure, schema-agnostic `BundleProjections(bundle []byte, baseSeq
uint64) ([]Projection, error)` in `internal/logclient/projection.go` — a faithful sibling of
`LeafHashes` that decodes one tlog-tiles entry bundle into per-leaf `{Seq, IsccID, NoteSchema,
RecordSHA256}` records (ADR-0008), reading the top-level `iscc_id` and the INNER `note.$schema`
plus the record-content SHA-256, interpreting nothing. The diff is exactly two new files (decoder +
golden test, 225 additive Go lines, zero deletions), all verification criteria pass, and the golden
is reviewer-mutation-proven non-vacuous.

**Verification:**
- [x] `mise run check` — GREEN (all 11 packages `ok`; build + vet + test).
- [x] `go test -run TestBundleProjections -count=1 ./internal/logclient` — PASS uncached, all 5
  subtests (main golden + empty + schema-agnostic + malformed-record + truncated-bundle).
- [x] `GOOS=js GOARCH=wasm go build ./internal/logclient` — exit 0 (file stays WASM-shareable).
- [x] `gofmt -l .` — empty.
- [x] `git diff --quiet HEAD -- go.mod go.sum internal/store/schema.sql` — exit 0 (no dep/schema change).
- [x] `go list -f '{{join .Imports "\n"}}' ./internal/logclient | grep -E 'internal/store|database/sql'`
  — empty (store/`database/sql` stay un-imported by logclient).
- [x] Assertion: two-record bundle (declaration + deletion) at `baseSeq=256` → length 2 with
  `Seq={256,257}` and verbatim `NoteSchema={iscc-note-0.8.0.json, iscc-note-delete-0.8.0.json}`; IDs
  round-trip; `RecordSHA256 == sha256.Sum256(record)` recomputed independently. Confirmed.
- [x] Scope discipline — exactly 2 new files (1 non-test `projection.go` + 1 test), purely additive;
  nothing from `## Not In Scope` touched (no store import, no `iscc_id→seq` query, no PollHub wiring,
  no ISCC-ID parsing).
- [x] Gate integrity — unpushed-commit scan clean: no `//nolint`/`t.Skip`/build-tag/swallowed-error;
  zero `.go` deletions, no assertions/tests weakened. (grep hits were all handoff prose, not code.)
- [x] Oracle gate — correctly N/A: pure JSON + content-SHA-256 fold, no signature/RFC-6962/Merkle/
  did:web/fsck path. `notecheck`/`derive_vkey.py` untouched.

**Reviewer verification beyond the criteria:**
- Cross-checked the envelope shape against `cauldron/iscc-hub/specs/iscc-log.md` §5.1 (`{$schema,
  iscc_id, note}`, deletion discriminator `note.$schema`) and §5.2 (seq N == leaf index) — the
  decoder reads the correct fields and `Seq = baseSeq + i` is the spec sequence number.
- Mutation 1 (reverted): read a constant/outer `$schema` instead of `env.Note.Schema` → FAILS both
  the declaration-vs-deletion golden and the schema-agnostic case → the inner-schema claim is real.
- Mutation 2 (reverted): prepend `0x00` to the content hash (leaf-hash style) → FAILS the
  `RecordSHA256` assertion → the content-hash-vs-leaf-hash distinction is genuinely pinned.
- Working tree restored and confirmed byte-identical to HEAD after both mutations.

**Issues found:** (none)

**Next:** The store writer — `store.RecordProjections`/`RecordProjection` upserting `[]Projection`
into `iscc_index`. Decide the `iscc_id` BLOB encoding vs the `iscc_id_str` TEXT column (schema.sql
has both; the projection carries only the raw `ISCC:`-prefixed string, so the writer owns the
BLOB-decode mapping). That slice unblocks the `iscc_id → []seq` read query and, with it, wiring
`VerifyInclusionEvidence` into `PollHub` (resolve a sampled leaf index, pass `SQLiteFetcher.ReadTile`
straight in) — the remaining M2 second-half gap.

**Notes:**
- Open issues unchanged — none resolved by this slice (all 7 touch store/follower/accept paths, not
  projection). 6 `normal` + 1 `low`; the 6 `normal` still block DONE. No new issues filed: the slice
  is correct, scope-clean, purity-clean, well-tested.
- The `net/http`/`net/url`/`os` entries in the `logclient` *package* closure are pre-existing (via
  `didresolve.go`), NOT introduced by `projection.go` — the file-level import set is exactly the four
  required imports and the WASM build is green. Verify purity by the WASM build, never by grepping the
  package dep list.
- Pushed to `origin/develop`.
