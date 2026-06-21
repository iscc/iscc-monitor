## 2026-06-21 — Review of: `iscc_index` store writer + `iscc_id → []seq` read-back

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** Added `internal/store/iscc_index.go` — the store half of the M2 `iscc_index`
projection (ADR-0008): `RecordProjections` (idempotent per-row `ON CONFLICT(seq) DO UPDATE`
upsert) and `SeqsForISCCID` (one-to-many `iscc_id → []seq` reader over the existing BLOB index).
The diff is scope-clean (2 additive files + handoff), matches `next.md` exactly, store stays a
leaf, and both load-bearing surfaces (idempotent upsert, hub-scoped lookup) are mutation-proven
non-vacuous by the reviewer.

**Verification:**
- [x] `mise run check` — GREEN (all 11 packages `ok`; build + vet + test).
- [x] `go test -run TestRecordProjections -count=1 ./internal/store` — PASS (uncached).
- [x] `go test -run TestSeqsForISCCID -count=1 ./internal/store` — PASS (uncached).
- [x] `gofmt -l .` — empty (no formatting failures).
- [x] One-to-many: declaration@256 + deletion@257 same `IsccID`, distinct schemas →
  `SeqsForISCCID` returns `[256 257]` ordered (`TestSeqsForISCCIDOneToMany`); descending-input
  variant returns `[42 100 300]` (`TestSeqsForISCCIDOrderedDescendingInput`).
- [x] Idempotency: two writes at seq 256 → `COUNT(*) WHERE seq=256 == 1` AND `note_schema`/
  `record_sha256` round-trip to the *second* write (`TestRecordProjectionsIdempotent`).
- [x] Schema-agnostic: `"iscc-note-future-9.9.9"` + empty `IsccID` both persist and read back
  verbatim; empty id looked up via `SeqsForISCCID("")` (`TestRecordProjectionsSchemaAgnostic`).
- [x] Leaf-purity: `go list -f '{{join .Imports}}' ./internal/store | grep -E
  'internal/logclient|net/http'` is empty.
- [x] `git diff --quiet HEAD~1 -- internal/store/schema.sql go.mod go.sum` exits 0 (no schema/dep
  change).
- [x] Gate-integrity: no `//nolint`/`t.Skip`/swallowed-err/build-tag/deleted-assertion in the
  unpushed diff.

**Mutation checks (reviewer, both reverted):**
- Drop the `hub_id` filter from `SeqsForISCCID` → `TestSeqsForISCCIDScopedByHub` FAILS (`[10 20]`
  vs `[10]`).
- `ON CONFLICT … DO UPDATE` → `DO NOTHING` → `TestRecordProjectionsIdempotent` FAILS on
  "second write wins" (stale `note_schema` + `record_sha256`). A green-but-wrong upsert/scope
  cannot ship.

**Issues found:** (none) — no new issues filed; no prior issue resolved by this slice (the open
`normal` issues are all in `follower`/`logclient`/`checkpoints.go`, untouched here).

**Next:** Wire the `iscc_index` projection into the `PollHub` tile-ingestion loop — decode each
ingested entry bundle via `logclient.BundleProjections`, copy `Projection → ProjectionRecord` at
the call site (store stays a leaf), `RecordProjections`, then resolve a sampled `iscc_id →
leafIndex` via `SeqsForISCCID` and feed `logclient.VerifyInclusionEvidence` over the
`SQLiteFetcher` (`ReadTile`). This re-arms the oracle/inclusion-cross-check gate (currently N/A for
the pure CRUD slices) against the hub's own `IsccLogInclusionProof`. It also touches `follower`, so
it is the natural moment to weigh the open `normal` follower issues (frozen-hub advance,
`CheckpointAt` ordered `LIMIT 1`, `AcceptCheckpoint` did.json re-fetch, tile `p`/`width` duplication).

**Notes:**
- **Import deviation from `next.md` is correct, not a flag.** `next.md` proposed `context,
  database/sql, errors, fmt`; advance shipped `context + fmt` only. The reader's absent case is the
  natural empty `rows.Next()` result, not an `sql.ErrNoRows` sentinel, so `database/sql`/`errors`
  are genuinely unused (gofmt/vet would reject them). Behavior matches `next.md` exactly.
- **Oracle/conformance gate correctly N/A** — plain CRUD + BLOB round-trip with synthetic in-test
  records; no signature/RFC-6962/Merkle/did:web/`fsck`-rebuild path. `notecheck`/`derive_vkey.py`/
  `fsck` untouched. The gate re-arms when `VerifyInclusionEvidence` is wired into `PollHub` (next
  slice).
- **Unwired-until-M2 export seam by design** (like `RecordTile`/`LookupHubKey`/`LeafHashes` were) —
  no production caller this slice, `go vet` clean, not dead code. Nothing from `## Not In Scope` was
  touched (no follower/PollHub wiring, no ISCC-ID codec, no `note_schema` validation, no
  `schema.sql`/`go.mod`/`go.sum` change).
- M1 → OTS milestones are not all independently confirmed against `target.md` Verify criteria, and
  open `normal` issues remain — so not DONE. No human-only decision is open — not STOP. Pushing on
  PASS.
