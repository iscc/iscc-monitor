# Next Work Package

## Step: Certificate §4 SIGNING KEY — render the cached did:web key that signed the accepted checkpoint

## Advances
M-UI (Evidence Ledger frontend) Verify criterion — the certificate-clause requirement:

> the **realm-wide certificate** (`/inclusion/{iscc_id}`, keyed by the self-describing ISCC-IDv1 …) for
> a known id renders the numbered evidence clauses (subject + position; checkpoint `(size, root)`;
> inclusion proof; **signing key**; anchor state; full per-id record history incl. any deletion) …

§1 SUBJECT, §2 CHECKPOINT, and §3 INCLUSION PROOF are PASS-verified; `HasClause4..6` are all still
`false` and never set (`handler.go:160`, the empty `§4` placeholder at `cert.html:356-361`). This step
closes the **§4 signing key** sub-clause — the next unmet piece of the single open M-UI certificate
criterion, continuing the established clause-by-clause arc (the review handoff `**Next:**` from `96f6ed9`
names "§4 SIGNING KEY → §5 → §6"). Milestone work, not a preempting issue.

## Goal
Render the certificate's §4 SIGNING KEY clause: the did:web-resolved Ed25519 key that *actually signed*
the §2 accepted checkpoint — its key id derived from that checkpoint's own raw signature line, then
looked up in the `hub_keys` cache. This grounds the certificate's "the hub vouched for this with key X"
claim in the irreplaceable signed checkpoint, and fails closed (omits §4, never fabricates a key) when
the key is not cached.

## Scope
- **Create**: (none)
- **Modify** (2 non-test source files, within the ≤3 budget):
  - `/workspace/iscc-monitor/internal/certificate/handler.go` — capture the accepted checkpoint's raw
    bytes from `CheckpointAt` (currently discarded), derive `key_id` via `KeyIDFromCheckpoint`, look up
    the cached key via `LookupHubKey`, populate the §4 `certData` fields + `HasClause4`; add the §4
    view-model fields with evergreen docstrings; update the file/`buildData`/`certData` docstrings to
    describe §4.
  - `/workspace/iscc-monitor/internal/certificate/cert.html` — fill the existing empty `§4 SIGNING KEY`
    `clause-value` placeholder (`<div class="clause-value"></div>`, cert.html:356-361) with the key
    fields, reusing the existing `clause-mono` / `clause-note` classes (no new CSS).
  - `/workspace/iscc-monitor/internal/certificate/handler_test.go` — test file, not counted toward the
    ≤3 budget. Add the two §4 tests + thread a real signed-note checkpoint `Raw` into `fixtureStoreTiled`.
- **Reference** (read before implementing):
  - `/workspace/iscc-monitor/.claude/context/learnings/certificate.md` — package-local mechanics:
    buffer-then-200 500 discipline, base64-Std cross-surface, the `html/template` `+`→`&#43;` escape
    trap, the §2 `CheckpointAt` read that already exposes the raw bytes.
  - `/workspace/iscc-monitor/.claude/context/learnings/didweb.md` — `HubKey` semantics; did:web is the
    only key source (ADR-0009); the `unresolvable`/`unverified`/rotation distinctions (context only —
    §4 reads the *cache*, it does NOT re-resolve did.json).
  - `/workspace/iscc-monitor/internal/logclient/checkpointkey.go` — `KeyIDFromCheckpoint(raw) (name,
    keyID uint32, err)`: the pure (stdlib + sumdb/note) raw-checkpoint key-id recovery; rejects a
    non-note input with an error (no panic).
  - `/workspace/iscc-monitor/internal/store/checkpoints.go:62-78` (`HubKey` struct) and `:443-485`
    (`LookupHubKey(hubID, keyID) (HubKey, found, err)` — absent row is `(HubKey{}, false, nil)`, not an
    error; only a real query fault is non-nil).
  - `/workspace/iscc-monitor/internal/follower/follower.go:263-356` (`cacheHubKey` /
    `KeyIDFromCheckpoint` / `RecordHubKey`) — the production WRITE side §4 reads back; §4 is its mirror.
  - `/workspace/iscc-monitor/.claude/design/ISCC Monitor - Certificate.dc.html:65` — the §4 mockup: a
    mono DID line (`did:web:hub.iscc.id#key-2025`) + note "Ed25519, resolved from the hub's did:web
    document."

## Not In Scope
- §5 Bitcoin anchor and §6 record history (`HasClause5`/`HasClause6` stay `false`) — separate later
  sub-steps in the same arc.
- The downloadable proof-bundle assembler (`{checkpoint, inclusion/consistency proof, record bytes,
  hub key, ots?}`) — its own oracle-gated step; the §4 key it bundles can reuse this read path later.
- Re-resolving the hub's `did.json` live, or ANY `net`/`net/http` work — §4 reads the `hub_keys` cache
  only. Do not add an HTTP fetch to the handler.
- A CID 1.0 validity-window engine (`DIDKey.ValidAt`). The cache row carries a `Revoked` instant; you
  may surface it if present, but do not build validity/rotation evaluation here.
- The deferred `internal/registry` `hubDomain` `ForceQuery` fix — this step does not touch `registry.go`.
- Any change to `proofserve`'s `serveVerify` or to §1/§2/§3 logic — additive only.

## Implementation Notes
- **Derive the key id from the §2 accepted checkpoint, not synthetically.** `CheckpointAt` already
  returns the accepted checkpoint's raw bytes as its second return value (today discarded with `_` in
  `buildData`'s §2 branch). Capture it (`root, raw, found, err := st.CheckpointAt(...)`) and, INSIDE the
  `if found` / `data.HasClause2` region (so §4 is meaningful only alongside §2), call
  `name, keyID, err := logclient.KeyIDFromCheckpoint(raw)`. This is the exact key the hub signed the
  accepted checkpoint with — the right key to display; it equals `KeyIDFromVerifier` of the resolved
  vkey (proven by `logclient/checkpointkey_test.go`). The `name` return is unused for rendering (the
  DID is built from the resolved `data.Domain`); discard or assert it as you prefer.
- **Read the cache, fail closed.** `key, found4, err := st.LookupHubKey(r.Context(), hub.HubID, keyID)`.
  Set `HasClause4 = true` ONLY on `found4`. On a `KeyIDFromCheckpoint` error (malformed sig line — e.g.
  the cheap `[]byte("raw")` fixtures) or a cache miss (`!found4`), leave §4 unrendered: an honest
  decline, NOT a 500, NOT a fabricated key (the same discipline as §3's honest tile-gap). A real
  `LookupHubKey` DB error is a 500, buffered before any 200 (`return certData{},
  http.StatusInternalServerError`) — the buffer-then-200 invariant is already in place.
- **View-model fields** (add to `certData` with evergreen docstrings like the existing §2/§3 fields):
  - `SigningKeyDID string` — `"did:web:" + data.Domain` (the hub's did:web identifier; matches ADR-0009
    "domain ownership is identity").
  - `SigningKeyID string` — `fmt.Sprintf("%08x", keyID)`, the BE-uint32 signed-note keyhash in hex
    (matching how the codebase prints key ids, e.g. `LookupHubKey`'s `%08x` error format).
  - `SigningKeyMultibase string` — `key.PubkeyZ`, the z6Mk… multibase (may be empty if the cache row
    stored NULL pubkey_z; render that chip conditionally in the template).
  - (optional) `SigningKeyRevoked string` — only set when `key.Revoked` is non-zero (RFC-3339); omit
    otherwise. Keep these names parallel to the §2 `Checkpoint*` / §3 `Proof*` fields.
- **Template (`cert.html` §4 block):** render the DID mono line + the hex key id + (conditionally) the
  multibase chip, plus the mockup note "Ed25519, resolved from the hub's did:web document." Reuse
  `clause-mono` / `clause-note` (same markup as §2/§3). No external/CDN URL; no new `<style>` rule.
- **Correctness rule (learnings index):** did:web is the only key source (ADR-0009). §4 must read the
  cached resolution (`hub_keys`), never invent a key; a cache miss is an honest "key not yet resolved"
  decline, not an affirmative claim. The displayed key MUST be the one tied to the accepted checkpoint
  (derive via `KeyIDFromCheckpoint`), so the certificate can never show a key that did not sign what §2
  vouches for. Also: coverage honesty (ADR-0001) — never render a guarantee the accepted state does not
  support.
- **`html/template` escape (learnings):** the hex key id is `[0-9a-f]` and the z6Mk multibase is base58
  (no `+`/`/`), so the `+`→`&#43;` entity escaping is unlikely to bite, but a test asserting on rendered
  values should `html.UnescapeString(body)` first defensively (the §3 test already does this).
- **Test fixture — make §4 testable without weakening §3.** `fixtureStoreTiled` (handler_test.go:499)
  currently passes `Raw: []byte("raw")` to `AdvanceAccepted`, which `KeyIDFromCheckpoint` rejects (the
  honest-decline path — fine for the §3 tests). Thread the checkpoint `Raw` in as a helper parameter
  (or add a thin `fixtureStoreTiledKey` wrapper) so the existing §3 callers keep the cheap
  `[]byte("raw")` and ONLY the §4 happy-path test supplies a REAL signed note. For the real note, read
  `testdata/live/sb0.iscc.id_checkpoint` (its sig line yields keyID `0x40b74463`, pinned in
  `logclient/checkpointkey_test.go`) and pass it as `CheckpointRecord.Raw`; then `st.RecordHubKey(ctx,
  store.HubKey{HubID: target, KeyID: 0x40b74463, PubkeyRaw: <32 bytes>, PubkeyZ: "z6Mk…test",
  ResolvedAt: <some time>})`. `KeyIDFromCheckpoint` does not verify the signature, so the live sb0 note
  seeds an sb1-indexed fixture fine (it only reads the BE-uint32 keyhash).
  - Locate the repo-root `testdata/` from the test's package dir — the certificate package is two levels
    under root, so the path is `filepath.Join("..", "..", "testdata", "live", "sb0.iscc.id_checkpoint")`
    (verify against how a sibling package reads it; `logclient` uses a `readCheckpoint` helper with its
    own relative base — do not assume the same base).
- **Two new tests:**
  - `TestCertificateSigningKey` — tiled fixture WITH a real note `Raw` + a seeded `RecordHubKey`:
    assert the body contains `§4 SIGNING KEY`, the `did:web:<domain>` line, the `40b74463` key id, and
    the seeded multibase. §1/§2/§3 must also still render (regression).
  - `TestCertificateSigningKeyUncached` — same real-note tiled fixture but NO `RecordHubKey`: body has
    §1/§2/§3 but NOT `§4 SIGNING KEY` (the honest cache-miss decline; status 200).
  - Existing `TestCertificateInclusionProof*` / `TestCertificate*` must stay green; if you change
    `fixtureStoreTiled`'s signature, update its callers.
- **Non-vacuity (mandatory — review reproduces it):** forcing `HasClause4 = true` unconditionally (or
  rendering §4 on a cache miss) must make `TestCertificateSigningKeyUncached` FAIL; restoring the gate
  passes. State this in the test docstring and confirm locally before handoff.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -count=1 -run TestCertificate ./internal/certificate` passes uncached (all existing §1/§2/§3
  tests plus `TestCertificateSigningKey` and `TestCertificateSigningKeyUncached`).
- Oracle/conformance gate (the signed-note key-id path is touched):
  `go test -count=1 ./internal/logclient ./internal/follower ./cmd/notecheck` all `ok`.
- **Mutation (non-vacuity):** forcing `HasClause4 = true` unconditionally makes
  `go test -run TestCertificateSigningKeyUncached ./internal/certificate` FAIL; restoring the gate
  passes. (Run, confirm, revert — leave the tree clean.)
- WASM/purity unaffected: `GOOS=js GOARCH=wasm go build ./internal/index ./internal/didweb` exits 0
  (no `net`/`net/http` added to any WASM-shared leaf).
- No new dependency: `git diff --stat go.mod go.sum` is empty.

## Done When
`mise run check` is green and the certificate renders a §4 SIGNING KEY clause for a known id whose
did:web key (derived from the accepted checkpoint and read from `hub_keys`) is cached, while honestly
omitting §4 — no 500, no fabricated key — on a cache miss, proven non-vacuous by the two new
`TestCertificateSigningKey*` tests.
