# Next Work Package

## Step: Wire `LeafHashes` + `SQLiteFetcher` into `fsck.New(...).Check(...)` — the M2 root-rebuild conformance slice

## Goal
Land `logclient.RunFsck(...)`, the wiring that runs tessera's `fsck.New(origin, verifier, fetcher,
LeafHashes, opts).Check(ctx)` over a mirror fetcher, and prove it end-to-end: a conformance test
synthesizes a small tlog-tiles log in-process, seeds its checkpoint + entry bundles + hash tiles into
a real `store.SQLiteFetcher`, and asserts `RunFsck` rebuilds the signed root (and *fails* when a
mirrored BLOB is corrupted). This is the first slice to re-arm the trust-root oracle gate for the
**mirror path** and delivers the first half of M2's Verify bar ("fsck rebuilds each accepted root from
the `SQLiteFetcher`").

## Scope
- **Create**: `internal/logclient/fsck.go` — `RunFsck(ctx, vkey, origin string, f fsck.Fetcher) error`
  (the only non-test source file).
- **Modify**: `go.mod` + `go.sum` — run `go mod tidy` so `github.com/transparency-dev/tessera/fsck`
  and its require-graph enter the closure (verified by a throwaway probe: importing `tessera/fsck`
  forces a tidy; `tessera@v1.0.2` is already a direct require and `formats` is in the module cache).
- **Create**: `internal/logclient/fsck_test.go` — the conformance test, in external `package
  logclient_test` so it may import both `logclient` and `internal/store` without adding any production
  import edge.
- **Reference**:
  - `/home/dev/go/pkg/mod/github.com/transparency-dev/tessera@v1.0.2/fsck/fsck.go` — the real `fsck`
    package: `Fetcher` interface (3 methods, byte-identical to `SQLiteFetcher`), `New(origin string,
    verifier note.Verifier, f Fetcher, bundleHasher func([]byte) ([][]byte, error), opts Opts) *Fsck`,
    `(*Fsck).Check(ctx) error`, `Opts{N uint}`. `Check` fetches the checkpoint via the fetcher's
    `ReadCheckpoint`, re-hashes bundles with `bundleHasher`, re-derives tiles, and compares the rebuilt
    root to `cp.Hash` (line 158).
  - `cauldron/iscc-hub/conformance/runfsck/main.go` — the reference invocation (`fsck.New(*origin, v,
    src, leafHasher, fsck.Opts{N: 1})`, lines 64–68); `RunFsck` ports this minus the HTTP fetcher +
    flag/`os` glue, passing the existing `logclient.LeafHashes` as `bundleHasher`.
  - `internal/logclient/leafhasher.go` — `LeafHashes(bundle []byte) ([][]byte, error)`, already the
    exact `func([]byte) ([][]byte, error)` shape `fsck.New` wants.
  - `internal/store/fetcher.go` — `SQLiteFetcher{Store,HubID}` already satisfies `fsck.Fetcher`
    structurally (`ReadCheckpoint`/`ReadTile(ctx,l,i,p)`/`ReadEntryBundle(ctx,i,p)`); the test passes
    it straight in. Note the p↔width map: `p==0→256`, else `int(p)` (`widthForP`).
  - `internal/store/tiles.go` — `RecordTile`/`RecordEntryBundle` (+ `RecordCheckpoint` in
    `checkpoints.go`) are the test's seed-side writes (BLOB + width + observedAt).
  - `internal/logclient/proofbuilder_test.go` — the in-test ground-truth pattern to reuse: build a
    `testonly.New(rfc6962.DefaultHasher)` tree, derive hash-tile BLOBs via
    `api.HashTile{Nodes:…}.MarshalText()` and the `nodeHash` helper. This slice *additionally* needs
    entry-bundle BLOBs (`api.EntryBundle{Entries:…}.MarshalText()`) and a signed checkpoint.
  - `internal/logclient/verify.go` — mirror its `note.NewVerifier(vkey)` construction; the body framing
    `"<origin>\n<size>\n<base64(root)>\n"` is the exact text the test must sign.
  - `cauldron/iscc-hub/conformance/notecheck/main.go` — confirms `note.NewVerifier` / `note.Open` are
    the signed-note tooling; the test signs with `note.NewSigner`/`note.Sign` over a generated keypair.

## Not In Scope
- **The inclusion cross-check against the hub's own `evidence.IsccLogInclusionProof`** (the *second
  half* of M2's Verify bar). It needs real captured `IsccLogInclusionProof` fixtures and an inclusion
  `ProofBuilder` path — its own later slice. This step delivers only the root-rebuild half.
- **Live tile-ingestion writer** — making `PollHub` mirror real tiles/bundles from a hub is a separate
  M2 slice; this step seeds the store synthetically in-test, never over the network.
- **Calling `RunFsck` from the follower / a periodic fsck loop** — `RunFsck` lands as an
  unused-until-wired export seam (like the consistency triggers, `IsFull`, `LeafHashes`), `go
  vet`-clean. Do not add a production caller.
- **Capturing real on-disk `testdata/live/` tile fixtures** — not required once the log is synthesized
  in-process; defer real-hub capture to the live-ingestion slice.
- **CI / `notecheck` workflow** (the open `normal` issue) — its natural companion, but a separate step;
  define-next will pick it next so the mirror path faces the external oracle in CI.

## Implementation Notes
- **`RunFsck` is thin glue.** Signature: `func RunFsck(ctx context.Context, vkey, origin string, f
  fsck.Fetcher) error`. Body: `v, err := note.NewVerifier(vkey)` (wrap on error, mirroring
  `VerifyCheckpoint`), then `return fsck.New(origin, v, f, LeafHashes, fsck.Opts{N: 1}).Check(ctx)`
  wrapped with `%w`. Imports are exactly `context` + `fmt` + `golang.org/x/mod/sumdb/note` +
  `github.com/transparency-dev/tessera/fsck` (+ same-package `LeafHashes`). **Take the `fsck.Fetcher`
  interface, NOT the concrete `store.SQLiteFetcher`**, so production `logclient` gains no `store`
  import edge; the test supplies the concrete fetcher.
- **Correctness rule (learnings — entry-bundle leaf hasher + SQLiteFetcher).** `fsck` re-hashes each
  entry bundle with `LeafHashes`, re-derives the lower tiles, and compares the rebuilt root to the
  checkpoint's `cp.Hash`. Seed entry bundles whose `LeafHashes` output matches the tree's leaf hashes,
  and hash tiles whose bottom row equals `rfc6962.DefaultHasher`'s node hashes — else `Check`
  legitimately fails. Use ONE `testonly.Tree` as the single source of truth for both leaf and node
  hashes (the `nodeHash` helper from `proofbuilder_test.go`), so prover and verifier stay independent
  of the fetcher under test.
- **Entry-bundle BLOB framing.** `LeafHashes` decodes via `api.EntryBundle{}.UnmarshalText`, so encode
  the seed bundles with `api.EntryBundle{Entries: [][]byte{…}}.MarshalText()`. Each `e` in `Entries`
  is the raw leaf *preimage*; `fsck` runs `rfc6962.DefaultHasher.HashLeaf(e)`. Build the
  `testonly.Tree` from those SAME preimages (`tree.AppendData(e)`), so the tree's leaf hashes equal
  `HashLeaf(e)` and the rebuilt root matches `tree.Hash()`.
- **Width discipline (learnings — tlog-tiles layout + SQLiteFetcher).** Start with a small
  within-one-tile log (e.g. 3–10 leaves): one partial entry bundle at index 0 and one partial level-0
  hash tile at index 0, both `width == n`. Seed with `RecordEntryBundle(…, width=n, …)` and
  `RecordTile(…, level=0, index=0, width=n, …)`. The `SQLiteFetcher` p↔width map means `fsck`
  requesting partial `p=n` resolves to `width=n`. A within-one-tile log avoids the 256-leaf boundary
  for the first green; an optional second case crossing 256 (full tile 0 + partial tile 1, the
  `proofbuilder_test.go` shape) strengthens it but is not required for the Done bar.
- **Signed checkpoint.** Generate an Ed25519 keypair + the C2SP vkey in-test (`note.GenerateKey(rand,
  name)` returns `(skey, vkey)`; `name` = `origin`). Sign the body `"<origin>\n<n>\n<base64(tree.Hash())>\n"`
  with `note.Sign(&note.Note{Text: body}, signer)`; seed the signed bytes via `RecordCheckpoint`.
  `RunFsck` verifies with that same `vkey`. `origin` is any stable name (e.g. `"sb0.iscc.id/log"`); it
  need not be a live hub since the key is synthetic. **Do not reuse the real testnet checkpoint
  fixtures** — their root commits a ~10183-leaf tree whose tiles are not mirrored here.
- **Oracle gate APPLIES (RFC-6962 root-rebuild crypto) — make the test non-vacuous (learnings: "a
  green-but-wrong verify must not ship").** Assert BOTH, as two subtests: (1) `RunFsck` returns `nil`
  over the correctly-seeded store; (2) `RunFsck` returns a non-nil error after corrupting one mirrored
  tile OR entry-bundle BLOB (flip a byte, re-`RecordTile`/`RecordEntryBundle` at the same key),
  proving the rebuild genuinely compares against the checkpoint root rather than trivially passing.
  This mutation is baked into the committed test, not a throwaway. Document in the test that `fsck` is
  an in-process structural **self-check** (it shares the monitor's code); the fully-independent oracle
  (`notecheck`) is the deferred CI companion.
- **go.mod cost (learnings — tessera/merkle dep hygiene).** `go mod tidy` will add `tessera/fsck`'s
  require-graph (`k8s.io/klog/v2`, `go.opentelemetry.io/otel*`, `golang.org/x/sync`,
  `transparency-dev/formats`) to the `// indirect` block + `go.sum`. Keep the `go` directive at `go
  1.24.0` with **no** `toolchain` line (drop any auto-injected one). Confirm `go mod tidy && git diff
  --exit-code -- go.mod go.sum` is idempotent after the commit.
- **WASM nuance (learnings).** `logclient` already pulls `net/http` via `didresolve.go`, so file-level
  WASM purity is the invariant for the OTHER files, not `fsck.go` — `fsck` is a server-side mirror
  check, never WASM. Do not attempt to keep `fsck.go` WASM-clean.

## Verification
- `mise run check` is green (`go build ./...` + `go vet ./...` + `go test ./...`, all packages `ok`).
- `gofmt -l .` is empty.
- `go test -count=1 -run TestRunFsck ./internal/logclient` passes uncached, with at least two
  subtests: one asserting `RunFsck` returns `nil` over the correctly-seeded `SQLiteFetcher`, one
  asserting it returns a non-nil error after a one-byte corruption of a mirrored tile/bundle BLOB.
- `go mod tidy && git diff --exit-code -- go.mod go.sum` exits 0 (tidy is idempotent after the commit).
- `grep -c '^go 1.24.0$' go.mod` is 1 and `grep -c '^toolchain' go.mod` is 0 (no toolchain pin).
- The trust-root goldens still reproduce: `go test -count=1 -run 'VerifierKey|Origin'
  ./internal/didweb/ ./internal/logclient/` passes (verifier-key + origin goldens unchanged).

## Done When
`mise run check` is green and `go test -run TestRunFsck ./internal/logclient` passes with both the
correctly-seeded `nil` case and the corrupted-BLOB non-nil case, `go mod tidy` is idempotent, and the
existing trust-root goldens are unchanged — proving `fsck.New(...).Check(...)` rebuilds the signed
root from the local `SQLiteFetcher` and rejects a tampered mirror.
