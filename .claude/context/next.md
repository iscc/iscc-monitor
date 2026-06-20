# Next Work Package

## Step: Pure did:web document parser (`internal/didweb/resolve.go`)

## Goal
Parse a hub's `/.well-known/did.json` bytes into its Ed25519 verifier key — the next M1 unit per the
PASS handoff. Doing the **pure parse** (bytes → key + validity, no HTTP yet) first keeps the unit
golden-testable against a captured live fixture and chains it into the already-verified
`pubkeyFromDID` / `verifierKey`, so the follower step can later wire the HTTP fetch at the
outbound-fetch seam without re-deriving the trust root.

## Scope
- **Create**:
  - `/workspace/iscc-monitor/internal/didweb/resolve.go` — pure `parseDIDDocument([]byte) (...)`:
    JSON-decode a did.json, select the assertion-method verification method, extract its
    `publicKeyMultibase` (z6Mk… did:key) and any CID 1.0 validity fields, return the 32-byte pubkey
    (via the existing `pubkeyFromDID`) plus the parsed key metadata. No `net`/`os`/`sql` imports.
  - `/workspace/iscc-monitor/internal/didweb/resolve_test.go` — table-driven golden + error test
    (test file, not counted toward the 3-file limit).
  - `/workspace/iscc-monitor/internal/didweb/testdata/sb0.iscc.id_did.json` — captured live fixture
    (data file, not counted). Real bytes captured below; do not hand-edit them.
  - `/workspace/iscc-monitor/internal/didweb/testdata/sb1.amlet.id_did.json` — captured fixture whose
    `publicKeyMultibase` is the **recorded** sb1 did:key `z6MkiNW46AUjNmKTV2YNyFi9ANG9wbfYQQoUQgADGwScd9jk`
    (the one the existing `verifierKey` golden vector asserts), so the resolve→vkey chain stays a single
    coherent golden. See Implementation Notes re: the live sb1 key rotation.
- **Modify**: none. Do **not** edit `vkey.go` (its `pubkeyFromDID`/`verifierKey` already do the math and
  are oracle-verified); `resolve.go` calls them in-package.
- **Reference** (read, do not import):
  - `/workspace/iscc-monitor/internal/didweb/vkey.go` — call `pubkeyFromDID` / `verifierKey` from here.
  - `/workspace/iscc-monitor/.claude/adr/0009-didweb-trust-root.md` — key source + status taxonomy.
  - `/workspace/iscc-monitor/.claude/plans/cosmic-baking-octopus.md` lines 56-60, 105, 137-138, 168-170
    (resolver shape; `hub_keys(... pubkey_z, revoked_at ...)`; `unresolvable`/`unverified` semantics).
  - `/workspace/iscc-monitor/cauldron/iscc-hub/iscc_hub/schema.py` lines ~124-182 (DID/verificationMethod
    field shapes — gitignored Python reference, do not import).

## Not In Scope
- **No HTTP / no I/O.** Do not add `net/http`, a `Resolver` struct that fetches, the `did:web:<domain>`
  → `https://<domain>/.well-known/did.json` URL mapping, retries, or caching. That is the follower-step
  seam and lands next. This step is bytes-in → key-out only.
- No `unresolvable`/`unverified` **status assignment**, no `hub_keys` SQLite table, no signature
  verification against the key. Parsing returns data + errors; status mapping is a later store/follower
  concern.
- No new external module dependency — JSON via `encoding/json` (stdlib); keep `go.mod` dependency-free.
- Do not touch `logclient`, `config`, `registry`, or `cmd/`.

## Implementation Notes
- **The fixtures are the oracle.** Capture them exactly as served (already fetched 2026-06-20):
  - `sb0.iscc.id_did.json` (write these bytes verbatim):
    `{"id": "did:web:sb0.iscc.id", "verificationMethod": [{"id": "did:web:sb0.iscc.id#z6MkqbHELZopsq6eKrn6qxiAgRoVvwp2Vp7mPfKsvrbYmGwJ", "type": "Multikey", "controller": "did:web:sb0.iscc.id", "publicKeyMultibase": "z6MkqbHELZopsq6eKrn6qxiAgRoVvwp2Vp7mPfKsvrbYmGwJ"}], "authentication": ["did:web:sb0.iscc.id#z6MkqbHELZopsq6eKrn6qxiAgRoVvwp2Vp7mPfKsvrbYmGwJ"], "assertionMethod": ["did:web:sb0.iscc.id#z6MkqbHELZopsq6eKrn6qxiAgRoVvwp2Vp7mPfKsvrbYmGwJ"], "capabilityDelegation": ["did:web:sb0.iscc.id#z6MkqbHELZopsq6eKrn6qxiAgRoVvwp2Vp7mPfKsvrbYmGwJ"], "capabilityInvocation": ["did:web:sb0.iscc.id#z6MkqbHELZopsq6eKrn6qxiAgRoVvwp2Vp7mPfKsvrbYmGwJ"]}`
  - For `sb1.amlet.id_did.json`, reuse the **same document shape** but set every `z6Mk…` occurrence to
    the recorded did:key `z6MkiNW46AUjNmKTV2YNyFi9ANG9wbfYQQoUQgADGwScd9jk` and `id`/`controller` to
    `did:web:sb1.amlet.id`. This keeps the parse→`verifierKey` result equal to the existing golden
    `sb1.amlet.id/log+22b08f3e+ATo2ruguSdJGh11PS76osrQf6OZKrufzzwH/HMwE3a8/`.
- **Real-world finding to record (for `review`/learnings), NOT to act on this step:** the *live*
  sb1 did.json now serves a **rotated** key `z6MkmwqgJABz2DCeESCSqx6JXg2CwASEUvBzxERWV3HZ8yyt`, which
  differs from the `derive_vkey.py` / golden-test key `z6MkiNW46…`. The pinned fixture deliberately uses
  the recorded key so the golden chain stays consistent; do not "fix" the oracle to the live key here.
  This is exactly why fixtures are captured snapshots, not live fetches.
- **Verification-method selection.** The live docs list the same key under `verificationMethod`,
  `assertionMethod`, etc. For v1 take the verification method referenced by `assertionMethod` (the hub
  signs checkpoints as an assertion; `cryptosuite eddsa-jcs-2022` / `proofPurpose assertionMethod` per
  `schema.py`). `assertionMethod` entries may be either an inline object or a string DID-URL reference
  (`#fragment`) into `verificationMethod` — handle the string-reference case by resolving the fragment to
  the matching `verificationMethod[i].id`. Accept `type` of `Multikey` (live) and tolerate `Ed25519VerificationKey2020`.
- **Key bytes.** Read `publicKeyMultibase`, pass it straight to the existing `pubkeyFromDID` (it already
  strips the `z`, asserts the `0xED 0x01` multicodec, guards short keys, returns bytes `[2:34]`).
  Do not re-implement base58/multicodec — reuse the verified function (Correctness rule: keep the
  defensive `len(raw) < 34` guard intact, learnings).
- **CID 1.0 validity (ADR-0009, plan `revoked_at`).** Parse optional `revoked` (and, if present,
  `validFrom`/`validUntil`) on the verification method into a struct field; the live docs omit them, so
  treat absence as "currently valid." Do **not** implement now-vs-window enforcement logic this step —
  just surface the parsed timestamps so the follower can decide later. Keep it minimal (YAGNI).
- **Errors, not panics.** Return a wrapped error (never panic / swallow) on: invalid JSON, no
  verification method, no resolvable assertion key, missing `publicKeyMultibase`, or a `pubkeyFromDID`
  failure. Functional, short, pure functions; package-level docstring already covers the file's purpose —
  add a one-line file-purpose comment and evergreen docstrings per function.
- Do NOT use `t.Skip`, `//nolint`, build tags, or swallow errors to pass the gate (target quality bar).

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all exit 0).
- `gofmt -l /workspace/iscc-monitor` prints nothing.
- `go test -run TestParseDIDDocument ./internal/didweb` passes and asserts, for the captured fixtures,
  that the parsed pubkey fed through `verifierKey(origin, pub)` equals the recorded golden vectors:
  - sb0 (origin `sb0.iscc.id/log`) → `sb0.iscc.id/log+40b74463+AaV+ivnly67hhzQSQfGqCBP3PlOV2NBcmfGyzGdE2ZE5`
  - sb1 (origin `sb1.amlet.id/log`) → `sb1.amlet.id/log+22b08f3e+ATo2ruguSdJGh11PS76osrQf6OZKrufzzwH/HMwE3a8/`
- `go test -run TestParseDIDDocument ./internal/didweb` also covers error cases: malformed JSON, a
  document with empty `verificationMethod`, and one whose `publicKeyMultibase` is missing — each returns
  a non-nil error.
- `go test -run TestVerifierKey ./internal/didweb` still passes (no regression to the existing golden).
- Purity assertion (mechanical): `resolve.go` imports no `net`, `net/http`, `os`, or `database/sql` —
  e.g. `go list -deps ./internal/didweb` shows no `net/http`.

## Done When
`mise run check` is green, `gofmt -l` is empty, and `go test -run TestParseDIDDocument ./internal/didweb`
passes with the sb0/sb1 fixtures parsing to pubkeys whose `verifierKey` byte-matches the recorded golden
vectors, plus the three error cases returning errors — all from a pure (`net`/`os`/`sql`-free) parser.
