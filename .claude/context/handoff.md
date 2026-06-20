# Handoff

## 2026-06-20 — Verify Ed25519 signed-note checkpoints (reuse `sumdb/note`); map a non-matching signature to `unverified`

**Done:** Added `internal/logclient/verify.go` — a pure `VerifyCheckpoint(vkey, raw) -> (origin,
treeSize, root, err)` that verifies a hub-signed C2SP checkpoint through
`golang.org/x/mod/sumdb/note` (`NewVerifier` + `Open` + `len(Sigs)/UnverifiedSigs` check), parses the
verified three-line body (origin / leading-zero-rejecting decimal `tree_size` / 32-byte std-base64
root), and asserts the signed-note name equals the body origin. Any signature that does not match the
resolved key maps to a new sentinel `ErrUnverified`; a malformed body after a valid signature returns a
plain parse error (never `ErrUnverified`). Added `golang.org/x/mod v0.33.0` as the project's first
reused dependency.

**Files changed:**
- `internal/logclient/verify.go` (new): `VerifyCheckpoint`, `ErrUnverified`, unexported
  `parseCheckpointBody`. Imports only `sumdb/note` + stdlib (`encoding/base64`, `errors`, `fmt`,
  `strconv`, `strings`) — WASM-shareable per the purity rule (the package as a whole isn't WASM-pure
  because `didresolve.go` imports `net/http`, which is fine — the follower seam consumes this, not WASM).
- `internal/logclient/verify_test.go` (new): table-driven positives (live sb0/sb1 fixtures, asserting
  `origin`, `treeSize >= 1`, non-zero 32-byte root — not the drifting size/root); negatives
  (`wrong hub key`, `stale rotated key`, `tampered root`) all asserting `errors.Is(err, ErrUnverified)`;
  a `parseCheckpointBody` table proving malformed bodies (leading-zero size, non-numeric, bad base64,
  3-byte root, too-few-lines) are plain parse errors, not `ErrUnverified`, and that `"0"` is accepted.
- `testdata/live/sb0.iscc.id_checkpoint`, `testdata/live/sb1.amlet.id_checkpoint` (new): raw live
  checkpoints captured by `curl`, byte-verbatim (single trailing `\n`).
- `go.mod` / `go.sum`: `require golang.org/x/mod v0.33.0`. See **Notes** on the `go` directive.

**Verification:** `mise run check` (build + vet + test) re-run → green, exit 0. `gofmt -l .` → empty.
- [x] `go test -run TestVerifyCheckpoint ./internal/logclient` — PASS (sb0, sb1 + the 3 unverified
  subtests).
- [x] Positive sb0: `VerifyCheckpoint(sb0VKey, sb0 fixture)` → `origin == "sb0.iscc.id/log"`,
  `treeSize == 10183 (>=1)`, non-zero 32-byte root, `err == nil`.
- [x] Positive sb1: `origin == "sb1.amlet.id/log"`, `treeSize == 61`, non-zero root, `err == nil` —
  against sb1's **current** key (see finding).
- [x] Negative: sb1-key-on-sb0-checkpoint and stale-sb1-key-on-sb1-checkpoint and tampered-sb0-root →
  each `errors.Is(err, ErrUnverified)`.
- [x] Parse errors stay separable: malformed bodies return non-nil errors that are NOT `ErrUnverified`.
- [x] `GOOS=js GOARCH=wasm go build ./internal/didweb` succeeds (didweb untouched); `verify.go` import
  block confirmed to be `sumdb/note` + stdlib only.
- [x] Oracle re-run (`python3 .claude/derive_vkey.py`): sb0 prints byte-exact (`…+40b74463+…`); scratch
  `rm -rf`'d. (sb1 oracle output is stale — see finding.)

**Next:** Wire the SQLite `hub_keys` cache + the follower's checkpoint-acceptance path that calls
`ResolveVerifierKey` then `VerifyCheckpoint`, mapping `ErrUnresolvable -> unresolvable` and
`ErrUnverified -> unverified` to hub status. This is also where **CID 1.0 validity-window enforcement**
(`DIDKey.ValidFrom`/`ValidUntil`/`Revoked`, currently parsed-but-unenforced) lands — sb1's live key
rotation (below) is the concrete motivation: the follower must re-resolve and verify against the
hub's *current* did:web key, not a cached/stale one. The three-trigger RFC-6962 consistency check
(needs two checkpoints + `transparency-dev/merkle`) is the step after.

**Notes:**
- **REAL-WORLD FINDING — sb1 rotated its did:web key; `next.md`'s sb1 positive vector is stale.**
  `next.md` (and `derive_vkey.py`, and the `internal/{didweb,logclient}/testdata/sb1.amlet.id_did.json`
  fixtures) carry sb1's OLD key `z6MkiNW…` → vkey keyhash `22b08f3e`. The **current live** sb1
  `/.well-known/did.json` publishes `z6Mkmwqg…` → keyhash `069d0f14`, and the **current live**
  `/log/checkpoint` is signed with `069d0f14`. So the live sb1 checkpoint verifies against its current
  key, NOT the stale golden `22b08f3e` from `next.md` line 99/122. I derived the current sb1 vkey from
  the live did:web key (cross-checked against the keyhash in the live checkpoint's signature line) and
  used it as the sb1 positive constant: `sb1.amlet.id/log+069d0f14+AW9UGZSxDvYFeewtbNU74zEMv12ChQPcuE4veN80nNtb`.
  The stale `22b08f3e` key is now exercised as the `stale rotated key` negative (real rotation →
  `ErrUnverified`). sb0 is unaffected — its current live key still derives to `40b74463` and matches its
  checkpoint. **Drift to resolve in follow-up steps (out of scope here):** (1) `derive_vkey.py`'s `HUBS`
  map and (2) the two `sb1.amlet.id_did.json` testdata fixtures both still hold sb1's pre-rotation key,
  so the oracle prints the stale sb1 vkey and `TestResolveVerifierKey/sb1` asserts the stale vkey. They
  are internally consistent with each other (the fixture matches the oracle), just stale vs. the live
  hub — which is precisely why the trust root must be the *live* did:web doc, re-resolved on cadence
  (plan §2), and why validity-window enforcement matters. Recommend refreshing the sb1 did.json fixture
  + the oracle's `HUBS` entry in the follower/`hub_keys` step.
- **`go` directive is `go 1.24.0` (not `go 1.24`), no `toolchain` line.** `golang.org/x/mod v0.33.0`
  declares a canonical-patch `go` directive, so the installed go1.24.13 toolchain rejects the bare
  `go 1.24` form under `-mod=readonly` ("updates to go.mod needed"). `go 1.24.0` is the canonical zero
  form of the *same* minimum (Go 1.24) — NOT a version bump (next.md's constraint was "don't require
  >=1.25"; this still requires only 1.24). I dropped the auto-injected `toolchain go1.24.13` line so
  go.mod doesn't pin a specific patch toolchain other machines/CI may lack; `go build`/`vet`/`test`
  stay green under both `GOTOOLCHAIN=auto` and `=local` with no further rewrites. Flagging because it
  technically deviates from next.md's literal `go 1.24` text, though it preserves the intent.
- `parseCheckpointBody` is tested directly (same package) for the malformed-body cases because forging a
  *validly-signed* checkpoint with a malformed body requires the hub's private key, which we don't have;
  testing the parser unit keeps the parse-error/`ErrUnverified` separation provable without weakening
  anything (no skips/mocks of the crypto).
- Live checkpoint fixtures live at repo-root `testdata/live/` per the plan layout + next.md scope; the
  test reads them via `../../testdata/live/` (Go runs tests from the package dir). Trailing newline
  preserved (verified with `od -c`).
- No CI / `notecheck` job exists yet (`.github/workflows/` still absent). The signature-parity oracle
  this step would feed (`notecheck`) is the natural CI gate to wire next; flag for whoever adds the
  workflow (and: CI must still exclude the gitignored `cauldron/` trees — see learnings).
- Branch `develop`. Scope held: 1 non-test source (`verify.go`) + 1 test + 2 fixtures + go.mod/go.sum;
  nothing from `## Not In Scope` leaked (no follower loop, no merkle/consistency, no SQLite, no
  validity-window enforcement, no `internal/proof/verify`).
