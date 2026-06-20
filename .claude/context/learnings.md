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
