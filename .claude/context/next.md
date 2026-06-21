# Next Work Package

## Step: Wire RunFsck into PollHub over the live SQLiteFetcher mirror

## Goal
Give `logclient.RunFsck` its first production caller: after `PollHub` mirrors a verified hub's
tiles/bundles via `ingestTiles`, run `fsck.New(...).Check(ctx)` over the local `SQLiteFetcher` to
rebuild the accepted root from the mirror and cross-check it against the signed checkpoint root.
This re-arms the RFC-6962 root-rebuild oracle gate on the live path and lands the FIRST half of M2's
Verify bar ("`fsck` rebuilds each accepted root from the `SQLiteFetcher`").

## Scope
- **Create**: `internal/follower/fsck_test.go` (a `package follower` test that seeds a
  `testonly.Tree`-backed consistent mirror and proves `PollHub`/the helper fscks it)
- **Modify**: `internal/follower/follower.go` (add an `fsckMirror` helper and call it on the
  verified, non-violation path of `PollHub`, after `ingestTiles`)
- **Reference**:
  - `/workspace/iscc-monitor/internal/logclient/fsck.go` — `RunFsck(ctx, vkey, origin string, f
    fsck.Fetcher) error`, the seam to call (already built + conformance-tested)
  - `/workspace/iscc-monitor/internal/logclient/fsck_test.go` — the existing `testonly.Tree` →
    consistent-mirror seeding pattern (`seedMirror`, `encodeBundle`, the level-0 `api.HashTile` fill,
    the in-test `note.GenerateKey` signed checkpoint, and the `RejectsCorruptedTile`/`...Bundle`
    negative cases) to port the new test's fixture from
  - `/workspace/iscc-monitor/internal/follower/equivocation_test.go` — `buildEquivTree` /
    `level0TileBytes` / `seedMirrorTiles` show how the follower package builds a
    `testonly.Tree`-consistent local mirror over `RecordTile`
  - `/workspace/iscc-monitor/internal/store/fetcher.go` — `SQLiteFetcher{Store, HubID}` (the
    `fsck.Fetcher` the call passes; `ReadCheckpoint`/`ReadTile`/`ReadEntryBundle`)
  - `/workspace/iscc-monitor/internal/logclient/didresolve.go` line 93 —
    `ResolveVerifierKey(ctx, fetcher, baseURL) (vkey string, didweb.DIDKey, error)`, the vkey source
  - `/workspace/iscc-monitor/internal/logclient/origin.go` line 48 — `Origin(baseURL) (string, error)`,
    the `<domain>/log` origin `RunFsck` needs
  - `/workspace/iscc-monitor/cauldron/iscc-hub/conformance/runfsck/main.go` — the original `runfsck`
    reference `RunFsck`/`LeafHashes` were ported from (read-only oracle context, never import)

## Not In Scope
- The **inclusion cross-check** vs the hub's own `evidence.IsccLogInclusionProof` (the SECOND half of
  M2's Verify) — the sibling slice; it needs captured `IsccLogInclusionProof` fixtures.
- Capturing **byte-accurate live sb0/sb1 tile/entry-bundle fixtures** over the network into
  `testdata/live/`. The follower-level test proves the wiring with an in-process `testonly.Tree`
  mirror (byte-accurate to its own signed root — the same standard `fsck_test.go` uses and the review
  already accepted), so no live capture is required for this step. Defer live-fixture capture if ever
  needed.
- `iscc_index` projection, serving `inclusion`/`consistency`/`entries` from the store, OTS, the M3
  REST surface — all later milestones.
- Touching `internal/store` (keep it a leaf; the follower owns the `RunFsck` wiring and the vkey).
- The `cmd/notecheck` `low` issue (loop-skipped).

## Implementation Notes
- **Where (placement).** In `PollHub`, on the verified, non-violation path, add the fsck step
  **after** `ingestTiles` succeeds and **before** the final
  `recordVerdict(m, hubID, status, false, observedAt)` (the mirror must be populated first). Keep it a
  small helper, e.g. `fsckMirror(ctx, st, fetcher, hubID, baseURL)`, called as
  `if err := fsckMirror(...); err != nil { return status, fmt.Errorf("follower.PollHub: hub %d: %w", hubID, err) }`.
  Do NOT run it on the freeze path nor on non-verified verdicts.
- **Getting vkey + origin.** `RunFsck` needs `(vkey, origin, fsck.Fetcher)`. Resolve the vkey via
  `logclient.ResolveVerifierKey(ctx, fetcher, baseURL)` (returns `(vkey string, didweb.DIDKey,
  error)`; take the vkey, ignore the key — validity was already checked by `AcceptCheckpoint`) and the
  origin via `logclient.Origin(baseURL)`. The fetcher is `store.SQLiteFetcher{Store: st, HubID:
  hubID}` — identical to the equivocation branch's construction in `checkConsistency`. Wrap each error
  with `%w`. (`info.Origin` is also available if you thread `info` in, but `Origin(baseURL)` is the
  already-used, self-contained call.)
- **Error-vs-violation discipline (ADR-0006 — the load-bearing subtlety; learnings "Error vs.
  violation discipline" + "fsck root-rebuild wiring").** A `RunFsck` failure here is a *root-rebuild
  mismatch or a mirror fault*, NOT a self-consistency violation — it must **not** freeze the hub
  (freezing is reserved for the three triggers in `checkConsistency`). Treat a non-nil `RunFsck` error
  as a genuine fault returned up to the caller (the established `PollHub` pattern: surface it; the
  checkpoint is already recorded/advanced above, so a transient fault is re-attempted next poll). Do
  **not** swallow it into a clean verdict and do **not** call `freeze`. Document this in the helper
  docstring and extend the `PollHub` package comment, mirroring the existing `ingestTiles` comment
  block.
- **Honesty (learnings "fsck root-rebuild wiring").** `RunFsck` is an *in-process structural
  self-check* — it shares the monitor's own `LeafHashes`/RFC-6962 code, so it is NOT the
  fully-independent oracle (`notecheck` is). State this plainly in the docstring; do not over-claim
  external independence.
- **Test fixture (port `fsck_test.go`).** Build ONE internally-consistent `testonly.Tree` and serve
  it through a follower-package fetcher; do NOT reuse `ingest_test.go`'s `mirrorFetcher` — it serves
  *inconsistent* synthetic per-URL bytes, so the rebuild would always fail. Use a within-one-tile size
  (e.g. `fsckLeaves = 5`, `< 256`) for the first green so there is exactly one partial level-0 tile +
  one partial entry bundle. Two viable drive paths, prefer (a):
  - **(a) end-to-end `PollHub`** — generate a keypair with `note.GenerateKey(rand.Reader, origin)`,
    serve a `did:web` document advertising that key's multibase (model it on the committed
    `internal/logclient/testdata/*_did.json` shape), and sign the checkpoint body
    `"<origin>\n<size>\n<base64(root)>\n"` with it; serve the level-0 `api.HashTile` + framed entry
    bundle at the tile/bundle URLs. `PollHub` then verifies, mirrors via `ingestTiles`, and fscks —
    one assertion that `PollHub` returns `(StatusVerified, nil)`.
  - **(b) direct helper** (fallback if reconstructing a `did.json` is too large for this step) — seed
    a consistent mirror + signed checkpoint directly into the store exactly as `fsck_test.go`'s
    `seedMirror` does, then call `fsckMirror(...)` and assert nil. Note `fsckMirror` resolves the vkey
    via `ResolveVerifierKey`, which needs a `did.json` from the fetcher — so path (b) may need a small
    fetcher serving the generated key's `did.json` too. Either is acceptable.
- **Mutation / non-vacuity (REQUIRED).** Include a negative subtest: corrupt one mirrored tile or
  bundle BLOB after ingest (re-`RecordTile`/`RecordEntryBundle` with a flipped byte at the same key,
  as `fsck_test.go`'s `RejectsCorruptedTile`/`RejectsCorruptedBundle` do) and assert the fsck step now
  returns a non-nil error. A green-but-wrong `RunFsck` that ignores the root must fail this case.
- **Imports.** Production follower imports must stay `{context, fmt, internal/logclient,
  internal/metrics, internal/store, internal/tiles, log/slog}` (no new prod import — `tessera/fsck` is
  reached transitively through `logclient.RunFsck`, not imported by the follower). `merkle/testonly` +
  `tessera/api` + `golang.org/x/mod/sumdb/note` are **test-only**. Store stays a leaf (no reverse dep).
  `go.mod`/`go.sum` stay byte-identical (`tessera/fsck` already in the closure via
  `internal/logclient/fsck.go`).
- **Oracle gate APPLIES** for this slice (RFC-6962 root-rebuild crypto on the live path) and is
  satisfied by the in-test `testonly.Tree` ground truth (prover) vs `RunFsck`/`LeafHashes` (verifier)
  being independent code paths, plus the mutation negative case. `notecheck`/`derive_vkey.py` stay the
  external oracles and are unchanged by this wiring.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -run TestPollHubFsck -count=1 ./internal/follower` passes — the good-mirror case returns
  nil and the corrupted-mirror case returns non-nil (both asserted in the one test).
- `go test -run 'TestPollHub|TestIngest|TestWidthForP|TestEquivocation' -count=1 ./internal/follower`
  still passes (existing verified/fork/shrink/equivocation/ingest paths unaffected).
- `git diff --quiet HEAD -- go.mod go.sum` exits 0 (no dependency change).
- `GOOS=js GOARCH=wasm go build ./internal/didweb` exits 0 (trust-root WASM leaf unaffected).
- Production follower import set unchanged: `go list -f '{{join .Imports "\n"}}' ./internal/follower`
  lists no package outside `{context, fmt, internal/logclient, internal/metrics, internal/store,
  internal/tiles, log/slog}` plus stdlib.
- Mutation check (run manually, then revert): forcing the new `fsckMirror` helper to `return nil`
  makes the corrupted-mirror subtest of `TestPollHubFsck` FAIL — proving the rebuild genuinely
  compares against the signed root.

## Done When
`PollHub` runs `RunFsck` over the live `SQLiteFetcher` mirror on every verified, non-violation poll,
a tampered mirror BLOB surfaces a non-nil fault (without freezing the hub), and all Verification
criteria pass with `mise run check` green.
