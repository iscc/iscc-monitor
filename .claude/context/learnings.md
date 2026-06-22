# Learnings — Index

Durable, cross-cutting rules are below and **always loaded**. Per-package detail lives in
`.claude/context/learnings/<name>.md`. **When your step touches a path, look it up in the
pointer table and Read that detail file** before writing or reviewing — a cold subagent has
no other memory of it.

`review` appends findings to the matching detail file (create it + add a pointer row if the
area is new), and promotes a finding to the rules below **only if it is durable AND
cross-cutting** — true even if that package were deleted, and needed even when a step does not
touch it. Everything package-local stays in the detail file. See `README.md` for the rotation rule.

## Correctness rules (seeded from `.claude/plans/cosmic-baking-octopus.md`)

- **Origin = `<domain>/log`**, never the bare domain. Used for the signed-note `name` and the
  verifier key. One `origin()` helper, golden-tested against both live hubs. (Highest-probability bug.)
- **`iscc_id → seq` is one-to-many, and verification is schema-agnostic (ADR-0008).** Index by `seq`;
  store the raw `note.$schema`; lookups return a list. Unknown schemas are indexed + proof-able but
  never interpreted and never gate checkpoint acceptance.
- **Partial-tile discipline (ADR-0005).** Mark a tile/bundle BLOB `is_full` (immutable) only at
  `width == 256`; re-fetch partials (`.p/<W>`) every poll and overwrite. Never promote a partial.
- **A self-consistency violation freezes, never crashes (ADR-0006).** Three triggers —
  fork / shrink / equivocation. Persist both raw checkpoints + proof permanently, set `frozen=1`,
  alert once, keep polling evidence-only at a backed-off cadence, no auto-unfreeze, survive restart,
  other hubs unaffected.
- **Coverage honesty (ADR-0001).** Record `monitored_since`; state guarantees *from coverage start*.
  A late start backfills tiles for full data + forward guarantees but cannot retroactively detect
  pre-coverage equivocation.
- **SQLite single writer per DB (ADR-0005/0007).** WAL + `busy_timeout`; one goroutine owns all
  writes per network file; network fetch happens *outside* the write transaction.
- **OTS never blocks** the follower loop; calendars are best-effort with backoff + redundancy.
- **did:web is the only key source (ADR-0009).** Resolution failure → `unresolvable` (keep
  mirroring). A signature matching no listed key → `unverified`. Domain-compromise is out of scope.
- **`proof/verify` is pure** (no `net`/`os`/`sqlite` imports) — it is shared by the server,
  `verify-for-me`, and the WASM build. Keep it import-clean or the WASM build breaks.
- **On a self-verifiable surface, gate a rendered `✓`/Merkle assertion on a re-VERIFICATION, not a
  status flag.** A built proof is not a verified proof. Any surface that renders a ✓ a reader trusts
  (certificate, proof-bundle assembler, the in-browser verifier) must re-run `proof.VerifyInclusion`
  against the accepted root before asserting validity — never gate on a flag (`hub.Frozen`, etc.) read
  from a separate, concurrently-updated row, which leaves a TOCTOU window (the HTTP server runs beside
  the follower; a fork poll overwrites mirror tiles before the freeze commits). Fail-closed re-verify
  subsumes the flag and catches both steady-state and the race.

## Go / tooling conventions (durable)

- `CGO_ENABLED=0` everywhere → **do not use `go test -race`** (the race detector needs cgo) and do
  not pull cgo-dependent deps. `modernc.org/sqlite` is the pure-Go SQLite driver for this reason.
- Port from `cauldron/` reference copies; never `import` them (they are gitignored, not a module dep).
- **`cauldron/` breaks a fresh `go build ./...`**: its reference trees need external deps the root
  module lacks. The local fix is gitignored stub `go.mod` files at `cauldron/iscc-hub/` +
  `cauldron/tessera/` (separate modules → Go skips them). These are NOT committed, so any CI running
  `go build ./...` on a fresh checkout that includes `cauldron/` will hit the same failure — CI must
  either not check out `cauldron/`, add the same stubs, or use a `go.work` exclude. (Verified: entire
  `cauldron/` is `.gitignore`d, so it never reaches CI from this repo's tree anyway.)
- **Verifier-key golden test is the trust-root oracle gate.** `internal/didweb` derivation matches
  `.claude/derive_vkey.py` byte-for-byte (keyid = BE-uint32 of `SHA-256(name||0x0A||0x01||pub)[:4]`,
  base64 **Std** padded). Go added a defensive `len(raw) < 34` check before the multicodec assert —
  the Python port would index-panic on a short key instead. Keep this guard when porting crypto.
- **Oracle parity is reproducible and external, not self-referential.** Reviewer can re-run
  `python3 .claude/derive_vkey.py` (Python 3.11 present, no deps) → both golden vectors print exactly
  (`sb0…+40b74463+…`, `sb1…+22b08f3e+…`), and the testdata `publicKeyMultibase` values equal the
  oracle's `HUBS` map. Side effect: the oracle writes `.claude/.scratch/` (NOT gitignored) — `rm -rf`
  it after running so it doesn't dirty the tree.
- **`internal/didweb` purity nuance:** `go list -deps` shows `os` in the closure even for a pure
  parser, because `fmt` transitively imports `os`. That is stdlib and unavoidable; the load-bearing
  rule (no `net`/`net/http`/`database/sql`, WASM-shareable) holds — verify with
  `GOOS=js GOARCH=wasm go build ./internal/didweb`, not by grepping `os` out of the dep list.

> Package-specific did:web / logclient transport seam notes moved out of this section into
> `learnings/didweb.md` and `learnings/logclient.md` — Read those when touching that code.

## Detail index

| Area / path | Detail file | Gist |
| --- | --- | --- |
| `internal/logclient` | `learnings/logclient.md` | signed-note verify; shrink/fork/equivocation triggers; consistency/inclusion proof builders; leaf hasher; fsck glue; did:web resolver + checkpoint/tile transport + AcceptCheckpoint seam |
| `internal/proof/verify` | `learnings/proof-verify.md` | pure WASM-shareable inclusion-verifier core shared by server + verify-for-me + WASM; three-way (true,nil)/(false,nil)/(false,err) verdict contract; arg-order hidden; golden 4-leaf vector |
| `internal/didweb` | `learnings/didweb.md` | DocumentURL W3C mapping; ValidAt half-open window; parseTime fail-closed; assertionMethod polymorphism; WASM-pure verifier seam |
| `internal/tiles` | `learnings/tiles.md` | layout re-export; PartialTileSize / TileCoords / BundleCoords boundary math (the p→width / index-1 partial trap) |
| `internal/store` | `learnings/store.md` | single-writer SQLite leaf; SQLiteFetcher p→width read-back; iscc_index; AdvanceAccepted tx; coverage set-once; modernc v1.46.1 pin |
| `internal/follower` | `learnings/follower.md` | PollHub chain; ingest-before-checkConsistency order (closed critical); freeze/evidence-only re-poll; hub_keys cache; slog at the loop boundary |
| `internal/tilesserve, internal/proofserve, internal/corsmw, cmd/iscc-monitor` | `learnings/http-surface.md` | raw mirror + computed inclusion/consistency/entries routes; CORS; Cache-Control/ETag/conditional-GET; trailing-slash subtree mounting; /healthz |
| `cmd/iscc-monitor` | `learnings/cmd-monitor.md` | wires the M1 leaves; thin main owns the one os.Exit; Origin re-export; bare-struct Loop construction |
| `cmd/notecheck` | `learnings/notecheck.md` | faithful port of the reference notecheck; the in-repo external signature oracle shelled out in CI; reject-guard SIGPIPE nuance |
| `.github/workflows/` (ci.yml + pages.yml) | `learnings/ci.md` | CI: one CGO_ENABLED=0 job inlining mise check + the notecheck oracle shell-out (pipefail/SIGPIPE caveat); Pages: verifier-site build→deploy, copies byte-pinned verify.wasm, artifact CNAME is a no-op under Actions (custom domain is a repo-settings step) |
| `internal/metrics, internal/metricshttp` | `learnings/metrics.md` | stdlib-only Prometheus renderer; glossaryStatus mapping (rotated→unverified, frozen override); recordVerdict single mutation point |
| `internal/dashboard` + `store.ListHubs` | `learnings/dashboard.md` | server-rendered `GET /` hub list; `/`-mount-vs-exact-path-guard; store-provable status subset (inactive>frozen>verified); coverage-honesty render; ListHubs leaf read |
| `internal/badge` | `learnings/badge.md` | five-status `HubStatusBadge` SSR partial; silhouettes ported verbatim from the `.dc.html`; fail-closed label-from-fixed-table; `Render`+`Source`+`PartialName` surface; pure WASM-shareable leaf; only 3/5 store-provable |
| `internal/web` + dashboard `<link>` | `learnings/web.md` | embedded DS token + self-hosted woff2 `go:embed` leaf; `/_ds/` SUBTREE mount (one handler, path switch); `no-cache`+strong-ETag+304 (no `immutable` on stable paths); `serveFont` traversal guard; `noExternalCDN` bans third-party origins only (same-origin `url(` OK); dashboard ban relies on scheme-less `h.origin` |
| `internal/registry` | `learnings/registry.md` | domains-only ADR-0009 parser; fails closed on URL-shaped lines; preserves order, no dedupe |
| `internal/config` | `learnings/config.md` | pure env-value leaf; DB/REALM required; Frozen>=Normal cross-check ties to the back-off |
| `internal/index` | `learnings/index.md` | pure WASM-shareable ISCC-IDv1 decoder; 80-bit layout (realm=SubType nibble, hub_id=body&0xFFF, ts=body>>12); all 4 header nibbles fail-closed incl. Length=0; golden vector grounded in hub schema.py example |
| `internal/certificate` | `learnings/certificate.md` | realm-wide `GET /inclusion/{iscc_id}` cert; first caller of decode→resolve→ListHubs→SeqsForISCCID chain; fail-closed 200 verdicts + buffer-then-200; §1 gated on accepted-tree cap `seqs[0] < LastSize` + canonicalized `ISCC:`-prefixed lookup key; §2 renders accepted `(size, root)` via `CheckpointAt`, base64-Std cross-surface |
| `internal/ots` | `learnings/ots.md` | OpenTimestamps confirmed-check adapter; pure parse/classify (no I/O); NOT WASM-pure (keep out of WASM-shared closures); `ReadFromFile` panics→`recoverParse` fail-closed; `>MaxInt64` height guard landed before int64 cast; bundled `examples/*.ots` are the external `ots verify` oracle |
| `internal/otsclient` | `learnings/otsclient.md` | real `follower.Upgrader`+`Stamp` calendar-HTTP adapter; lives outside follower (anchoring import isolation); `database/sql` count 1 is correct (store.OTSRecord seam); upgrade only PENDING sequences; `UpgradeSequence` panics on uncomputable ops + uses timeout-less `http.DefaultClient` (two open `normal` issues); fixtures = ots oracle copied verbatim |
| `internal/verifier` | `learnings/verifier.md` | Surface-C standalone Independent Verification page (`monitor.iscc.codes`); ONE static artifact (no per-request data), NOT mounted in buildMux (different origin); target read CLIENT-side from `URLSearchParams(location.search)` so the pre-generated `index.html` works on GitHub Pages; pure stdlib leaf, `/_ds/` paths are literals synced to `web.*`; chrome identity is `.codes` not an instance (correct mockup divergence); no-CDN body ban golden-tested; `readTarget` is a usability guard (more permissive than the Go `parseTarget` on opaque URLs — filed `normal`) |
| `cmd/wasm` (+ `cmd/wasm/verifyadapter`) | `learnings/cmd-wasm.md` | GOOS=js WASM entrypoint; untagged file in `package main` breaks `go build ./...` → split tagged `main.go` from a non-main `verifyadapter` subpackage; pure base64-Std adapter preserving proof/verify's three-way verdict at the marshaling boundary; integer guard is now untagged+linux-tested `verifyadapter.SafeIndex` (NaN/Inf branch needs a `wantMsg` pin or the table goes vacuous); certificate is the first SSR caller (JSON data island + `/_ds/` loader, verified live) |
| `cmd/verifier-site` | `learnings/verifier-site.md` | Surface-C static-site generator (the `monitor.iscc.codes` deploy build command); renders `verifier.Handler`+`web.Handler` over httptest into `<out>` at URL paths; fonts enumerated from `fonts.css`; fail-closed on a 404 (handler-served path, NOT a const rename — both move together); copies the byte-pinned `verify.wasm` (hash == `web.WasmVerifyHash`); two internal deps only; non-atomic output (partial tree on error — filed `low`) |
