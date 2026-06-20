# Handoff

## 2026-06-20 — Review of: Pure checkpoint-acceptance decision (`AcceptCheckpoint`) + close the `parseTime` fail-open gap

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added `internal/logclient/accept.go` — a pure `Status` enum
(`StatusVerified/Unverified/Unresolvable/Rotated` + `String()`) and `AcceptCheckpoint`, which composes
`ResolveVerifierKey → VerifyCheckpoint → DIDKey.ValidAt(observedAt)` into the four-way ADR-0009 verdict
(zero `CheckpointInfo` on every non-verified outcome), and made `internal/didweb`'s `parseTime` return
an error on a non-empty-but-unparseable validity timestamp so a garbled window fails *closed* to
`StatusUnresolvable`. Diff is tightly scoped (2 non-test files + 2 test files), import-clean, oracle
parity intact, and every Verification criterion passes. Independently confirmed the four-way mapping is
real (sb1's prior key forces `unverified`, not a false `unresolvable`) and the trust root is untouched.

**Verification:**
- [x] `mise run check` (build + vet + test) — green; `go build ./...` + `go vet ./...` exit 0.
- [x] `gofmt -l .` — empty (no formatting failures).
- [x] `go test -run TestAcceptCheckpoint ./internal/logclient` — PASS (6 subcases: verified, unverified,
  unresolvable×2 [not-found + malformed-revoked], rotated×2 [validUntil + revoked in the past]).
- [x] `go test -run TestParseDIDDocument ./internal/didweb` — PASS (existing sb0/sb1 goldens + the two
  new malformed-timestamp error subcases).
- [x] Assertion: sb0 fixture + sb0 checkpoint + `observedAt=now` → `StatusVerified`,
  `CheckpointInfo.TreeSize == 10183`, non-zero root, `Origin == sb0.iscc.id/log`.
- [x] Assertion: did.json with `"revoked":"not-a-date"` → `StatusUnresolvable` (NOT `StatusVerified`) —
  the closed `parseTime` fail-open issue.
- [x] Assertion: sb0 signer key with `validUntil`/`revoked` before `observedAt` → `StatusRotated`,
  distinct from unverified/unresolvable.
- [x] `GOOS=js GOARCH=wasm go build ./internal/didweb` exits 0 (parser fix imports only stdlib).
- [x] Trust-root oracle: `python3 .claude/derive_vkey.py` prints both golden vectors byte-exact
  (`sb0…+40b74463+…`, `sb1…+22b08f3e+…`) — derivation path untouched; `.claude/.scratch/` removed.
- [x] Gate-integrity sweep over all unpushed commits (`origin/develop..HEAD`): no `//nolint`, no
  `t.Skip`, no swallowed errors, no removed assertions, no loosened gate, `go.mod`/`go.sum`/`mise.toml`
  untouched.

**Issues found:** (none) — closed the open `normal` `parseTime` fail-open issue (verified fixed + tested);
`issues.md` now empty.

**Next:** Wire the SQLite `hub_keys` cache + the stateful follower poll loop that calls
`AcceptCheckpoint` and persists the verdict — only `StatusVerified` advances accepted state; the other
three are recorded findings while mirroring continues. That step also refreshes the sb1 did.json fixture
+ `derive_vkey.py` `HUBS` to the current key (`069d0f14`). The three-trigger RFC-6962 consistency check
(fork/shrink/equivocation via `transparency-dev/merkle`) is the step after, once tiles land in
`testdata/live/`.

**Notes:**
- **Caller contract to carry into the follower:** `AcceptCheckpoint` returns a non-nil `error` (paired
  with `StatusUnverified`'s zero value) only for a genuine fault — a verified-but-garbled body (a
  non-`ErrUnverified` `VerifyCheckpoint` parse error). So the follower **must check `err` before the
  status**. This keeps "signature didn't match" (a status) separable from "body was garbage" (an error).
  The follower decides how to surface that fault (it is neither a clean four-way verdict nor a
  resolution failure).
- Validity ordering is load-bearing and correctly implemented: `ValidAt(observedAt)` is evaluated
  **only after** a good signature, so an out-of-window key whose signature also fails is `unverified`,
  not `rotated`. The two `rotated` subcases use the *correct* sb0 signer key, so the signature genuinely
  verifies before the window check — the path is real, not contrived.
- No `notecheck` CI oracle job exists in this repo yet (no `cauldron/` compile path, no CI workflow),
  and there is no `internal/proof/` package yet — both are expected pre-M1-store. The active trust-root
  oracle at this stage is `derive_vkey.py`, which is green; nothing in this diff touches Merkle/
  consistency/proof code.
- M1 remains partially met: the four pure primitives (did:web chain, signed-note verify, validity
  predicate, composed acceptance decision) are done and tested, but there is still no SQLite store, no
  follower loop, no binary entrypoint, no consistency check, no coverage/metrics. Loop stays CONTINUE.
