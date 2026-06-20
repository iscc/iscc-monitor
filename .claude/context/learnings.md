# Learnings

High-signal pitfalls, patterns, and verified conventions accumulated by the `review` role. Keep each
entry specific and actionable. The entries below are **seeded from the build plan's "Correctness
rules"** — the load-bearing gotchas — so the loop knows them from iteration 1.

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

## Go / tooling conventions

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
- **did:web `assertionMethod` is polymorphic** — an entry is either a JSON string (`#fragment` DID-URL
  ref into `verificationMethod`) or an inline object. The clean decode is `[]json.RawMessage` then
  try-string-first (a JSON object fails to unmarshal into a Go `string`, so it falls through to the
  inline path unambiguously). Live testnet docs use the string-ref form.
- **`DocumentURL` W3C did:web mapping is implemented + golden-tested** (`internal/didweb/url.go`): MSID
  is colon-split, each segment `url.PathUnescape`d, first segment = `host[:port]`, rest = path; no path
  → `/.well-known/did.json`, path → `/<segs>/did.json`. Live hubs (no path/port) hit `.well-known`.
  Errors on missing `did:web:` prefix, empty MSID, empty host, bad percent-encoding. `net/url` is the
  URL-parsing half only — it does **not** pull `net`/`net/http` into the closure, so WASM build stays
  green (verified). When the follower lands, it owns the actual fetch; `DocumentURL` stays pure.
- **Export surface for the follower seam is now `DocumentURL` + `ParseDIDDocument` + `VerifierKey`**
  (in `internal/didweb`). `pubkeyFromDID`/`keyID`/`b58decode` deliberately stay package-private — the
  follower derives keys via the three exported entrypoints only. Mechanical renames did not move the
  derived bytes (oracle parity reconfirmed).
- **Networked did:web resolver lives in `internal/logclient` (`didresolve.go`), never `didweb`.** It
  imports `net/http`, so putting it in `didweb` would break the WASM build. `ResolveVerifierKey(ctx,
  Fetcher, baseURL)` wires `origin → did:web:<host%3Aport> → DocumentURL → Fetch → ParseDIDDocument →
  VerifierKey`; the `Fetcher` 1-method seam keeps tests offline (fake + `httptest.NewTLSServer`). Host
  is recovered as `TrimSuffix(origin(baseURL), "/log")` — reuse, not a second host parser. The
  verifier key embeds the *fixture's* origin (`sb0.iscc.id/log`), not the fetch host, so the
  `httptest` golden asserts only the `<addr>/log` prefix, not the full key.
- **did:web colon must be percent-encoded before `DocumentURL` (`strings.Replace(host, ":", "%3A",
  1)`).** Otherwise a `host:port` splits at the colon into a path segment. Live hubs have no port so
  this is exercised only by the `httptest` random-port path — but it is required there and matches the
  `did:web:example.com%3A3000` golden.
- **CID 1.0 validity is a pure predicate, `DIDKey.ValidAt(now) bool` (`validity.go`, `time`-only).**
  Half-open `[ValidFrom, ValidUntil)` with `Revoked` revoking at/after its instant; each zero field =
  "no constraint" so a zero `DIDKey` is always valid (matches `parseTime` + the live "currently valid"
  fixtures). Boundaries compare with `Before` only (never `==`/`After`), guarded by `!IsZero()`. The
  follower must call this and treat out-of-window as **not-`verified`** (rotation/revocation) — a
  *distinct* outcome from `ErrUnverified` (signature matches no key) and `ErrUnresolvable`.
- **Malformed-validity-timestamp fail-open is CLOSED (verified).** `parseTime` now returns
  `(time.Time, error)`: empty → `(zero, nil)` (unconstrained, golden-test untouched); non-empty +
  unparseable → wrapped error, which `ParseDIDDocument` propagates and `ResolveVerifierKey` maps to
  `ErrUnresolvable`. So a hub serving `"revoked":"not-a-date"` collapses to `StatusUnresolvable`, never
  `verified`. The fix lives in the parser, NOT `ValidAt` — the 12-case `ValidAt` boundary golden and
  the `derive_vkey.py` vectors are unchanged (reconfirmed `40b74463`/`22b08f3e`).
- **`FetchCheckpoint(ctx, Fetcher, baseURL)` is the transport-only checkpoint fetch** (`logclient/
  checkpoint.go`, imports only `context`+`fmt`): `origin(baseURL)` → `"https://"+name+"/checkpoint"`
  by concatenation (NOT a second `net/url` parse), returns bytes verbatim, wraps both `origin()` and
  Fetcher errors with `%w` so a 404's `errors.Is(err, os.ErrNotExist)` survives. It does NOT wrap in
  `ErrUnresolvable` — that sentinel is did:web-only; a checkpoint-fetch fault is a plain transport
  error the follower classifies separately.
- **An `httptest` round-trip can prove *fetch* but NOT *verify* against a live-host URL** — `Accept`/
  `ResolveVerifierKey` derive the verifier key from `origin(baseURL)`, and the key is origin-bound
  (`SHA-256(name||…)`). A `127.0.0.1:<port>` host derives a different origin than the fixture's
  `sb0.iscc.id/log` signature → always `ErrUnverified`. The correct pattern (used by both
  `TestAcceptCheckpoint` and `TestFetchCheckpointOverHTTP`): fetch over real HTTP via `srv.URL`, then
  verify the fetched bytes with `baseURL="https://sb0.iscc.id"` + a fake Fetcher for the did.json. Do
  not expect `AcceptCheckpoint(srv.URL, …)` to yield `StatusVerified` with captured fixtures.
- **`AcceptCheckpoint` is the pure 4-way verdict seam the follower consumes** (`logclient/accept.go`):
  composes `ResolveVerifierKey → VerifyCheckpoint → DIDKey.ValidAt(observedAt)` into
  `StatusVerified/Unverified/Unresolvable/Rotated`. `observedAt` is injected (never `time.Now()`),
  validity is checked **only after a good signature** (out-of-window-AND-bad-sig is `unverified`, not
  `rotated`), and `CheckpointInfo{Origin,TreeSize,Root}` is the **zero value on every non-verified
  verdict**. Contract gotcha for the follower: a non-`ErrUnverified` `VerifyCheckpoint` error (a
  verified-but-garbled body) is returned as a non-nil `error` alongside `StatusUnverified`'s zero —
  **callers must check `err` before the status**, mirroring `VerifyCheckpoint`.

## Checkpoint signed-note verification (`logclient/verify.go`)

- **Signed-note keyhash is independently checkable from the raw sig line.** A C2SP sig line decodes
  to `keyhash(BE-uint32, 4 bytes) || ed25519_sig(64 bytes)`; the vkey's middle `+<hex>+` field IS that
  keyhash. Reviewer can confirm a fixture's signer without trusting the author: sb0 sig→`40b74463`,
  sb1 sig→`069d0f14` (both 64-byte sigs), matching their vkeys. sb1's live key really did rotate
  (`22b08f3e`→`069d0f14`), so the "stale rotated key→ErrUnverified" negative is a genuine real-world
  case, not a contrived one.
- **`note.Open` success check mirrors the notecheck oracle exactly:** `len(n.Sigs) == 0 ||
  len(n.UnverifiedSigs) != 0` → reject. Watch for M7: a future *cosigner* sig line whose key isn't in
  the verifier list lands in `UnverifiedSigs`, so this strict check would reject an otherwise-valid
  hub checkpoint that also carries a witness cosig. Fine for v1 (single hub sig); revisit at gossip.
- **`x/mod v0.33.0` forces the `go` directive to canonical-patch form `go 1.24.0`.** The dep declares
  `go 1.24.0`; under `-mod=readonly` a bare `go 1.24` consumer fails with "updates to go.mod needed".
  `go 1.24.0` is the same minimum (still Go 1.24, NOT ≥1.25) — accept it; it is not a version bump.
  Auto-injected `toolchain go1.24.13` lines should be dropped so go.mod doesn't pin a patch CI may lack.
- **`note.Open` returns `n.Text` as the clean trailing-`\n` body**, so `strings.Split(text, "\n")`
  yields a trailing empty element; `len >= 3` + reading `lines[0:3]` is correct and tolerates C2SP
  extension lines after the root. Verify `note.Open`'s framing — never re-parse the sig line by hand.
- **`KeyIDFromCheckpoint(raw)` recovers `(name, keyID)` from the raw checkpoint via `note.Open(raw,
  note.VerifierList())` + `errors.As(err, &*note.UnverifiedNoteError)`, reading `ue.Note.
  UnverifiedSigs[0].{Name,Hash}`.** This is library-exact: with an empty verifier list every well-formed
  sig line hits `*UnknownVerifierError` and is appended to `UnverifiedSigs` (note.go:569) with `Hash =
  BE-uint32(sig[0:4])` already decoded (note.go:554); `len(n.Sigs)==0` then returns `*UnverifiedNoteError`
  (note.go:597). A malformed note returns `errMalformedNote` (a plain `errors.New`, NOT a pointer type),
  so `errors.As` is correctly false and the garbled path wraps cleanly — no `[0]` index, no panic.
  Reviewer re-decoded the sb0 fixture sig line independently in Python (`base64 → struct ">I" → [:4]`):
  `name='sb0.iscc.id/log'`, `keyhash=0x40b74463`, 64-byte ed25519 sig — confirming the golden from
  ground truth, not the author. Oracle gate correctly N/A (reads an already-decoded keyhash; no
  verify/proof/merkle/didweb path; go.mod/go.sum/schema byte-identical). Still the pure prerequisite —
  wiring it into `PollHub` to consult `LookupHubKey` and skip the 2nd did.json fetch is the next slice.

## Consistency triggers (`internal/logclient/consistency.go`)

- **Shrink is the one size-only trigger; it lands dep-free.** `CheckShrink(prev, next uint64) bool ==
  prev > 0 && next < prev`. The `prev > 0` guard is load-bearing: it stops the fresh-store
  `FollowState{}.LastSize == 0` from being misread as a shrink (any `next` over a zero prior is growth).
  Fork (same size, different root) is the next dep-free slice (a `[rootBytes]byte` compare at equal
  size); only equivocation (RFC-6962 consistency proof) needs `transparency-dev/merkle` + tile
  fixtures. `ViolationShrink ViolationKind = "shrink"` is the exact `store.Violation.Kind` /
  `violations.kind` string — kept as a logclient string type so store stays import-free of logclient,
  mirroring how `Status` rides on `CheckpointRecord`.
- **`CheckShrink`/`ViolationShrink` are an intentional unused-until-wired export seam, not dead code.**
  `next.md` scoped the `follower.PollHub` wiring (map `LastSize→prev`, `TreeSize→next`; on true →
  `RecordViolation` + `Freeze` + alert-once) as a deliberately separate later slice. `go vet` is clean;
  do not flag the new exported symbols as dead. The shrink check is pure arithmetic, so the
  conformance/oracle gate (`notecheck`/`derive_vkey.py`/`fsck`) is correctly N/A here — it only trips
  once the merkle-backed equivocation slice lands.
- **Equivocation is the merkle-backed third trigger and lands in the same file (`consistency.go`).**
  `CheckEquivocation(prevSize, prevRoot, nextSize, nextRoot [rootBytes]byte, proof [][]byte) (violated,
  err)` calls `proof.VerifyConsistency(rfc6962.DefaultHasher, …, prevRoot[:], nextRoot[:])` ONLY on the
  strictly-growing path (`prevSize>0 && nextSize>prevSize`); ANY verify failure → `(true, nil)` (the
  proof not verifying IS the evidence, never a poll error — ADR-0006). The `prevSize==0 || nextSize<=
  prevSize` guard short-circuits BEFORE `VerifyConsistency` (proven by feeding garbage proof+roots and
  still getting false). `ViolationEquivocation = "equivocation"` matches `violations.kind`. The golden
  is real ground truth: `testonly.New(rfc6962.DefaultHasher)` builds an append-only tree, `HashAt(M/N)`
  gives roots and `ConsistencyProof(M,N)` the valid proof — the prover (`ConsistencyProof`) and the
  verifier (`VerifyConsistency`) are *independent* code paths in merkle, so the cross-check is not a
  tautology. **Reviewer re-ran two mutations (and removed them):** (1) force `(false,nil)` always →
  corrupted-root + corrupted-proof cases FAIL; (2) drop the growing guard → all three boundary-skip
  cases FAIL. So a green-but-wrong verify cannot ship. Oracle gate APPLIES (this is RFC-6962 crypto)
  and is satisfied by the in-test merkle ground truth; `notecheck`/`derive_vkey.py`/`fsck` are N/A (no
  signature/did:web/tile path). `testonly.Tree` was used instead of `next.md`'s literal
  `compact.RangeFactory` — strictly stronger (same hasher, less bespoke test code), not a shortcut.
- **`merkle v0.0.2` is dep-clean on the 1.24 toolchain.** `go.mod` directive stays `go 1.24.0`, no
  `toolchain` line, `go mod tidy` is a no-op, `go mod verify` passes. `go-cmp v0.6.0` enters `go.sum`
  ONLY as a transitive test-dep of `merkle/testonly` (`go mod why` → "main module does not need" it);
  it is NOT a direct require and is CGO-free.
- **`CheckEquivocation`'s `err` return is unreachable-by-type.** The `[rootBytes]byte` array params make
  a wrong-length root unrepresentable, so `err` is always `nil` from this pure layer. The doc was
  trimmed to say so (it previously promised a `(false, non-nil err)` malformed-root path that cannot
  fire); the `wantErr` test column + return are kept for signature symmetry / future slice-param
  loosening. `CheckEquivocation`/`ViolationEquivocation` are an intentional unused-until-wired export
  seam (same as shrink/fork) — `go vet` clean, not dead code.
- **Fork is the second dep-free trigger and landed in the same file as shrink.** `CheckFork(prevSize,
  prevRoot [rootBytes]byte, nextSize, nextRoot [rootBytes]byte) bool == prevSize > 0 && nextSize ==
  prevSize && nextRoot != prevRoot`. Same-size + differing root only; growth/shrink/identical-root all
  false, and the `prevSize > 0` guard keeps the fresh-store zero from misreading. Uses array `!=` (Go
  elementwise on fixed-size `[32]byte`) — `consistency.go` stays import-free of `bytes`/any dep
  (verified: only `bytes` occurrence is the comment explaining why none is needed). `ViolationFork
  ViolationKind = "fork"` matches the `violations.kind` string. Only equivocation (RFC-6962
  consistency-proof) now remains deferred to the merkle-backed slice. Test guard `if rootA == rootB
  { t.Fatal }` makes the "different root" cases non-vacuous.

## tlog-tiles layout seam (`internal/tiles`)

- **`internal/tiles` is a thin re-export of `tessera/api/layout`, not a reimplementation** — wrappers
  `TilePath`/`EntriesPath`/`PartialTileSize` delegate 1:1 (matching arg order: `TilePath(level, index
  uint64, p uint8)`, `EntriesPath(n uint64, p uint8)`), plus `const TileWidth/TileHeight = layout.*`
  and the one project predicate `IsFull(width int) bool == width == TileWidth`. The package's compiled
  closure is `api/layout` only, and `api/layout`'s own closure is **stdlib-only** — so `net/http`/
  `database/sql` stay out and the WASM build is green (verified). `IsFull` takes `int` (not `uint8`)
  because the column-convention full sentinel 256 cannot fit in `uint8`; `next.md` left the choice open
  and `int` is the clean pick.
- **`next.md`'s `PartialTileSize(0,0,300)==44` golden was WRONG — advance correctly pinned `0`.** Per
  tessera `tile.go`: `sizeAtLevel=300`, `fullTiles=300/256=1`, `index 0 < fullTiles` → **0** (the first
  256-leaf tile is *full*); the leftover 44 spill into index **1**. Reviewer re-ran the real
  `layout.PartialTileSize` independently: `(0,0,300)=0`, `(0,1,300)=44`, `(0,0,44)=44`, `(0,0,256)=0`.
  Downstream load-bearing: the SQLiteFetcher slice must address the 44-leaf partial of a 300-leaf tree
  at **index 1**, never index 0. All four `TilePath`/`EntriesPath` golden strings are verbatim from
  tessera's `api/layout/paths_test.go` (ground truth, not author-asserted).
- **tessera v1.0.2 is a clean dep on the 1.24 toolchain.** Its go directive is `go 1.24.0`; the heavy
  otel/klog/formats/`x/crypto`/backoff deps land in `go.sum` as module-graph requirements only (never
  compiled, so absent from `go.mod`'s indirect block and the `internal/tiles` closure) — same pattern as
  go-cmp for merkle. `go mod tidy` is a verified no-op, `go mod verify` passes, directive stays `go
  1.24.0` with no `toolchain` line. The tessera require graph bumped `x/sys 0.37→0.41` (benign, pure-Go).
- **`IsFull`/`internal/tiles` are an intentional unused-until-wired export seam** (like the consistency
  triggers) — `go vet` clean, not dead code; the SQLiteFetcher store-key slice is its first caller. The
  oracle/conformance gate is correctly N/A here (pure path strings, no signature/RFC-6962/did:web/fsck
  path); it re-arms at the SQLiteFetcher + `fsck` slice.

## SQLiteFetcher / mirror read-back (`internal/store/tiles.go` + `fetcher.go`)

- **`next.md`'s "`var _ fsck.Fetcher = SQLiteFetcher{}` AND byte-identical go.mod" was a contradiction
  — advance correctly resolved it by copying the interface, not importing it.** Reviewer reconfirmed
  from source: `go list -deps github.com/transparency-dev/tessera/fsck` pulls `net/http`, `otel`, and
  `klog` (via `fsck` → `tessera/client` + `errgroup`), so importing `fsck` even in a `_test.go` would
  break store's leaf purity AND force `go mod tidy` to add indirect requires. The test instead declares
  a local `fsckFetcher` interface that is **byte-for-byte identical** to `fsck.Fetcher@v1.0.2`
  (`ReadCheckpoint`/`ReadTile(ctx,l,i uint64,p uint8)`/`ReadEntryBundle(ctx,i uint64,p uint8)` — names,
  params, types, returns all match; diffed against `fsck/fsck.go:38-42`) and pins it with `var _
  fsckFetcher = SQLiteFetcher{}`. Equivalent drift-detection guarantee, no dep-closure leak,
  go.mod/go.sum byte-identical, store stays a leaf (`.Imports` = `context crypto/sha256 database/sql
  embed errors fmt internal/tiles modernc.org/sqlite os time`, no `net/http`). Not a gate dodge.
- **The p↔width translation is the load-bearing bug surface and is pinned correctly.** `widthForP(p)`:
  `p==0 → tiles.TileWidth (256)`, else `int(p)`; a full-tile request (`p==0`) must look up width 256,
  not 0. `RecordTile`/`RecordEntryBundle` set `is_full=1` only when `tiles.IsFull(width)` (width==256)
  via `boolToInt`; partials overwrite in place through the composite-PK `ON CONFLICT … DO UPDATE`
  (TestRecordTilePartialOverwrite asserts row count stays 1). `LatestCheckpointRaw` is the
  size-agnostic `ORDER BY tree_size DESC LIMIT 1` read (distinct from size-keyed `CheckpointAt`).
- **The partial→full fallback wraps `os.ErrNotExist` on BOTH legs, so `errors.Is` survives a double
  miss.** `ReadTile`/`ReadEntryBundle` retry at width 256 only when `p>0 && errors.Is(err,
  os.ErrNotExist)`; if the full leg also misses it returns *that* wrapped `os.ErrNotExist`
  (TestFetcherReadTilePartialNoFallbackNoFull). `readTileAt`/`readEntryBundleAt` are the no-fallback
  inner reads; a `p==0` miss returns the wrapped sentinel directly. Matches tessera's
  `PartialOrFullResource` (which is `internal/` and not importable) exactly.
- **Oracle/conformance gate correctly N/A for this slice** — plain CRUD + BLOB round-trip with
  synthetic in-test bytes; no signature/RFC-6962/Merkle/did:web/`fsck`-rebuild path. The actual
  `fsck.New(...).Check(...)` root-rebuild + inclusion cross-check is the *next* conformance slice
  (needs real tile fixtures) and is the point where this seam first faces the trust-root oracle.

## Monitor binary (`cmd/iscc-monitor`)

- **The binary is the only consumer that wires all four M1 leaves**: `config.Load(os.LookupEnv)` →
  `os.ReadFile(RealmPath)` → `registry.Parse` → `store.Open` → `registerHubs` → `follower.Loop.Run`.
  `main` stays thin (owns the single `os.Exit`); all branching lives in the testable `registerHubs`
  (`Loop.Run` is correctly untested — blocking ticker select). Verified end-to-end offline: a long
  `NORMAL=10m` means no tick fires, SIGINT exits 0, and the `hubs` rows persist with
  `origin == <domain>/log` (sb0→`sb0.iscc.id/log`, sb1→`sb1.amlet.id/log`) — never the bare domain.
- **`Origin(baseURL)` is a one-line re-export of the private `origin`, NOT a second deriver** — the
  golden `TestOrigin` vectors cover it because `Origin` delegates; `TestOriginExport` only re-asserts
  the two live-hub vectors. The private `origin` body and its two internal callers stayed byte-identical.
- **`Loop.lastPoll` is lazily inited inside `Tick` (`if l.lastPoll == nil`), so the binary can build
  `&follower.Loop{…}` without setting the unexported `lastPoll`** — no nil-map-write panic. Confirms
  the follower author handled the bare-struct construction the binary relies on.
- **`Run` returns `ctx.Err()` unwrapped**, so `main.go`'s `err != context.Canceled` (a `==`, not
  `errors.Is`) is correct for a `signal.NotifyContext(os.Interrupt)` cancel → clean exit 0. If `Run`
  ever wraps the cancel error, switch to `errors.Is`; today the bare `==` holds (smoke-verified).
- **Whitespace-only `RealmPath` now fails cleanly at the binary's `os.ReadFile`** with the path named
  (`read realm document "   ": open …: no such file or directory`), closing the config presence-only
  gap noted in earlier learnings — the binary owns the fs error, config owns presence.

## Follower composition (`internal/follower`)

- **`PollHub` is the first real caller composing the M1 chain + store CRUD** (`follower.go`):
  `FetchCheckpoint → AcceptCheckpoint → (only on StatusVerified) RecordCheckpoint → AdvanceFollowState`.
  Verified independently: `go list -deps ./internal/store` stays a single self-only line (store is a
  leaf), `./internal/follower` pulls in `logclient`+`store`(+transitive `didweb`) — direction
  follower → {logclient, store}, never the reverse, so `net/http` never enters the store closure.
- **The verified-path assertion `FollowState.LastSize == 10183` is non-vacuous** — `10183` is line 2
  of the `testdata/live/sb0.iscc.id_checkpoint` fixture (the signed tree size), so it proves the value
  flowed `info.TreeSize → AdvanceFollowState → persisted cursor`. The complementary non-advancing test
  asserts `== 0` after a mismatching-key `StatusUnverified`, so neither case is vacuously satisfied by
  the fresh-store zero.
- **Garbled-body fault returns `(status, wrapped-err)` where status is `AcceptCheckpoint`'s
  `StatusUnverified` zero** — meaningless when err != nil. `PollHub` honors the err-before-status
  contract (returns the wrapped err and persists nothing); callers of `PollHub` must do the same.
- **Freeze wiring composes the two pure verdicts in `PollHub` via `checkConsistency` + `freeze`
  helpers** (`follower.go`). Order is load-bearing: on `StatusVerified`, read `FollowState`, run
  shrink-then-fork BEFORE `RecordCheckpoint`/`AdvanceFollowState`; on a true verdict
  `RecordViolation` + `RecordCheckpoint`(evidence, no advance) + `Freeze`, then alert iff
  `!wasFrozen`. A violation returns `(StatusVerified, nil)` — freezes, never crashes (ADR-0006).
  The `AlertFunc func(int64,string)` is a func seam (YAGNI, not an interface); the `modernc.org/sqlite`
  blank import is added to the follower *test only* (production imports stay `{context,fmt,logclient,
  store,time}`, store stays a leaf, go.mod/go.sum byte-identical).
- **Re-detection of a fork relies on `CheckpointAt`'s `LIMIT 1` returning the PRIOR root, not the
  contradictory one.** The freeze path records the contradicting checkpoint as evidence, so after the
  first detection there are two rows at the same `tree_size` (prior seed root + new root). On the next
  poll `CheckpointAt(LIMIT 1, no ORDER BY)` returns the lower-rowid (prior/seed) row, so
  `CheckFork(prior != new)` re-fires and a 2nd `violations` row is recorded (re-detection = evidence).
  This is correct today (SQLite returns rowid order) and verified stable over 20 runs, but it is an
  *implicit* dependency on insert order — the equivocation/merkle slice that changes how the prior root
  is selected must preserve "compare against the prior accepted root, not the contradicting evidence."
- **Fork-test non-vacuousness comes from the `kind == "fork"` (not "shrink") assertion at equal size,
  NOT the `sb0FixtureRootB64` guard.** That guard compares raw seed bytes to a base64 *string*, so it
  can never trip (and the seed is 33 bytes — `copy` into `[32]byte` truncates harmlessly). The real
  proof the fork branch fired is: verified observation at size 10183 with the real decoded root vs a
  seeded distinct root, asserting kind `"fork"` distinctly from the size-only shrink path.
- **The poll loop (`loop.go`) is pure cadence over `PollHub` — `due()` is the only testable unit, and
  the back-off works *because a freeze returns `(StatusVerified, nil)`*.** `Tick` marks `lastPoll[hub]
  = now` only on a nil-error `PollHub`, and a freeze is a nil error, so a just-frozen hub *does* get
  its `lastPoll` recorded → the next due decision correctly uses the longer `Frozen` interval. If a
  later change ever made freeze return a non-nil error, the frozen hub would be left unmarked and
  re-polled every `Normal` tick (no back-off) — keep freeze on the nil-error path. `due()` uses `>=`
  (exactly-at-interval is due); zero `lastPoll` is always due (fresh hub polled on tick 1, restart
  re-polls all — harmless, `PollHub` is idempotent on an unchanged checkpoint).
- **`Run` is deliberately untested and that is correct here** — it is a 12-line `select` over
  `ctx.Done()`/`ticker.C` with `defer ticker.Stop()` and one documented `_ = l.Tick(ctx, t)` (a flaky
  hub must not abort the network loop; `Tick` already surfaces the error to its caller, so this is not
  gate-dodging). All branching logic lives in the injected-`now` `Tick` + pure `due()`, both covered;
  the spec forbids wall-clock sleeps so testing `Run` would mean sleeping. The single swallowed error
  is justified inline. Verify `time.Now()` never appears in `loop.go` (the ticker delivers `t` via
  `ticker.C`) — the only wall-clock source is `time.NewTicker(l.Normal)`.

## hub_keys cache wiring (`internal/follower` + `internal/logclient/keyid.go`)

- **`cacheHubKey` is wired ONLY on the verified, non-violation `PollHub` path** (after
  `AdvanceFollowState`, mirroring coverage placement), never inside `freeze` and never on a
  non-verified verdict — the fork/shrink/unverified tests assert `countRows(…, "hub_keys")==0`, the
  verified test asserts exactly 1 row with `key_id==0x40b74463` + 32-byte `pubkey_raw` + refresh-in-place.
  A `ResolveVerifierKey` failure here is wrapped (`cache hub key: %w`) and surfaced, never swallowed.
- **`KeyIDFromVerifier` recovers the key id from the vkey STRING, it does not re-derive crypto.**
  `SplitN(vkey, "+", 3)` (n=3 load-bearing: sb0's base64 tail `AaV+ivnly67…` itself has a `+`, so a
  plain `Split` over-splits), require 3 fields, `ParseUint(parts[1], 16, 32)`. The middle `+<hex>+`
  field IS the signed-note keyhash — reviewer independently decoded the sb0 checkpoint sig line
  (`base64→ raw[:4]`) to `40b74463` with a 64-byte sig, matching the golden vector, so the trust-root
  value is confirmed from the fixture, not the author. Oracle gate correctly N/A (string parse, not a
  derivation; `go.mod`/`go.sum`/`schema.sql` byte-identical), but the golden still pins it to
  `VerifierKey`'s `"%s+%08x+%s"` output so it cannot silently diverge.
- **The cache-hit fast path now skips the SECOND did.json fetch (`cacheHubKeyFast`).** On a warm cache
  `cacheHubKey` recovers `(name, keyID)` from raw (`KeyIDFromCheckpoint`), asserts `name ==
  Origin(baseURL)` (`sb0.iscc.id/log`, verified against fixture line 1), hits `LookupHubKey`, and
  `RecordHubKey`-refreshes in place — no `ResolveVerifierKey`. The first verified poll still resolves
  twice (cold cache → miss → `cacheHubKeyResolve` fallback). The `+1` fetch-count assertion is
  non-vacuous: a broken name-guard/lookup would fall through to +2 and fail the test. Fall-through
  cases (key-id miss, name mismatch, cache miss) return `(false, nil)`; genuine faults
  (origin/query/RecordHubKey) wrap a non-nil error and are never swallowed. The remaining FIRST resolve
  (inside `AcceptCheckpoint`, drives the `ValidAt` window check) is the next, larger efficiency slice.
- **The fast path reuses the *cached* `Revoked`/`PubkeyRaw` on a hit, NOT a re-resolve — and that is
  safe.** A same-`key_id` pubkey edit is cryptographically near-impossible (`key_id =
  SHA-256(name||0x0A||0x01||pub)[:4]` → different pubkey ⇒ different key_id ⇒ cache miss ⇒ full
  resolve), and a `revoked_at`/window edit is still caught by `AcceptCheckpoint`'s first resolve every
  poll (which gates `StatusVerified` before `cacheHubKey` ever runs). The `hub_keys` row is an
  identity/availability cache, never the verification authority. A fully cache-only window-honoring
  path would first need a `valid_from`/`valid_until` schema column (explicitly Not In Scope here).
- **`LookupHubKey(ctx, hubID, keyID)` is the read side of the cache and is now landed** — the exact
  column-by-column inverse of `RecordHubKey` (`pubkey_raw`→`[]byte`, `pubkey_z`/`revoked_at`/
  `resolved_at` via `sql.NullString`/`sql.NullInt64`→`""`/zero-time), `HubID`/`KeyID` reconstructed
  from the in-args (never re-scanned), absent row → `(HubKey{}, false, nil)` per `FollowState`/
  `Coverage`. **The `uint32` key id never has to be recovered from the signed `int64` column on read**
  (it comes from the lookup arg), so high-bit ids like `0xdeadbeef` round-trip losslessly —
  independently verified with a throwaway high-bit test (PASS, then removed). `LIMIT 1` (no `ORDER BY`)
  is sound because `RecordHubKey`'s UPDATE-then-INSERT keeps ≤1 row per `(hub_id, key_id)`. Still
  unwired into `PollHub`/verification (deliberate next slice). Oracle gate correctly N/A — pure CRUD,
  `go.mod`/`go.sum`/`schema.sql` byte-identical (`git diff --quiet HEAD~1..HEAD` exit 0).

## Realm registry (`internal/registry`)

- **`Parse([]byte) ([]Entry, error)` is the pure domains-only membership leaf (ADR-0009).** Line-based,
  drops blank/`#`-comment lines, trims, preserves input order (no sort/dedupe — reconciliation is the
  store-coupled wiring step's job). Fails closed on URL-shaped lines: `://` (scheme) or `/` (path) →
  wrapped error naming the bad line; returns `nil` entries alongside the error. Imports are exactly
  `{bufio bytes fmt strings}` — verified no `net`/`net/http`/`os` in the closure (oracle gate correctly
  N/A: no proof/verify/didweb/merkle path touched, go.mod/go.sum byte-identical). `Entry.BaseURL =
  "https://"+Domain`; the follower derives origin+vkey from BaseURL inside `PollHub`, so the registry
  intentionally carries no key field and never reaches for the package-private `logclient.origin`.
- **The golden fixture's `sb0.iscc.id`/`sb1.amlet.id` are the real testnet hubs**, consistent with the
  existing `didweb`/`logclient`/`follower` fixtures and `derive_vkey.py`'s HUBS — not invented. The
  registry→`HubTarget` mapping is deferred to wiring (needs `HubID` from `store.UpsertHub`), so
  `Loop`/`HubTarget`/`cmd/` stay untouched here, as scoped.

## Config loader (`internal/config`)

- **`Load(get func(key string) (string, bool)) (Config, error)` is the pure startup-value leaf**
  (`config.go`, imports exactly `{fmt time}`). `get` is injected (the binary backs it with
  `os.LookupEnv` next step), so the package touches no env/fs/net and is 100%-covered by a map-backed
  fake. Keys are `ISCC_MONITOR_{DB,REALM,NORMAL,FROZEN}`; `DB`/`REALM` required (absent OR empty →
  named error), intervals default `Normal=5m`/`Frozen=1h`, parsed via `time.ParseDuration`. The keys
  are package constants the binary references symbolically and are not yet load-bearing against any
  external contract — renamable cheaply at wiring time (e.g. if a flag surface is preferred).
- **The load-bearing cross-check is `Frozen >= Normal`** — it ties config to `loop.go due()`'s
  back-off (a frozen hub polls on the *longer* `Frozen` interval, ADR-0006). Verified `Loop.Normal`/
  `Loop.Frozen` are `time.Duration` fields config feeds directly; `Frozen == Normal` is accepted,
  `Frozen < Normal` rejected. The `frozen < normal` error names BOTH keys; the test asserts only that
  `keyFrozen` appears, which holds.
- **A *present but* non-positive interval (`0s`, `-1m`) is rejected** (named error) — a deliberate
  addition beyond the literal `>0` spec wording; absent keys still take the positive defaults. Oracle
  gate correctly N/A (no proof/verify/didweb/merkle/fsck path; go.mod/go.sum byte-identical). Note:
  `required` does NOT trim whitespace, so a whitespace-only path (`" "`) passes config and would fail
  later at `os.ReadFile`/`store.Open` — acceptable (config validates presence, the binary owns I/O),
  but the binary should surface that fs error clearly.

## SQLite store (`internal/store`)

- **`modernc.org/sqlite` pin is `v1.46.1` (last version requiring only `go 1.24.0`).** `v1.46.2`+ bump
  the `go.mod` directive to `≥1.25.0` and would fail the gate on the pinned 1.24 toolchain. After
  `go get`, the directive stayed `go 1.24.0` with no `toolchain` line; `go mod verify` + `go mod tidy`
  are both clean (reviewer reconfirmed — tidy produces zero diff). The reset-the-directive route is NOT
  viable for v1.52.0 (genuinely won't build on 1.24); the pin is the correct fix.
- **`PRAGMA foreign_keys=ON` is per-connection — `SetMaxOpenConns(1)` makes it stick.** Reviewer
  independently confirmed FK enforcement is live (orphan `hub_keys` insert rejected with SQLITE error
  787) and `journal_mode=wal`, `busy_timeout=5000`, `synchronous=1(NORMAL)`, `MaxOpenConnections=1` are
  all applied in order on `Open`. When the read-pool split lands at serving, FKs + WAL pragmas must be
  re-asserted on the read connections too (per-connection state does not carry across a larger pool).
- **Schema columns match the plan's "SQLite schema" block verbatim** (all 9 core tables, `iscc_id`
  non-unique indexed for the one-to-many `iscc_id→seq`, UNIQUE on checkpoints/ots `(hub_id,tree_size,
  root)`, PKs on tiles/entry_bundles). No `network` column (ADR-0007, grep confirms comment-only). No
  `cosigs` table (M7-deferred). The `_ = db.Close()` on `Open`'s error paths is the correct idiom
  (preserve the original error; don't mask it with the close error).
- **Typed CRUD seam for the follower is `UpsertHub`/`RecordCheckpoint`/`FollowState`/`AdvanceFollowState`
  (`checkpoints.go`), and `store` stays a leaf** — `go list -deps ./internal/store` shows zero internal
  iscc-monitor deps; the `Status` string is carried on `CheckpointRecord` (the `checkpoints` table has
  **no** status column, so it is intentionally not persisted) rather than importing `logclient`. Keep
  it that way so net/http never enters the store closure.
- **`RecordCheckpoint` dedupe relies on `ON CONFLICT(...) DO NOTHING` + `RowsAffected()`:** real insert
  → `n>0` → `LastInsertId`/`inserted=true`; conflict → `n==0` → SELECT the id back/`inserted=false`,
  nil err. The modernc driver returns `RowsAffected==0` on `DO NOTHING`, so this branch is load-bearing
  and is the correct way to tell "first sighting" from "re-observed" without an error.
- **`AdvanceFollowState` upsert omits `frozen` from the `DO UPDATE SET`** (`ON CONFLICT(hub_id) DO
  UPDATE SET last_size=excluded.last_size`), so a hub frozen via the freeze path stays frozen across an
  advance (ADR-0006, no auto-unfreeze) — proven by raw `UPDATE frozen=1` then asserting `frozen` still 1
  after advance. Nothing in this layer ever sets or clears `frozen`; only the (future) freeze path sets it.
- **Zero `time.Time` → NULL convention (`unixOrNil`):** a zero `ObservedAt` writes SQL NULL, not `0`, so
  "never observed" stays distinct from the unix epoch. `FollowState` reads `last_size`/`last_error`
  through `sql.NullInt64`/`sql.NullString` so a partial/absent row degrades to the zero value, never an
  error — an unknown `hubID` returns `FollowState{}` + nil err by design (follower treats it as "never polled").
- **Freeze path seam is `RecordViolation` (plain INSERT) + `Freeze` (upsert `frozen=1`).** Verified the
  two upserts compose: `Freeze` writes `INSERT … (hub_id, frozen) VALUES (?,1) ON CONFLICT(hub_id) DO
  UPDATE SET frozen=1`, `AdvanceFollowState` omits `frozen` from its `DO UPDATE`, so advance-after-freeze
  keeps `frozen=1` AND moves `last_size` (no auto-unfreeze, ADR-0006) — `TestFreezeNoAutoUnfreeze` is
  non-vacuous (asserts both flags). `RecordViolation` is `LastInsertId`-only (no `RowsAffected` dance —
  no `ON CONFLICT`), so re-detection yields distinct ids/rows (re-detection is itself evidence). The
  `Kind` string rides on the `Violation` struct exactly as `Status` rides on `CheckpointRecord`, keeping
  store import-free of `logclient`.
- **The `net`/`net/netip`/`net/url` in `go list -deps ./internal/store` are from `modernc.org/sqlite`,
  NOT iscc-monitor code.** The load-bearing invariant is "no `net/http` in the store closure" — verify
  with `go list -deps ./internal/store | grep '^net/http'` (empty) and that the package's own `.Imports`
  are exactly `context database/sql embed errors fmt time` + the sqlite driver. Do not flag the bare
  `net` lines as a leak.
- **`RecordHubKey` is the did:web key cache write — a guarded `UPDATE … WHERE hub_id=? AND key_id=?`
  then `INSERT` on zero `RowsAffected` (the `SetCoverage` idiom; `hub_keys` has no UNIQUE so no
  `ON CONFLICT`).** The UPDATE rewrites *all* mutable columns, so a re-resolve genuinely tracks the
  DID doc as source of truth — verified it clears `pubkey_z` back to NULL when the multibase drops out
  (not just append). `nullStringOrNil` (empty→NULL) joins `unixOrNil` (zero-time→NULL) so "no
  multibase"/"not revoked" stay distinct from `""`/epoch. FK is genuinely enforced (orphan insert →
  SQLite `FOREIGN KEY constraint failed (787)`, independently reconfirmed). `key_id uint32→int64` cast
  mirrors `RecordCheckpoint`'s `uint64→int64`. Store stays a leaf (zero internal deps, no `net/http`).
  Oracle gate N/A — plain CRUD, no proof/verify/didweb/merkle path, go.mod/go.sum byte-identical. The
  *reader* and follower→store wiring (map `ResolveVerifierKey`'s `DIDKey`→`HubKey`) are the next slice,
  intentionally deferred.
- **Coverage set-once is a guarded `UPDATE … WHERE monitored_since_size IS NULL` keyed on the SIZE
  column being NULL — and that guard is correct even for a size-0 start.** The first `SetCoverage`
  writes `int64(size)` (so the column is NOT NULL afterward, even when size==0), making every re-call a
  silent no-op regardless of size; the start is immutable at size 0 too (reviewer added a throwaway
  `TestCoverageZeroSizeStillSet` — PASS — then removed it; the committed suite does not cover the
  size-0 edge but the live `PollHub` only ever records `info.TreeSize` from a verified checkpoint).
  `SetCoverage` intentionally ignores `RowsAffected` (zero-rows-after-set is the correct non-error
  case), mirroring how `next.md` scoped it. `Coverage` reads both columns through `sql.NullInt64` and a
  zero `monitored_since_time` (NULL via `unixOrNil`) degrades to a zero `time.Time` with `Set` still
  true — so "coverage started, time unknown" is representable but unreachable from `PollHub` (which
  always injects a real `observedAt`). Wiring lives ONLY on the verified, non-violation `PollHub` path
  (between `RecordCheckpoint` and `AdvanceFollowState`), kept out of `freeze`, so a contradictory
  observation never starts coverage (ADR-0001) — the fork/unverified tests assert `cov.Set == false`.
