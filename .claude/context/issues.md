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

## Certificate §6 RECORD HISTORY omits the per-record `· at` timestamp the mockup shows
- **Priority:** normal
- **Source:** [review] (visual pass vs the §6 mockup region)
- **What / where / how to verify:** The certificate mockup `.claude/design/ISCC Monitor -
  Certificate.dc.html:68` renders each §6 row as `label` + `seq N · at` — a per-record
  timestamp. The landed §6 (`internal/certificate/handler.go:591-613`, `cert.html:376-378`)
  renders only `{{.Label}} · seq {{.Seq}}` with no time, because the projection it reads
  (`store.RecordRow` = `Seq`/`IsccID`/`NoteSchema`, `iscc_index.go:80-84`) carries no
  per-record timestamp column. The named-region's primary affordance (kind + seq +
  deletion note) is complete and correct; the missing `· at` is cosmetic and does not
  affect certification correctness. Surfacing it cleanly needs a store change: add a
  timestamp to the `iscc_index` projection (written by `RecordProjections`) and surface it
  via `RecordAt`, then render it in the §6 row — a schema change touching store + follower
  ingest, larger than this clause. Fix when §6 (or a step that adds a record timestamp to
  the projection) is next touched. Verify fixed: a §6 row renders `label · seq N · <time>`
  and a test asserts the time component is present for a seeded record.
- **Spec:** target.md M-UI certificate Verify criterion (record history); `.dc.html` §6
  region line 68; CLAUDE.md "Projection" (a derived view — adding a column is additive).

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

## `/` realm-index sub-region deltas vs the mockup (logo, instance-identity copy, Checkpoint/Anchor columns)
- **Priority:** normal
- **Source:** [review] (visual pass vs the Realm-Index mockup, after the named-region parity landed)
- **What / where / how to verify:** The three HEADLINE landmark regions (claim-lookup hero, per-row
  dossier link, instance-identity masthead) now render and the lone `critical` is closed — but the
  ADR-0012 visual pass against `.claude/design/ISCC Monitor - Realm Index.dc.html` shows four remaining
  sub-region deltas the parity step deferred as constraint-wins (all flagged in that handoff):
  (1) **No logo** — **CLOSED (reviewer-confirmed `6a442b4`).** The self-hosted logo now renders on ALL SIX
  SSR mastheads (`/`, dossier, certificate, the three proofserve surfaces); the ADR-0012 visual pass shows no
  remaining "no logo" delta on any surface. No further action — kept here only as a resolved sub-item record.
  (2) **Static instance identity + realm name** — the mockup shows `monitor.iscc.id` / "instance operated
  by ISCC Foundation · ISCC mainnet" and a "REALM REGISTER · ISCC MAINNET" subtitle; the live page renders
  generic static copy ("monitor instance" / "independent Trust & Transparency service" and a bare "Realm
  register") because the identity is not env-configurable. Needs the deferred config-driven instance
  identity (domain / operator / realm name) before it can be honest per-deployment.
  (3) **Checkpoint-size + Bitcoin-anchor data columns absent** — the mockup's ledger has `Checkpoint` and
  `Anchor` columns; the live grid renders `#`/Hub·domain/Coverage since/Observed size/Status only, because
  `store.HubSummary` carries no per-hub checkpoint-size-vs-observed split or OTS anchor state for the index.
  Surfacing them is a store-projection change (add the fields to `ListHubs`/`HubSummary` + render the
  columns) — do NOT add a store read until that projection lands.
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

## Pages custom domain is not bound by the artifact CNAME under Actions-based deploy — needs a one-time repo-settings step (else `/_ds/` asset paths break on the project URL)
- **Priority:** normal
- **Source:** [review] (Codex P2, reviewer-confirmed against the GitHub Pages Actions mechanism)
- **What / where / how to verify:** `.github/workflows/pages.yml:49-50` copies the tracked
  `.github/pages/CNAME` to `dist/CNAME` to set the `monitor.iscc.codes` custom domain — but with the
  modern Actions-based Pages deploy (`actions/deploy-pages@v4`), GitHub IGNORES a `CNAME` in the uploaded
  artifact; the custom domain comes from the repository **Settings → Pages** (or the API), and the Pages
  source must additionally be switched to "GitHub Actions". Both are one-time repo-config steps a workflow
  file cannot assert. Consequence: on a fresh setup, until the custom domain is configured in Settings,
  the deploy lands at the default project URL `iscc.github.io/iscc-monitor/`, where the verifier page's
  root-absolute asset references (`href="/_ds/tokens.css"`, `src="/_ds/wasm_exec.js"`, etc.,
  reviewer-confirmed in the generated `index.html`) resolve against the apex (`iscc.github.io/_ds/...`)
  and 404 — the page renders chrome-less and the WASM never loads. On the apex custom domain
  `monitor.iscc.codes` the same root-absolute paths resolve correctly, so the artifact is right; only the
  domain binding is the gap. NOT a code defect and does NOT block this increment (the workflow correctly
  builds + uploads the byte-pinned tree; the handoff already flags the human settings step; ADR-0003 +
  next.md explicitly chose the tracked-CNAME approach, which is the correct mechanism for a branch-based
  source and a harmless intent-documenting no-op under Actions). Codex's "broken absolute asset paths"
  framing is REAL but contingent on the custom domain not being configured. Fix when `pages.yml` (or the
  deploy docs) is next touched: either (a) add a short `## GitHub Pages setup` doc note (in CLAUDE.md or a
  README) that the human must set the custom domain + "GitHub Actions" source once in repo Settings, OR
  (b) keep the artifact CNAME AND document that it is a no-op under Actions, so the binding mechanism is
  not silently assumed. Verify fixed: the deploy docs name the one-time Settings/API custom-domain step,
  or the workflow/docs make the Actions-CNAME no-op explicit. (Operationally: a human confirms Pages
  source = "GitHub Actions" and custom domain = `monitor.iscc.codes` + the DNS CNAME on first deploy.)
- **Spec:** ADR-0003 "Pages-from-repo ties the deployed WASM to a public commit" + `.codes` custom domain;
  target.md WASM "the verifier artifact … published value"; `learnings/ci.md` Pages-CNAME-no-op nuance.

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

