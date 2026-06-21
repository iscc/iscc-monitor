<!-- area: internal/tilesserve, internal/proofserve, internal/corsmw, cmd/iscc-monitor (/healthz) -->
<!-- indexed-as: http-surface.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# HTTP read surface — tilesserve, proofserve, corsmw

Read this when a step touches the area above. Durable cross-cutting rules live in
the index (`.claude/context/learnings.md`); the package-local mechanics are here.

## tlog-tiles HTTP read surface (`internal/tilesserve/handler.go`)

- **`Handler(f store.SQLiteFetcher) http.Handler` is a pure static-byte read transport — the oracle
  gate is correctly N/A and verified so, not asserted.** `go list -deps ./internal/tilesserve | grep
  -E 'internal/proof|logclient|didweb|merkle|note|rfc6962'` is EMPTY — the served bodies are opaque
  BLOBs (the verifier computes proofs locally in a later slice), so no signature/RFC-6962/Merkle/
  did:web/fsck path is introduced. `notecheck`/`derive_vkey.py`/CI parity job all untouched + green;
  they re-arm at the proof-serving / `verify-for-me` slice. Same leaf-wrapping posture as `metricshttp`
  (which keeps `net/http` out of the pure `metrics` leaf): `tilesserve → store`, never the reverse —
  `go list -deps ./internal/store` shows neither `tilesserve` nor `net/http`, so store stays a leaf.
- **The routing is robust at every malformed boundary, reviewer-probed against the real tessera parser
  (not just the 9 test subcases).** `serveTile` trims `tile/` then `strings.Cut(rest, "/")` on the FIRST
  slash only (so `tile/0/001.p/44` → `level="0"`, `index="001.p/44"`, `width=44` passes straight as the
  fetcher's `p`); a slash-less `tile/0` → `Cut` `ok=false` → 400; `tile/0/` and `tile/0/abc` →
  `layout.ParseTileLevelIndexPartial` error → 400. The `tile/entries/` case MUST precede `tile/` in the
  switch (entries is a sub-prefix) — confirmed present. tessera's returned `width` IS the `SQLiteFetcher`
  `p` (shared 0==full convention), so it is passed through unmapped, matching `widthForP`.
- **The body-equality + status assertions are mutation-proven non-vacuous (reviewer, reverted).** (1)
  `w.Write([]byte("X"))` instead of the BLOB → all four 200-byte-equal subtests FAIL; (2) collapsing the
  404 mapping to 200 → both the never-mirrored-tile and unmatched-path subtests FAIL. A green-but-wrong
  handler that served constant bytes or 200'd a missing row cannot ship. The single `_, _ = w.Write`
  swallow is the documented metricshttp idiom (the 200 is already on the wire), not a gate dodge.
- **Intentional unwired export seam (like the prior M2 seams): no production caller yet — the binary
  wiring (per-hub route prefix, hub→origin router, read-only pool) is the deferred next slice that
  `next.md` scoped out.** `go vet` clean, not dead code. The handler takes `store.SQLiteFetcher` by
  value (exported `Store *Store` + `HubID int64` fields), so the binary constructs one per hub with no
  new store surface. go.mod/go.sum/schema byte-identical; `GOOS=js GOARCH=wasm go build ./internal/didweb`
  still exits 0 (WASM purity rides on `didweb`, untouched by this net/http leaf-wrapper).
- **Per-route `Cache-Control` keys on the parsed path-API `width`, NOT the store width — `immutable :=
  (width == 0)`.** `writeBlob(w, data, immutable bool)` sets `public, max-age=31536000, immutable` for
  FULL tiles/bundles and `no-cache` for partials + the size-varying checkpoint (`serveCheckpoint` passes
  `false` unconditionally). The predicate is correct because `layout.ParseTile*` returns `width == 0` for
  a full resource and the actual leaf count (`> 0`) for a partial — reviewer re-derived from ground truth
  that `tile/0/000`/`tile/entries/000` parse to width 0 and `tile/0/001.p/44` parses to width 44 (the
  same vocab the SQLiteFetcher documents as `p == 0 → full`, distinct from the store column's 256=full).
  `no-cache` (NOT `no-store`) is deliberate: a partial is overwritten in place every poll (ADR-0005), so
  it must be cacheable-with-revalidation, never pinned immutable. The partial-tile assertion is the
  load-bearing guard — reviewer mutated the predicate to always-immutable and the `partial_tile` subtest
  FAILED (reverted), so a partial wrongly marked immutable cannot ship.
- **Header-order + no-conflict are safe: both `Content-Type` and `Cache-Control` are set before the
  first `w.Write` (the 200 freezes the header map on first write), and the `corsmw` wrap sets only
  `Access-Control-*` (verified), so it neither duplicates nor conflicts with the per-route
  `Cache-Control`.** Error paths (`http.Error` 400/404/405/500) intentionally carry no `Cache-Control`.
  Oracle gate correctly N/A (pure HTTP header wiring on opaque BLOBs — no signature/RFC-6962/Merkle/
  did:web/fsck/proof path); no new import, so go.mod/go.sum/schema byte-identical; `immutable` directive
  string appears in exactly one place (the const), `no-store` absent. `proofserve` size-dependent surfaces
  (`/inclusion`/`/consistency`/`/entries`) still have no cache policy (tied to `LastSize`, later slice).
- **Conditional GET landed: a strong content ETag + `If-None-Match` → 304 on every `writeBlob` 200.**
  `etag := fmt.Sprintf("\"%x\"", sha256.Sum256(data))` — quoted lowercase hex, STRONG (no `W/` prefix).
  All three validators (`Content-Type`/`Cache-Control`/`ETag`) are `Set` BEFORE the `If-None-Match`
  branch, so a 304 still carries `ETag` + `Cache-Control` per RFC 7232 §4.1 (reviewer wrote a throwaway
  test, removed: a full-tile 304 echoes both the ETag and `…immutable` Cache-Control). The match is
  `inm == "*" || inm == etag` — exact-token-or-wildcard only, no comma-separated list parser (a client
  echoes the exact tag the server sent). `writeBlob` now takes `r *http.Request`; the 304 path
  `w.WriteHeader(304)` + bare `return` writes no body. Reviewer independently re-derived the seeded
  full-tile ETag in Python (`sha256(0x11 * 8192)` → `"a44d83e2…"`), confirming Go's `%x` over the
  `[32]byte` array matches the test's expectation — the tag is genuinely content-derived, not constant.
  Oracle gate correctly N/A (opaque-BLOB header wiring); no new dep (`crypto/sha256`+`fmt` stdlib),
  go.mod/go.sum/schema byte-identical, didweb WASM leaf untouched + builds green. The `_, _ = w.Write`
  is the pre-existing post-status-write idiom (not in the added-line diff), NOT a swallowed-error dodge.

## Computed inclusion proof HTTP surface (`internal/proofserve/handler.go`)

- **Nesting a per-hub `http.ServeMux` under `StripPrefix` changes the strip discipline — strip `"/"+
  Origin`, NOT `"/"+Origin+"/"`.** When the inner handler is itself a `ServeMux` (here: `/inclusion` →
  proofserve, `/` → tilesserve), it 301-redirects any path missing its leading slash, so the strip must
  leave the leading slash on the suffix. The earlier tilesserve-only mount stripped the full
  trailing-slash prefix and worked only because `tilesserve` `TrimPrefix`-tolerates a slash-less path.
  The mount prefix still keeps its trailing slash to arm subtree matching. Reviewer-confirmed with a
  standalone mux: exact `/inclusion` wins over the `/` subtree; `/checkpoint`, `/tile/...`, and even
  `/inclusion/x` fall through to tilesserve (the last 404s there, fine — not a real proof route).
- **Oracle gate APPLIES here (RFC-6962 inclusion crypto) and is reviewer-mutation-proven non-vacuous.**
  Three independent paths meet in `TestInclusionServedProofVerifies`: the `testonly.Tree` prover (owns
  the real proof + root), `InclusionProofFromTiles` inside the handler (recompute over the mirror the
  test wrote), and `proof.VerifyInclusion` (verifier). Reviewer reverted-mutated the handler twice: (1)
  serve `proof = nil` → FAIL "wrong proof size 0, want N" across boundary leaves; (2) build for
  `leafIndex+1` → FAIL (wrong-leaf verifies / 500 at the last leaf). A green-but-wrong handler cannot
  ship. `notecheck`/`derive_vkey.py` correctly N/A — the checkpoint signature is never re-parsed or
  served here (the served `{type,treeSize,leafIndex,inclusionProof}` omits `checkpoint`; the client
  refetches `/checkpoint` from the mirror; `AcceptCheckpoint` owns signature/root).
- **Default seq is `seqs[0]` and it is genuinely the lowest committed seq** because `SeqsForISCCID`
  is `ORDER BY seq` ASC — so the documented "first committed seq" default is deterministic, not
  arbitrary. An explicit `&index=<n>` must equal one of the committed seqs (else 400 via `selectSeq`),
  never a silently-substituted leaf; `parseUint` rejects any non-digit → 400, not a silent default.
  `leafIndex >= size` is guarded before building → 404 (never a 500/panic) on a stale/racing size.
- **Dep direction holds: `proofserve → {store, logclient}`, never the reverse.** `go list -deps
  ./internal/store | grep -E 'proofserve|net/http'` and `go list -deps ./internal/logclient | grep
  proofserve` both empty, so `net/http` stays out of the store/logclient closures. Proof is built from
  the LOCAL mirror only (`f.ReadTile`), never re-hitting the hub.

## Computed consistency proof HTTP surface (`internal/proofserve` + `CheckpointAt ORDER BY rowid`)

- **`GET /consistency?from=<n>` is the second proof route; the inner `Handler` is now a `switch
  r.URL.Path { /inclusion, /consistency, default 404 }` after the one shared 405 method-gate.** The
  per-hub mount in `cmd/iscc-monitor/main.go` `hubHandler` mounts ONE `proofserve.Handler(st,hubID)` at
  BOTH exact paths (`mux.Handle("/inclusion", proofs)` + `mux.Handle("/consistency", proofs)`); the
  handler's internal path switch dispatches. Both exact mounts beat the `/` → tilesserve subtree via
  `http.ServeMux` most-specific match — verified green by `TestMirror*` + the proofserve route tests.
- **Oracle gate APPLIES (RFC-6962 consistency crypto) and is reviewer-mutation-proven non-vacuous.**
  `TestConsistencyServedProofVerifies` reuses the inclusion `buildMirror` 300-leaf `testonly.Tree`
  fixture (records a prior checkpoint at each tested `from` via `recordPriorCheckpoint`) and for
  `from ∈ {1,200,255,256,299}` (straddling the 256-leaf tile boundary, `larger=300`) asserts the served
  proof byte-equals `tree.ConsistencyProof(from,300)` AND `proof.VerifyConsistency` ACCEPTS, with a
  sharp wrong-prior-root negative. Reviewer corrupted `encoded[0]` in `writeConsistency` → FAILS BOTH
  the byte-equal assert and `VerifyConsistency` ("calculated root does not match expected root"), then
  reverted → green. Arg-order gotcha (reconfirmed): `VerifyConsistency(hasher, size1, size2, proof,
  root1, root2)` — `proof` precedes the two roots, UNLIKE `VerifyInclusion` where `leafHash` precedes
  `proof`. `notecheck`/`derive_vkey.py` correctly N/A — the consistency response carries no checkpoint
  and re-parses no signature (the client refetches `/checkpoint`; `AcceptCheckpoint` owns sig/root).
- **The degenerate `from == 0` and `from == LastSize` cases are a 200 with an empty `consistencyProof`
  array, not a 400** — `ConsistencyProofFromTiles` returns a nil proof without touching the fetcher.
  `from == 0` SKIPS the `CheckpointAt(from)` row requirement (there is no checkpoint at size 0 by
  construction); `from == LastSize` still REQUIRES the accepted-size row (`buildMirror` records it). The
  literal `next.md` status table ("unknown `from` → 404") and the degenerate note conflicted only at
  `from == 0`; the `from > 0` guard around `CheckpointAt` resolves it. `TestConsistencyDegenerateBoundaries`
  pins both. Status mapping (all pinned): missing/non-numeric → 400; `LastSize==0` → 404; `from>LastSize`
  → 400 (RFC-6962 `M ≤ N`); unrecorded `from` → 404; non-GET → 405; tile-miss `os.ErrNotExist` → 404.
- **`CheckpointAt`'s `ORDER BY rowid LIMIT 1` fix is correct because `id INTEGER PRIMARY KEY` aliases
  `rowid` in SQLite — so rowid is monotonic by insertion and the first-recorded (prior accepted) row is
  returned over a later same-`tree_size` contradicting-evidence row.** `RecordCheckpoint` dedupes on
  `UNIQUE(hub_id, tree_size, root)`, so two DIFFERENT roots at one size are two rows (the fork-evidence
  case); the prior accepted root was recorded first → lowest rowid. Reviewer reversed the order to `DESC`
  → `TestCheckpointAtDeterministicOnFork` FAILS (returns the contradicting row), then reverted → green.
  Store stays a leaf (returns `[]byte`, no `logclient` type). **Knock-on:** the follower-test comment at
  `follower_test.go:280` claiming the query is "unordered … non-deterministic" is now STALE; rewiring
  `TestPollHubFork` to re-detect via a second `PollHub` (not a direct `freeze`) is the remaining
  follow-up — tracked in issues.md, out of scope for the store-only slice.

## Computed record-bytes HTTP surface (`/entries` + `internal/logclient/entries.go`)

- **`GET /entries?index=<seq>` completes M2's proof surface (3-of-3 served) and is the simplest of the
  three — a pure decode + index, NOT crypto.** `logclient.RecordBytesFromBundle(bundle, offset)` is a
  verbatim `leafhasher.go` sibling: `api.EntryBundle{}.UnmarshalText` then `eb.Entries[offset]` returned
  RAW (the JCS-canonical envelope, never re-hashed/verified). It imports exactly `errors`+`fmt`+
  `tessera/api` (WASM-pure, verified `GOOS=js GOARCH=wasm` green); `ErrLeafOutOfBundle` is the sentinel
  the handler maps via `errors.Is` (offset >= `len(Entries)` -> 404, a partial bundle not yet covering the
  leaf). Served `application/octet-stream` via `writeRecord` (post-status write-drop convention, like
  `tilesserve.writeBlob`). Oracle gate correctly N/A (no signature/RFC-6962/Merkle/did:web/fsck path);
  go.mod/go.sum/schema byte-identical. Three exact mounts (`/inclusion`,`/consistency`,`/entries`) now
  share one `proofserve.Handler` via the inner path switch; all beat `/`->tilesserve by most-specific match.
- **`next.md`'s `p == 0` (full-bundle request) note was WRONG for the final partial bundle — advance
  correctly fixed it to `p := tiles.PartialTileSize(0, bundleIndex, size)`.** Passing `p == 0`
  unconditionally queries the mirror at width 256, but the final bundle of any non-multiple-of-256 tree
  (and every tree < 256 leaves) is stored only at its partial width; `SQLiteFetcher`'s partial->full
  fallback fires ONLY for `p > 0` (`errors.Is(err, os.ErrNotExist) && p > 0` in `fetcher.go`), so `p == 0`
  would 404 a leaf that IS in the accepted tree. `PartialTileSize` (entry bundles are level-0) returns the
  exact `p`-qualifier the proof builders already use -> a partial later promoted to full still resolves.
  This is the same partial-bundle gotcha the inclusion/consistency builders handle; remember it for any
  future bundle/tile read keyed on an absolute index. The 300-leaf test (`{256,260,299}` land in the
  44-leaf partial bundle 1) and the 5-leaf binary test both exercise it — both would 404 under `p == 0`.
- **`serveEntries` accepted-tree guards mirror `serveInclusion` exactly:** `FollowState.LastSize == 0`
  -> 404 "no accepted checkpoint"; `seq >= LastSize` -> 404 "leaf not covered by accepted checkpoint";
  bundle-miss `os.ErrNotExist` -> 404; `ErrLeafOutOfBundle` -> 404; missing/non-numeric `index` (reused
  `parseUint`) -> 400; non-GET -> 405. The index is the absolute leaf **seq**, schema-agnostic (ADR-0008) —
  this route interprets nothing (no ISCC-ID codec, no `note.$schema`). Reviewer mutation-proved the tests
  non-vacuous: forcing the extractor to `eb.Entries[0]` FAILED the golden across the bundle boundary AND
  the binary routing test (`record-0` vs `record-2`), then reverted -> green.

## CORS middleware (`internal/corsmw`)

- **`corsmw.Handler(next)` is the monitor's single CORS policy leaf, wrapped once at the lone mux
  convergence point (`buildMux` returns `corsmw.Handler(mux)`).** `serveMetrics` feeds `buildMux(...)`
  straight to `http.Server.Handler`, so one wrap covers the single listener + every mounted subtree
  (metrics, healthz, per-hub mirror/proof). Sets `Access-Control-Allow-Origin: *` BEFORE delegating (so
  it lands on 200/404/405/500 alike, since inner handlers `WriteHeader` via `http.Error` freezes the
  header map); on `OPTIONS` also sets `Allow-Methods: "GET, OPTIONS"` + `Allow-Headers: "*"`, writes 204,
  and returns WITHOUT calling `next` (inner GET-only handlers would 405 a preflight, blocking the real
  GET). Wildcard `*` is correct + simplest: the monitor serves public, credential-free, read-only data,
  so no per-origin allow-list and NO `Allow-Credentials` (browser rejects it paired with `*`).
- **The OPTIONS-skip is double-guarded in the test** — the `tt.inner` for that case `t.Error`s if run AND
  the outer asserts the `ran` sentinel is false; the generic `Allow-Origin == "*"` assert runs for all
  three cases so it also covers the preflight + the non-200 `http.Error` path. Closure is `net/http`+
  stdlib only (`go list -deps` has no `store`/`logclient`); go.mod/go.sum/schema byte-identical; oracle
  gate correctly N/A (pure HTTP header wiring, no signature/RFC-6962/Merkle/did:web/fsck/proof path).
  `corsmw` is NOT on the WASM-shared verifier path (that rides `internal/didweb`) but stays stdlib-only.
