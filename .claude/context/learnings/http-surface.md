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

- **settled:** the `/consistency` RFC-6962 proof seam is landed + stable (oracle gate APPLIES, mutation-
  proven by corrupting `encoded[0]`, reverted). Three durable traps survive: (1)
  **`VerifyConsistency(hasher, size1, size2, proof, root1, root2)`** — `proof` precedes the two roots,
  UNLIKE `VerifyInclusion` (`leafHash` precedes `proof`); (2) degenerate `from == 0` / `from == LastSize`
  → 200 with empty `consistencyProof` (nil proof, fetcher untouched); `from == 0` SKIPS the
  `CheckpointAt(from)` row requirement, `from == LastSize` still REQUIRES it. Status mapping (pinned):
  missing/non-numeric → 400; `LastSize==0` → 404; `from>LastSize` → 400 (`M ≤ N`); unrecorded `from` →
  404; tile-miss → 404. (Detail in git history pre-2026-06-21.)
- **`CheckpointAt`'s `ORDER BY rowid LIMIT 1` is correct because `id INTEGER PRIMARY KEY` aliases `rowid`
  in SQLite — rowid is monotonic by insertion, so the first-recorded (prior accepted) row wins over a
  later same-`tree_size` contradicting-evidence row** (`RecordCheckpoint` dedupes on
  `UNIQUE(hub_id, tree_size, root)`, so two roots at one size are two rows = the fork-evidence case).
  Mutation `DESC` → `TestCheckpointAtDeterministicOnFork` FAILS, reverted. Store stays a leaf.

## Computed record-bytes HTTP surface (`/entries` + `internal/logclient/entries.go`)

- **settled:** `GET /entries?index=<seq>` completes M2's served proof surface (3-of-3) — a pure decode +
  index, NOT crypto. `logclient.RecordBytesFromBundle` (`UnmarshalText` then `eb.Entries[offset]`, RAW)
  is WASM-pure; `ErrLeafOutOfBundle` → 404 via `errors.Is`; served `application/octet-stream`. Three exact
  mounts (`/inclusion`,`/consistency`,`/entries`) share one `proofserve.Handler` via the inner path switch.
  Mutation-proven (extractor `eb.Entries[0]`) then reverted. (Detail in git history at-2026-06-21.) Two
  durable traps survive below.
- **Bundle reads keyed on an absolute index must compute `p := tiles.PartialTileSize(0, bundleIndex,
  size)`, NEVER pass `p == 0` unconditionally.** The final bundle of any non-multiple-of-256 tree (and
  every tree < 256 leaves) is stored only at its partial width; `SQLiteFetcher`'s partial→full fallback
  fires ONLY for `p > 0`, so `p == 0` would 404 a leaf that IS in the accepted tree. `serveEntries` AND
  `serveVerify` both do this; copy it for any future bundle/tile read by absolute index.
- **`serveEntries` accepted-tree guards mirror `serveInclusion` exactly (this is the contract every
  record-facing route must follow):** `LastSize == 0` → 404; **`seq >= LastSize` → 404 "leaf not covered
  by accepted checkpoint"**; bundle-miss `os.ErrNotExist` → 404; `ErrLeafOutOfBundle` → 404; missing/
  non-numeric `index` → 400; non-GET → 405. The index is the absolute leaf **seq**, schema-agnostic
  (ADR-0008) — nothing interpreted. **The `>= LastSize` accepted-tree cap is the contract EVERY
  record-facing route must follow** — `serveRecords`/`ListRecords` now enforces it too (see below).
  `iscc_index` can hold projections ABOVE `LastSize` (ingest writes them before accept; a freeze/fault
  leaves them), so any record route that omits the cap shows unaccepted leaves whose `entries?index=`
  links would then 404.

## HTML record list at `/records` (`serveRecords` + `store.ListRecords`)

- **settled:** the no-JS, newest-first (`seq DESC`), seq-cursor-paginated record list is landed + correct
  (DS shell, no `<table>`, no CDN, unquoted `[data-status=…]` CSS so the negative overlay assert stays
  honest, badge partial reuse, render-into-buffer-then-200; pure store-read, oracle gate N/A; store stays
  a leaf; go.mod/go.sum/schema byte-identical). All three record-list defects are closed (`ListRecords`
  now takes `(…, last, hasFrom, from, n)`; `serveRecords` threads `fs.LastSize` + `hasFrom = fromErr ==
  nil`). (1) `last` caps BOTH the
  `COUNT(*)` total and the page `SELECT` with `AND seq < ?` so the list never shows a leaf the accepted
  checkpoint omits; (2) the page-size clamp moved to `if n > maxPageSize { n = maxPageSize }` BEFORE the
  `int(n)` conversion; (3) `from` is no longer overloaded — `hasFrom` carries present/absent so `from=0`
  is a real seq-0 cursor. `parseUint` gained an overflow reject. All three mutation-proven (reviewer
  re-ran each: drop-ceiling → `total 6 want 4`; post-`int()` clamp → whole 300-row index; re-overload
  → seq 0 unreachable), reverted. Codex concurred (no findings). **Durable lessons for any seq-cursor
  pagination here:** never overload `0` as both a cursor value and a sentinel (carry a `has-from` bool
  or a 1-based/`+1` cursor); clamp page size while still `uint64` BEFORE the `int()` conversion (a huge
  `n` wraps `int(n)` negative and modernc SQLite reads a negative `LIMIT` as UNLIMITED); apply the same
  accepted-tree ceiling to the COUNT and the windowed SELECT (an uncapped total still lies in the
  "showing N of TOTAL" line). `parseUint` had no overflow guard (it wrapped silently in `n = n*10 + …`),
  so it now rejects overflow with its existing error type — every shared caller treats it as a malformed
  value, no behaviour change for empty/non-numeric.

## verify-for-me JSON verdict (`/verify` + `serveVerify`)

- **settled + durable trap:** `/verify` INVERTS the other proof routes' status mapping — an id-shaped
  fault (missing/unknown id, no-accepted-checkpoint, leaf-out-of-tree, tile-not-mirrored) is a
  `200 {verified:false, reason}`, NOT 4xx/5xx; non-200 is reserved for genuine infra faults only (DB
  read error, `CheckpointAt found==false` at accepted size, or a non-`os.ErrNotExist` proof error → 500).
  The `included`/`verified` field is a REAL `proof.VerifyInclusion` against the persisted accepted root
  (oracle gate APPLIES; mutation-proven `included:=true` → `TestVerifyInclusionIsNonVacuous` FAILS,
  reverted). `hub_status` is store-provable only (`frozen` else `verified`). Remember the inverted posture
  for any future verify-for-me surface (the bundle assembler is the stronger client-verifies path).

## HTML log browser at the hub-log root (`GET /` + `serveBrowser`)

- **settled:** the `GET /<domain>/log/` browser is landed + Evidence-Ledger-dressed (DS token/font shell,
  no `<table>`, no CDN URL — matches `/`). Pure store-read render (oracle gate N/A): reads only
  `FollowState` + `CheckpointAt`, base64-Std encodes the root verbatim, render-into-`bytes.Buffer`-then-200
  with post-200 write-drop, `html/template` auto-escape; `proofserve` stays free of
  `internal/metrics`/`internal/dashboard` (`go list -deps` empty); store stays a leaf; go.mod/go.sum
  byte-identical. The status cell renders through the five-status `hubStatusBadge` partial overlaid with
  the in-memory verdict via `proofserve`'s own local `StatusSource` + `overlayStatus` (verbatim
  `dashboard.overlayStatus` precedence; can only flip store-`verified` → `unresolvable`/`unverified`;
  `inactive` unreachable here). (Detail in git history pre/at-2026-06-21.) Three durable traps below.
- **Mux mount trap:** the bare-`/` hub-log root is a 5th `proofserve.Handler` route, but an
  `http.ServeMux` cannot hold both an exact `/` AND a subtree `/` (the subtree pattern `/` IS the bare-`/`
  match). `cmd/iscc-monitor` `hubHandler` makes the `/` slot a tiny dispatch `http.HandlerFunc`: send
  `r.URL.Path == "/"` to proofserve, delegate every deeper path to `tilesserve`; the four exact proof
  mounts still win by most-specific match. `/<domain>/log` (no slash) → 301 to trailing slash.
- **Coverage-honesty status mapping (ADR-0001):** DB error or `CheckpointAt found==false` at the accepted
  size → 500 (real store-inconsistency fault); `LastSize == 0` (followed-but-unpolled) → **200** "No
  accepted checkpoint yet", NEVER a 404 and NEVER a fabricated `(0,"")`. Both `browser.html` branches
  invoke the badge partial so an unpolled hub still shows its honest overlay status.
- **CSS-literal trap (cross-cutting for any DS-dressed SSR surface with a negative `data-status` assert):**
  `TestBrowserRendersInMemoryStatus` asserts the body contains NO `data-status="verified"` (proving the
  overlay won). The DS badge-color/frozen-tint selectors must therefore use the UNQUOTED CSS attribute
  form (`[data-status=verified]`, valid CSS for identifier values), NOT `dashboard.html`'s QUOTED
  `[data-status="verified"]` — the quoted form would emit that literal into the rendered `<style>` and
  falsely fail the negative assert. `dashboard.html` gets away with quoted selectors only because its
  render test has no such negative assertion. Use the unquoted form on any future surface that both
  carries a no-`data-status="X"` assert AND inlines the badge color block.

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
