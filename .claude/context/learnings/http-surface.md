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
- **Default seq is `seqs[0]`, the lowest committed seq** (`SeqsForISCCID` is `ORDER BY seq` ASC →
  deterministic). Explicit `&index=<n>` must equal a committed seq (else 400 via `selectSeq`); non-digit
  → 400; `leafIndex >= size` → 404 (never 500/panic) on a stale size.
- **Dep direction holds: `proofserve → {store, logclient}`, never the reverse** (`net/http` stays out of
  the store/logclient closures — both `go list -deps … | grep proofserve` empty). Proof is built from the
  LOCAL mirror only (`f.ReadTile`), never re-hitting the hub.

## Computed consistency proof HTTP surface (`internal/proofserve` + `CheckpointAt ORDER BY rowid`)

- **settled:** the `/consistency` RFC-6962 proof seam is landed + stable (oracle gate APPLIES, mutation-
  proven by corrupting `encoded[0]`). Durable traps: (1) **`VerifyConsistency(hasher, size1, size2,
  proof, root1, root2)`** — `proof` precedes the two roots, UNLIKE `VerifyInclusion`; (2) degenerate
  `from == 0`/`from == LastSize` → 200 with empty proof (`from == 0` SKIPS the `CheckpointAt(from)` row
  requirement, `from == LastSize` still REQUIRES it). Status: missing/non-numeric → 400; `LastSize==0` →
  404; `from>LastSize` → 400; unrecorded `from`/tile-miss → 404. (Detail in git history pre-2026-06-21.)
- **`CheckpointAt`'s `ORDER BY rowid LIMIT 1` is load-bearing: `id INTEGER PRIMARY KEY` aliases `rowid`
  (monotonic by insertion), so the first-recorded (prior accepted) row wins over a later same-`tree_size`
  contradicting-evidence row** (two roots at one size = two rows = the fork-evidence case). `DESC` would
  pick the fork row — keep ASC. (Mutation-proven `TestCheckpointAtDeterministicOnFork`; git history.)

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

- **settled:** the no-JS, `seq DESC`, seq-cursor-paginated record list is landed + correct (DS shell, no
  `<table>`/CDN, unquoted `[data-status=…]` CSS, buffer-then-200, pure store-read), and the full 4-col
  mockup head `Seq · Type · ISCC-ID · Logged` is COMPLETE: `Logged` = verbatim `RecordRow.NoteTimestamp`
  RFC-3339 (`&mdash;` ENTITY fallback for the NULL-timestamp common case); `Type` = per-row badge mapped
  from the verbatim `note.$schema` (Declaration/Deletion/Unknown record type) via a handler-local
  `recordRowVM` (embeds `store.RecordRow`, adds render-time `Kind`+`KindKey`) since `RecordRow` has no
  `Kind` — projection lives in the VM, NOT a store column (no DB-migration trigger). `recordKind` (label)
  + `recordKindKey` (CSS token) switch on the SAME schema constants so they cannot drift; derive the key
  from the schema, never from the label. All template/VM only, store stays a leaf. (Detail at-2026-06-22.)
  Durable rules this surface bakes in: (1) **a rendered head lists ONLY columns with a data cell** (never
  promise a head a row lacks); (2) badge CSS uses the UNQUOTED `[data-kind=…]` selector (the quoted form
  leaks a `data-kind="…"` literal into `<style>` and trips a negative body assert — same trap as
  `[data-status=…]`); (3) **test grounding uses HARDCODED literal `note.$schema`/`NoteTimestamp`**, NOT the
  package constants — `buildMirror` seeds neither, and a constant-vs-constant test goes vacuous (the open
  `record_test.go` trap); reverting a constant must FAIL the test (reviewer mutation-proven here).
  **Durable lessons for any seq-cursor pagination here:** never overload `0` as both a cursor value and a
  sentinel (carry a `has-from` bool or a `+1` cursor); clamp page size while still `uint64` BEFORE the
  `int()` conversion (a huge `n` wraps `int(n)` negative and modernc SQLite reads a negative `LIMIT` as
  UNLIMITED); apply the accepted-tree ceiling to the COUNT and the windowed SELECT alike (an uncapped
  total lies in the "showing N of TOTAL" line). `parseUint` rejects overflow with its existing error type.
- **Chrome/breadcrumb/head dressing (part-2a, landed):** `Handler`/`serveRecords` now take `domain string`
  + `dashboard.Identity` (resolved ONCE at construction via a local `resolveIdentity` + byte-identical
  `instanceFallback`/`operatorFallback` consts — the now-4× duplication is the tracked `low`). Only the
  `/records` HTML surface reads them; the proof/bytes/verdict routes ignore them. Two durable points: (1)
  the `← <domain> dossier` breadcrumb MUST be the ABSOLUTE site-root `href="/{{.Domain}}"` — the dossier
  is mounted at the site root OUTSIDE the `/log/` subtree, so a relative `../` walk is wrong (the record
  rows are relative because they share the subtree); (2) head name + breadcrumb use the bare `{{.Domain}}`,
  NOT a fabricated display name (`HubSummary` has no `name`) — same constraint-win the dossier head makes.
  Part-2b (pager rework: top+bottom "seq X–Y of Z") and the single-record-page chrome are separate slices.

## HTML single-record page at `/record?index=<seq>` (`serveRecord` + `store.RecordAt`)

- **settled:** the `/record` page is landed + correct — `serveRecord` copies `serveEntries`'
  accepted-tree-capped bundle read verbatim, renders bytes as source of truth (missing `iscc_index`
  projection → 200 "no projection indexed", ADR-0008), kind-label constants hold the FULL wire URIs
  byte-matching `projection_test.go`. (Detail in git history at-2026-06-21.) Two durable traps survive:
  (1) **any surface interpreting `note.$schema` matches the FULL wire URI** (`projection_test.go`/
  `fsck_test.go`), never CLAUDE.md's prose short name; once a real schema URI renders, scope a no-CDN
  `http://` body ban to the template/head region (up to `</style>`), not the verbatim record fields.
  (2) **Tie a schema-match test to GROUND TRUTH, not the constant under test** — `record_test.go`'s
  `schemaForSeq` + `recordKind` switch on the SAME constants, so reverting both leaves the suite green
  (mutation-verified); a future `note.$schema`→label test must seed a HARDCODED literal URI (open `low`).

## Mirrored OTS proof at `/checkpoint.ots` (`serveOTS` + `store.OTSForRoot`)

- **settled:** `GET /checkpoint.ots` serves the mirrored OpenTimestamps proof for the accepted
  `(size, root)` verbatim as `application/octet-stream` (oracle gate N/A — opaque-byte serve of an
  already-stored proof; production proofserve stays OFF the non-WASM `internal/ots`/`internal/otsclient`
  closure, `go list -deps` empty — the `opentimestamps` parse is test-only). Resolution copies
  `serveVerify` (`FollowState.LastSize` → `CheckpointAt`); `writeOTS` mirrors `tilesserve.writeBlob`
  (strong content-ETag + `If-None-Match`→304) but with `Cache-Control: no-cache`, NOT `immutable`: the
  proof is overwritten in place on the pending→Bitcoin-confirmed upgrade, so it must revalidate. All
  four cases mutation-proven (mount removed / sentinel guard dropped / truncated serve / 404→500),
  reverted; Codex clean. `/checkpoint.ots` is an EXACT `mux.Handle` in `hubHandler` (Mux-mount trap) and
  is a DIFFERENT artifact from the raw `/checkpoint` signed-note BLOB tilesserve serves under `/`.
- **The empty-OTSBytes sentinel is the load-bearing edge case:** a row CAN exist for the accepted root
  yet carry zero `OTSBytes` (stamped at observation but not yet calendar-submitted). `found == true` is
  NOT sufficient — guard `!found || len(rec.OTSBytes) == 0` → 404 "root not yet anchored", else a client
  gets an unparseable zero-byte `.ots`. `OTSForRoot` returns a plain miss `(…, false, nil)` for an
  un-anchored root, so a not-yet-stamped root is an honest 404, never a 5xx. `LastSize == 0` → 404 (no
  root to anchor); a `CheckpointAt found==false` at the accepted size is a real store inconsistency → 500
  (same as `serveVerify`).

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
  `FollowState` + `CheckpointAt`, base64-Std root, render-into-`bytes.Buffer`-then-200, `html/template`
  auto-escape; store stays a leaf. Status cell uses the five-status `hubStatusBadge` overlaid via
  proofserve's local `StatusSource` + `overlayStatus` (verbatim `dashboard.overlayStatus` precedence; can
  only flip store-`verified` → `unresolvable`/`unverified`; `inactive` unreachable here). **Dep note (as
  of part-2a):** proofserve now DOES import `internal/dashboard` (for the plain `Identity` struct only) —
  the load-bearing rule is `internal/metrics` stays OUT of the closure and no `net/http`/`database/sql` is
  pulled via dashboard; `database/sql` IS present but pre-existing through `store`. Coverage-honesty
  (ADR-0001): DB error / `CheckpointAt found==false` at accepted size → 500; `LastSize == 0`
  (followed-but-unpolled) → **200** "No accepted checkpoint yet", never 404 / fabricated `(0,"")`.
- **Mux mount trap:** the bare-`/` hub-log root is a 5th `proofserve.Handler` route, but an
  `http.ServeMux` cannot hold both an exact `/` AND a subtree `/` (the subtree pattern `/` IS the bare-`/`
  match). `cmd/iscc-monitor` `hubHandler` makes the `/` slot a tiny dispatch `http.HandlerFunc`: send
  `r.URL.Path == "/"` to proofserve, delegate every deeper path to `tilesserve`; the four exact proof
  mounts still win by most-specific match. `/<domain>/log` (no slash) → 301 to trailing slash.
- **CSS-literal trap (cross-cutting for any DS-dressed SSR surface with a negative `data-status` assert):**
  `TestBrowserRendersInMemoryStatus` asserts the body contains NO `data-status="verified"` (proving the
  overlay won). The DS badge-color/frozen-tint selectors must therefore use the UNQUOTED CSS attribute
  form (`[data-status=verified]`, valid CSS for identifier values), NOT `dashboard.html`'s QUOTED
  `[data-status="verified"]` — the quoted form would emit that literal into the rendered `<style>` and
  falsely fail the negative assert. `dashboard.html` gets away with quoted selectors only because its
  render test has no such negative assertion. Use the unquoted form on any future surface that both
  carries a no-`data-status="X"` assert AND inlines the badge color block.

## CORS middleware (`internal/corsmw`)

- **settled:** `corsmw.Handler(next)` is the single CORS leaf wrapped ONCE at the lone mux convergence
  point (`buildMux` returns `corsmw.Handler(mux)`), so one wrap covers every subtree. Sets
  `Access-Control-Allow-Origin: *` BEFORE delegating (lands on 200/404/405/500 alike — `http.Error`
  freezes the header map); on `OPTIONS` sets `Allow-Methods`/`Allow-Headers`, writes 204, and returns
  WITHOUT calling `next` (else inner GET-only handlers 405 the preflight). Wildcard `*` is correct + NO
  `Allow-Credentials` (public, credential-free, read-only data; browser rejects creds paired with `*`).
  stdlib-only closure; oracle gate N/A. (Detail in git history at-2026-06-22.)
