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
- **Fork is the second dep-free trigger and landed in the same file as shrink.** `CheckFork(prevSize,
  prevRoot [rootBytes]byte, nextSize, nextRoot [rootBytes]byte) bool == prevSize > 0 && nextSize ==
  prevSize && nextRoot != prevRoot`. Same-size + differing root only; growth/shrink/identical-root all
  false, and the `prevSize > 0` guard keeps the fresh-store zero from misreading. Uses array `!=` (Go
  elementwise on fixed-size `[32]byte`) — `consistency.go` stays import-free of `bytes`/any dep
  (verified: only `bytes` occurrence is the comment explaining why none is needed). `ViolationFork
  ViolationKind = "fork"` matches the `violations.kind` string. Only equivocation (RFC-6962
  consistency-proof) now remains deferred to the merkle-backed slice. Test guard `if rootA == rootB
  { t.Fatal }` makes the "different root" cases non-vacuous.

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
