# Handoff

## 2026-06-20 — Pure did:web document parser (`internal/didweb/resolve.go`)

**Done:** Implemented `parseDIDDocument([]byte) (DIDKey, error)` — a pure (no `net`/`os`/`sql`)
parser that JSON-decodes a hub's `did.json`, resolves the verification method referenced by
`assertionMethod` (inline object or `#fragment` string reference into `verificationMethod`), extracts
`publicKeyMultibase` through the existing oracle-verified `pubkeyFromDID`, and surfaces optional CID
1.0 validity timestamps (`validFrom`/`validUntil`/`revoked`) without enforcing them. Parse →
`verifierKey` byte-matches the recorded sb0/sb1 golden vectors against captured live fixtures.

**Files changed:**
- `internal/didweb/resolve.go` (new): `parseDIDDocument`, `assertionKey` (verification-method
  selection), `parseTime`, and the `DIDKey`/`didDocument`/`verificationMethod` types. Calls in-package
  `pubkeyFromDID`; `vkey.go` untouched.
- `internal/didweb/resolve_test.go` (new): table-driven golden test (sb0/sb1 fixtures → `verifierKey`
  == recorded vectors, plus validity fields asserted zero), five error cases (malformed JSON, empty
  `verificationMethod`, missing `publicKeyMultibase`, no `assertionMethod`, bad multibase), and an
  inline-object `assertionMethod` case.
- `internal/didweb/testdata/sb0.iscc.id_did.json` (new): captured live fixture, verbatim bytes from
  `next.md`.
- `internal/didweb/testdata/sb1.amlet.id_did.json` (new): fixture using the recorded golden did:key
  `z6MkiNW46…` (not the live rotated key — see Notes).

**Verification:** `mise run check` → green (`go build`/`vet`/`test` all exit 0).
- [x] `gofmt -l .` prints nothing.
- [x] `go test -run TestParseDIDDocument ./internal/didweb` — PASS. sb0 →
  `sb0.iscc.id/log+40b74463+AaV+ivnly67hhzQSQfGqCBP3PlOV2NBcmfGyzGdE2ZE5`; sb1 →
  `sb1.amlet.id/log+22b08f3e+ATo2ruguSdJGh11PS76osrQf6OZKrufzzwH/HMwE3a8/`. All three+ error cases
  return non-nil errors.
- [x] `go test -run TestVerifierKey ./internal/didweb` — PASS (no regression to the existing golden).
- [x] Purity — `resolve.go` directly imports only `encoding/json`, `fmt`, `time`; the non-test
  dependency closure has no `net`, `net/http`, or `database/sql`. (`go list -deps` lists `os` only
  because `fmt` transitively pulls it in — verified `fmt`'s own Deps include `os`; nothing we wrote
  imports it. `os` in the test closure is `os.ReadFile` for fixtures.) `GOOS=js GOARCH=wasm go build
  ./internal/didweb` succeeds, confirming it stays WASM-shareable.

**Next:** Wire the did:web HTTP fetch at the outbound-fetch seam (the follower step): map
`did:web:<domain>` → `https://<domain>/.well-known/did.json`, inject a `*http.Client` / `Fetcher`,
fetch + call `parseDIDDocument`, and map outcomes to status (`unresolvable` on fetch/parse failure).
`DIDKey`, `parseDIDDocument`, and the `pubkeyFromDID`/`verifierKey` helpers are unexported today;
export them (or a small public surface) when the follower needs them across the package boundary. The
`hub_keys(... pubkey_z, revoked_at ...)` cache and now-vs-window validity enforcement are also still
pending and belong to the store/follower step.

**Notes:**
- **Recorded real-world finding (per `next.md`, not acted on):** the *live* `sb1.amlet.id` did.json
  now serves a **rotated** key `z6MkmwqgJABz2DCeESCSqx6JXg2CwASEUvBzxERWV3HZ8yyt`, which differs from
  the `derive_vkey.py` / golden-test key `z6MkiNW46…`. The pinned `sb1` fixture deliberately uses the
  recorded golden key so the resolve→vkey chain stays one coherent golden — this is exactly why
  fixtures are captured snapshots, not live fetches. When the follower lands key rotation / re-resolve
  logic, the sb1 live key will need its own up-to-date fixture/vector separate from this golden chain.
- **Validity fields surfaced, not enforced (YAGNI per `next.md`):** `parseTime` returns zero on empty
  or unparseable input; the live fixtures omit `validFrom`/`validUntil`/`revoked`, so all three are
  zero ("currently valid"). Deciding what an unparseable/expired value means is the follower's call,
  not this pure parser's — flagged here so `review` knows the lenient parse is intentional.
- **Verification-method selection** takes the first `assertionMethod` entry (hub signs checkpoints as
  an assertion). Handles both the live string-reference form and an inline-object form; accepts `type`
  of `Multikey` (live) and tolerates `Ed25519VerificationKey2020` (the `type` field is parsed but not
  gated — key bytes are the source of truth via `pubkeyFromDID`).
- Scope: exactly the files `next.md` listed (1 source + 1 test + 2 data fixtures); `vkey.go`
  untouched; nothing from `## Not In Scope` (no HTTP, no `Resolver` struct, no status assignment, no
  `hub_keys` table, no new module deps — still `go.mod` dependency-free).
