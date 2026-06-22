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

## Certificate §5 BITCOIN ANCHOR does not bind the OTS proof's committed digest to §2's accepted root
- **Priority:** normal
- **Source:** [review] (Codex P2, reviewer-confirmed against the library + the OTS write path)
- **What / where / how to verify:** `internal/certificate/handler.go:843-845` sets `HasClause5=true` and
  renders "block N" whenever `ots.Confirmed(rec.OTSBytes)` parses a Bitcoin attestation — but
  `ots.Confirmed` (`internal/ots/ots.go:58`) only classifies the proof's attestations; it never compares
  the parsed `File.Digest` (the 32-byte SHA-256 the proof commits to, exposed by
  `opentimestamps@v0.4.0` `ots.go:61`) against the §2 accepted `root`. So a stored OTS row whose
  `ots_bytes` commit to a DIFFERENT digest than §2's root would render the §5 anchor as if §2's root were
  Bitcoin-confirmed when it is not — a false anchor claim on a Tier-1 self-verifiable surface. This is the
  always-loaded "gate a rendered ✓/anchor on a re-VERIFICATION, not a classify-only flag" rule applied to
  the §5 anchor assertion. NOT currently exploitable: the production write path
  (`follower.OTSTick`→Stamper→`MarkOTSStamped`, Upgrader→`MarkOTSUpgraded` in `otsloop.go:144-179`) always
  submits/upgrades the row's OWN `r.Root` digest, so a mismatched (root-key, proof-digest) row is
  unreachable; only a buggy `RecordOTS` — or the §5 tests, which seed `hello-world.txt.ots` against an
  arbitrary tree root for fixture convenience — produces one. Does NOT block this increment's stated goal
  (the three honest §5 states render correctly from production-written rows). Fix when §5 / `ots.Confirmed`
  is next touched: have `Confirmed` (or a sibling) surface `File.Digest` and require
  `bytes.Equal(file.Digest, root)` before `HasClause5=true`; an unbound proof declines §5 like an
  unparseable one. Verify fixed: a §5 test seeds a confirmed proof under a root that does NOT match the
  proof's digest and asserts §5 is OMITTED; reverting the digest check makes it FAIL.
- **Spec:** learnings.md always-loaded "re-verify a rendered ✓, not a status flag"; target.md M-UI
  certificate Bitcoin-anchor Verify criterion; ADR-0001 fail-closed; CLAUDE.md "Verifiable cache".

## The production OTS stamp path has neither a panic-recover nor a per-request timeout (the upgrade path has both)
- **Priority:** normal
- **Source:** [review] (Codex P1+P2, reviewer-confirmed against the library source)
- **What / where / how to verify:** This advance newly wired `otsclient.Stamp` onto the live background
  goroutine (`cmd/iscc-monitor/main.go` `stampFunc()` → `runOTSLoop` → `OTSTick`'s Stamper call), but
  `Stamp` (`internal/otsclient/client.go:121`) has NONE of the two guards the sibling upgrade path got in
  the prior hardening slice (`safeUpgrade`/`recoverRead`). TWO defects on the same call:
  (1) **panic → process crash.** `Stamp` calls `opentimestamps.Stamp`, which parses the calendar
  response via `parseCalendarServerResponse` → `parseTimestamp`/`readInstruction`
  (`opentimestamps@v0.4.0/parsers.go`) — the IDENTICAL panic-prone parser family `recoverRead`
  (`client.go:136`) was created to guard (the otsclient learning: "the library over-reads its buffer on
  truncated / non-.ots bytes → a slice-bounds panic"). A malformed/truncated calendar stamp response
  therefore panics, and because the Stamper runs inside `runOTSLoop`'s goroutine with no recover, the
  panic crashes the WHOLE monitor (violates the always-loaded "OTS never crashes the follower", ADR-0004).
  (2) **stall → goroutine hang.** `opentimestamps.Stamp` uses `http.DefaultClient.Do` with no deadline
  (`stamp.go:21`) and `stampFunc` passes the process ctx (no timeout), so a calendar that accepts the POST
  but never finishes the body hangs the OTS goroutine forever, starving all later pending rows + future
  ticks. The upgrade path already solved exactly this with `safeUpgrade`'s
  `context.WithTimeout(ctx, upgradeTimeout=30s)` (`client.go:158`). Not currently exploitable in tests
  (the Stamper is injected/faked offline) and best-effort by design, but a live calendar can now trigger
  both. Does NOT block this increment's stated goal (the stamp→upgrade transit works); it is a latent
  production-correctness defect on a freshly-live path. Fix when the stamp path is next touched: add a
  `safeStamp` wrapper mirroring `safeUpgrade` — derive `context.WithTimeout(ctx, stampTimeout)` AND a
  `recover()`-to-error guard around `opentimestamps.Stamp`/the response parse, and route `Stamp` through it
  (the same FFI-boundary pattern as `recoverRead`/`safeUpgrade`). Verify fixed: a unit test feeds `Stamp`
  (or `safeStamp`) a malformed calendar response and asserts a wrapped error (no panic), and a stalled
  request returns a deadline-exceeded error rather than hanging; reverting the guard makes that test panic/hang.
- **Spec:** ADR-0004 "OTS never blocks / never crashes the follower"; learnings.md always-loaded OTS rule;
  `learnings/otsclient.md` `safeUpgrade`/`recoverRead` precedent ("keep ALL upgrade calls routed through
  safeUpgrade … do not strip it" — the stamp path needs the symmetric guard).

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

## Hub-List `hubDomain` accepts a trailing `?` (ForceQuery fail-open against the bare-host contract)
- **Priority:** normal
- **Source:** [review] (Codex P2, reviewer-confirmed)
- **What / where / how to verify:** `internal/registry/registry.go` `hubDomain` (line 188) now rejects
  `u.Path != "" || u.RawQuery != "" || u.Fragment != ""`, but `net/url` represents a bare trailing `?`
  (e.g. `https://sb0.iscc.id?`) as `ForceQuery == true` with `RawQuery == ""`, so the guard does NOT
  fire and `ParseHubList` accepts the url. `u.String()` round-trips the delimiter (`"https://sb0.iscc.id?"`),
  and `Hub.URL` is retained for callers, so the query delimiter survives despite the docstring's
  "path/query/fragment not allowed" contract. Reviewer-confirmed: `hubDomain("https://sb0.iscc.id?")`
  returns `("sb0.iscc.id", nil)` (no error). The trailing-`#` case (`https://sb0.iscc.id#`) Go drops
  on round-trip (harmless), so only `?`/`ForceQuery` is load-bearing. Same fail-open class as the two
  path/missing-hub_id gaps just closed; not currently exploitable (live wiring deferred, fixture uses
  clean `https://host` urls), but a trust-root-adjacent resolver should fully enforce its stated
  contract before the certificate page consumes it. Fix when `hubDomain` is next touched: add
  `|| u.ForceQuery` to the line-188 reject; add a `TestParseHubListErrors` case with url
  `https://sb0.iscc.id?` asserting the "not a bare host base url" fragment. Verify fixed: `ParseHubList`
  with a `https://host?` url returns a non-nil error + nil list, and reverting the `u.ForceQuery` clause
  makes that test FAIL.
- **Spec:** next.md "Fail closed, like Parse" Implementation Note; ADR-0010 Hub-List schema; CLAUDE.md
  registry-rejects-URL-shapes precedent (a bare host base url carries no query).

## Certificate §4 AND the proof bundle build `did:web:` + raw domain, mis-rendering a `host:port` hub's DID
- **Priority:** normal
- **Source:** [review] (Codex P2, reviewer-confirmed — now on TWO surfaces)
- **What / where / how to verify:** NOTE: the proof-bundle assembler now carries this bug on a SECOND
  surface — `internal/certificate/handler.go` `serveBundle` sets `bundle.Hub.DID = "did:web:" +
  data.Domain` (same string §4 builds). Fix BOTH sites together when next touched. `handler.go:485` sets
  `data.SigningKeyDID = "did:web:" + data.Domain`. `internal/registry` explicitly supports `host:port`
  domains (`registry.go` docstring: `a kept line is a bare host (e.g. "sb0.iscc.id", optionally
  "host:port")`), and the codebase's own `didweb.DocumentURL` (`url.go:17-19`) documents the
  method-specific id's first segment as the **percent-encoded** `host[:port]`. So a hub configured as
  `localhost:8443` renders `did:web:localhost:8443`, which per the did:web method denotes host
  `localhost` with path segment `8443` — a DIFFERENT DID than the key was resolved from. The §4 clause
  would name the wrong DID. Reviewer-confirmed against the resolver + registry contracts. NOT currently
  exploitable (the testnet realm fixture uses clean `sb0.iscc.id`/`sb1.amlet.id`; the displayed key id
  `40b74463` is correct and the certificate is an explicitly-Tier-1 "re-verify yourself" surface), so it
  does not block progress — same latent fail-open class as the `hubDomain` ForceQuery gap below. Fix when
  §4 (or a sibling DID-building surface) is next touched: `%3A`-encode the port in the domain→DID
  conversion (reuse the resolver's encoding, do not hand-roll). Verify fixed: a §4 test with a
  `host:port`-domain hub renders `did:web:host%3Aport`, and reverting the encode makes it FAIL.
- **Spec:** ADR-0009 did:web is the only key source; W3C did:web method (port `%3A` encoding);
  `internal/didweb/url.go` DocumentURL contract; `internal/registry` `host:port` support.

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

## Certificate tier-2 honesty header overstates "This browser re-verifies" on the no-JS baseline
- **Priority:** normal
- **Source:** [review] (Codex P2, partially confirmed — the no-JS-overstatement half)
- **What / where / how to verify:** `internal/certificate/cert.html:465` — the `{{if .HasBundle}}` honesty
  header reads, unconditionally and present-tense, "This browser re-verifies the proof below for you, and
  you can download the bundle and re-verify it offline." The tier-2 verifier is progressive enhancement,
  so with JavaScript disabled (or on a WASM load/parse failure) NO browser verdict runs — yet this
  server-rendered copy still asserts the browser re-verifies. The actual verdict panel below
  (`cert.html:480-483`, `id="tier2-result"`) IS honest and conditional ("Re-verify the downloadable
  bundle yourself — or, with JavaScript enabled, this browser re-checks…"), so the two regions disagree
  on the no-JS baseline: the header promises active re-verification while the panel hedges it. On a Tier-1
  self-verifiable surface, honesty copy is load-bearing (target.md M-UI). Reviewer-confirmed by serving a
  certifiable id with JS disabled (the header text renders verbatim, no verdict appears). Does NOT block
  progress (the feature works; the verdict panel itself is honest; the no-JS baseline renders every
  clause). Fix when the honesty copy is next touched: make the `HasBundle` header describe only the
  available bundle/offline path (e.g. "you can download the bundle and re-verify it offline; with
  JavaScript enabled, this browser also re-checks the proof below") so the static copy never claims a
  verdict that may not have run — let the script's panel be the sole asserter of an actual re-verification.
  Verify fixed: the served `HasBundle` header copy does not state in the present tense that the browser
  re-verifies, and a test asserts the no-JS header is consistent with the conditional panel default.
  NOTE: Codex's companion claim — that the `!HasBundle` branch shows stale "lands in a later release"
  copy — is a FALSE POSITIVE and was dismissed: that copy renders ONLY when there is no bundle (no
  verifier wired), which is accurate (`grep "land in a later release" /tmp/cert-fresh.html` → 0 on a
  certifiable page).
- **Spec:** target.md M-UI two-tier honesty (the certificate is the monitor's account; the user verifies);
  CLAUDE.md "Write evergreen comments/copy that describe the current state"; `learnings/certificate.md`
  two-tier-honesty copy rules.

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

## Surface-C `readTarget` accepts opaque-scheme monitor forms (`https:example.com`) the Go `parseTarget` rejected — JS port is more permissive
- **Priority:** normal
- **Source:** [review] (Codex P2, reviewer-confirmed against both URL parsers)
- **What / where / how to verify:** `internal/verifier/verifier.html:550-553` (`readTarget`) ports the
  former Go `parseTarget` rules to the browser's WHATWG `new URL()`, but the two parsers disagree on the
  opaque-path / scheme-relative form. `new URL("https:example.com")` yields `protocol="https:"` +
  `host="example.com"`, so the JS guard PASSES it and returns the RAW string `"https:example.com"` (not
  the normalized `u.href`); the later fetch concatenates the raw value, so the loader leaves the honest
  baseline and runs a live attempt instead of declining. Go's `net/url.Parse("https:example.com")` (the
  original `parseTarget`) instead yields `Host=""`, so the server-side guard REJECTED it — the JS port is
  strictly MORE permissive on this edge form (reviewer-reproduced in node + Go: `https:example.com`,
  `http:foo.bar/x`, `https:example.com:8443` all pass the JS guard but fail the Go one). NOT a
  trust/security defect and NOT exploitable: the browser RESOLVES the raw fetch URL to exactly the same
  host the parser reported (`https:example.com` → `https://example.com/inclusion/…` — never a "wrong host"
  / SSRF), the returned bundle is RE-VERIFIED by WASM against the hub-signed root (the monitor is never in
  the trust path), and an unreachable host yields the documented honest `error` render — never a false
  `verified`/`failed`. So Codex's "fetch the wrong URL" framing is overstated; the only real delta is the
  guard is sloppier than its Go original on a malformed input that still resolves correctly. Does NOT block
  this increment (the static artifact works; well-formed targets run; all gates green; `readTarget` is a
  documented usability guard, NOT a trust boundary — `learnings/verifier.md`). A byte-for-byte port is not
  achievable here because WHATWG `new URL` and Go `net/url` genuinely differ on opaque paths. Fix when
  `readTarget` is next touched: return the PARSED `u.href` (or `u.toString()`) instead of the raw
  `monitor` so the normalized URL is what flows downstream — and/or reject when `u.href`'s origin/path
  prefix does not match the raw input, so the guard's own normalization is the single source of truth.
  Verify fixed: a JS-level test (or the deploy harness) feeds `monitor=https:example.com` and asserts the
  fetch URL begins `https://example.com/` (normalized), or that the raw opaque form is declined; reverting
  the normalization makes it FAIL.
- **Spec:** next.md Surface-C Implementation Note "Port the validation verbatim, in JS" / "Mirror
  `parseTarget`'s rules"; `learnings/verifier.md` "`parseTarget`/`readTarget` is a usability guard, NOT a
  trust boundary"; CLAUDE.md "Verifiable cache" (the client re-verifies; the monitor is not trusted).

## The WASM verifier proves only inclusion math — it never checks the checkpoint signature or binds the record to the requested id (monitor stays in the trust path)
- **Priority:** normal
- **Source:** [review] (Codex P1, reviewer-confirmed against the verify core; affects BOTH tier-2 callers)
- **What / where / how to verify:** `isccVerifyInclusion` (`cmd/wasm/verifyadapter/verify_adapter.go`
  `VerifyJSON` → `internal/proof/verify.VerifyInclusion`) verifies ONLY that `record` hashes into a tree
  of `size` leaves with `proof` → `root` (RFC-6962 inclusion). It does NOT (1) verify the checkpoint
  note signature against the hub's did:web key, nor (2) bind the returned `record` to the requested
  `target.id`. On Surface C this is acute: the verifier's whole mission is that "the instance you point it
  at is never in the trust path", yet a malicious/compromised monitor can return a bundle whose
  record+proof+root are internally consistent (forged unsigned checkpoint, or a DIFFERENT declaration's
  record) and the browser renders the green `verified` state — putting the monitor BACK in the trust path.
  The copy overstates this: `verifier.html:449` lists "Check the signature against the hub's did:web key"
  as a step the verifier WILL run, and `verifier.html:601` reports "✓ … re-verified this inclusion proof
  against the **hub-signed** checkpoint root" — but neither the signature nor a did:web resolution runs.
  This is the SAME verifier-core scope the certificate's same-origin tier-2 already ships
  (`cert.html:565`), so it is NOT a regression introduced here and does NOT block this increment (its
  Verify is met); but it is more serious cross-origin. NOT currently exploitable on the testnet (the
  fixture monitor is honest), but it is a real trust-root honesty gap. Fix when the WASM verifier scope
  is next expanded: extend the verifier (or a sibling export) to (a) verify the checkpoint note signature
  against a did:web key fetched/resolved in the browser, and (b) assert the record decodes to the
  requested `target.id`, gating `verified` on ALL THREE; until then, narrow the success copy + drop the
  unrun did:web step from the record block so the page does not claim a signature/key check it skips.
  Verify fixed: a bundle with a valid inclusion proof but a checkpoint signed by a non-did:web key, or a
  record whose id != the requested id, renders `error`/`failed`, NOT `verified`; reverting the added
  checks makes that test FAIL.
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

