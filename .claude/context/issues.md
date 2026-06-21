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

## Adopt the iscc-lib Go codec + bump the toolchain to Go 1.26 (ADR-0011)
- **Priority:** normal
- **Source:** [human]
- **What / where / how to verify:** Land ADR-0011's adoption — reuse `iscc/iscc-lib` for ISCC
  en/decoding instead of owning a second hand-rolled codec. Sequence it **before** more M-UI feature
  work (it is a foundational stack change). Steps:
  1. **Bump the locked toolchain Go 1.24 → 1.26** across `mise.toml` (`go = "1.26"`), `go.mod` (`go
     1.26.x`, plus a `toolchain` line only if the pinned patch needs it), `.github/workflows/ci.yml`,
     and `.devcontainer/` — `iscc-lib`'s `packages/go/go.mod` requires `go 1.26.1`.
  2. **Add `github.com/iscc/iscc-lib/packages/go` (pinned v0.5.0)** to `go.mod` / `go.sum`.
  3. **Make the dependency real + encode the migration trigger** with ONE tripwire/parity test in
     `internal/index` (a `_test.go` / sibling test package — NOT `iscc.go`) that imports iscc-lib and
     asserts `IsccDecode` currently **rejects** a known ISCC-IDv1 (`"MAIGHFECJMOPMIAB"`, Version 1)
     with a Version error. When [iscc/iscc-lib#43](https://github.com/iscc/iscc-lib/issues/43) lands,
     that assertion flips RED and signals: migrate `internal/index.Decode` to iscc-lib and delete the
     port. Production `internal/index/iscc.go` must stay a **pure stdlib-only leaf** (no iscc-lib
     import) so `GOOS=js GOARCH=wasm go build ./internal/index` still succeeds — only the test imports
     iscc-lib.
  - **Verify fixed:** `go.mod` requires Go 1.26 and `iscc-lib/packages/go` v0.5.0; the tripwire test
    is present and green (asserts today's rejection); `mise run check` is green under a Go 1.26
    toolchain; `GOOS=js GOARCH=wasm go build ./internal/index` still succeeds.
  - **Toolchain caveat:** local dev at filing time is Go 1.24; this increment must run where mise can
    provision Go 1.26. Do **not** flip `go.mod`'s `go` directive without the 1.26 toolchain present —
    it reds the gate for the whole module.
- **Spec:** ADR-0011; target.md "Stack (locked — ADR-0003, ADR-0011)"; iscc/iscc-lib#43.

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

## Certificate §4 builds `did:web:` + raw domain, mis-rendering a `host:port` hub's DID
- **Priority:** normal
- **Source:** [review] (Codex P2, reviewer-confirmed)
- **What / where / how to verify:** `internal/certificate/handler.go:485` sets
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

