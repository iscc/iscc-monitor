# Handoff

## 2026-06-20 — Review of: Add pure `logclient.KeyIDFromCheckpoint` — recover the signed-note keyhash from raw checkpoint bytes

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** Added the pure `KeyIDFromCheckpoint(raw []byte) (name string, keyID uint32, err error)`
helper to `internal/logclient`. It recovers the C2SP signed-note `(name, keyhash)` from a raw
checkpoint via `note.Open(raw, note.VerifierList())` + `errors.As` on `*note.UnverifiedNoteError`,
reading the already-decoded `UnverifiedSigs[0].{Name,Hash}`. Clean, library-exact, scope-tight (one
production file + one test file + context docs); all gates green.

**Verification:**
- [x] `mise run check` (build + vet + test) — green, all 7 packages ok.
- [x] `gofmt -l internal/logclient/checkpointkey.go internal/logclient/checkpointkey_test.go` — empty.
- [x] `gofmt -l .` (whole tree) — empty.
- [x] `go test -run TestKeyIDFromCheckpoint ./internal/logclient` — PASS (3 tests, incl. 3 garbled
  subcases). Golden asserts `name == "sb0.iscc.id/log"`, `keyID == 0x40b74463` on the real sb0 fixture.
- [x] Cross-check `KeyIDFromCheckpoint(raw).keyID == KeyIDFromVerifier(sb0VKey)` — PASS (both paths agree).
- [x] Garbled input (`"not a note"`, empty, nil) returns non-nil error and does not panic — PASS.
- [x] Import block is exactly `{errors, fmt, golang.org/x/mod/sumdb/note}`; no `net`/`os`/`sqlite` in
  the file (the only "net/os/sqlite" string is the docstring noting their absence).
- [x] `go.mod`/`go.sum`/`internal/store/schema.sql` byte-identical since HEAD~1 (`git diff --quiet` exit 0).
- [x] Quality-gate integrity — scanned unpushed commits (`@{upstream}..HEAD`): no `//nolint`, `t.Skip`,
  `SkipNow`, build-tag exclusion, swallowed error, or deleted assertion in code. The only scan hits are
  prose inside `handoff.md`/`state.md`.
- [x] Oracle/conformance gate correctly N/A — no signature *verification*, RFC-6962/consistency, or
  proof/merkle/didweb/fsck code touched; this reads a keyhash `note.Open` already decoded. Trust-root
  value `0x40b74463` independently re-decoded from the fixture sig line in Python (`base64 → ">I"
  struct → sig[:4]`): `name='sb0.iscc.id/log'`, 64-byte ed25519 sig — confirmed from ground truth.

**Issues found:** (none)

**Next:** Wire `KeyIDFromCheckpoint` into the verified `PollHub` path so `cacheHubKey` consults
`store.LookupHubKey(hubID, keyID)` on a cache hit and skips the second did.json fetch. Intended shape
(from the prior handoff): `KeyIDFromCheckpoint(raw) → (name, keyID)`, assert `name == origin`, then
`LookupHubKey(hubID, keyID)`; on a hit, reuse the cached key instead of calling `ResolveVerifierKey`
again. Watch the schema gap: the cached `HubKey` row carries only `revoked_at` (no
`valid_from`/`valid_until`), so a full CID-1.0 validity-window re-check from the cache alone is not yet
possible — a `hub_keys` schema step may need to precede a fully window-honoring fast path.

**Notes:**
- The garbled-input safety is library-exact, not just defensive: `errMalformedNote` is a plain
  `errors.New` (not a pointer type), so `errors.As(err, &*UnverifiedNoteError)` is correctly false and
  the wrap path fires — the `[0]` index is never reached on a bad note. The `len(UnverifiedSigs) < 1`
  and nil-error guards are genuine belt-and-suspenders against a future library change.
- This is the pure prerequisite only; `follower.go`, `AcceptCheckpoint`, schema, and go.mod are all
  untouched as scoped. M1 is not yet DONE — reader→follower wiring + the equivocation (RFC-6962
  consistency) trigger remain open (per `state.md`), so the loop continues.
- Pushed to `origin/develop`.
