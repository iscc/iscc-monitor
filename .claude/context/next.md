# Next Work Package

## Step: Pure checkpoint-acceptance decision (`AcceptCheckpoint`) + close the `parseTime` fail-open gap

## Goal
Compose the three existing verification primitives (`ResolveVerifierKey` → `VerifyCheckpoint` →
`DIDKey.ValidAt`) into ONE pure function that returns the four-way hub-status verdict the follower
will consume, and make a malformed CID 1.0 validity timestamp fail *closed* at the parse boundary.
This is the natural seam between today's pure primitives and the upcoming stateful follower, and it
resolves the open `normal` `parseTime` issue — both prerequisites for the SQLite `hub_keys` + follower
loop that follows.

## Scope
- **Create**:
  - `/workspace/iscc-monitor/internal/logclient/accept.go` — a pure `Status` enum + `AcceptCheckpoint` decision function.
  - `/workspace/iscc-monitor/internal/logclient/accept_test.go` — table-driven test (test file, not counted).
- **Modify**:
  - `/workspace/iscc-monitor/internal/didweb/resolve.go` — make `ParseDIDDocument` reject a
    non-empty-but-unparseable `validFrom`/`validUntil`/`revoked` (fail-closed); keep absent → zero
    (unconstrained) unchanged. (1 of ≤3 non-test/doc files.)
- **Reference**:
  - `/workspace/iscc-monitor/internal/logclient/verify.go` — `VerifyCheckpoint` signature, `ErrUnverified`.
  - `/workspace/iscc-monitor/internal/logclient/didresolve.go` — `ResolveVerifierKey` signature, `ErrUnresolvable`, the `Fetcher` seam.
  - `/workspace/iscc-monitor/internal/didweb/validity.go` — `DIDKey.ValidAt(now) bool`.
  - `/workspace/iscc-monitor/internal/didweb/resolve.go` — `parseTime`, `DIDKey`, `ParseDIDDocument`.
  - `/workspace/iscc-monitor/.claude/adr/0009-didweb-trust-root.md` — the `verified`/`unresolvable`/`unverified` taxonomy; an out-of-window key is rotation/revocation, not `unverified`.
  - `/workspace/iscc-monitor/.claude/context/issues.md` — the `parseTime` fail-open issue this step closes.

## Not In Scope
- **No SQLite / `hub_keys` cache / persistence of any kind.** This step is pure and in-memory; the
  store lands in the next step and consumes `AcceptCheckpoint`.
- **No follower loop, no polling, no goroutine, no `cmd/` binary, no freeze/alert.** Decision only.
- **No RFC-6962 three-trigger consistency check** (fork/shrink/equivocation) — that is the step after
  the store, and needs tiles in `testdata/live/` that do not exist yet.
- **No sb1 fixture / `derive_vkey.py` `HUBS` refresh to `069d0f14`.** That belongs with the `hub_keys`
  cache step; touching it here would mix concerns and risk the golden-vector oracle. The existing
  `22b08f3e` fixture (and the genuine "stale rotated key → `ErrUnverified`" negative) stays as-is.
- **No change to `DIDKey.ValidAt` or `validity_test.go`** — the fail-closed fix lives in the parser,
  not the predicate, so the 12-case boundary golden test is untouched.

## Implementation Notes
- **Decision shape.** Add an exported status type to `accept.go`, e.g.
  `type Status int` with `const ( StatusVerified Status = iota; StatusUnverified; StatusUnresolvable;
  StatusRotated )` plus a `String()` for logs/tests. The names must map onto the ADR-0009 taxonomy:
  `verified` / `unverified` / `unresolvable`, with the out-of-window case as a **distinct** outcome
  (rotation/revocation) — per learnings it is *not* `unverified` and *not* `unresolvable`.
- **`AcceptCheckpoint` signature.** Keep it pure and dependency-injected via the existing `Fetcher`
  seam so tests stay offline. Suggested:
  `func AcceptCheckpoint(ctx context.Context, fetcher Fetcher, baseURL string, raw []byte, observedAt time.Time) (Status, CheckpointInfo, error)`
  where `CheckpointInfo` carries the verified `{Origin string; TreeSize uint64; Root [32]byte}` on
  success (zero on non-`verified`). `observedAt` is passed in (never `time.Now()` inside) so the
  decision is deterministic and table-testable — mirror the `DIDKey.ValidAt(now)` discipline.
- **Ordering of the verdict (this is load-bearing):**
  1. `ResolveVerifierKey(ctx, fetcher, baseURL)` — if `errors.Is(err, ErrUnresolvable)` →
     `StatusUnresolvable`. (A malformed validity timestamp now also collapses here via the parser fix
     below — fail-closed.)
  2. `VerifyCheckpoint(vkey, raw)` — if `errors.Is(err, ErrUnverified)` → `StatusUnverified`. A
     well-signed-but-malformed body (non-`ErrUnverified` parse error) is a real error: return it as a
     wrapped `error`, NOT a status (keep "signature didn't match" separable from "body was garbage",
     exactly as `VerifyCheckpoint` already does).
  3. Only if the signature verified, evaluate `key.ValidAt(observedAt)`: in-window → `StatusVerified`;
     out-of-window → `StatusRotated`. **Validity is checked only after a good signature** — an
     out-of-window key whose signature also fails is `unverified`, not rotated.
- **`parseTime` fail-closed fix (`internal/didweb/resolve.go`).** Today `parseTime` returns the zero
  time for BOTH absent and non-empty-unparseable input, so a garbled `revoked` silently fails open.
  Change `ParseDIDDocument` to distinguish: introduce a parse helper that returns
  `(time.Time, error)` — empty string → `(zero, nil)` (unconstrained, unchanged); non-empty +
  `time.Parse` failure → a wrapped error. `ParseDIDDocument` returns that error (wrapped, like its
  other failures), which `ResolveVerifierKey` already maps to `ErrUnresolvable`. Net effect: a hub
  serving `"revoked": "not-a-date"` resolves to `StatusUnresolvable`, never `verified`. Update the
  `parseTime`/`ParseDIDDocument` doc comments to match the fail-closed semantics.
- **Correctness rules in play (learnings.md):** "did:web is the only key source — resolution failure →
  `unresolvable`; a signature matching no listed key → `unverified`"; the out-of-window outcome is the
  *distinct* rotation/revocation case (learnings, `ValidAt` entry) — do not fold it into `unverified`.
  Keep `accept.go` import-clean of `database/sql`/`sqlite` (it may import `context`/`time`/`errors`
  and the sibling primitives; it is the follower seam, not part of the WASM-pure `internal/proof`).
- **Tests** drive `AcceptCheckpoint` through a fake `Fetcher` returning fixture `did.json` bytes (reuse
  `internal/logclient/testdata/sb0.iscc.id_did.json` + `testdata/live/sb0.iscc.id_checkpoint`) and
  assert on the returned `Status` — never on internals. Cover all four statuses: `verified` (sb0 real
  checkpoint + sb0 did.json + `observedAt` = now), `unverified` (a fetcher returning a mismatching key,
  or a tampered sig byte), `unresolvable` (a fetcher returning `os.ErrNotExist` AND a fetcher returning
  a did.json with a malformed `revoked`), and `rotated` (a did.json fixture whose key matches the
  checkpoint signer but carries `validUntil` in the past relative to `observedAt`). For the
  malformed-timestamp and expired-window cases, construct the did.json bytes inline in the test (do not
  add a committed fixture) so the golden fixtures stay stable.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` exit 0; `gofmt -l .` empty).
- `go test -run TestAcceptCheckpoint ./internal/logclient` passes (all four status subcases).
- `go test -run TestParseDIDDocument ./internal/didweb` passes (existing tests still green after the
  parser change) AND a new subcase asserts a non-empty unparseable `revoked`/`validUntil` makes
  `ParseDIDDocument` return a non-nil error.
- Assertion: `AcceptCheckpoint(ctx, fakeSb0, "https://sb0.iscc.id", sb0CheckpointBytes, time.Now())`
  returns `StatusVerified` with `CheckpointInfo.TreeSize == 10183`.
- Assertion: a fetcher returning a did.json with `"revoked":"not-a-date"` yields `StatusUnresolvable`
  (NOT `StatusVerified`) — the closed `parseTime` fail-open issue.
- Assertion: a did.json whose key matches the sb0 checkpoint signer but with `validUntil` before
  `observedAt` yields `StatusRotated` (distinct from `StatusUnverified` and `StatusUnresolvable`).
- `GOOS=js GOARCH=wasm go build ./internal/didweb` still exits 0 (parser fix imports only stdlib).

## Done When
`AcceptCheckpoint` returns the correct four-way verdict for the verified/unverified/unresolvable/rotated
fixtures, a malformed validity timestamp fails closed to `StatusUnresolvable`, and every Verification
criterion passes with `mise run check` green.
