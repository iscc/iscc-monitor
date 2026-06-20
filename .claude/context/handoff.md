# Handoff

## 2026-06-20 — Review of: Verify Ed25519 signed-note checkpoints (`sumdb/note`); map non-matching signature to `unverified`

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `internal/logclient/verify.go` adds a pure `VerifyCheckpoint(vkey, raw) -> (origin,
treeSize, root, err)` that verifies a hub-signed C2SP checkpoint via `golang.org/x/mod/sumdb/note`,
parses the verified three-line body (origin / leading-zero-rejecting decimal size / 32-byte std-base64
root), asserts signer-name == origin, and maps any non-matching signature to a new sentinel
`ErrUnverified` while keeping malformed-but-signed bodies as plain parse errors. Clean scope (1
source + 1 test + 2 live fixtures + go.mod/go.sum), reuses the oracle's exact `note.Open` /
`len(Sigs)==0 || len(UnverifiedSigs)!=0` pattern, and every verification criterion passes. The
real-world sb1 key rotation was handled honestly and is independently confirmed below.

**Verification:**
- [x] `mise run check` (build + vet + test) — green, exit 0; full no-cache `go test ./...` also green.
- [x] `gofmt -l .` — empty.
- [x] `go test -run TestVerifyCheckpoint ./internal/logclient` — PASS (sb0, sb1, + 3 unverified subtests).
- [x] Positive sb0 — `origin == "sb0.iscc.id/log"`, `treeSize >= 1`, non-zero 32-byte root, `err == nil`.
- [x] Positive sb1 — `origin == "sb1.amlet.id/log"`, `treeSize >= 1`, non-zero root, `err == nil`
  against sb1's CURRENT key (`069d0f14`); the cryptographic pass is itself proof the derived key is right.
- [x] Negative (wrong key / stale rotated key / tampered root) — each `errors.Is(err, ErrUnverified)`.
- [x] Parse errors separable — `TestParseCheckpointBody` proves malformed bodies (leading-zero size,
  non-numeric, bad base64, 3-byte root, too-few-lines) return non-nil errors that are NOT `ErrUnverified`;
  `"0"` accepted.
- [x] WASM purity — `GOOS=js GOARCH=wasm go build ./internal/didweb` succeeds; `verify.go` imports only
  `sumdb/note` + stdlib, and `sumdb/note` pulls no `net`/`net/http`/`database/sql` (WASM-shareable).
- [x] Oracle parity — `python3 .claude/derive_vkey.py`: sb0 byte-exact (`…+40b74463+…`); scratch removed.
  sb1 oracle prints the STALE `22b08f3e` (the oracle's `HUBS` map is pre-rotation) — internally
  consistent with the testdata did.json fixtures; flagged as known drift (see Notes).
- [x] **Independent trust-root cross-check (reviewer, not author):** decoded the raw sig lines —
  sb0 sig keyhash `40b74463`, sb1 sig keyhash `069d0f14`, both 64-byte Ed25519 sigs — matching the
  test's `sb0VKey` / `sb1VKey`, with `22b08f3e` correctly distinct. sb1's rotation is real, so the
  "stale rotated key → ErrUnverified" negative is genuine, not contrived.
- [x] Gate integrity — no `//nolint`, `t.Skip`, build-tag exclusion, swallowed error, loosened gate, or
  deleted assertion across all unpushed source commits.

**Issues found:** (none blocking)
- Known, documented drift (out of scope here, file for the follower/`hub_keys` step): `derive_vkey.py`'s
  `HUBS` map and both `sb1.amlet.id_did.json` testdata fixtures still hold sb1's pre-rotation key
  (`22b08f3e`), so the oracle + `TestResolveVerifierKey/sb1` assert the stale vkey. They are internally
  consistent (fixture matches oracle), just stale vs. the live hub — which is exactly why the live
  did:web doc must be re-resolved on cadence with validity-window enforcement.

**Next:** Wire the SQLite `hub_keys` cache + the follower's checkpoint-acceptance path
(`ResolveVerifierKey` → `VerifyCheckpoint`), mapping `ErrUnresolvable → unresolvable` and
`ErrUnverified → unverified` to hub status. This is also where CID 1.0 validity-window enforcement
(`DIDKey.ValidFrom`/`ValidUntil`/`Revoked`, parsed-but-unenforced today) lands — sb1's live rotation
is the concrete motivation. Refresh the sb1 did.json fixture + `derive_vkey.py` `HUBS` entry to the
current key (`069d0f14`) in that step. The three-trigger RFC-6962 consistency check (two checkpoints +
`transparency-dev/merkle`) is the step after.

**Notes:**
- **Conformance/oracle gate scope for this step:** fsck root-rebuild and inclusion cross-check are N/A
  (no SQLiteFetcher, Merkle code, or `internal/proof` exists yet — all later steps). The trust-root
  oracle that *does* apply here is signature/vkey parity, verified two ways: `derive_vkey.py` golden
  parity (sb0 byte-exact) AND the live fixtures actually verifying under `note.Open` (the strongest
  proof). The code mirrors the `notecheck` reference oracle's success check verbatim.
- **No CI / `notecheck` job exists** (`.github/workflows/` still absent). The signature-parity oracle
  this step feeds is the natural CI gate; flag for whoever adds the workflow. CI must also exclude the
  gitignored `cauldron/` trees (see learnings) — `cauldron/` is fully `.gitignore`d so it never reaches
  CI from this tree, but a fresh `go build ./...` that includes it would break without the stubs.
- **`go 1.24.0` directive** (not bare `go 1.24`): verified `x/mod v0.33.0` declares `go 1.24.0`, so a
  bare consumer fails under `-mod=readonly` ("updates to go.mod needed"). `go 1.24.0` is the same
  minimum (Go 1.24, NOT ≥1.25) — a justified canonical-form change, not a version bump. No `toolchain`
  line pinned. This technically deviates from next.md's literal `go 1.24` text but preserves the intent.
- **M7 watch:** the strict `len(UnverifiedSigs) != 0 → reject` check would reject a hub checkpoint that
  also carries a witness cosig line (unknown to the verifier list). Correct for v1 single-hub-sig;
  revisit at the gossip/cosigner step. Captured in learnings.
- Pushed to `origin/develop` on PASS (remote configured). Branch: `develop`.
