<!-- area: internal/tilesserve, internal/proofserve, internal/corsmw, cmd/iscc-monitor (/healthz) -->
<!-- indexed-as: http-surface.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# HTTP read surface — tilesserve, proofserve, corsmw

Read this when a step touches the area above. Durable cross-cutting rules live in
the index (`.claude/context/learnings.md`); the package-local mechanics are here.

## tlog-tiles HTTP read surface (`internal/tilesserve/handler.go`)

- **settled:** the static-BLOB mirror (`/checkpoint`, `/tile/...`, `/tile/entries/...`) is landed +
  stable — opaque-byte read transport (oracle gate N/A), `tilesserve → store` (store a leaf),
  `strings.Cut` routing with `tile/entries/` before `tile/`, per-route `Cache-Control` keyed on `width`
  (`immutable := width==0`, partials `no-cache`), strong content ETag + `If-None-Match` → 304; all
  mutation-proven, reverted. (Detail in git history pre-2026-06-21.) Durable trap: the path-API `width`
  vocab (`0 == full`) differs from the store column's `256 == full`, so map deliberately.

## Computed inclusion proof HTTP surface (`internal/proofserve/handler.go`)

- **settled:** the `/inclusion` RFC-6962 proof seam is landed + stable; oracle gate APPLIES and was
  mutation-proven non-vacuous (`TestInclusionServedProofVerifies`; `proof=nil`/`leafIndex+1` both FAIL),
  `notecheck`/`derive_vkey.py` correctly N/A (served object omits `checkpoint`; client refetches it).
  (Detail in git history pre-2026-06-21.) Durable trap: strip discipline is `"/"+Origin`, NOT `+"/"`, so
  the inner `ServeMux` keeps its leading slash, exact mounts beat the `/` subtree, and a nested mux does
  not 301-redirect.
- **Default seq is `seqs[0]`, the lowest committed seq** (`SeqsForISCCID` is `ORDER BY seq` ASC, so the
  "first committed seq" default is deterministic). An explicit `&index=<n>` must equal a committed seq
  (else 400 via `selectSeq`); `parseUint` rejects non-digits → 400; `leafIndex >= size` → 404 (never
  500/panic) on a stale/racing size.
- **Dep direction holds: `proofserve → {store, logclient}`, never the reverse.** `go list -deps
  ./internal/store | grep -E 'proofserve|net/http'` and `go list -deps ./internal/logclient | grep
  proofserve` both empty, so `net/http` stays out of the store/logclient closures. Proof is built from
  the LOCAL mirror only (`f.ReadTile`), never re-hitting the hub.

## Computed consistency proof HTTP surface (`internal/proofserve` + `CheckpointAt ORDER BY rowid`)

- **settled:** the `/consistency` RFC-6962 proof seam is landed + stable (oracle gate APPLIES, mutation-
  proven by corrupting `encoded[0]`). Durable traps: (1) **`VerifyConsistency(hasher, size1, size2,
  proof, root1, root2)`** — `proof` precedes the two roots, UNLIKE `VerifyInclusion`; (2) degenerate
  `from == 0`/`from == LastSize` → 200 with empty proof (`from == 0` SKIPS the `CheckpointAt(from)` row
  requirement, `from == LastSize` still REQUIRES it). Status: missing/non-numeric → 400; `LastSize==0` →
  404; `from>LastSize` → 400; unrecorded `from`/tile-miss → 404. (Detail in git history pre-2026-06-21.)
- **`CheckpointAt`'s `ORDER BY rowid LIMIT 1` is correct because `id INTEGER PRIMARY KEY` aliases `rowid`
  in SQLite — rowid is monotonic by insertion, so the first-recorded (prior accepted) row wins over a
  later same-`tree_size` contradicting-evidence row** (`RecordCheckpoint` dedupes on
  `UNIQUE(hub_id, tree_size, root)`, so two roots at one size are two rows = the fork-evidence case).
  Mutation `DESC` → `TestCheckpointAtDeterministicOnFork` FAILS, reverted. Store stays a leaf.

## Computed record-bytes HTTP surface (`/entries` + `internal/logclient/entries.go`)

- **settled:** `GET /entries?index=<seq>` is a pure decode + index, NOT crypto
  (`RecordBytesFromBundle` is WASM-pure; `application/octet-stream`); the exact mounts share one
  `proofserve.Handler` via the inner path switch. Mutation-proven, reverted. (Detail at-2026-06-21.)
- **Bundle reads keyed on an absolute index must compute `p := tiles.PartialTileSize(0, bundleIndex,
  size)`, NEVER pass `p == 0` unconditionally.** The final bundle of any non-multiple-of-256 tree (and
  every tree < 256 leaves) is stored only at its partial width; `SQLiteFetcher`'s partial→full fallback
  fires ONLY for `p > 0`, so `p == 0` would 404 a leaf that IS in the accepted tree. `serveEntries`,
  `serveVerify`, AND `serveRecord` all do this; copy it for any future bundle/tile read by absolute index.
- **The `>= LastSize` accepted-tree cap is the contract EVERY record-facing route must follow:**
  `LastSize == 0` → 404; `seq >= LastSize` → 404 "leaf not covered by accepted checkpoint"; bundle-miss
  `os.ErrNotExist` → 404; `ErrLeafOutOfBundle` → 404; missing/non-numeric `index` → 400; non-GET → 405.
  Index is the absolute leaf **seq**, schema-agnostic (ADR-0008). `serveEntries`/`serveRecord` apply it
  to one leaf; `serveRecords`/`ListRecords` apply it to the COUNT + windowed SELECT. `iscc_index` can
  hold projections ABOVE `LastSize` (ingest writes them before accept; a freeze/fault leaves them), so
  any record route that omits the cap shows unaccepted leaves whose `entries?index=` links 404.

## HTML record list at `/records` (`serveRecords` + `store.ListRecords`)

- **settled:** the no-JS, newest-first (`seq DESC`), seq-cursor-paginated record list is landed + correct
  (DS shell, no `<table>`, no CDN, unquoted `[data-status=…]` CSS, badge partial reuse, buffer-then-200;
  pure store-read, oracle gate N/A; store stays a leaf; byte-identical go.mod/go.sum/schema). All three
  record-list defects mutation-proven closed, reverted; Codex concurred. (Detail at-2026-06-21.)
  **Durable lessons for any seq-cursor pagination here:** never overload `0` as both a cursor value and a
  sentinel (carry a `has-from` bool or a `+1` cursor); clamp page size while still `uint64` BEFORE the
  `int()` conversion (a huge `n` wraps `int(n)` negative and modernc SQLite reads a negative `LIMIT` as
  UNLIMITED); apply the accepted-tree ceiling to the COUNT and the windowed SELECT alike (an uncapped
  total lies in the "showing N of TOTAL" line). `parseUint` rejects overflow with its existing error type.

## HTML single-record page at `/record?index=<seq>` (`serveRecord` + `store.RecordAt`)

- **settled:** the `/record` page is landed + correct. `serveRecord` copies `serveEntries`'
  accepted-tree-capped bundle read verbatim (`p := tiles.PartialTileSize(0, bundleIndex, size)`,
  `RecordBytesFromBundle`, `>= LastSize` → 404, bundle-miss/`ErrLeafOutOfBundle` → 404, missing/
  non-numeric index → 400, non-GET → 405), renders bytes as source of truth (missing `iscc_index`
  projection → 200 "no projection indexed", not 404, ADR-0008), and the kind-label constants now hold
  the FULL wire URIs (`http://purl.org/iscc/schema/iscc-note-0.8.0.json` + `…delete…`), byte-matching
  `projection_test.go:19-20`, so real declarations/deletions label correctly. (Detail at-2026-06-21.)
- **Durable trap for any surface that interprets `note.$schema`:** match the FULL wire URI (see
  `projection_test.go`/`fsck_test.go`), never CLAUDE.md's prose short name. A no-CDN `http://` body ban
  must be scoped to the template/CDN region (head up to `</style>`), not the verbatim record fields, once
  a real schema URI renders into the page.
- **Tie a schema-match test to GROUND TRUTH, not to the constant under test.** `record_test.go`'s
  `schemaForSeq` returns the `schemaDeclaration`/`schemaDeletion` *constants*, and `recordKind` switches
  on the same constants — so reverting both constants to the wrong value leaves the whole suite green
  (mutation-verified). The label test cannot catch a constant regression. Any future test guarding a
  `note.$schema`→label map must seed a HARDCODED literal URI (or compare the constant against the
  `projection_test.go` literal) so the gate is non-vacuous. (Open `low` issue.)

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
