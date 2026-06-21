## 2026-06-21 — Review of: Wire the iscc_index projection into PollHub's entry-bundle ingestion

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` wired the two built-but-unwired `iscc_index` halves into a running projection:
`ingestEntryBundles` now folds every mirrored entry bundle via `logclient.BundleProjections(raw,
bundleIndex*tiles.TileWidth)` into `store.RecordProjections` immediately after `RecordEntryBundle`
(new `projectEntryBundle` helper), copying `logclient.Projection → store.ProjectionRecord`
field-by-field so the store stays a leaf. The change is scope-clean (1 production file + 2 test files),
both verified-path mutations are independently reproducible, and the fixture conversion keeps the
signed root self-consistent (fsck still rebuilds). A verified `PollHub` now populates `iscc_index`,
unblocking the later inclusion cross-check.

**Verification:**
- [x] `mise run check` — GREEN (11 packages `ok`, build + vet + test; uncached re-run all `ok`).
- [x] `go test -run TestPollHub -count=1 ./internal/follower` — PASS uncached (existing verified-path +
  mirror + fsck tests survive the `leafPreimages` JSON-envelope change).
- [x] `go test -run TestPollHubRecordsProjections -count=1 ./internal/follower` — PASS uncached; full
  verified `PollHub` over the 300-leaf mirror, fsck "Successfully fsck'd log with size 300", read-back
  `SeqsForISCCID == [seq]` for {0,5,255,256,260,299} (crosses the 256-leaf bundle boundary) + absent-id
  negative returns empty.
- [x] `gofmt -l .` — empty.
- [x] store stays a leaf — `go list .Imports ./internal/store | grep -E 'internal/logclient|net/http'`
  empty; full closure is `context crypto/sha256 database/sql embed errors fmt internal/tiles
  modernc.org/sqlite os time`.
- [x] `git diff --quiet HEAD -- internal/store/schema.sql go.mod go.sum` — exit 0 (no schema/dep change).
- [x] **Conformance / oracle gate** (slice touches the verified proof path) — full `./internal/follower`
  (fsck root-rebuild over `SQLiteFetcher`) + `./internal/logclient` (golden / inclusion / equivocation) +
  `./internal/didweb` all PASS uncached; `python3 .claude/derive_vkey.py` reproduces both golden vectors
  (`40b74463`/`22b08f3e`); `cmd/notecheck` accepts the real sb0 checkpoint (`OK sb0.iscc.id/log`) and
  rejects a corrupted one (exit non-zero). CI `notecheck` signature-parity job present + unchanged.
- [x] **Mutation re-verification (reviewer, reverted)** — (1) `BundleProjections(raw, 0)` →
  `TestPollHubRecordsProjections` FAILS (bundle-1 seqs collide onto bundle-0 under `ON CONFLICT(seq) DO
  UPDATE`); (2) skip `RecordProjections` → read-back empty, FAILS. Both prove the wiring non-vacuous.
  Tree clean after revert.
- [x] **Gate-integrity scan** (all unpushed commits) — no `//nolint`, `t.Skip`, build-tag, swallowed
  error, or deleted assertion/test; every error `%w`-wrapped and checked.
- [x] **Scope** — exactly 1 non-test production file (`internal/follower/ingest.go`) + 2 test files;
  nothing in `## Not In Scope` touched (no `VerifyInclusionEvidence` wiring, no ISCC-ID codec, no
  deletion projection, no `schema.sql`/`go.mod`/`go.sum`, open `normal` follower issues untouched).
- [x] WASM purity invariant — `GOOS=js GOARCH=wasm go build ./internal/didweb` (+ logclient) green.

**Issues found:** (none) — clean slice; no new issues filed, none resolved (all open `normal` follower
issues were correctly out of scope for this slice).

**Next:** Wire the inclusion cross-check — resolve a sampled `iscc_id → leafIndex` via `SeqsForISCCID`
and feed `logclient.VerifyInclusionEvidence` over the `SQLiteFetcher` (`ReadTile`) against the hub's own
`IsccLogInclusionProof`. This needs an `IsccLogInclusionProof` fixture (none captured yet) and is the
natural moment to weigh the open `normal` follower issues (frozen-hub advance, `CheckpointAt` ordering,
`AcceptCheckpoint` did.json re-fetch, tile `p`/`width` duplication) since it reworks the verified path
more deeply.

**Notes:**
- The `leafPreimages` fixture conversion was a mandatory test-fix, not just an additive test: once the
  bundle fold runs on every verified poll, the old `leaf-%d` plaintext is a genuine JSON-parse fault.
  The conversion is sound — the tree is rebuilt from the same JSON preimages framed into the bundles, so
  the signed root stays self-consistent (fsck green at size 300). Distinct per-leaf `iscc_id`
  (`ISCC:LEAF%08d`) makes the read-back a clean one-seq-per-id lookup.
- `TestIngestTilesWidthMapping`'s `recordingFetcher` now frames a valid one-record bundle for
  `/tile/entries/` URLs (hash-tile URLs are `tile/<digit>/`, no suffix collision); the width-mapping
  assertions are unchanged.
- Error discipline matches the existing ingest posture: a malformed record / store fault is wrapped
  `project entry bundle index %d: %w`, aborts the poll before accepted state advances, and never freezes
  the hub (ADR-0008 + ADR-0006).
- No remote push concern noted yet — pushing `develop` (working branch) per protocol.
