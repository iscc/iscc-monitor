# Handoff

## 2026-06-20 — Review of: Pure did:web document parser (`internal/didweb/resolve.go`)

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added `parseDIDDocument([]byte) (DIDKey, error)` — a pure (no `net`/`os`/`sql`)
parser that JSON-decodes a hub's `did.json`, resolves the `assertionMethod`-referenced verification
method (string `#fragment` ref or inline object), extracts `publicKeyMultibase` through the
oracle-verified `pubkeyFromDID`, and surfaces optional CID 1.0 validity timestamps without enforcing
them. Code is clean, short, pure functions with evergreen docstrings; scope is exactly the 4 files
`next.md` listed. Every verification criterion passes, and I independently confirmed trust-root parity
against the external `derive_vkey.py` oracle.

**Verification:**
- [x] `mise run check` (build + vet + test) — green, exit 0 (`internal/didweb`, `internal/logclient`).
- [x] `gofmt -l .` — prints nothing.
- [x] `go test -run TestParseDIDDocument ./internal/didweb` — PASS. sb0 →
  `sb0.iscc.id/log+40b74463+AaV+ivnly67hhzQSQfGqCBP3PlOV2NBcmfGyzGdE2ZE5`; sb1 →
  `sb1.amlet.id/log+22b08f3e+ATo2ruguSdJGh11PS76osrQf6OZKrufzzwH/HMwE3a8/`. Validity fields zero.
- [x] Error cases — `TestParseDIDDocumentErrors` PASS: malformed JSON, empty `verificationMethod`,
  missing `publicKeyMultibase`, no `assertionMethod`, bad multibase all return non-nil. Plus
  `TestParseDIDDocumentInlineAssertion` (inline-object form → same sb0 golden) PASS.
- [x] `go test -run TestVerifierKey ./internal/didweb` — PASS (no regression to existing golden).
- [x] Purity — `resolve.go` declares only `encoding/json`, `fmt`, `time`; non-test closure has no
  `net`/`net/http`/`database/sql`. (`os` appears only because `fmt` pulls it in — stdlib, unavoidable.)
  `GOOS=js GOARCH=wasm go build ./internal/didweb` succeeds → stays WASM-shareable.
- [x] **Oracle / trust-root parity (independent)** — re-ran `python3 .claude/derive_vkey.py`: both
  golden vectors print byte-for-byte equal to the test asserts, and the fixtures' `publicKeyMultibase`
  match the oracle's `HUBS` map. The resolve→vkey chain is anchored to the external oracle, not
  self-referential. (`notecheck` CI job not yet wired this early in M1 — no signature-verification code
  exists yet; flagged, not a gate failure.)
- [x] Gate-integrity scan over unpushed commits — no `//nolint`, `t.Skip`, build tags, swallowed
  errors, deleted assertions, or loosened gates. The only matches were the policy prose in
  `handoff.md`/`next.md` themselves.
- [x] Scope — exactly 1 source + 1 test + 2 fixtures + handoff; nothing from `## Not In Scope` (no
  HTTP, no `Resolver` struct, no status assignment, no `hub_keys` table); `go.mod` still
  dependency-free (no `require` block); `vkey.go` untouched.

**Issues found:** (none)

**Minor fixes by reviewer:** removed `.claude/.scratch/` (not gitignored) left behind by running the
`derive_vkey.py` oracle during verification — kept the tree clean.

**Next:** Wire the did:web HTTP fetch at the outbound-fetch seam (follower step): map
`did:web:<domain>` → `https://<domain>/.well-known/did.json`, inject a `Fetcher`/`*http.Client`, fetch
+ call `parseDIDDocument`, and map outcomes to status (`unresolvable` on fetch/parse failure,
`unverified` on signature mismatch). `parseDIDDocument`/`DIDKey`/`pubkeyFromDID`/`verifierKey` are all
unexported today — export the minimal surface the follower needs across the package boundary. The
`hub_keys(... pubkey_z, revoked_at ...)` cache and now-vs-window validity enforcement (consuming the
already-surfaced `ValidFrom`/`ValidUntil`/`Revoked`) belong to that store/follower step.

**Notes:**
- **sb1 fixture deliberately uses the recorded golden key `z6MkiNW46…`, not the live rotated key**
  `z6MkmwqgJABz2DCeESCSqx6JXg2CwASEUvBzxERWV3HZ8yyt` — verified the fixture contains zero occurrences of
  the rotated key. This keeps the resolve→vkey golden coherent (snapshot, not live fetch), per
  `next.md`'s explicit instruction. When the follower lands key-rotation/re-resolve, the live sb1 key
  needs its own current fixture/vector separate from this golden chain.
- **Validity fields surfaced, not enforced (intentional, YAGNI):** `parseTime` returns zero on empty or
  unparseable input. The follower decides what an unparseable/expired value means; the pure parser
  stays decision-free. Watch this lenient-parse choice when enforcement lands — an unparseable `revoked`
  silently becoming "valid" could be a future foot-gun if not re-examined at the enforcement seam.
- **`notecheck` external-oracle CI job** does not exist yet (no signature-verification code at M1).
  The trust-root oracle gate this iteration is `derive_vkey.py` parity, which I ran and confirmed.
- No remote push concern: working branch is `develop`, remote `origin` configured; pushing on PASS.
