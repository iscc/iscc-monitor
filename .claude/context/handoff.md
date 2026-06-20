# Handoff

## 2026-06-20 — Review of: Bootstrap Go module + golden-tested `origin()` and `verifierKey()`

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` bootstrapped the Go module (`github.com/iscc/iscc-monitor`, `go 1.24`) and
landed two pure M1 units: `origin()` (hub base URL → `<domain>/log`) in `internal/logclient` and the
did:key → C2SP verifier-key derivation in `internal/didweb`, a faithful port of
`.claude/derive_vkey.py`. The diff is tightly scoped (3 source files + 2 test files), all gates are
green, and the verifier-key derivation matches the Python oracle byte-for-byte for both live testnet
hubs. Trust-root golden-vector parity holds.

**Verification:**
- [x] `go.mod` declares `module github.com/iscc/iscc-monitor` — confirmed (`head -1 go.mod`).
- [x] `mise run check` green — `go build`/`vet`/`test` all exit 0.
- [x] `gofmt -l .` prints nothing — confirmed empty.
- [x] `go test -run TestVerifierKey ./internal/didweb` — PASS (sb0, sb1 both assert literal vectors).
- [x] `go test -run TestOrigin ./internal/logclient` — PASS (incl. trailing-slash, no-scheme, port,
  whitespace, and three error cases).
- [x] Oracle cross-check (trust root) — `python3 .claude/derive_vkey.py` emits
  `sb0.iscc.id/log+40b74463+AaV+ivnly67hhzQSQfGqCBP3PlOV2NBcmfGyzGdE2ZE5` and
  `sb1.amlet.id/log+22b08f3e+ATo2ruguSdJGh11PS76osrQf6OZKrufzzwH/HMwE3a8/`, byte-identical to the test
  literals the passing Go test asserts. Independently re-derived both keyids (`40b74463`, `22b08f3e`)
  from the hash preimage to confirm the BE-uint32 + `%08x` + base64-Std path.
- [x] Gate-integrity scan over unpushed commits — no `//nolint`, `t.Skip`, build tags, swallowed
  errors, or deleted assertions.
- [x] Purity — both packages are stdlib-only (no external module deps; go.mod dependency-free as
  scoped). `vkey.go` imports no `net`/`os`/`sql`; `origin.go` uses `net/url` (correct for logclient,
  and `proof/verify` does not exist yet so its purity rule is not yet in play).
- [x] Scope discipline — exactly the files `next.md` listed; nothing from `## Not In Scope` touched.

**Issues found:** (none)

**Next:** did:web resolution (`internal/didweb/resolve.go`): fetch `/.well-known/did.json`, parse
`verificationMethod` to the `z6Mk` did:key, honor CID 1.0 validity windows / `revoked`, and return the
pubkey that feeds `verifierKey`. Inject the HTTP client at the outbound-fetch seam
(`*http.Client` / a `Fetcher` interface) and test against a captured
`testdata/live/.../did.json` fixture (real fixture, not a mock). When `resolve.go` or the follower
needs them across package boundaries, export `origin`/`verifierKey` (or wire a small public surface) —
they are unexported today, which is correct for now.

**Notes:**
- **`cauldron/` build isolation — confirmed safe, no action needed in-repo.** The entire `cauldron/`
  tree is `.gitignore`d (verified via `git check-ignore`), including the stub `go.mod` files `advance`
  added at `cauldron/iscc-hub/` and `cauldron/tessera/`. So nothing reaches the repo or CI from this
  tree. The forward concern is real but belongs to whoever wires CI: a fresh `go build ./...` that
  includes a `cauldron/` checkout without those stubs will fail on the reference trees' external deps.
  CI should not check out `cauldron/`, or replicate the stubs / `go.work` exclude. Captured in
  learnings; not a gate failure this iteration.
- **Conformance scope:** the diff touches the trust root (verifier-key derivation), so the
  golden-vector oracle (`derive_vkey.py`) is the relevant gate and it passes. `notecheck`, `fsck`
  root-rebuild, and inclusion cross-checks do not exist yet (no proof/logclient-verify/fetcher code
  this step) and are correctly out of scope. No CI is configured yet (`.github/workflows/` absent), so
  there is no `notecheck` job to confirm green — flag for whoever sets up CI.
- **Push:** working branch is `develop`; remote `origin` is configured. Pushed on PASS.
- Minor defensive improvement worth keeping: `pubkeyFromDID` checks `len(raw) < 34` before the
  multicodec assert, avoiding the index panic the Python port would hit on a short key.
