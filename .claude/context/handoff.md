## 2026-06-21 — Review of: Serve GET /entries (computed record bytes) from the local mirror

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The `/entries?index=<seq>` route extracts a single accepted leaf's raw record bytes
from the hub's mirrored entry bundles and serves them verbatim as `application/octet-stream`, never
re-hitting the hub — completing M2's third and final computed-proof surface (inclusion, consistency,
entries all served). Core is one pure decode-and-index function `logclient.RecordBytesFromBundle`
(faithful `leafhasher.go` sibling, WASM-pure) plus an `ErrLeafOutOfBundle` sentinel, wired into
`proofserve.Handler` and the per-hub mux. Scope is exactly 3 production `.go` files + 3 test files;
nothing from `## Not In Scope` was touched.

**Verification:**
- [x] `mise run check` — green (build + vet + test, all 14 packages `ok`); re-run uncached
  `go test -count=1 ./...` → all `ok`; `gofmt -l .` empty.
- [x] `go test -run TestRecordBytesFromBundle ./internal/logclient` — PASS (in-range byte-equality,
  `ErrLeafOutOfBundle` via `errors.Is` at/past `len`, empty bundle, `%w`-wrapped truncated frame).
- [x] `go test -run 'TestServeEntries|TestEntries' ./internal/proofserve` — PASS (200 + exact bytes
  across the 256-leaf boundary `{0,5,255,256,260,299}`; 400 missing/non-numeric index; 404 seq≥LastSize,
  no accepted checkpoint, bundle not mirrored; 405 non-GET; Content-Type `application/octet-stream`).
- [x] `go test -run TestMirror ./cmd/iscc-monitor` — PASS (new `TestMirrorEntriesRoute` routes
  `/<origin>/log/entries?index=2` through the shared mux to the record bytes; inclusion + mirror +
  healthz sub-tests intact).
- [x] `entries.go` imports exactly `errors`+`fmt`+`tessera/api`; `GOOS=js GOARCH=wasm go build
  ./internal/logclient` exits 0.
- [x] `git diff --quiet HEAD -- go.mod go.sum internal/store/schema.sql` exits 0 (no dep/schema change).

**Independent verification (reviewer):**
- **Trust-root unaffected.** The new code introduces NO signature/RFC-6962/Merkle/did:web/fsck path
  (`RecordBytesFromBundle` = `EntryBundle.UnmarshalText` then `eb.Entries[offset]`, never re-hashed),
  so the oracle gate is correctly N/A for *this slice's* new code. Confirmed anyway that the diff
  living inside the trust-root package did not disturb it: the `logclient` + `didweb`
  verify/proof/fsck/inclusion conformance tests pass uncached, `derive_vkey.py` reproduces both
  golden vectors (`40b74463`/`22b08f3e`), and the independent `notecheck` external oracle held
  locally (accept `OK sb0.iscc.id/log` + reject-corrupted exit 1) — the same job CI runs on push.
- **Tests are mutation-proven non-vacuous.** Forcing `RecordBytesFromBundle` to return `eb.Entries[0]`
  always (wrong-leaf extractor) FAILED the golden across the bundle boundary (seq 5/255 in the full
  bundle, seq 260/299 in the 44-leaf partial) AND at the binary routing level (`record-0` vs
  `record-2`); reverted → green. A green-but-wrong extractor cannot ship.
- **Gate integrity:** scanned all unpushed commits — no `//nolint`/`t.Skip`/build-tag exclusion,
  no deleted assertion or swallowed error. The `_, _ = w.Write(record)` drop is the documented
  post-status write-drop convention (matching `tilesserve.writeBlob`/`metricshttp`), not a dodge.

**Issues found:** (none)

**Next:** The M2 proof surface is now complete (3-of-3 served). Strong candidates for `define-next`:
(1) the M3 cross-cutting HTTP slice — CORS + caching + conditional-GET (ETag/If-None-Match) headers
applied uniformly across `/inclusion`, `/consistency`, `/entries`, and the static mirror, now that all
three computed surfaces exist; or (2) drain an open `normal` issue (`TestPollHubFork` re-detection
cleanup, frozen-hubs-still-advance, `AcceptCheckpoint` context reuse, tile-writer `p`-vocab, the
`AdvanceAccepted` deep store method, or the `CheckConsistency` collapse). The proof-bundle JSON
assembly + verify-for-me (M3) is the larger feature arc after the HTTP plumbing lands.

**Notes:**
- **Deviation from `next.md`'s `p == 0` implementation note is a genuine correctness FIX, not a scope
  change (reviewer-validated).** `next.md` said "request the full bundle (`p == 0`)". Passing `p == 0`
  unconditionally queries width 256 for the FINAL partial bundle of any non-multiple-of-256 tree (and
  every tree < 256), which the mirror only holds at its partial width — and `SQLiteFetcher`'s
  partial→full fallback fires only for `p > 0`, so it would 404 a leaf that IS in the accepted tree.
  The author instead computes `p := tiles.PartialTileSize(0, bundleIndex, size)` (the exact pattern the
  proof builders use) so a partial later promoted to full still resolves via the fallback. Verified
  sound against `fetcher.go` (the fallback guard is `errors.Is(err, os.ErrNotExist) && p > 0`) and
  exercised by the 300-leaf test (seq 256-299 land in the 44-leaf partial bundle 1) and the 5-leaf
  binary test — both would 404 under the literal `p == 0`. Stays a `internal/tiles`-only computation,
  no new dependency, no API change.
- **Scope clean:** exactly `cmd/iscc-monitor/main.go`, `internal/logclient/entries.go`,
  `internal/proofserve/handler.go` (3 prod) + 3 test files. No `tilesserve` change, no CORS/caching/ETag
  header code, no range fetch, no proof-bundle JSON assembly, no open-issue cleanup. `/entries`
  (computed single-leaf `record_bytes`) is correctly distinct from `/tile/entries/<bundleindex>` (the
  raw 256-leaf whole-bundle BLOB served by `tilesserve` under `"/"`) — different route, different shape.
- **No issues resolved or stale-swept this iteration** — every open `normal` issue was explicitly
  deferred in `## Not In Scope`, so `issues.md` is unchanged. The six open issues (`TestPollHubFork`
  cleanup, frozen-advance, `AcceptCheckpoint` context reuse, tile-writer `p` vocab, `AdvanceAccepted`,
  `CheckConsistency` collapse, + one `low` `notecheck` cosmetic) remain for `define-next` to weigh.
- **Loop = CONTINUE, not DONE:** M2's Verify bar is complete but M3 (dashboard / verify-for-me / CORS /
  landing), WASM, and OTS milestones are entirely unstarted, and `normal` issues remain open.
