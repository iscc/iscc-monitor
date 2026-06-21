<!-- area: internal/didweb (url.go, validity.go, parse) -->
<!-- indexed-as: didweb.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# `internal/didweb` — did:web resolution & validity seam

Read this when a step touches the area above. Durable cross-cutting rules live in
the index (`.claude/context/learnings.md`); the package-local mechanics are here.

## did:web resolution & validity seam (from Go / tooling conventions)

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
