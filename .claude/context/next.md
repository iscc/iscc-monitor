# Next Work Package

## Step: Verify Ed25519 signed-note checkpoints (reuse `sumdb/note`); map a non-matching signature to `unverified`

## Goal
Give the monitor its first real checkpoint-acceptance primitive: a pure function that takes a
hub-signed C2SP checkpoint plus the verifier key `ResolveVerifierKey` already returns, verifies the
Ed25519 signature via `golang.org/x/mod/sumdb/note`, and yields `(origin, treeSize, root)` on success
— mapping any signature that does not match the resolved key to status `unverified` (distinct from the
`unresolvable` resolution failure already handled in `didresolve.go`). This is the last load-bearing
crypto unit before the follower/store can begin, and it adds the project's first reused dependency.

## Scope
- **Create**: `/workspace/iscc-monitor/internal/logclient/verify.go` — the signed-note checkpoint
  verifier (per the plan's `logclient/{origin.go,follower.go,verify.go}` layout). 1 non-test source.
- **Create**: `/workspace/iscc-monitor/internal/logclient/verify_test.go` — table-driven golden test
  (test file, not counted toward the 3-file limit).
- **Create**: `/workspace/iscc-monitor/testdata/live/sb0.iscc.id_checkpoint` and
  `/workspace/iscc-monitor/testdata/live/sb1.amlet.id_checkpoint` — real checkpoints captured once from
  the live hubs (golden fixtures, not counted as source).
- **Modify**: `/workspace/iscc-monitor/go.mod` — add `require golang.org/x/mod v0.33.0` (version
  constraint in Implementation Notes). `go.sum` is created by `go mod tidy`; that is expected output,
  not a separate scope item.
- **Reference** (read, do not import):
  - `/workspace/iscc-monitor/cauldron/iscc-hub/iscc_hub/checkpoint_note.py` — the exact wire format and
    key-id derivation to mirror; in particular `parse_checkpoint`, `verify_checkpoint`, and the
    body/signature layout.
  - `/workspace/iscc-monitor/cauldron/iscc-hub/conformance/notecheck/main.go` — the Go reference
    oracle: shows the `note.Open(body, note.VerifierList(v))` + `len(n.Sigs)`/`len(n.UnverifiedSigs)`
    pattern to follow.
  - `/workspace/iscc-monitor/internal/logclient/didresolve.go` — `ResolveVerifierKey` returns the
    `vkey` string this consumes; match its sentinel-error style (`ErrUnverified` parallel to
    `ErrUnresolvable`).
  - `/workspace/iscc-monitor/internal/didweb/vkey.go` — confirms the vkey string format
    `note.NewVerifier` parses.

## Not In Scope
- The follower loop, polling, or `tessera/client` wiring — no fetch of `/log/checkpoint` in production
  code. This step verifies bytes handed to it; the live fixtures are captured by hand (a one-time
  `curl`), not fetched at test time.
- The three-trigger fork/shrink/equivocation **consistency** check and RFC-6962 / Merkle math — a
  later step (needs two successive checkpoints + `transparency-dev/merkle`).
- Any SQLite store, `hub_keys` cache, or persistence.
- **CID 1.0 validity-window enforcement** (`DIDKey.ValidFrom`/`ValidUntil`/`Revoked`): leave them
  parsed-but-unenforced as today; enforcement belongs to the store/follower step.
- The pure `internal/proof/verify` package (Merkle inclusion/consistency, WASM) — distinct from this
  signed-note `logclient/verify.go`; do not create it here.
- Wiring `transparency-dev/formats/note` — not needed for the Ed25519 case (stdlib `sumdb/note`
  handles alg `0x01` natively, verified during scoping).

## Implementation Notes
- **Reuse, do not reimplement (ADR-0003 / target Stack).** Verification goes through
  `golang.org/x/mod/sumdb/note`, not a hand-rolled Ed25519 path. The flow, mirroring
  `notecheck/main.go`:
  ```
  v, err := note.NewVerifier(vkey)        // vkey is exactly what ResolveVerifierKey returns
  n, err := note.Open(body, note.VerifierList(v))
  // success iff err == nil && len(n.Sigs) >= 1
  ```
  Then parse the verified body's three lines (`<origin>\n<tree_size>\n<base64(root)>\n`) into
  `(origin, treeSize, root)`. Port the body-parsing rules from `checkpoint_note.py:parse_checkpoint`:
  decimal `tree_size` with **no leading zeros** (reject `"01"`, accept `"0"`), non-negative; root is
  **std** base64 and **exactly 32 bytes**. `note.Open` already enforces the signed-note framing
  (trailing-`\n` body, blank separator, `— <name> <base64>` sig line), so do not re-parse the
  signature line yourself — let `note.Open` own it.
- **Status mapping (learnings: "A signature matching no listed key → `unverified`").** Two distinct
  failure shapes from `note.Open` BOTH mean the checkpoint was not signed by the hub's resolved key →
  return a sentinel `ErrUnverified`:
  1. body present but the listed verifier's name+keyhash matched a sig line yet the signature is
     invalid → `n.UnverifiedSigs` populated, `n.Sigs` empty;
  2. no sig line matches the verifier's name/keyhash at all → `note.Open` returns the error
     `"note has no verifiable signatures"` (verified during scoping with the sb1 vkey on the sb0
     checkpoint).
  Suggested signature:
  ```go
  // ErrUnverified marks a checkpoint whose signature does not match the hub's
  // resolved did:web key (an internally-broken hub). Distinct from ErrUnresolvable.
  var ErrUnverified = errors.New("checkpoint signature unverified")

  // VerifyCheckpoint verifies raw against vkey and returns the signed (origin, treeSize, root).
  func VerifyCheckpoint(vkey string, raw []byte) (origin string, treeSize uint64, root [32]byte, err error)
  ```
  Keep `VerifyCheckpoint` **pure** — it imports only `sumdb/note` + stdlib (`encoding/base64`,
  `strconv`, `strings`, `errors`, `fmt`); NO `net`/`os`/`database/sql`. `sumdb/note` is pure Go (it
  ships in the std `cmd/vendor` tree), so this function stays WASM-shareable. (`net/http` already lives
  in this package via `didresolve.go`, so the package as a whole is not WASM-pure — that is fine; the
  follower seam, not WASM, consumes `VerifyCheckpoint`. The pure WASM verify path is the later
  `internal/proof/verify`.)
- **`go.mod` version constraint (verified during scoping).** `golang.org/x/mod@v0.35.0`+ require
  `go >= 1.25.0`, which breaks the pinned `go 1.24` directive. `golang.org/x/mod v0.33.0` works under
  go 1.24 and verifies the live sb0 checkpoint (`note.Open` → `verified=1 unverified=0`). Pin
  `v0.33.0`. Run `go mod tidy` to populate `go.sum`, then `mise run fmt`. Do **not** bump the module's
  `go 1.24` directive.
- **Capturing the live fixtures.** Fetchable now:
  `curl -fsS https://sb0.iscc.id/log/checkpoint` and `https://sb1.amlet.id/log/checkpoint`. Save the
  raw bytes verbatim (preserve the trailing newline; do not reformat) to the `testdata/live/` files.
  The sb0 fixture verifies against
  `sb0.iscc.id/log+40b74463+AaV+ivnly67hhzQSQfGqCBP3PlOV2NBcmfGyzGdE2ZE5` and sb1 against
  `sb1.amlet.id/log+22b08f3e+ATo2ruguSdJGh11PS76osrQf6OZKrufzzwH/HMwE3a8/` (the existing golden vkeys).
  Because the live tree grows, assert the parsed `origin` and that `treeSize >= 1` and `root` is 32
  non-zero bytes — do NOT hard-code the live `treeSize`/`root` (they drift). For an exact-value golden,
  optionally add a synthetic inline checkpoint constant (fixed origin+size+root+sig) only if you can
  produce one deterministically; otherwise the live fixture plus the negative/tamper cases below
  suffice.
- **Negative cases the test must cover** (assert `errors.Is(err, ErrUnverified)`): (a) the sb0
  checkpoint verified against the **sb1** vkey (name/keyhash mismatch → `"no verifiable signatures"`);
  (b) a tampered sb0 checkpoint (flip one base64 char in the root line) against the correct vkey (sig
  invalid → `UnverifiedSigs`). (c) A body with a leading-zero `tree_size` or a non-32-byte root must
  return a **parse** error — it must NOT be `ErrUnverified` when the signature itself was valid; keep
  the two concerns separable (a distinct sentinel or a plain wrapped error is fine).
- **Correctness rule (learnings):** origin is `<domain>/log`; the verified signed-note `name`
  (`n.Sigs[0].Name`) must equal the checkpoint's first body line — assert they match.
- Short, pure functions with evergreen docstrings; file starts with a one-line purpose docstring. Do
  NOT use `t.Skip`, `//nolint`, build tags, or swallow errors to pass the gate (target quality bar;
  learnings "Never weaken a gate").

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all exit 0).
- `gofmt -l /workspace/iscc-monitor` prints nothing.
- `go test -run TestVerifyCheckpoint ./internal/logclient` passes (all subtests).
- Positive: `VerifyCheckpoint("sb0.iscc.id/log+40b74463+AaV+ivnly67hhzQSQfGqCBP3PlOV2NBcmfGyzGdE2ZE5",
  <testdata/live/sb0.iscc.id_checkpoint bytes>)` returns `origin == "sb0.iscc.id/log"`, `treeSize >= 1`,
  a 32-byte non-zero `root`, and `err == nil`; same shape for sb1 with its vkey.
- Negative: the sb0 checkpoint verified against the sb1 vkey, and a one-char-tampered sb0 checkpoint
  against the sb0 vkey, each return `errors.Is(err, ErrUnverified) == true`.
- External oracle parity (re-run): `python3 /workspace/iscc-monitor/.claude/derive_vkey.py` prints both
  golden vkeys byte-exact (`sb0…+40b74463…`, `sb1…+22b08f3e…`) matching the vkeys the test uses;
  `rm -rf /workspace/iscc-monitor/.claude/.scratch` afterward so it does not dirty the tree.
- `GOOS=js GOARCH=wasm go build ./internal/didweb` still succeeds (didweb untouched, purity intact).

## Done When
`VerifyCheckpoint` verifies the live sb0/sb1 checkpoint fixtures against their derived vkeys, maps a
non-matching signature to `ErrUnverified`, and all Verification criteria pass with `mise run check`
green.
