# Issues

Lightweight backlog `define-next` can prioritize. Append entries; `review` deletes resolved ones.

**Format** — one entry per issue:

```
## <short title>
- **Priority:** critical | normal | low
- **Source:** [human] | [review] | [advance]
- **What / where / how to verify:** <the problem, its location, and the check that proves it fixed>
- **Spec:** <optional — target.md or an ADR section this is rooted in>
```

**Priority semantics:** `critical` preempts everything; `normal` is weighed against the state→target
gap; **`low` is skipped by the loop** (reserved for human-directed work). The `Source` tag records who
filed it and does **not** affect priority.

---

## Nil-Stamper + an empty-OTSBytes row falls through to the Upgrader instead of being left untouched
- **Priority:** low
- **Source:** [review] (Codex P3, reviewer-confirmed by probe)
- **What / where / how to verify:** `OTSTick`'s stamp guard is `if stamper != nil && len(r.OTSBytes) == 0`
  (`internal/follower/otsloop.go:144`). When `stamper == nil` AND a pending row has the empty-OTSBytes
  sentinel, the guard is false, so execution falls through to `up(ctx, r)` with empty bytes — the real
  Upgrader (`recoverRead`) parses empty bytes → error → a bogus back-off (`MarkOTSAttempted`,
  Attempts++), rather than leaving the row untouched. This contradicts the `OTSTick` docstring's
  nil-tolerant claim ("A nil Stamper skips stamping, leaving the row empty"). Reviewer-confirmed by a
  throwaway probe: nil Stamper + empty row → `upCalled == true`, `Attempts == 1` after the tick. NOT a
  production hazard — `stampFunc()` always wires a non-nil Stamper, so the live loop never hits this; it
  is a docstring-vs-code contract mismatch on the test-only nil path. Fix when `otsloop.go` is next
  touched: handle `len(r.OTSBytes) == 0` FIRST and `continue` when `stamper == nil` (skip the row), so the
  nil-tolerant contract the docstring states actually holds. Verify fixed: a test with a nil Stamper + an
  empty-OTSBytes row asserts the Upgrader is NOT invoked and the row's Attempts stays 0; reverting the
  guard reorder makes it FAIL. Low — production wires a non-nil Stamper, the suite is green.
- **Spec:** CLAUDE.md "Write evergreen comments that describe the current state" (docstring must match
  behavior); next.md Implementation Note "Prefer nil-tolerant, mirroring the Loop's nil-Logger discipline".

## No on-disk DB migration story — a column added to an existing table never reaches a pre-existing database
- **Priority:** normal
- **Source:** [review] (Codex P1, reviewer-confirmed against `store.Open`; codebase-wide pre-existing gap)
- **What / where / how to verify:** `store.Open` (`internal/store/sqlite.go:72`) applies the embedded
  `schema.sql` as one `db.Exec(schemaSQL)` whose every statement is `CREATE TABLE IF NOT EXISTS` (9
  tables, ZERO `ALTER TABLE`, no `PRAGMA user_version`, no migration framework — reviewer grep-confirmed).
  So a column added to an EXISTING table (here `iscc_index.note_timestamp`, but this applies to EVERY
  column ever added: `note_schema`, `record_sha256`, the OTS columns, the freeze columns, …) is a silent
  no-op on a database created before that commit. An upgraded node opening such a DB would then fail the
  new INSERT/SELECT paths with `no such column: note_timestamp`. This is **not a regression of the
  timestamp slice** — it is a pre-existing, codebase-wide property: every prior column landed the same
  way, and `next.md` Not-In-Scope explicitly chose it (dev DBs are ephemeral; no schema-versioning
  framework). Does NOT block this increment (fresh DBs — the only deployed kind so far — get the column;
  all gates green). It becomes a real operational hazard the first time the monitor needs an in-place
  upgrade over a populated production DB. Fix when a deliberate migration step is scheduled: introduce a
  `PRAGMA user_version`-gated (or `ALTER TABLE ADD COLUMN`-idempotent) migration mechanism applied on
  `Open` AFTER the `CREATE TABLE IF NOT EXISTS` pass, covering all post-bootstrap columns; this is a
  design decision (it is the project's FIRST migration mechanism), not a field-slice side effect. Verify
  fixed: opening a DB seeded with the pre-`note_timestamp` `iscc_index` DDL then running an ingest +
  `RecordAt` succeeds (column auto-added), with a test that seeds the old schema and asserts no
  `no such column` error.
- **Spec:** ADR-0007 one-file-per-network store; CLAUDE.md "Irreplaceable evidence" (a populated prod DB
  that cannot be upgraded in place is a backup/continuity risk); `next.md` Not-In-Scope migration note.

## Single-record label test is vacuous on the kind-label constant value
- **Priority:** low
- **Source:** [review] (mutation-found in the constant-fix review)
- **What / where / how to verify:** The constant-fix advance set `schemaDeclaration` /
  `schemaDeletion` (`internal/proofserve/handler.go:111-112`) to the correct full wire URIs — verified
  byte-equal to the golden `internal/logclient/projection_test.go:19-20` — so the production feature is
  CORRECT. But the guarding test cannot prove it: `record_test.go`'s `schemaForSeq` (lines 40-52)
  returns the `schemaDeclaration`/`schemaDeletion` *constants*, and `recordKind`
  (`handler.go:838-847`) switches on the *same constants*, so reverting BOTH constants to the old short
  forms leaves the entire proofserve record suite GREEN (reviewer mutation-verified: both reverts →
  `go test -run TestRecord ./internal/proofserve` still `ok`). The test is tied to the symbol under
  test, not to ground truth, so it would not catch a future regression of the constant value. Fix when
  `record_test.go` is next touched: make `TestRecordKindLabels` (or a sibling) seed a HARDCODED literal
  URI (`"http://purl.org/iscc/schema/iscc-note-0.8.0.json"` / `…delete…`) — or assert the constants
  equal those literals — so the gate is non-vacuous. Verify fixed: reverting either constant to a short
  form makes a proofserve test FAIL. Low — the production code is already correct; this only hardens the
  regression gate.
- **Spec:** target.md M-UI single-record Verify criterion; CLAUDE.md Testing ("tests covering
  implemented functionality" + use ground-truth data, not fixtures matched to the code).

## `-run TestOTS` does not catch the OTS store tests (`TestMarkOTSAttempted*`, `TestPendingOTS` back-off)
- **Priority:** low
- **Source:** [review] (reviewer-confirmed against next.md Verification line)
- **What / where / how to verify:** `next.md`'s Verification explicitly required naming the new tests so
  `go test -count=1 -run TestOTS ./internal/store ./internal/follower` catches them all ("name the new
  tests `TestOTS…` / `TestMarkOTSAttempted` / `TestPendingOTSBackoff` so this filter catches them all —
  the filter caveat the prior OTS review flagged"). The advance instead named the store tests
  `TestMarkOTSAttempted`, `TestMarkOTSAttemptedAbsent`, `TestMarkOTSAttemptedZeroNextRetryNull` and put
  the back-off assertions inside the existing `TestPendingOTS` — none of which match the `TestOTS` prefix.
  Reviewer-confirmed: `go test -v -run TestOTS ./internal/store` runs ONLY `TestOTSForRootAbsent`; the
  three `TestMarkOTSAttempted*` and the `TestPendingOTS` back-off path are silently skipped by that
  filter. The tests DO exist, are non-vacuous (reviewer reproduced the next_retry-filter + no-op-
  `MarkOTSAttempted` mutations), and run+pass under `mise run check` and the broader
  `-run 'TestOTS|TestMarkOTSAttempted|TestPendingOTS'` union — so this is a developer-convenience /
  spec-literal gap, NOT a coverage hole and NOT a gate weakening. The follower tests (`TestOTSTick*`,
  `TestOTSBackoff`) DO match the prefix. Fix when the OTS store tests are next touched: rename
  `TestMarkOTSAttempted*` → `TestOTSMarkAttempted*` (or add a `TestOTSPendingBackoff` wrapper) so the
  documented `-run TestOTS` shorthand catches the whole suite. Verify fixed: `go test -v -run TestOTS
  ./internal/store` lists every OTS store test. Low — the suite is green and complete under `mise run
  check`; only the shorthand filter under-selects.
- **Spec:** next.md Verification "name the new tests … so this filter catches them all"; CLAUDE.md
  Testing (clean, discoverable test output).

## `cmd/notecheck`'s `run` has a vestigial `out io.Writer` parameter
- **Priority:** low
- **Source:** [review]
- **What / where / how to verify:** `cmd/notecheck/main.go` `run(vkey string, in io.Reader, out
  io.Writer) (string, error)` never writes to `out` — it returns the signer name and `main` prints
  `OK %s` to `os.Stdout` itself. The param matches the literal signature `next.md` specified and is
  harmless (tests pass a throwaway buffer; `go vet` does not flag unused params), but the signature
  is misleading. Fix when `run` is next touched: drop `out`, OR have `run` print `OK %s` to `out` and
  let the test assert on it. Verify fixed: `out` is either gone or written to. Low — skipped by the loop.
- **Spec:** KISS / YAGNI (CLAUDE.md code standards); no spec contract.

## Hub-status overlay precedence is duplicated across dashboard, proofserve, AND dossier (now 3x)
- **Priority:** low
- **Source:** [review] (architecture review)
- **What / where / how to verify:** `internal/dashboard/handler.go:161-190`,
  `internal/proofserve/handler.go:600-635`, AND now `internal/dossier/handler.go:742-787` all implement
  the same five-status `overlayStatus` + `hubStatus` precedence verbatim — every docstring confesses it.
  The third copy landed with the hub dossier (deliberately, per its `next.md` Not-In-Scope), so the
  consolidation pressure is now 3x: a precedence fix is a three-site edit. Still `low` (no progress
  gate), but the move is more valuable now. The `internal/badge` package owns *rendering*
  the five statuses (silhouette + the single-source label table) but not *resolving* them, so the
  ADR-0010 visual-contract precedence (frozen/inactive are durable truths that win; only `verified`
  consults the live verdict, and only to adopt `unresolvable`/`unverified`) lives in two places keyed on
  two input types (`store.HubSummary`, which carries `.Active` → `inactive`, vs `store.FollowState`,
  which does not). A precedence fix in one silently diverges from the other; the log-browser overlay has
  no HTTP-seam test of its own. Deepen by moving resolution into `badge` (already the taxonomy owner) as
  one `Resolve(provable-status, live-verdict) → status` both handlers cross; the inactive case (only
  `HubSummary` has it) is decided before the seam so the resolver stays one function. Verify fixed: the
  overlay precedence exists in exactly one place, both handlers call it, and one test covers the
  five-status taxonomy. Pure locality deepening — contradicts no ADR.
- **Spec:** ADR-0010 five-status `HubStatusBadge` visual contract; CLAUDE.md "Hub status" glossary.

## Mirror write path leaks tile coordinates and the partial-`p` convention into the follower
- **Priority:** low
- **Source:** [review] (architecture review)
- **What / where / how to verify:** `internal/follower/ingest.go:58-83` walks `tiles.TileCoords` /
  `tiles.BundleCoords` and passes each coord's `Partial` (the tlog-tiles p qualifier) straight to
  `store.RecordTile` / `store.RecordEntryBundle`, then projects via `logclient.BundleProjections` →
  `store.RecordProjections`. The "Mirror" (glossary: the complete copy of a hub's tiles + entries) has
  no single owner — coordinate enumeration, the p→width convention, projection, and BLOB writes are
  split between the follower's ingest path and the store's 22-method CRUD surface, so the follower must
  learn the tile layout to drive storage. Deepen by absorbing the walk + p + projection + writes behind
  one deep Mirror seam (e.g. `Sync(hubID, treeSize, fetcher)`); the follower stops referencing tile
  coordinates and the store's per-tile methods go private behind it. Verify fixed: `ingest.go` no longer
  references `tiles.*Coords` or a `Partial` qualifier, and the mirror round-trip is tested through the
  single Mirror interface. NOTE: this is **not** "add a store interface" — there is exactly one SQLite
  adapter (ADR-0005), so that would be a hypothetical seam with one adapter; Mirror still writes to the
  same SQLite store and `store.SQLiteFetcher` stays its read side. Larger move — wants a design/grilling
  pass before building.
- **Spec:** ADR-0005 single SQLite store; CLAUDE.md "Mirror" glossary.

## Add a scaling trip-wire: writer-wait time + per-network DB file size metrics
- **Priority:** low
- **Source:** [human]
- **What / where / how to verify:** The single-file-per-network store (ADR-0007) is right for
  10s–100s of hubs, but two axes can eventually bind: the single writer (`SetMaxOpenConns(1)`,
  `internal/store/sqlite.go`) serializing all hubs' poll-commits, and per-network file size (one
  high-traffic hub at millions/day bloating the shared file). Expose two Prometheus metrics via the
  existing `internal/metrics` registry so the bind is visible *before* it hurts, not discovered under
  load: (1) writer-wait / commit latency — how long a poll-commit waits on or holds the single
  connection (a rising p99 is the writer-contention signal); (2) per-network DB file size in bytes
  (e.g. `os.Stat` on the `.db` file, refreshed per poll cycle). Both feed the "revisit per-hub files
  or rebuildable-bulk tiering" decision recorded in ADR-0007. Verify fixed: `GET /metrics` exposes a
  writer-wait/commit-latency series and a DB-file-size gauge, both labelled per network, with a test
  asserting they appear. Low — skipped by the loop; reserved for when load planning resumes.
- **Spec:** ADR-0007 "Why network-level and not hub-level" (the trip-wire it names); CLAUDE.md
  `GET /metrics` surface.

## proofserve repeats the `os.ErrNotExist`→404 mapping that the sibling tilesserve already centralised
- **Priority:** low
- **Source:** [review] (architecture review)
- **What / where / how to verify:** The three proof routes in `internal/proofserve/handler.go` —
  `serveInclusion` (198-202), `serveConsistency` (287-291), `serveEntries` (349-353) — each spell out the
  same `errors.Is(err, os.ErrNotExist) → 404 "… not mirrored", else → 500` mapping inline. The sibling
  `internal/tilesserve/handler.go:145-152` already lifts this into one `writeReadError(w, err)` reused by
  all its routes; a change to the not-mirrored mapping is a three-site edit in proofserve. Fix: lift one
  proofserve-local `writeReadError` and call it from the three proof routes. **Exclude `serveVerify`** —
  it deliberately maps a not-yet-mirrored tile/bundle (`os.ErrNotExist`) to a 200 verdict
  (handler.go:399-400), not a 404, so it must NOT share the helper. Verify fixed: the not-mirrored→404
  mapping for the three proof routes lives in one helper and `serveVerify`'s 200 behaviour is unchanged.
  Cosmetic locality only.
- **Spec:** `internal/tilesserve` `writeReadError` pattern; no spec contract.

## `noExternalCDN`'s `stripLineComments` still strips whitespace-prefixed protocol-relative CDN URLs
- **Priority:** low
- **Source:** [review] (Codex P2, reviewer-confirmed by probe)
- **What / where / how to verify:** The quoted-delimiter over-strip (`src="//cdn..."`, `url("//cdn...")`)
  is now CLOSED — `stripLineComments` (`internal/web/web_test.go:57`) treats `//` as a comment only at
  line-start or when preceded by whitespace, so a `"`-preceded protocol-relative URL survives and trips
  the ban (`TestNoExternalCDNProtocolRelative`, mutation-proven). Codex flags the residual narrower case:
  a `//` preceded by **whitespace** is still stripped, so the (rare, mostly-invalid HTML / valid-but-odd
  CSS) whitespace-before-URL forms `<script src = //cdn.jsdelivr.net/x.js>` and `url( //cdn.example/x.woff2)`
  are truncated before `cdn.`/`jsdelivr` and the ban misses them. Reviewer-confirmed by probe (both forms
  → `cdn.present=false`). This is **not a regression**: the prior `:`-only guard stripped these same forms
  too (reviewer-verified), and NO served asset (tokens.css/fonts.css/byte-verbatim `wasm_exec.js`) uses a
  whitespace-prefixed protocol-relative URL — the hole is latent, same class as before, and strictly
  narrower than what this advance fixed. Does NOT block progress: the increment's stated goal (the
  quoted-delimiter `//cdn.` trips the ban) is fully met, all gates green, and the gate is strictly
  stronger than its prior state. Low because the form is not realistic in a hand-authored asset and the
  loop skips lows; promote only if a real asset needs a whitespace-tolerant URL. Fix when the helper is
  next touched: a tokenizer-grade check (treat `//` as a comment only OUTSIDE a quoted string / `url(...)`
  token), not another preceding-byte blocklist — a per-delimiter list will keep losing edge forms. Verify
  fixed: a test feeds `noExternalCDN` `<script src = //cdn.jsdelivr.net/x.js>` and `url( //cdn.example/x)`
  and asserts the ban FIRES for both; reverting the tokenizer makes them pass (regress).
- **Spec:** target.md M-UI hard CDN-free constraint; `learnings/web.md` `noExternalCDN` bans third-party
  origins; CLAUDE.md "Never weaken a quality gate to pass" (the fix is the root cause, not the gate).

## `/` realm-index: only the "recent declarers checked" hero footer remains vs the mockup (logo, instance-identity, Checkpoint/Anchor all CLOSED)
- **Priority:** normal
- **Source:** [review] (visual pass vs the Realm-Index mockup, after the named-region parity landed)
- **What / where / how to verify:** The three HEADLINE landmark regions (claim-lookup hero, per-row
  dossier link, instance-identity masthead) now render and the lone `critical` is closed — but the
  ADR-0012 visual pass against `.claude/design/ISCC Monitor - Realm Index.dc.html` shows four remaining
  sub-region deltas the parity step deferred as constraint-wins (all flagged in that handoff):
  (1) **No logo** — **CLOSED (reviewer-confirmed `6a442b4`).** The self-hosted logo now renders on ALL SIX
  SSR mastheads (`/`, dossier, certificate, the three proofserve surfaces); the ADR-0012 visual pass shows no
  remaining "no logo" delta on any surface. No further action — kept here only as a resolved sub-item record.
  (2) **Static instance identity + realm name** — **CLOSED for `/` (reviewer-confirmed `b30b84e`).** The
  `/` masthead now renders three operator-supplied strings (`dashboard.Identity{Instance, Operator, Realm}`)
  flowing env → binary → page via `ISCC_MONITOR_INSTANCE` / `ISCC_MONITOR_OPERATOR` / `ISCC_MONITOR_REALM_NAME`
  (the last distinct from config's required realm-document PATH `ISCC_MONITOR_REALM`), with handler-side
  fail-safe fallback to today's static copy when unset. Visual pass confirms the live binary renders
  `monitor.iscc.id` / "instance operated by ISCC Foundation · ISCC mainnet" / "REALM REGISTER · ISCC MAINNET"
  matching the mockup; mutation-proven non-vacuous (template binding + wiring). The SAME `dashboard.Identity`
  value still needs threading into the OTHER five SSR mastheads (dossier, certificate, the three proofserve
  surfaces) — tracked separately as the follow-on arc; `internal/verifier` stays excluded (`.codes` chrome).
  Kept here only as a resolved sub-item record for the `/` surface.
  (3) **Checkpoint-size + Bitcoin-anchor data columns** — **CLOSED (reviewer-confirmed `b74931f`).** The
  `/` ledger now renders the mockup's six columns `# | Hub · domain | Coverage since | Checkpoint | Anchor
  | Status`; `store.HubSummary` gained a read-only `Anchor` projection (latest-stamped-root OTS status via
  a correlated subselect) and the Checkpoint cell re-purposes the accepted `LastSize`. Visual pass vs the
  mockup confirms column order + naming; mutation-proven non-vacuous; store stays a leaf. The per-hub
  (vs per-checkpoint) honesty design question Codex raised is its own `normal` below — kept here only as a
  resolved sub-item record.
  (4) **"Recent declarers checked" hero footer omitted** — needs a recent-lookup history the store does
  not track. None of these block progress (the headline-region parity Verify criteria are met); they are
  the named sub-steps to finish full `/` design-parity at the M-UI exit. Verify fixed: the served `/`
  carries config-driven instance identity + realm name and honest Checkpoint/Anchor columns (the logo is
  verified in its own extracted critical issue); the visual pass files no remaining sub-region delta.
- **Spec:** target.md M-UI design-parity "named-region" bar (the `/` realm-index region) + "Document chrome
  + instance identity"; ADR-0010 Evidence-Ledger handoff; ADR-0012 visual-pass.

## The WASM verifier never checks the checkpoint signature against the hub's did:web key (the signature half of the verifier-scope trust gap; id-binding half now CLOSED in source)
- **Priority:** normal
- **Source:** [review] (Codex P1, reviewer-confirmed against the verify core; affects BOTH tier-2 callers)
- **What / where / how to verify:** UPDATE: the **id-binding half** of this gap is now CLOSED IN SOURCE
  (advance `22f0420`): `verifyadapter.RecordCommitsID` binds the record's committed `iscc_id` to the
  requested id, and the 6-arg `isccVerifyInclusion` shim gates `verified` on it (the cross-origin
  `verifier.html:628` passes `target.id`). What REMAINS open is the **signature half**:
  `isccVerifyInclusion` (`VerifyJSON` → `internal/proof/verify.VerifyInclusion`) still verifies ONLY
  that `record`+`proof`+`size`→`root` (RFC-6962 inclusion) and the id-binding — it does NOT verify the
  checkpoint note signature against the hub's did:web key. So a malicious/compromised monitor can still
  return a bundle whose record+proof+root are internally consistent under a FORGED (unsigned / wrong-key)
  checkpoint and — provided the record commits the requested id — the browser renders the green
  `verified` state, trusting the monitor for the signature. The copy overstates this: `verifier.html:449`
  lists "Check the signature against the hub's did:web key" as a step the verifier WILL run, and
  `verifier.html:631` reports "✓ … re-verified this inclusion proof against the **hub-signed** checkpoint
  root" — but neither the signature nor a did:web resolution runs. This is the SAME verifier-core scope
  the certificate's same-origin tier-2 ships (`cert.html:565`), so it is NOT a regression and does NOT
  block progress; but it is more serious cross-origin. NOT currently exploitable on the testnet (the
  fixture monitor is honest). It needs a DESIGN PASS (browser did:web resolution + note-signature verify)
  — review flagged it as the design-first remainder. Fix when the WASM verifier scope is next expanded:
  extend the verifier (or a sibling export) to verify the checkpoint note signature against a did:web key
  fetched/resolved in the browser, gating `verified` on signature + id-binding + inclusion; until then,
  narrow the success copy + drop the unrun did:web step from the record block so the page does not claim a
  signature/key check it skips. Verify fixed: a bundle with a valid inclusion proof + matching id but a
  checkpoint signed by a non-did:web key renders `error`/`failed`, NOT `verified`; reverting the added
  signature check makes that test FAIL.
- **Spec:** CLAUDE.md "Verifier app" / "Proof bundle" / "Verifiable cache" (the monitor is NOT in the
  trust path; the client re-verifies signature + Merkle); ADR-0009 did:web is the only key source;
  learnings.md always-loaded "gate a rendered ✓ on a re-VERIFICATION" (a full re-verification includes
  the signature + id binding, not inclusion math alone); `learnings/cmd-wasm.md` `isccVerifyInclusion` scope.

## `pages.yml` actions target deprecated Node 20 — bump to current major versions
- **Priority:** low
- **Source:** [human] (deprecation warning surfaced on Pages run `27965858770`, 2026-06-22)
- **What / where / how to verify:** the live Pages deploy run annotated: "Node.js 20 is deprecated. The
  following actions target Node.js 20 but are being forced to run on Node.js 24: `actions/checkout@v4`,
  `actions/configure-pages@v5`, `actions/setup-go@v5`, `actions/upload-artifact@v4`." The deploy is GREEN
  today (GitHub force-runs them on Node 24), so this does NOT block — but the warning will become a hard
  failure once GitHub removes the Node 20 shim (see github.blog/changelog/2025-09-19-deprecation-of-node-20).
  Fix when `pages.yml` is next touched: bump the pinned action majors to their current Node-24 releases
  (`actions/checkout@v5`, `actions/setup-go@v6`, `actions/upload-artifact@v5`, `actions/configure-pages` +
  `actions/deploy-pages` to their latest), confirming each new major's inputs still match this workflow's
  usage. Verify fixed: a dispatched `pages.yml` run on `develop` is green with NO Node-20 deprecation
  annotation. (Also re-check `ci.yml` for the same pinned actions while there.)
- **Spec:** CLAUDE.md "Building the Surface-C verifier site" (`.github/workflows/pages.yml` is the publish
  workflow); `learnings/ci.md`.

## `cmd/verifier-site` `generate` writes non-atomically — a mid-run error leaves a partial deploy tree
- **Priority:** low
- **Source:** [review] (Codex P3, reviewer-confirmed against the code)
- **What / where / how to verify:** `cmd/verifier-site/main.go` `generate` writes `index.html` first
  (`main.go:66`) and then render-then-writes each `/_ds/` asset in the `for _, p := range paths` loop
  (`main.go:85-93`). If a LATER step fails — a future `/_ds/` path that 404s (the fail-closed branch),
  or a `writeFile` error (e.g. `_ds` already exists as a *file* under a reused `-out`) — `generate`
  returns an error but `index.html` (and any already-written assets) are ALREADY on disk, leaving a
  partially-updated tree in a reused `dist/`. The generator's stated contract ("fails closed … rather
  than writing a partial site", `main.go` docstring + next.md) is honored at the run level (it errors →
  `os.Exit(1)` → CI/`TestGenerate` catches it, so a broken deploy is NEVER silently published), but NOT
  at the output level: the directory itself is left half-written. NOT a current hazard — `TestGenerate`
  uses a fresh `t.TempDir()`, the happy path materializes the full 14-file tree, and the Pages publish
  workflow (next sub-step) gates on the non-zero exit. Reviewer-confirmed by inspection (write-before-
  later-render ordering). Fix when the generator is next touched: stage into a temp dir and
  `os.Rename` it into place on success, OR buffer every handler response (collect all `(path, body)`
  pairs) before the first `writeFile`, so `outDir` is updated atomically. Verify fixed: force a mid-run
  render error (e.g. inject a 404 path) and assert `outDir` is left unchanged (no stale `index.html`);
  reverting the staging makes it FAIL. Low — skipped by the loop; the run-level fail-closed is intact.
- **Spec:** next.md "fail closed so a broken deploy is caught … not in production"; `main.go` docstring
  ("errors rather than writing a partial site"); learnings.md always-loaded fail-closed discipline;
  `learnings/verifier-site.md` non-atomic-output note.

## Realm-index `/` Anchor column is per-hub (latest-stamped root), not tied to the displayed Checkpoint — a design-honesty question for the M-UI exit
- **Priority:** normal
- **Source:** [review] (Codex P2, reviewer-triaged — implementation is spec-faithful; the design question is real)
- **What / where / how to verify:** The `/` ledger Anchor cell (`internal/store/hubs.go:50-51` +
  `internal/dashboard/handler.go anchorLabel`) projects the hub's LATEST-STAMPED-root OTS status
  (`SELECT o.status … ORDER BY o.stamped_at DESC, o.id DESC LIMIT 1`), while the same row's Checkpoint
  cell shows `f.last_size` (the accepted tree size). The two are DECOUPLED: nothing ties the chosen OTS
  row's `tree_size`/`root` to the displayed checkpoint. Codex's framing (it could render "confirmed" for
  an older root while the newer checkpoint is shown, "overstating current anchoring") is technically
  accurate but is the EXPECTED steady state, not a defect: OTS is async/best-effort (ADR-0004, never
  blocks the poll), so the displayed checkpoint is almost always AHEAD of the latest Bitcoin-confirmed
  anchor. The column is — by design AND by the mockup (`anchorState` is a free-standing per-hub property,
  `.dc.html:88-103`) — a per-HUB "this hub anchors its roots" indicator, NOT a per-checkpoint
  attestation. The AUTHORITATIVE per-checkpoint claim already exists in **certificate §5**, which binds
  `OTSForRoot(hubID, treeSize, root)` to the §2 accepted root via `ots.ConfirmedFor`. This increment is
  spec-faithful (matches `next.md`'s "latest-stamped-root" projection + the mockup), all gates green,
  mutation-proven — so it does NOT block progress. The open question for the M-UI exit / a design pass:
  should the realm-index Anchor cell (a) stay a per-hub activity indicator (current, mockup-faithful),
  (b) gain a distinct label that makes the "latest confirmed anchor, not this checkpoint" semantics
  explicit, or (c) tie to `o.tree_size = f.last_size` — but (c) is REJECTED without a design pass because
  it would render "not anchored" for virtually every actively-polling hub (the newest checkpoint is rarely
  confirmed yet) and defeat the column. Verify resolved: the design pass records the chosen semantics and,
  if (b), the realm-index Anchor label distinguishes hub-anchoring-activity from a per-checkpoint claim;
  the certificate §5 per-root surface stays the authoritative per-checkpoint attestation.
- **Spec:** CLAUDE.md "Bitcoin anchoring" (Bitcoin-only meaning) + "Coverage" (never imply a guarantee
  the data does not support); ADR-0004 OTS async/best-effort; ADR-0010 Evidence-Ledger honesty;
  `.claude/design/ISCC Monitor - Realm Index.dc.html` per-hub anchorState model; `learnings/dashboard.md`
  per-hub-vs-per-checkpoint Anchor note; `internal/certificate/handler.go` §5 authoritative per-root surface.

## Masthead identity fallback consts are now duplicated across dashboard + dossier + certificate (3x) instead of one shared resolve leaf
- **Priority:** low
- **Source:** [review] (filed alongside the dossier masthead-identity slice `413efe8`; updated when the cert copy landed `3c64097`)
- **What / where / how to verify:** The masthead-identity arc has now copied `instanceFallback` /
  `operatorFallback` + a private `resolveIdentity` into THREE packages: `internal/dashboard/handler.go:114-115`
  (the original `Identity.resolve` owner), `internal/dossier/handler.go:107-108`, and now
  `internal/certificate/handler.go:174-175` (advance `3c64097`). All are LITERALS byte-identical with a "MUST
  stay byte-identical" comment, because neither package can import the other's unexported consts and exporting
  `dashboard.resolve` would push each slice to a 4th prod file (over the ≤3 budget). This is a documented,
  commented, mutation-proven duplication, not a defect — but a future change to the static masthead copy is now
  a THREE-site edit (FOUR once the proofserve mastheads land) that can silently diverge. Fix when the
  masthead-identity arc finishes across all surfaces: lift `Identity` + the fallback consts + a single exported
  `Resolve` into ONE owner (the `internal/dashboard` package already owns the type, or a tiny new shared leaf)
  that dossier/cert/proofserve all import, so the fallback exists once. Verify fixed: the
  `instanceFallback`/`operatorFallback` literals appear in exactly one package and every masthead resolves
  through it; a test asserting dashboard+dossier+cert render the SAME fallback line passes. Low — the consts are
  currently byte-identical and the duplication is commented; this only removes the divergence risk once the arc
  is complete (best folded WITH the proofserve masthead slice, the natural 4th-copy trigger).
- **Spec:** CLAUDE.md DRY ("Reduce code duplication even if refactoring requires extra effort"); next.md
  Implementation Note (per-package helper chosen to stay ≤3 prod files, consolidation deferred).

## Stale `.chrome-identity` CSS comment in dashboard.html still says "static copy in this skeleton"
- **Priority:** low
- **Source:** [review] (observed during the dossier masthead-identity review)
- **What / where / how to verify:** `internal/dashboard/dashboard.html:75-76` carries the comment "The
  instance-identity block: this deployment's domain + operator/realm. It is static copy in this skeleton (a
  config-driven identity is a separate concern)." — inaccurate since the dashboard masthead became
  config-driven in `b30b84e` (the block now renders `{{.Instance}}`/`{{.Operator}}` from
  `dashboard.Identity`). The dossier slice wrote an ACCURATE comment on its ported copy
  (`internal/dossier/dossier.html`) but correctly left the dashboard untouched (it was out of scope and a
  4th prod file). Violates CLAUDE.md "write evergreen comments that describe the current state". Fix when
  `dashboard.html` is next touched: update the comment to match the dossier's accurate wording. Verify
  fixed: the comment no longer says "static copy in this skeleton". Low — cosmetic; the rendered output is
  already correct.
- **Spec:** CLAUDE.md "Write evergreen comments that describe the current state, not historical changes".

## No public-facing root `README.md` — the project has no human-facing front door
- **Priority:** normal
- **Source:** [human]
- **What / where / how to verify:** The repository root has **no `README.md`** — the only README is the
  CID context pack's `.claude/context/README.md` (loop-internal), and `CLAUDE.md` is agent-facing project
  instructions, not a human overview. So a person landing on the repo (GitHub, a fresh clone, the future
  GHCR image's "source" link) gets no front-door explanation of what iscc-monitor is, how to build it, or
  how to run an instance. Add a tracked root `README.md` that states (1) **what it is** — the independent
  Trust & Transparency service for the ISCC-Hub network: follows every hub's tlog-tiles transparency log,
  verifies Ed25519 signatures + RFC-6962 consistency, mirrors the logs, and publishes verifiable evidence,
  framed honestly as a *verifiable cache*, not a trusted oracle; (2) **the stack** (Go 1.26,
  `CGO_ENABLED=0`, single binary `cmd/iscc-monitor`); (3) **build + run** — a copy-pasteable snippet that
  builds the binary and starts it against the testnet realm (the `ISCC_MONITOR_DB` / `ISCC_MONITOR_REALM` /
  `ISCC_MONITOR_ADDR` / cadence env config), plus the quality gate `mise run check`; (4) **pointers to the
  specs** (`.claude/prd`, `.claude/adr`, the glossary in `CLAUDE.md`). Keep it human-facing and evergreen;
  do **not** duplicate the full env-var table — link `CLAUDE.md` "Running a local dev instance" as the
  authoritative source so the two never drift. Verify fixed: `README.md` exists at the repo root, renders a
  project overview + a build/run snippet that actually starts the binary, and links the spec dirs;
  `target.md` "Done When" requires it, so DONE is not reachable until it exists.
- **Spec:** target.md "Done When" (now requires a root README); CLAUDE.md project overview + "Running a
  local dev instance"; memory `docs-layout-convention` (`.claude/` = agentic docs, public docs elsewhere).

## `.dockerignore` secret/sidecar globs are slashless — they only exclude CONTEXT-ROOT files, not nested ones
- **Priority:** low
- **Source:** [review] (Codex P2, reviewer-confirmed against Docker's `filepath.Match` vs git basename matching)
- **What / where / how to verify:** The `.dockerignore` (advance `a15a9f4`) now lists `.env`, `.env.*`,
  `*.db-wal`, `*.db-shm` (`/workspace/iscc-monitor/.dockerignore:23-36`) — closing the ROOT-level gap — but
  these are SLASHLESS patterns. Docker's `.dockerignore` uses Go `filepath.Match`, where a slashless
  pattern matches ONLY a file directly under the build-context root; `.gitignore`, by contrast, matches the
  basename at ANY depth. Reviewer-confirmed: `git check-ignore` IGNORES `deploy/.env` and
  `data/monitor.db-wal`, but Docker would NOT exclude them — so a nested secret/sidecar (e.g. `deploy/.env`,
  `data/monitor.db-wal`) is still sent to the build stage by `COPY . .`. The intent ("a superset of the
  gitignore's never-commit set") therefore holds only for root-level files. `**/auth.json` already uses the
  correct recursive form. NOT a leak in the shipped artifact (the final stage only `COPY --from=build`s the
  binary, never the context) and NOT a CI issue (a fresh checkout has none of these files) — a latent
  defense-in-depth gap, same class as the now-closed root-level one, strictly narrower. Does NOT block
  progress; all gates green. Fix when `.dockerignore` is next touched: use recursive forms — `**/.env`,
  `**/.env.*`, `**/*.db-wal`, `**/*.db-shm` (mirroring the already-recursive `**/auth.json`) — so the
  exclusion matches the gitignore at any depth. Verify fixed: a throwaway `deploy/.env` /
  `data/monitor.db-wal` in the working tree is NOT in the build context (a test stage `RUN ls` cannot see
  them, or `docker build --progress=plain` shows them excluded).
- **Spec:** repo `.gitignore` "Local secrets / state — never commit"; ADR-0013 server packaging;
  `learnings/ci.md` (`.dockerignore` matching is not `.gitignore` matching).

## `publish.yml` `workflow_dispatch` can push the floating `:develop` tag from a non-develop ref
- **Priority:** normal
- **Source:** [review] (Codex P2, reviewer-confirmed against the workflow)
- **What / where / how to verify:** `.github/workflows/publish.yml` (advance `a15a9f4`) triggers on
  `push: [develop]` AND `workflow_dispatch`, but the `publish` job has NO ref guard — it pushes the
  floating `ghcr.io/iscc/iscc-monitor:develop` tag unconditionally. A maintainer can dispatch
  `workflow_dispatch` against ANY ref (a feature branch, an old commit), and that run would publish the
  selected ref's code as `:develop` — so infra pulling `:develop` could receive non-develop code (the
  immutable `:sha-<short>` tag is unaffected, since it is keyed on the actual SHA). This mirrors the
  in-repo `pages.yml` precedent (intentionally — `next.md` told advance to mirror its shape), so it is a
  pre-existing repo convention, NOT a regression introduced here, and `workflow_dispatch` is
  maintainer-only (not exposed to outside contributors). All gates green; does NOT block progress. Fix
  when the publish/pages workflows are next touched: guard the publish job (or just the `:develop` tag
  step) on `if: github.ref == 'refs/heads/develop'`, so a manual dispatch from a non-develop ref does NOT
  move `:develop` (it could still push only the immutable `:sha-<short>`). Apply the same guard to
  `pages.yml` for consistency (it has the identical unguarded `workflow_dispatch`). Verify fixed: the
  publish job is gated on the develop ref; a `workflow_dispatch` from a feature branch does not update
  `:develop`.
- **Spec:** ADR-0013 server packaging (GHCR publish); the GHCR issue's "`develop` (floating) AND
  `sha-<short>` (immutable)" tag contract — the floating tag must track develop only.

## `deploy/OPERATING.md` quick-start snippets omit the required `ISCC_MONITOR_REALM` — they do not boot
- **Priority:** critical
- **Source:** [review] (Codex P1, reviewer-confirmed against `config.Load` + Dockerfile)
- **What / where / how to verify:** The new operability doc's "State, volume & backup" section
  (`deploy/OPERATING.md:65-68`) claims "a fresh container has a valid `ISCC_MONITOR_REALM` out of the
  box", and the Compose quick-start (`deploy/OPERATING.md:192-193`) and `docker run` snippet
  (`deploy/OPERATING.md:209-215`) both OMIT `ISCC_MONITOR_REALM` with a comment that it "defaults to the
  baked `/etc/iscc-monitor/realm.txt`". This is FALSE: `internal/config.Load` calls `required(get,
  keyRealm)` (`internal/config/config.go:123`), so `ISCC_MONITOR_REALM` is a REQUIRED env var, and the
  Dockerfile only `COPY`s the realm FILE — it sets NO `ENV` (the image has zero `ENV` lines) and no Go
  code defaults `RealmPath`. So both quick-start snippets exit at startup with `config: required key
  "ISCC_MONITOR_REALM" is missing` — the doc's headline "copy-pasteable" deliverable does not boot. This
  blocks closing the persistence `critical` (the doc is its evidence) and is the active step's own
  deliverable. Fix: in BOTH snippets set `ISCC_MONITOR_REALM=/etc/iscc-monitor/realm.txt` explicitly and
  correct the "valid out of the box" sentence (the FILE is baked; the VAR is not) — OR add `ENV
  ISCC_MONITOR_REALM=/etc/iscc-monitor/realm.txt` to the Dockerfile so the "out of the box" claim becomes
  true (then the snippets may legitimately omit it). Verify fixed: a reader copy-pasting either snippet
  gets a config that supplies `ISCC_MONITOR_REALM` (or the Dockerfile sets the `ENV`), and the
  "valid out of the box" sentence matches whichever path was chosen.
- **Spec:** the active step's M-Deploy operability-doc Verify item (the doc must let an operator deploy
  CORRECTLY); `learnings/config.md` "baked realm FILE is not a set realm-config VAR"; the persistence
  `critical` below it is meant to close.

## `deploy/OPERATING.md` quick-start uses a fresh named volume that uid 65532 cannot write
- **Priority:** normal
- **Source:** [review] (Codex P2, reviewer-confirmed against Docker named-volume default ownership)
- **What / where / how to verify:** The doc correctly STATES "The volume directory must be writable by
  uid 65532" (`deploy/OPERATING.md:56-59`), but the quick-start snippets then mount a FRESH Docker named
  volume (`monitor-data:/data`, `deploy/OPERATING.md:197` Compose + `:209-215` `docker run`) without any
  init/chown step. A fresh named volume's mount root is `root:root` `0755` by default, while the image
  runs as the non-root uid 65532 — so `store.Open` cannot create `/data/monitor.db` and fails with a
  permission-denied error. The runnable snippet thus contradicts the requirement the same doc states two
  sections earlier. NOT a code defect — the uid-65532 contract IS the correct design; the gap is that the
  copy-pasteable example does not show how to satisfy it. Fix when the doc is next touched (fold with the
  `ISCC_MONITOR_REALM` critical above): add a one-line note/step on preparing the volume so uid 65532 can
  write it — a pre-`chown 65532:65532` init container / `docker run --user`-aware init, or a bind mount to
  a host dir already owned by 65532 — so the quick start actually boots. Verify fixed: the quick-start
  shows a volume-preparation step (or a 65532-writable bind mount) consistent with the stated uid-65532
  requirement, so a copy-paste does not fail at `store.Open` with permission denied.
- **Spec:** ADR-0013 server packaging (non-root uid 65532); the persistence `critical`'s "(c) the
  uid/permissions the non-root container user needs on the volume dir" line; CLAUDE.md "smallest
  reasonable changes" (the doc must be runnable as written).

---

<!-- The entries below are pre-deployment asks from the iscc-infra ops side, raised
     while preparing a testnet TEST INSTANCE at https://monitor-test.iscc.io on an
     existing DigitalOcean box (Docker Compose + caddy-docker-proxy). They are framed
     as what the deploy needs FROM this repo, not loop-internal defects. Filed 2026-06-22.
     Durable target: these are the work items of milestone **M-Deploy** in target.md
     (ratified in ADR-0013). issues.md is ephemeral; M-Deploy / ADR-0013 are the
     standing spec the loop verifies against — re-derive these if this list is pruned. -->

## Publish a deployable container image to GHCR (Dockerfile + push workflow)
- **Priority:** low
- **Source:** [human] (iscc-infra ops, pre-deploy blocker)
- **STATUS — code-complete (advance `760213b` Dockerfile + `a15a9f4` publish workflow):** the production
  multi-stage `Dockerfile` (static `CGO_ENABLED=0` binary → distroless/static nonroot, non-root uid 65532,
  CA roots, ~28 MB, version-stamped, fail-fast on empty VERSION) AND `.github/workflows/publish.yml`
  (push-to-`develop` + `workflow_dispatch`, `packages: write`, build-push tagging `:develop` + `:sha-<short>`
  with a non-empty `VERSION` build-arg) both exist and are reviewer-verified (Dockerfile via the static-ELF
  build half + the CI `docker` /healthz smoke; publish.yml via YAML-validity + tag/permission/trigger
  inspection — Docker is CI-only on the dev host). What REMAINS is purely iscc-infra repo-settings work,
  explicitly OUT of the loop's scope per `target.md` M-Deploy "Out of the loop's scope": make the GHCR
  package public OR issue infra a `read:packages` token. Demoted to `low` (was `critical`) — the loop has
  delivered everything code-closable; the residual is a one-time human/infra step that does not gate DONE
  here. Kept as a tracking record until the human confirms the package is pullable.
- **What / where / how to verify (original ask):** There was no production Dockerfile (only
  `.devcontainer/Dockerfile`) and no image-publish workflow — `.github/workflows/ci.yml`
  only builds+vets+tests, and `pages.yml` deploys the SEPARATE `.codes` verifier site, not
  the server. iscc-infra deploys via Docker Compose + caddy-docker-proxy and needs a
  *pullable image*, not a source build on the box. Ask: add a multi-stage `Dockerfile` that
  builds the `cmd/iscc-monitor` static binary (Go 1.26, `CGO_ENABLED=0`; it is already
  pure-Go incl. `modernc.org/sqlite`, so a `scratch`/distroless final stage with no libc
  works) running as a NON-root uid, plus a workflow that builds and pushes to
  `ghcr.io/iscc/iscc-monitor` on push to `develop`, tagged BOTH `develop` (floating) and
  `sha-<short>` (immutable, so infra can pin a known-good build and roll back). Make the
  GHCR package public, or hand infra a `read:packages` token. The server image is
  self-contained: it embeds and serves its own `/_ds/` assets incl. `verify.wasm`
  (`internal/web`), so it needs NEITHER the Pages site NOR any CDN at runtime. Fold in a
  build stamp — pass the git SHA via `-ldflags` and surface it (on `/healthz` JSON or a tiny
  `GET /version`) so infra can confirm exactly which build is live. Verify fixed:
  `docker run ghcr.io/iscc/iscc-monitor:develop` with the required env starts and serves
  `/healthz` = 200; `docker image inspect` shows a non-root user and a small (<~30 MB)
  image; the running git SHA is reported by the binary.
- **Spec:** ADR-0003 `CGO_ENABLED=0` static build; CLAUDE.md "single binary configured
  entirely through environment variables".

## Persistence contract for the SQLite DB volume + acknowledge the in-place migration hazard
- **Priority:** critical
- **Source:** [human] (iscc-infra ops, stateful deploy)
- **What / where / how to verify:** `ISCC_MONITOR_DB` is the single SQLite file holding the
  whole network's state, including what the glossary calls *irreplaceable evidence*
  (observed checkpoints, split-view pairs, OTS proofs). For a persistent deploy infra needs,
  documented: (a) the recommended in-container path to back a named Docker volume with (the
  `.db` plus its `-wal`/`-shm` siblings if WAL is on); (b) confirmation that consistently
  backing up that one file captures all durable state; (c) the uid/permissions the non-root
  container user needs on the volume dir. Separately, this repo's own backlog already carries
  **"No on-disk DB migration story"** (normal) — `store.Open` is `CREATE TABLE IF NOT EXISTS`
  only, so a column added in a later image silently never reaches a pre-existing DB and the
  new code path fails with `no such column`. For a *throwaway testnet* instance we can accept
  "recreate the volume on schema change", but I want that acknowledged as the operating
  assumption until the migration mechanism lands — otherwise the first `:develop` image bump
  over a populated volume breaks the instance. Verify fixed: docs state the DB path + volume
  + "back up this one file" contract and the non-root uid, and link the migration issue as
  the known constraint with "recreate volume on schema change" as the interim policy.
- **Spec:** ADR-0007 per-network DB + evidence retention; ADR-0005 single SQLite store;
  existing issue "No on-disk DB migration story".

## Decide which routes are safe to publish at the public vhost (especially /metrics)
- **Priority:** critical
- **Source:** [human] (iscc-infra ops, exposure/security)
- **What / where / how to verify:** Behind caddy-docker-proxy at `monitor-test.iscc.io`,
  every route on the single `:9464` mux is internet-facing: the dashboard / dossier / mirror
  / proof surfaces (public *by design* — the monitor is a "verifiable cache"), `/healthz`,
  AND `/metrics` (Prometheus, exposing operational internals — poll failures, violation
  counts, per-hub status). Because `/metrics` and the mirror share ONE listener
  (`serveMetrics` builds one `http.Server` over one mux, `main.go:179-180`), infra cannot
  separate them by port — only deny a path at the proxy. Decide: is a public `/metrics`
  intended (common and fine for this class of service), or should infra deny `/metrics` (and
  anything else) at Caddy and scrape it only on the internal network? Confirm no route needs
  auth and none is unsafe to expose (the monitor holds no signing key today — consistent with
  the cosigning milestone being deferred — so there is no secret to leak; please confirm).
  Verify fixed: a documented allow/deny list of public paths for the vhost, which infra then
  enforces in the Caddy labels.
- **Spec:** CLAUDE.md "Verifiable cache" (public by design); iscc-infra gotcha "only the
  reverse proxy may publish 80/443; bind debug ports to 127.0.0.1".

## Document egress + resource footprint for box sizing
- **Priority:** critical
- **Source:** [human] (iscc-infra ops, sizing)
- **What / where / how to verify:** The candidate box (DO `iscc.ai`, 206.189.52.39) already
  runs search-test (cap 1.5 GB) + the status page, so this instance must be sized to fit.
  Document the outbound egress the monitor needs — HTTPS to each hub's `/log` tiles and
  `/.well-known/did.json` (did:web key resolution, ADR-0009) and to the OTS calendar
  `https://alice.btc.calendar.opentimestamps.org` (`internal/otsclient/client.go:40`) — so
  egress policy is a conscious choice (DO default-allows egress; it just needs stating). And
  give rough steady-state numbers for the testnet realm (2 hubs): resident memory, CPU, and
  especially the **disk-growth rate** of the mirror BLOBs per hub over time, so infra can
  size the volume and set a DO disk-usage alert before it bites. Verify fixed: a short
  "deployment footprint" note lists the egress endpoints and ballpark RAM / CPU /
  disk-growth for an N-hub realm.
- **Spec:** ADR-0004 OTS calendar transport; ADR-0009 did:web resolution; ADR-0007 mirror
  growth / per-network DB sizing.

