# Handoff

## 2026-06-20 — Add pure `logclient.KeyIDFromCheckpoint` — recover the signed-note keyhash from raw checkpoint bytes

**Done:** Added `KeyIDFromCheckpoint(raw []byte) (name string, keyID uint32, err error)` to
`internal/logclient`. It recovers the signer name and signed-note key id directly from a raw
checkpoint's signature line via `note.Open(raw, note.VerifierList())` (empty verifier list), reading
the decoded `(Name, Hash)` off the `*note.UnverifiedNoteError`'s first unverified sig. This breaks the
chicken-and-egg so the verified `PollHub` path can consult `store.LookupHubKey(hubID, keyID)` before
re-resolving did.json — without touching `follower.go`, `AcceptCheckpoint`, the schema, or go.mod.

**Files changed:**
- `internal/logclient/checkpointkey.go` (new): the pure helper. Uses `errors.As` (not a string match)
  to extract `*note.UnverifiedNoteError`; wraps any non-`UnverifiedNoteError` (incl. the unreachable
  nil) with `%w`; guards `len(UnverifiedSigs) >= 1` before indexing `[0]`. Imports exactly
  `{errors, fmt, golang.org/x/mod/sumdb/note}`. Docstring flags M7 multi-sig (cosigner) selection as
  out of scope (`[0]` is the single hub sig in v1).
- `internal/logclient/checkpointkey_test.go` (new): golden + cross-check + garbled-input tests.
- `.claude/context/handoff.md`: this file.

**Verification:** `mise run check` → green, all 7 packages ok. Per criterion:
- [x] `mise run check` (build + vet + test) green.
- [x] `gofmt -l internal/logclient/checkpointkey.go internal/logclient/checkpointkey_test.go` → empty.
- [x] `go test -run TestKeyIDFromCheckpoint ./internal/logclient` → PASS. Asserts on the real sb0
  fixture: `name == "sb0.iscc.id/log"`, `keyID == 0x40b74463`.
- [x] Cross-check: `KeyIDFromCheckpoint(raw).keyID == KeyIDFromVerifier(sb0VKey)` (both `0x40b74463`).
  Reuses the `sb0VKey` constant already in `verify_test.go` (same package) — the two recovery paths
  agree, no key re-derivation needed.
- [x] Garbled input: `"not a note"`, empty, and nil all return a non-nil error and do NOT panic
  (table test).
- [x] Import block is exactly `{errors, fmt, golang.org/x/mod/sumdb/note}`; `go.mod`/`go.sum`
  byte-identical (`git diff --quiet HEAD` exit 0).

**Next:** The follower→store wiring this prerequisite unblocks: thread `LookupHubKey` into the verified
`PollHub` path so `cacheHubKey` skips the second did.json fetch on a cache hit. The intended shape —
`KeyIDFromCheckpoint(raw)` to get `(name, keyID)`, assert `name == origin`, then
`store.LookupHubKey(hubID, keyID)`; on a hit within the CID-1.0 validity window, reuse the cached key
instead of calling `ResolveVerifierKey` again. Note the cached `HubKey` row carries only `revoked_at`
(no `valid_from`/`valid_until`), so a full window re-check from the cache alone is not yet possible —
that schema gap may need its own step before the fast path can fully honor rotation/revocation windows.

**Notes:**
- The nil-error branch of `note.Open` is unreachable with an empty verifier list (no sig can verify),
  but is handled defensively: `errors.As(nil, &ue)` is false → falls into the wrap path, and
  `fmt.Errorf("...: %w", nil)` yields a non-nil error (no panic, no `[0]` index). Both the
  zero-`UnverifiedSigs` guard and the non-`UnverifiedNoteError` guard are belt-and-suspenders against
  a library change.
- Oracle/conformance gate is correctly **N/A**: no signature *verification*, consistency, or proof
  logic changed — this only reads the keyhash `note.Open` already decoded (no crypto, no
  proof/merkle/fsck path), and `go.mod`/`go.sum`/`schema.sql` are byte-identical. The trust-root value
  `0x40b74463` is pinned by the golden test and independently cross-checked against `KeyIDFromVerifier`.
- Scope held to one production file + one test file + handoff; `follower.go`, `AcceptCheckpoint`, the
  schema, and go.mod all untouched as required.
