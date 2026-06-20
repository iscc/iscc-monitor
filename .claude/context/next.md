# Next Work Package

## Step: Bootstrap Go module + golden-tested `origin()` and `verifierKey()`

## Goal
Turn the repo from "specs only" into a buildable Go module and land M1's highest-leverage pure units:
`origin()` (the `<domain>/log` single source of truth, the highest-probability bug per learnings) and
`verifierKey()` / `keyID()` (did:key → C2SP signed-note verifier-key string), golden-tested byte-for-byte
against `.claude/derive_vkey.py` for both live testnet hubs. This flips `mise run check` from red to green
and grounds every later crypto/follower unit on a verified key derivation.

## Scope
- **Create**:
  - `go.mod` (via `go mod init github.com/iscc/iscc-monitor`; Go 1.24)
  - `internal/logclient/origin.go` — pure `origin(baseURL string) (string, error)` returning `<domain>/log`
  - `internal/didweb/vkey.go` — pure `pubkeyFromDID`, `keyID`, `verifierKey` (port of `derive_vkey.py`)
  - `internal/didweb/vkey_test.go` — table-driven golden test (both hubs) — test file, not counted
  - `internal/logclient/origin_test.go` — table-driven test for `origin()` — test file, not counted
- **Modify**: none (no existing Go code; `mise.toml` already correct, do not touch)
- **Reference** (read, do not import — all gitignored / Python):
  - `/workspace/iscc-monitor/.claude/derive_vkey.py` — the exact derivation to port and the golden vectors
  - `/workspace/iscc-monitor/.claude/plans/cosmic-baking-octopus.md` lines 100-133, 151-152, 191
  - `/workspace/iscc-monitor/cauldron/iscc-hub/iscc_hub/checkpoint_note.py` — confirm keyid/wire format matches

## Not In Scope
- did:web **resolution** (`internal/didweb/resolve.go`): fetching `/.well-known/did.json` over HTTP, key
  validity windows, revocation. This step only does the pure pubkey→vkey math on a given did:key string.
- The `follower.go` / `verify.go` consistency logic, `tessera/client` wrapping, any SQLite, config, or registry.
- `cmd/iscc-monitor/main.go` — no entrypoint yet; the module compiles via `internal/` packages alone.
- Pulling in any external module dependency (`sumdb/note`, `mr-tron/base58`, etc.). Implement base58btc decode
  and the SHA-256 keyid inline — keep `go.mod` dependency-free this step. (Swap to a vetted dep later if needed.)

## Implementation Notes
- **Port faithfully from `derive_vkey.py`** — the format is exact and load-bearing:
  - `verifierKey = "<name>+<keyid:08x>+<base64(0x01 || pub32)>"` where `name` is the origin.
  - `keyID = big-endian uint32 of SHA-256(name || 0x0A || 0x01 || pub)[:4]` (note the literal `0x0A` newline
    byte and the `0x01` Ed25519 alg byte both appear in the hash preimage AND in the base64 body).
  - `pubkeyFromDID`: strip the leading `z` (multibase btc), base58-decode, assert the first two bytes are
    `0xED 0x01` (ed25519-pub multicodec), return bytes `[2:34]`. Preserve leading-zero bytes in base58 decode
    (count leading `1`s → that many `0x00` prefix bytes), matching the Python `b58decode`.
  - Use base64 **standard** encoding with padding (`base64.StdEncoding`), matching Python `b64encode`.
- **`origin()` is the single source of truth** (learnings Correctness rule #1, plan line 151): accept a hub
  base URL (e.g. `https://sb0.iscc.id`), parse with `net/url`, return `host + "/log"` (e.g. `sb0.iscc.id/log`).
  Strip any scheme/trailing slash; never return the bare domain. Return an error on an unparseable/empty host.
  Keep it pure (only `net/url` + `strings`) — `didweb/vkey.go` will eventually call it, but for THIS step the
  golden test passes the literal origin string straight into `verifierKey` to match the Python vectors exactly.
- Functional, pure, short functions per house style; package + file docstring comment at top of each file.
- **Correctness rule (learnings):** Origin = `<domain>/log`, never the bare domain — golden-test both hubs.
- Expected golden outputs (from running the oracle just now — assert these literals in the test):
  - `sb0.iscc.id/log` → `sb0.iscc.id/log+40b74463+AaV+ivnly67hhzQSQfGqCBP3PlOV2NBcmfGyzGdE2ZE5`
  - `sb1.amlet.id/log` → `sb1.amlet.id/log+22b08f3e+ATo2ruguSdJGh11PS76osrQf6OZKrufzzwH/HMwE3a8/`
  - DIDs: sb0 = `z6MkqbHELZopsq6eKrn6qxiAgRoVvwp2Vp7mPfKsvrbYmGwJ`,
    sb1 = `z6MkiNW46AUjNmKTV2YNyFi9ANG9wbfYQQoUQgADGwScd9jk`
- Do NOT use `t.Skip`, `//nolint`, build tags, or swallow errors to pass the gate (target.md quality bar).

## Verification
- `go mod init` succeeds and `/workspace/iscc-monitor/go.mod` declares `module github.com/iscc/iscc-monitor`.
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all exit 0).
- `gofmt -l /workspace/iscc-monitor` prints nothing.
- `go test -run TestVerifierKey ./internal/didweb` passes and asserts both literal golden vectors above.
- `go test -run TestOrigin ./internal/logclient` passes and asserts
  `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and `origin("https://sb1.amlet.id") == "sb1.amlet.id/log"`.
- Cross-check: `python3 /workspace/iscc-monitor/.claude/derive_vkey.py` output for both hubs equals the Go
  `verifierKey` output for the same `(origin, did)` pairs.

## Done When
`mise run check` is green, `gofmt -l` is empty, and both golden tests pass with the Go `verifierKey`/`origin`
output byte-matching `derive_vkey.py` for `sb0.iscc.id/log` and `sb1.amlet.id/log`.
