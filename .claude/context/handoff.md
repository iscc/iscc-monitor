## 2026-06-21 — Certificate §4 SIGNING KEY — render the cached did:web key that signed the accepted checkpoint

**Done:** The realm-wide Certificate of Inclusion (`GET /inclusion/{iscc_id}`) now renders a §4 SIGNING
KEY clause for a certifiable id: the key id is derived from the §2 accepted checkpoint's OWN raw
signature line (`logclient.KeyIDFromCheckpoint`), looked up in the `hub_keys` cache
(`store.LookupHubKey`), and rendered as the hub's `did:web:<domain>` identifier + hex key id (+ the
cached z6Mk multibase). It fails closed — §4 is omitted (never a 500, never a fabricated key) on a
malformed sig line or a cache miss.

**Files changed:**
- `internal/certificate/handler.go`: capture the accepted checkpoint's raw bytes from `CheckpointAt`
  (was discarded with `_`); add the §4 branch inside the `HasClause2` guard
  (`KeyIDFromCheckpoint` → `LookupHubKey` → populate `SigningKey*` + `HasClause4` on a cache hit);
  add four §4 view-model fields (`SigningKeyDID`, `SigningKeyID`, `SigningKeyMultibase`,
  `SigningKeyRevoked`) with evergreen docstrings; update the file/`certData`/`buildData` docstrings
  to describe §4. New imports: `fmt`, `time`.
- `internal/certificate/cert.html`: fill the empty `§4 SIGNING KEY` `clause-value` placeholder with
  the DID + key-id mono line, conditional multibase + revoked chips, and the mockup note ("Ed25519,
  resolved from the hub's did:web document."). Reuses existing `clause-mono`/`clause-note` classes —
  no new CSS, no CDN URL.
- `internal/certificate/handler_test.go` (test): thread a `checkpointRaw []byte` parameter into
  `fixtureStoreTiled` (existing §3 callers keep `[]byte("raw")`; §4 callers pass a real signed note);
  add `liveCheckpointRaw`/`hubIDForDomain` helpers and the `sb0CheckpointKeyID` (0x40b74463) pin; add
  `TestCertificateSigningKey` (happy path: real note + seeded `RecordHubKey` → §4 renders, §1-§3
  regress) and `TestCertificateSigningKeyUncached` (no `RecordHubKey` → §1-§3 render, §4 absent, 200).

**Verification:** `mise run check` → green (all 21 packages `ok`, build + vet + test). Per criterion:
- [x] `go test -count=1 -run TestCertificate ./internal/certificate` → PASS (all §1/§2/§3 + the two
  new §4 tests).
- [x] Oracle/conformance gate (signed-note key-id path touched): `go test -count=1 ./internal/logclient
  ./internal/follower ./cmd/notecheck` → all `ok`.
- [x] Mutation (non-vacuity): changing the §4 gate `if found4 {` → `if found4 || true {` makes
  `TestCertificateSigningKeyUncached` FAIL (uncached hub then renders §4); reverted, tree clean,
  test passes. (The plain `HasClause4 = true` would be unreachable-dead under the `found4` branch;
  `|| true` is the same logical mutation and compiles.)
- [x] `GOOS=js GOARCH=wasm go build ./internal/index ./internal/didweb` → exit 0 (no `net`/`net/http`
  added to any WASM-shared leaf; the handler reads the `hub_keys` cache only, no HTTP fetch).
- [x] `gofmt -l .` empty; `git diff --stat go.mod go.sum` empty (no new dependency).

**Next:** §5 BITCOIN ANCHOR — render the OTS calendar-asserted anchor state for the accepted
checkpoint (the §5 mockup at `.dc.html:66` is a status dot + `block N · <UTC time>` + the "run `ots
verify`" note; ADR glossary: OTS is calendar-asserted here, authoritative check is `ots verify`). It
will read the OTS/anchor store seam (whatever the follower persists) and, like §4, fail closed —
omit §5 when no anchor is recorded. After §6 RECORD HISTORY (the per-id `SeqsForISCCID` list incl. any
deletion record, the last unmet sub-clause), the downloadable proof-bundle assembler closes the M-UI
certificate criterion; it reuses this §4 key read-path + the §3 build+verify crypto path and must keep
the oracle/conformance gate green.

**Notes:**
- The §4 key id is derived from the checkpoint's raw bytes, NOT synthesized, so the certificate can
  never display a key that did not sign what §2 vouches for — the durable "show only what the accepted
  state supports" rule (ADR-0001/0009). The §3 callers' synthetic `[]byte("raw")` checkpoints exercise
  the honest §4 decline for free (`KeyIDFromCheckpoint` rejects them), so `TestCertificateInclusionProof*`
  also implicitly cover "§4 omitted on a non-note checkpoint".
- `SigningKeyRevoked` is surfaced as-is from the cached `revoked_at` instant when non-zero; this clause
  deliberately does NOT evaluate CID 1.0 validity windows (`DIDKey.ValidAt`) — out of scope per next.md.
  No fixture currently seeds a revoked key, so the conditional revoked chip is untested-but-trivial
  (a single `{{if .SigningKeyRevoked}}` guard mirroring the multibase chip); flag for review if you
  want a revoked-key test added.
- Test fixture detail: the live sb0 checkpoint note (keyID 0x40b74463) seeds an sb1-indexed fixture
  because `KeyIDFromCheckpoint` reads only the BE-uint32 keyhash, not the signature — matches the
  next.md guidance and the `logclient/checkpointkey_test.go` pin.
- Scope: exactly 2 non-test source files + the test file (within the ≤3 budget). Working tree clean of
  out-of-scope changes (the agent-browser workstream files from the prior handoff are not present).
