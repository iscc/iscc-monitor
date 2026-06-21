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
