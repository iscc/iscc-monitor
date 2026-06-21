<!-- area: internal/tilesserve, internal/proofserve, internal/corsmw, cmd/iscc-monitor (/healthz) -->
<!-- indexed-as: http-surface.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# HTTP read surface — tilesserve, proofserve, corsmw

Read this when a step touches the area above. Durable cross-cutting rules live in
the index (`.claude/context/learnings.md`); the package-local mechanics are here.

## tlog-tiles HTTP read surface (`internal/tilesserve/handler.go`)

- **settled:** the static-BLOB mirror (`/checkpoint`, `/tile/...`, `/tile/entries/...`) is landed and
  stable — pure opaque-byte read transport (oracle gate N/A), `tilesserve → store` (store stays a leaf,
  `net/http` out of its closure), `strings.Cut`-on-first-slash routing with `tile/entries/` ordered
  before `tile/`, per-route `Cache-Control` keyed on the parsed `width` (`immutable := width==0`,
  partials `no-cache`), and a strong content ETag + `If-None-Match` → 304. All body-equality / 404 /
  partial-immutable / ETag asserts were mutation-proven non-vacuous and reverted. (Detail in git
  history pre-2026-06-21.) The one durable trap to remember: the path-API `width` vocab (`0 == full`)
  differs from the store column's `256 == full`, so map deliberately.

## Computed inclusion proof HTTP surface (`internal/proofserve/handler.go`)

- **settled:** the `/inclusion` RFC-6962 proof seam is landed + stable — strip discipline (`"/"+Origin`,
  NOT `+"/"`, so the inner `ServeMux` keeps its leading slash and exact mounts beat the `/` subtree),
  oracle gate APPLIES and was mutation-proven non-vacuous (`TestInclusionServedProofVerifies`: tree
  prover vs `InclusionProofFromTiles` vs `proof.VerifyInclusion`; reverted-mutated `proof=nil` and
  `leafIndex+1` both FAIL), and `notecheck`/`derive_vkey.py` correctly N/A (the served object omits
  `checkpoint`; the client refetches `/checkpoint`; `AcceptCheckpoint` owns sig/root). (Detail in git
  history pre-2026-06-21.) The one durable trap: strip leaves the leading slash so a nested mux does not
  301-redirect.
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

- **settled:** the `/consistency` RFC-6962 proof seam is landed + stable — one `proofserve.Handler`
  mounted at both exact paths via the inner path switch (both beat the `/` subtree by most-specific
  match), oracle gate APPLIES and was mutation-proven non-vacuous (`TestConsistencyServedProofVerifies`
  over the 300-leaf fixture; corrupting `encoded[0]` FAILS both byte-equal + `VerifyConsistency`), and
  `notecheck`/`derive_vkey.py` correctly N/A. (Detail in git history pre-2026-06-21.) The two durable
  traps: **`VerifyConsistency(hasher, size1, size2, proof, root1, root2)`** — `proof` precedes the two
  roots, UNLIKE `VerifyInclusion` where `leafHash` precedes `proof`; and the degenerate-boundary rules
  below.
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

## verify-for-me JSON verdict (`/verify` + `serveVerify`)

- **`/verify` INVERTS the other proof routes' status mapping: an id-shaped fault is a 200 verdict, never
  4xx/5xx.** `serveVerify` composes the same machinery the other three routes use (FollowState ->
  CheckpointAt -> SeqsForISCCID -> ReadEntryBundle -> InclusionProofFromTiles -> proof.VerifyInclusion),
  but missing-id / unknown-id / no-accepted-checkpoint / leaf-out-of-tree / tile-not-mirrored ALL return
  `200 {verified:false, reason}`. Non-200 is reserved for genuine infra faults ONLY (DB read error on
  FollowState/CheckpointAt/SeqsForISCCID, a `CheckpointAt found==false` at the accepted size = a real
  store inconsistency, or a non-`os.ErrNotExist` proof build/read error -> 500). This is the documented
  weaker "caller trusts the verdict" path — remember it when adding tests for any future verify-for-me
  surface (the bundle assembler is the stronger client-verifies path with different posture).
- **Oracle gate APPLIES (RFC-6962 inclusion crypto) and is reviewer-mutation-proven non-vacuous.** The
  verdict's `included`/`verified` is a REAL `proof.VerifyInclusion` against the persisted accepted root,
  not a stub. Reviewer reverted-mutated `serveVerify` to ignore the Merkle result (`included := true`):
  `TestVerifyInclusionIsNonVacuous` (corrupted accepted root) FAILS all three asserts, reverted -> green.
  `hub_status` is the store-provable subset only (`frozen` if `FollowState.Frozen`, else `verified` —
  ADR-0006: only signature-verified checkpoints advance `LastSize`); the richer
  unverified/unresolvable/rotated statuses live in the in-memory metrics registry and are deliberately
  NOT threaded into proofserve (a documented limitation, not a defect).

## HTML log browser at the hub-log root (`GET /` + `serveBrowser`) — closes M3 4/4

- **The bare-`/` hub-log root is a 5th `proofserve.Handler` route, but the per-hub mux can't mount it as
  an exact pattern — an `http.ServeMux` cannot hold both an exact `/` AND a subtree `/` (the subtree
  pattern `/` IS the bare-`/` match).** Fix in `cmd/iscc-monitor` `hubHandler`: the `/` slot is a tiny
  dispatch `http.HandlerFunc` that sends `r.URL.Path == "/"` to `proofs` (proofserve) and delegates every
  deeper path to `tilesserve`. The four exact proof mounts (`/inclusion`/`/consistency`/`/entries`/
  `/verify`) still win by most-specific match. Reviewer end-to-end-confirmed through the full `buildMux`
  (throwaway, then removed): `GET /<domain>/log/` → 200 text/html "Log Browser"; `/<domain>/log/checkpoint`
  → 200 octet-stream (still tilesserve); `POST /<domain>/log/` → 405 (the shared method-gate at the top
  of `proofserve.Handler` covers it); `/<domain>/log` (no slash) → 301 to trailing slash (subtree).
- **`serveBrowser` is a pure store-read render (oracle gate N/A) — same posture as `/verify`'s
  store-provable subset, NOT the crypto routes.** Reads only `FollowState` + `CheckpointAt`, base64-Std
  encodes the root verbatim (never recomputed); render-into-`bytes.Buffer`-then-200, post-200 write-drop;
  `html/template` (NOT text) so root/status auto-escape. Status mapping: DB error or `CheckpointAt
  found==false` at the accepted size → 500 (the real store-inconsistency fault); `LastSize == 0`
  (followed-but-unpolled) → **200** "no accepted checkpoint yet" (ADR-0001 coverage honesty, never a 404
  and never a fabricated `(0,"")`). Mutation-proven non-vacuous: dropping `{{.Root}}` and `{{.Size}}`
  each FAIL `TestBrowserExposesAcceptedCheckpoint` (reviewer re-ran independently, reverted → green).
  go.mod/go.sum/schema byte-identical; store stays a leaf. Minor: an unpolled non-frozen hub renders
  "Status: verified" (the store-provable subset only knows frozen-vs-verified), softened by the explicit
  no-coverage sentence — consistent with `serveVerify`, not a regression.

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
