# Handoff

## 2026-06-20 — Bootstrap Go module + golden-tested `origin()` and `verifierKey()`

**Done:** Bootstrapped the Go module (`github.com/iscc/iscc-monitor`, `go 1.24`) and landed the two
highest-leverage pure M1 units: `origin()` (hub base URL → `<domain>/log`) in `internal/logclient`
and the did:key → C2SP verifier-key derivation (`pubkeyFromDID`/`keyID`/`verifierKey`, a faithful
port of `.claude/derive_vkey.py`) in `internal/didweb`. Both are golden-tested byte-for-byte against
the Python oracle for the two live testnet hubs. `mise run check` is now green.

**Files changed:**
- `go.mod`: created via `go mod init`; module `github.com/iscc/iscc-monitor`, directive normalized to
  `go 1.24` (matches the plan/CLAUDE.md, not the `1.24.13` toolchain patch).
- `internal/logclient/origin.go`: pure `origin(baseURL) (string, error)` using `net/url` + `strings`;
  defaults a scheme when absent so a bare domain still parses into Host, returns `<host>/log`, errors
  on empty/host-less input.
- `internal/didweb/vkey.go`: pure `b58decode` (leading-zero preserving), `pubkeyFromDID` (strip `z`,
  assert `0xED 0x01` multicodec, return `[2:34]`), `keyID` (BE uint32 of
  `SHA-256(name||0x0A||0x01||pub)[:4]`), `verifierKey` (`<name>+<keyid:08x>+<StdEncoding base64(0x01||pub)>`).
- `internal/didweb/vkey_test.go`: table-driven golden test asserting both literal verifier keys, plus
  error cases for `pubkeyFromDID`.
- `internal/logclient/origin_test.go`: table-driven test (scheme/no-scheme/trailing-slash/port/
  whitespace) plus error cases.

**Verification:** `mise run fmt` then `mise run check` → green (`go build ./...`, `go vet ./...`,
`go test ./...` all pass). `gofmt -l .` prints nothing.
- `go mod init` succeeded; `go.mod` declares the module → PASS.
- `mise run check` green → PASS.
- `gofmt -l` empty → PASS.
- `go test -run TestVerifierKey ./internal/didweb` asserts both golden literals → PASS.
- `go test -run TestOrigin ./internal/logclient` asserts both hub origins → PASS.
- Oracle cross-check: `python3 .claude/derive_vkey.py` prints
  `sb0.iscc.id/log+40b74463+AaV+ivnly67hhzQSQfGqCBP3PlOV2NBcmfGyzGdE2ZE5` and
  `sb1.amlet.id/log+22b08f3e+ATo2ruguSdJGh11PS76osrQf6OZKrufzzwH/HMwE3a8/` — byte-identical to the
  Go `verifierKey` output asserted in the test → PASS.

**Next:** did:web resolution (`internal/didweb/resolve.go`): fetch `/.well-known/did.json`, parse
`verificationMethod` to the `z6Mk` did:key, honor CID 1.0 validity windows / `revoked`, returning the
pubkey that feeds `verifierKey`. Inject the HTTP client at the outbound-fetch seam (`*http.Client` /
`client.Fetcher`) and test against a captured `testdata/live/.../did.json` fixture. `origin()` and
`verifierKey` are currently unexported (only intra-package callers/tests); export or wire them when
`resolve.go` / the follower needs them across package boundaries.

**Notes:**
- **`cauldron/` build isolation (decision, please confirm):** `go build ./...` from the repo root
  descended into `cauldron/` and failed on the reference trees' missing external deps
  (`transparency-dev/*`, `x/mod/sumdb/note`, etc.). I added a stub `go.mod` at `cauldron/iscc-hub/`
  and `cauldron/tessera/` so Go treats them as separate modules and skips them. Both stubs are
  **gitignored** (`git check-ignore` confirms) so they are not committed and don't touch the repo —
  they only keep the local build clean. This is the standard Go way to exclude a vendored subtree and
  weakens no gate (our packages still build/vet/test). If CI runs `go build ./...` against a fresh
  `cauldron/` checkout without these stubs, it will hit the same failure — flagging for `review` in
  case CI needs the same isolation (or a `go.work`/build-tag approach) there.
- Kept `go.mod` dependency-free as scoped: inline base58btc decode (`math/big`) and SHA-256 keyid, no
  `mr-tron/base58` / `sumdb/note` yet.
- `internal/proof/verify` purity rule is not yet exercised (no proof package this step), but both new
  packages are pure (stdlib only) and import-clean.
- `.claude/.scratch/*.pub` are byproducts of running the oracle; left untracked, not committed.
