# Next Work Package

## Step: Add pure `logclient.KeyIDFromCheckpoint` — recover the signed-note keyhash from raw checkpoint bytes

## Goal
Add a pure helper that extracts the C2SP signed-note `(name, keyID)` directly from a raw checkpoint's
signature line — **without** fetching did.json. This is the missing piece that lets the verified
`PollHub` path consult `store.LookupHubKey(hubID, keyID)` *before* re-resolving the key (the handoff's
"thread `LookupHubKey` into `PollHub` to skip the second did.json fetch"). The cache lookup key is the
keyhash; today the only source of the keyhash is `ResolveVerifierKey`, the very fetch we want to skip —
this helper breaks that chicken-and-egg without touching `AcceptCheckpoint`'s signature or the schema.

## Scope
- **Create**: `internal/logclient/checkpointkey.go` — `KeyIDFromCheckpoint(raw []byte) (name string, keyID uint32, err error)`
- **Create**: `internal/logclient/checkpointkey_test.go` — golden + edge-case tests (test file, not counted)
- **Modify**: (none — the only production file is the new one)
- **Reference**:
  - `/home/dev/go/pkg/mod/golang.org/x/mod@v0.33.0/sumdb/note/note.go` — `note.Open` (lines 504–600),
    `note.Signature{Name, Hash, Base64}` (line 450), `*UnverifiedNoteError{Note}` (lines 462–464),
    `VerifierList()` (line 415). On an empty verifier list every sig line lands in `UnverifiedSigs` with
    its `Hash` already decoded as `binary.BigEndian.Uint32(sig[0:4])` (note.go:554), and `Open` returns
    `*UnverifiedNoteError` because `len(n.Sigs) == 0` (note.go:596–597).
  - `/workspace/iscc-monitor/internal/logclient/keyid.go` — the sibling `KeyIDFromVerifier` (recovers the
    same keyhash from the *vkey string*); the new helper recovers it from the *raw checkpoint*. Same
    `uint32` id type, same value for the same hub.
  - `/workspace/iscc-monitor/internal/logclient/verify.go` — already calls `note.Open` for the framing;
    follow its idiom. Learnings rule: "never re-parse the sig line by hand — let `note.Open` own it."
  - `/workspace/iscc-monitor/testdata/live/sb0.iscc.id_checkpoint` — golden vector (signer
    `sb0.iscc.id/log`, keyhash `0x40b74463`; the follower test `readCheckpoint(t, "sb0.iscc.id_checkpoint")`
    loads it from `testdata/live/`).

## Not In Scope
- **Do NOT modify `internal/follower/follower.go`.** Wiring `cacheHubKey` to consult `LookupHubKey` (the
  cache-hit fast path that actually skips the fetch) is the *next* slice; this step lands only the pure
  prerequisite so the follower change stays a small, separately-verifiable increment.
- **Do NOT change `AcceptCheckpoint`'s signature** to surface its already-resolved key (explicitly out of
  scope per the prior `next.md` and learnings; this helper is the alternative that avoids that refactor).
- Do not add `valid_from`/`valid_until` columns to `hub_keys` or attempt full CID-1.0 window re-checking
  from the cache — the cached row carries only `revoked_at`; the schema is unchanged this step.
- No new `go.mod`/`go.sum` dependency — `golang.org/x/mod/sumdb/note` is already wired.
- No `transparency-dev/merkle`, equivocation trigger, tile fixtures, structured logs, or `/metrics`.

## Implementation Notes
- **Use `note.Open`, never a hand-written sig-line parser.** Call `note.Open(raw, note.VerifierList())`
  — pass `note.VerifierList()` with no args for an *empty* list (not `nil`). With no known verifier every
  sig line lands in `UnverifiedSigs`, and `Open` returns a `*note.UnverifiedNoteError` whose embedded
  `.Note` exposes `UnverifiedSigs[0].Name` and `.Hash` (the BE-uint32 keyhash the library already decoded).
- **Extract via `errors.As`**, not a string match: `var ue *note.UnverifiedNoteError; if
  errors.As(err, &ue) { … }`. On that path read `ue.Note.UnverifiedSigs`. A non-`UnverifiedNoteError`
  error (a malformed note) is a real parse failure — wrap it with `%w` and return. A `nil` error from
  `Open` is unreachable with an empty verifier list (no sig can verify), but if it ever returns one,
  treat zero `UnverifiedSigs` as an error rather than indexing `[0]`.
- **Guard `len(ue.Note.UnverifiedSigs) >= 1`** before `[0]` (defensive; `Open` only emits the error after
  appending ≥1, but never index-panic). Take `[0].Name` and `[0].Hash` — checkpoints carry one hub sig in
  v1. (M7 cosigner note: a second unverified sig line could appear later; `[0]` is the hub sig today. A
  future multi-sig selection is out of scope and flagged in the docstring, not handled here.)
- **Return shape `(name string, keyID uint32, err error)`** mirrors `VerifyCheckpoint`'s field-style
  return and `KeyIDFromVerifier`'s `uint32`. Return `name` (the signer name, `sb0.iscc.id/log`) so the
  follower can later assert it equals the hub origin before trusting a cache hit — do not discard it.
- **Purity / imports:** exactly `{errors, fmt, golang.org/x/mod/sumdb/note}` — stdlib + the already-wired
  note dep, no `net`/`os`/`sqlite`. `binary` is NOT needed (note already decoded the `Hash`).
- **Correctness rule (learnings):** the vkey's middle `+<hex>+` field and the sig line's keyhash are the
  *same* value, so `KeyIDFromCheckpoint` and `KeyIDFromVerifier` must agree for one hub. The golden test
  pins both to `0x40b74463` for sb0, catching any future divergence.
- Start the file with a docstring: it recovers the keyhash from the *raw checkpoint* (answering "which
  cached key id signed this, before resolving did.json"), distinct from `keyid.go` which reads the vkey
  string after resolution.

## Verification
- `mise run check` is green (build + vet + test, all packages).
- `gofmt -l internal/logclient/checkpointkey.go internal/logclient/checkpointkey_test.go` prints nothing.
- `go test -run TestKeyIDFromCheckpoint ./internal/logclient` passes, asserting on the real sb0 fixture
  (`testdata/live/sb0.iscc.id_checkpoint`): `name == "sb0.iscc.id/log"` and `keyID == 0x40b74463`.
- Cross-check in the same test: `KeyIDFromCheckpoint(raw)`'s `keyID` equals
  `KeyIDFromVerifier(VerifierKey("sb0.iscc.id/log", sb0pub))` for the sb0 fixture — the two recovery
  paths agree (derive `sb0pub` via the existing didweb/logclient fixtures or pin the known `0x40b74463`).
- Garbled-input case: `KeyIDFromCheckpoint([]byte("not a note"))` returns a non-nil error and does NOT
  panic.
- `internal/logclient/checkpointkey.go`'s import block is exactly `{errors, fmt,
  golang.org/x/mod/sumdb/note}` (no new dep beyond what logclient already pulls).

## Done When
`logclient.KeyIDFromCheckpoint` recovers `("sb0.iscc.id/log", 0x40b74463)` from the sb0 checkpoint
fixture, agrees with `KeyIDFromVerifier`, errors cleanly (no panic) on garbled input, and `mise run check`
is green with `gofmt` clean — all without touching `follower.go`, `AcceptCheckpoint`, the schema, or go.mod.
