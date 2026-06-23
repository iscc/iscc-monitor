<!-- area: internal/logclient (verify.go, consistency.go, proofbuilder.go, inclusioncheck.go, leafhasher.go, fsck.go, projection.go, checkconsistency.go, didresolve/checkpoint/tilefetch/accept transport) -->
<!-- indexed-as: logclient.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# `internal/logclient` — verify, triggers, proofs, transport seam

Read this when a step touches the area above. Durable cross-cutting rules live in
the index (`.claude/context/learnings.md`); the package-local mechanics are here.

## Transport, did:web resolver & AcceptCheckpoint seam (from Go / tooling conventions)

- **Networked did:web resolver lives in `internal/logclient` (`didresolve.go`), never `didweb`.** It
  imports `net/http`, so putting it in `didweb` would break the WASM build. `ResolveVerifierKey(ctx,
  Fetcher, baseURL)` wires `origin → did:web:<host%3Aport> → DocumentURL → Fetch → ParseDIDDocument →
  VerifierKey`; the `Fetcher` 1-method seam keeps tests offline (fake + `httptest.NewTLSServer`). Host
  is recovered as `TrimSuffix(origin(baseURL), "/log")` — reuse, not a second host parser. The
  verifier key embeds the *fixture's* origin (`sb0.iscc.id/log`), not the fetch host, so the
  `httptest` golden asserts only the `<addr>/log` prefix, not the full key.
- **`FetchCheckpoint(ctx, Fetcher, baseURL)` is the transport-only checkpoint fetch** (`logclient/
  checkpoint.go`, imports only `context`+`fmt`): `origin(baseURL)` → `"https://"+name+"/checkpoint"`
  by concatenation (NOT a second `net/url` parse), returns bytes verbatim, wraps both `origin()` and
  Fetcher errors with `%w` so a 404's `errors.Is(err, os.ErrNotExist)` survives. It does NOT wrap in
  `ErrUnresolvable` — that sentinel is did:web-only; a checkpoint-fetch fault is a plain transport
  error the follower classifies separately.
- **`FetchTile`/`FetchEntryBundle` (`logclient/tilefetch.go`) are the tile/bundle transport siblings of
  `FetchCheckpoint` — same shape, imports only `context`+`fmt`+`internal/tiles`.** The one structural
  difference from `FetchCheckpoint`: `TilePath`/`EntriesPath` return *slash-less* paths (`tile/1/000`),
  so the URL is `"https://"+name+"/"+tiles.TilePath(...)` (leading slash on the literal), whereas
  `FetchCheckpoint` appends `"/checkpoint"`. Paths come from `tiles.TilePath`/`tiles.EntriesPath` (never
  hand-built), both `origin()` and Fetch errors `%w`-wrapped so a 404's `os.ErrNotExist` survives. The
  test reuses the in-package `fakeFetcher` (records `gotURL`) and anchors `wantURL` on the four tessera
  golden path strings from `internal/tiles/layout_test.go` prefixed with `https://sb0.iscc.id/log/`, so
  the URL asserts are tessera ground truth, not author-asserted. Intentional unwired export seam (no
  production caller yet — first caller is the M2 `PollHub` tile-ingestion loop that walks `TileCoords`/
  `BundleCoords` and writes via `RecordTile`/`RecordEntryBundle`); `go vet` clean, not dead code. Oracle
  gate correctly N/A (pure URL construction + transport, no signature/RFC-6962/Merkle/did:web/fsck path),
  same posture as `FetchCheckpoint`; go.mod/go.sum byte-identical (`tessera/api/layout` already in the
  closure), WASM leaves (`tiles`/`didweb`) still build green.
- **An `httptest` round-trip can prove *fetch* but NOT *verify* against a live-host URL** — `Accept`/
  `ResolveVerifierKey` derive the verifier key from `origin(baseURL)`, and the key is origin-bound
  (`SHA-256(name||…)`). A `127.0.0.1:<port>` host derives a different origin than the fixture's
  `sb0.iscc.id/log` signature → always `ErrUnverified`. The correct pattern (used by both
  `TestAcceptCheckpoint` and `TestFetchCheckpointOverHTTP`): fetch over real HTTP via `srv.URL`, then
  verify the fetched bytes with `baseURL="https://sb0.iscc.id"` + a fake Fetcher for the did.json. Do
  not expect `AcceptCheckpoint(srv.URL, …)` to yield `StatusVerified` with captured fixtures.
- **`AcceptCheckpoint` is the pure 4-way verdict seam the follower consumes** (`logclient/accept.go`):
  composes `ResolveVerifierKey → VerifyCheckpoint → DIDKey.ValidAt(observedAt)` into
  `StatusVerified/Unverified/Unresolvable/Rotated`. `observedAt` is injected (never `time.Now()`),
  validity is checked **only after a good signature** (out-of-window-AND-bad-sig is `unverified`, not
  `rotated`), and `CheckpointInfo{Origin,TreeSize,Root}` is the **zero value on every non-verified
  verdict**. Contract gotcha for the follower: a non-`ErrUnverified` `VerifyCheckpoint` error (a
  verified-but-garbled body) is returned as a non-nil `error` alongside `StatusUnverified`'s zero —
  **callers must check `err` before the status**, mirroring `VerifyCheckpoint`.
- **`AcceptCheckpoint` now returns a 4-tuple `(Status, CheckpointInfo, VerifiedContext, error)`; the
  `VerifiedContext{VKey, Key}` is the resolved did:web key context, populated ONLY on `StatusVerified`
  and zero on every non-verified verdict (incl. `StatusRotated`, which resolved a key but failed
  `ValidAt` — accept.go:124-125 still returns the zero context, so a rotated key never leaks into the
  follower's cache/fsck reuse).** `PollHub` threads `vctx` into `cacheHubKeyResolve` (cold cache-miss
  upsert) and `fsckMirror`, which dropped their own `ResolveVerifierKey` calls. The reuse is in-bounds
  because the context was resolved + `ValidAt`-checked *this poll* (ADR-0009); it is never cached
  across polls. After this, a cold verified poll resolves `did.json` once (was 3); production has **no**
  `ResolveVerifierKey` caller left in `follower.go` (verified by grep).
- **A verified poll CANNOT make "0" warm-path did.json fetches — `next.md` asked for the physically
  impossible.** `AcceptCheckpoint` has no internal cache and unconditionally calls `ResolveVerifierKey`
  (accept.go:108), and the `cacheHubKeyFast` cache-hit is consulted only *after* `AcceptCheckpoint`
  returns. So every verified poll (cold or warm) irreducibly fetches `did.json` exactly once. The
  advance author correctly implemented the true value (`TestPollHubCacheHitSkipsDidFetch`: cold = 1,
  warm window = +1) and documented the `next.md` "0" as an arithmetic slip in the handoff — a
  legitimate physically-driven deviation, NOT a gate dodge. The load-bearing win is removing the
  *extra* cold resolves in `cacheHubKeyResolve` + `fsckMirror` (3→1), which the test does pin.

## Checkpoint signed-note verification (`logclient/verify.go`)

- **Signed-note keyhash is independently checkable from the raw sig line.** A C2SP sig line decodes
  to `keyhash(BE-uint32, 4 bytes) || ed25519_sig(64 bytes)`; the vkey's middle `+<hex>+` field IS that
  keyhash. Reviewer can confirm a fixture's signer without trusting the author: sb0 sig→`40b74463`,
  sb1 sig→`069d0f14` (both 64-byte sigs), matching their vkeys. sb1's live key really did rotate
  (`22b08f3e`→`069d0f14`), so the "stale rotated key→ErrUnverified" negative is a genuine real-world
  case, not a contrived one.
- **`note.Open` success check mirrors the notecheck oracle exactly:** `len(n.Sigs) == 0 ||
  len(n.UnverifiedSigs) != 0` → reject. Watch for M7: a future *cosigner* sig line whose key isn't in
  the verifier list lands in `UnverifiedSigs`, so this strict check would reject an otherwise-valid
  hub checkpoint that also carries a witness cosig. Fine for v1 (single hub sig); revisit at gossip.
- **`x/mod v0.33.0` forces the `go` directive to canonical-patch form `go 1.24.0`.** The dep declares
  `go 1.24.0`; under `-mod=readonly` a bare `go 1.24` consumer fails with "updates to go.mod needed".
  `go 1.24.0` is the same minimum (still Go 1.24, NOT ≥1.25) — accept it; it is not a version bump.
  Auto-injected `toolchain go1.24.13` lines should be dropped so go.mod doesn't pin a patch CI may lack.
- **`note.Open` returns `n.Text` as the clean trailing-`\n` body**, so `strings.Split(text, "\n")`
  yields a trailing empty element; `len >= 3` + reading `lines[0:3]` is correct and tolerates C2SP
  extension lines after the root. Verify `note.Open`'s framing — never re-parse the sig line by hand.
- **`KeyIDFromCheckpoint(raw)` recovers `(name, keyID)` from the raw checkpoint via `note.Open(raw,
  note.VerifierList())` + `errors.As(err, &*note.UnverifiedNoteError)`, reading `ue.Note.
  UnverifiedSigs[0].{Name,Hash}`.** This is library-exact: with an empty verifier list every well-formed
  sig line hits `*UnknownVerifierError` and is appended to `UnverifiedSigs` (note.go:569) with `Hash =
  BE-uint32(sig[0:4])` already decoded (note.go:554); `len(n.Sigs)==0` then returns `*UnverifiedNoteError`
  (note.go:597). A malformed note returns `errMalformedNote` (a plain `errors.New`, NOT a pointer type),
  so `errors.As` is correctly false and the garbled path wraps cleanly — no `[0]` index, no panic.
  Reviewer re-decoded the sb0 fixture sig line independently in Python (`base64 → struct ">I" → [:4]`):
  `name='sb0.iscc.id/log'`, `keyhash=0x40b74463`, 64-byte ed25519 sig — confirming the golden from
  ground truth, not the author. Oracle gate correctly N/A (reads an already-decoded keyhash; no
  verify/proof/merkle/didweb path; go.mod/go.sum/schema byte-identical). Still the pure prerequisite —
  wiring it into `PollHub` to consult `LookupHubKey` and skip the 2nd did.json fetch is the next slice.
- **`CheckpointSizeFromRaw(raw) (treeSize uint64, ok bool)` is the UNVERIFIED size sibling of
  `KeyIDFromCheckpoint` (`checkpointsize.go`).** Same `note.Open(raw, note.VerifierList())` + empty
  verifier list → `errors.As(err, &*note.UnverifiedNoteError)` → `ue.Note.Text` is the body, then parse
  line 2 EXACTLY as the unexported `parseCheckpointBody` (reject leading zeros except "0",
  `strconv.ParseUint(_, 10, 64)`). It does NOT verify the signature — the dossier Exhibit caller reads
  back already-signature-verified evidence (`Violation.RawA/RawB`) and only DISPLAYS each claimed size,
  so it must never pull a vkey/did:web resolution into the dossier. Fails closed `(0,false)` on any
  non-note / `<2`-line / leading-zero / non-decimal input. Pure (stdlib + `sumdb/note`), WASM-shareable
  (`GOOS=js GOARCH=wasm go build ./internal/logclient` exit 0). Oracle gate N/A — no
  signature/RFC-6962/Merkle/did:web/fsck path. Ground-truth test: the unverified size byte-EQUALS
  `VerifyCheckpoint`'s (independent line-2 parse) for the same sb0 fixture; off-by-one mutation FAILS it.

## Consistency triggers (`internal/logclient/consistency.go`)

- **Shrink is the one size-only trigger; it lands dep-free.** `CheckShrink(prev, next uint64) bool ==
  prev > 0 && next < prev`. The `prev > 0` guard is load-bearing: it stops the fresh-store
  `FollowState{}.LastSize == 0` from being misread as a shrink (any `next` over a zero prior is growth).
  Fork (same size, different root) is the next dep-free slice (a `[rootBytes]byte` compare at equal
  size); only equivocation (RFC-6962 consistency proof) needs `transparency-dev/merkle` + tile
  fixtures. `ViolationShrink ViolationKind = "shrink"` is the exact `store.Violation.Kind` /
  `violations.kind` string — kept as a logclient string type so store stays import-free of logclient,
  mirroring how `Status` rides on `CheckpointRecord`.
- **`CheckShrink`/`ViolationShrink` are an intentional unused-until-wired export seam, not dead code.**
  `next.md` scoped the `follower.PollHub` wiring (map `LastSize→prev`, `TreeSize→next`; on true →
  `RecordViolation` + `Freeze` + alert-once) as a deliberately separate later slice. `go vet` is clean;
  do not flag the new exported symbols as dead. The shrink check is pure arithmetic, so the
  conformance/oracle gate (`notecheck`/`derive_vkey.py`/`fsck`) is correctly N/A here — it only trips
  once the merkle-backed equivocation slice lands.
- **Equivocation is the merkle-backed third trigger and lands in the same file (`consistency.go`).**
  `CheckEquivocation(prevSize, prevRoot, nextSize, nextRoot [rootBytes]byte, proof [][]byte) (violated,
  err)` calls `proof.VerifyConsistency(rfc6962.DefaultHasher, …, prevRoot[:], nextRoot[:])` ONLY on the
  strictly-growing path (`prevSize>0 && nextSize>prevSize`); ANY verify failure → `(true, nil)` (the
  proof not verifying IS the evidence, never a poll error — ADR-0006). The `prevSize==0 || nextSize<=
  prevSize` guard short-circuits BEFORE `VerifyConsistency` (proven by feeding garbage proof+roots and
  still getting false). `ViolationEquivocation = "equivocation"` matches `violations.kind`. The golden
  is real ground truth: `testonly.New(rfc6962.DefaultHasher)` builds an append-only tree, `HashAt(M/N)`
  gives roots and `ConsistencyProof(M,N)` the valid proof — the prover (`ConsistencyProof`) and the
  verifier (`VerifyConsistency`) are *independent* code paths in merkle, so the cross-check is not a
  tautology. **Reviewer re-ran two mutations (and removed them):** (1) force `(false,nil)` always →
  corrupted-root + corrupted-proof cases FAIL; (2) drop the growing guard → all three boundary-skip
  cases FAIL. So a green-but-wrong verify cannot ship. Oracle gate APPLIES (this is RFC-6962 crypto)
  and is satisfied by the in-test merkle ground truth; `notecheck`/`derive_vkey.py`/`fsck` are N/A (no
  signature/did:web/tile path). `testonly.Tree` was used instead of `next.md`'s literal
  `compact.RangeFactory` — strictly stronger (same hasher, less bespoke test code), not a shortcut.
- **`merkle v0.0.2` is dep-clean on the 1.24 toolchain.** `go.mod` directive stays `go 1.24.0`, no
  `toolchain` line, `go mod tidy` is a no-op, `go mod verify` passes. `go-cmp v0.6.0` enters `go.sum`
  ONLY as a transitive test-dep of `merkle/testonly` (`go mod why` → "main module does not need" it);
  it is NOT a direct require and is CGO-free.
- **`CheckEquivocation`'s `err` return is unreachable-by-type.** The `[rootBytes]byte` array params make
  a wrong-length root unrepresentable, so `err` is always `nil` from this pure layer. The doc was
  trimmed to say so (it previously promised a `(false, non-nil err)` malformed-root path that cannot
  fire); the `wantErr` test column + return are kept for signature symmetry / future slice-param
  loosening. `CheckEquivocation`/`ViolationEquivocation` are an intentional unused-until-wired export
  seam (same as shrink/fork) — `go vet` clean, not dead code.
- **Fork is the second dep-free trigger and landed in the same file as shrink.** `CheckFork(prevSize,
  prevRoot [rootBytes]byte, nextSize, nextRoot [rootBytes]byte) bool == prevSize > 0 && nextSize ==
  prevSize && nextRoot != prevRoot`. Same-size + differing root only; growth/shrink/identical-root all
  false, and the `prevSize > 0` guard keeps the fresh-store zero from misreading. Uses array `!=` (Go
  elementwise on fixed-size `[32]byte`) — `consistency.go` stays import-free of `bytes`/any dep
  (verified: only `bytes` occurrence is the comment explaining why none is needed). `ViolationFork
  ViolationKind = "fork"` matches the `violations.kind` string. Only equivocation (RFC-6962
  consistency-proof) now remains deferred to the merkle-backed slice. Test guard `if rootA == rootB
  { t.Fatal }` makes the "different root" cases non-vacuous.

## Consistency-proof builder (`internal/logclient/proofbuilder.go`)

- **`ConsistencyProofFromTiles(ctx, fetch TileFetcher, smaller, larger uint64) ([][]byte, error)` is
  the proof *source* `CheckEquivocation` needs — a faithful port of tessera `client.ProofBuilder.
  ConsistencyProof + fetchNodes + nodeCache.GetNode` (`cauldron/tessera/client/client.go`) minus the
  otel spans, the net/http `TileFetcherFunc`, and the ephemeral-node map.** Reviewer diffed the `getNode`
  body line-for-line against tessera's `nodeCache.GetNode`: `layout.NodeCoordsToTileAddress` →
  `PartialTileSize(…, larger)` → `fetch` → `api.HashTile.UnmarshalText` → fold `t.Nodes[firstLeaf:
  lastLeaf]` through `compact.RangeFactory{Hash: rfc6962.DefaultHasher.HashChildren}` — identical. The
  dropped `m > pb.treeSize` bounds guard is correct to drop here (this layer is stateless; `larger` IS
  the tree size). Dropping the ephemeral-node check is correct too: `proof.Consistency`'s IDs only ever
  map to real tile nodes, and `nodes.Rehash` supplies the one ephemeral node itself — `getNode` is never
  asked for it.
- **`InclusionProofFromTiles(ctx, fetch, index, size)` is the inclusion sibling — structurally
  byte-identical to `ConsistencyProofFromTiles` with exactly two swaps: `proof.Inclusion(index, size)`
  (vs `proof.Consistency`) and `size` as the `getNode` `logSize` (vs `larger`).** Same per-call tile
  cache + `getNode` loop + `nodes.Rehash`; error wraps mirror the sibling (`compute node list` /
  `get node` / `rehash proof`, all `%w`). `ConsistencyProofFromTiles`/`getNode`/`tileKey`/`TileFetcher`
  stayed byte-identical (additive diff). No new import → go.mod/go.sum unchanged (`proof.Inclusion` was
  already in the closure). It is an unwired-until-M2 export seam (no production caller — the inclusion
  cross-check vs the hub's real `IsccLogInclusionProof` needs captured fixtures + the tile-ingestion
  writer, neither exists yet). Unlike tessera's `ConsistencyProof`, neither builder has an explicit
  `index < size` / `max > treeSize` guard before the `proof.*` call — they lean on `proof.Inclusion`/
  `proof.Consistency`'s own precondition (verified: `index == size` errors cleanly, never panics, never
  reaches the fetcher).
- **settled — both proof builders are oracle-gated (RFC-6962) + mutation-proven over the 300-leaf
  `testonly.Tree` boundary fixture (byte-equal vs `tree.*Proof` AND verify vs `proof.Verify*`, three
  independent merkle paths).** Durable trap to keep: `VerifyInclusion`'s arg order is `(hasher, index,
  size, leafHash, proof, root)` — `leafHash` precedes `proof`, UNLIKE `VerifyConsistency`'s `(…, proof,
  root1, root2)`. (Git history holds the full mutation log: wrong-leaf + node-corruption both FAIL.)
- **The `TileFetcher` signature is byte-identical to `store.SQLiteFetcher.ReadTile`** (`func(ctx
  context.Context, level, index uint64, p uint8) ([]byte, error)`), so the follower's equivocation wiring
  can pass `SQLiteFetcher.ReadTile` straight in — verified both signatures side by side. `larger` (not
  `smaller`) is the `logSize` fed to `PartialTileSize`, matching tessera's `nodeCache.logSize` (the
  partial qualifier reflects the *newer* tree). Empty-proof boundaries (`smaller==0`/`smaller==larger`)
  return a nil slice without touching the fetcher (`proof.Consistency` yields zero IDs) — the test's
  fail-on-fetch fetcher proves this. A genuine tile miss is `%w`-wrapped so `errors.Is(err,
  os.ErrNotExist)` survives, distinct from the "proof fails to verify = violation" verdict (that verdict
  is `CheckEquivocation`'s job, never this builder's).
- **`proofbuilder.go` is net-free though the `logclient` *package* is not.** `next.md` criterion "`go
  list -deps ./internal/logclient | grep net/http` empty" is UNSATISFIABLE for this package and was so at
  baseline — `net/http` enters via `didresolve.go` (the networked did:web resolver), confirmed
  pre-existing. The new file imports only `context`+`fmt`+`merkle/{proof,compact,rfc6962}`+`tessera/api
  {,/layout}`; the load-bearing purity invariant is the **`GOOS=js GOARCH=wasm` build**, which passes.
  Reviewer trimmed a stale file-comment that claimed `os` was imported (it never was — the sentinel rides
  the `%w` wrap, this file never references `os`).
- **The 22 tessera module-graph go.sum lines are now COMMITTED — tidy is idempotent (RESOLVED).** A
  deliberate go.sum-only commit (79e5e37) records the `h1:`/`/go.mod` checksums for tessera's transitive
  *require*-graph modules (otel/klog/x-crypto/formats/backoff) that `tessera/api` widens but **never
  compile** into a monitor package (`go mod why -m <each>` traces through `tessera/api.test` →
  `tessera`, never our packages). Now `go mod tidy && git diff --exit-code -- go.sum` exits 0 and a
  future CI tidy gate passes. Reviewer independently confirmed the recorded checksums are GENUINE, not
  fabricated: `go mod download` of three of the added modules resolved with **no** verification
  error/mismatch (Go recomputes the same `h1:` from source) and `go mod verify` → `all modules
  verified`. Purely additive (no existing go.sum entry rewritten), readonly build+test green, go.mod
  byte-untouched. The trust root is unaffected — didweb/logclient/follower conformance tests pass
  uncached and `derive_vkey.py` reproduces both golden vectors (`40b74463`/`22b08f3e`).
- **Validate go.sum-only commits against the proxy, not just `go mod verify`.** `go mod verify` checks
  the *local cache* against go.sum (tautological if both were written together); `go mod download -x
  <module>@<version>` on a fresh module re-fetches from the proxy and recomputes the `h1:` — a mismatch
  there is the real fabricated-checksum oracle. Combine with `go mod why -m` (proves a require-graph
  module never reaches a monitor package) to confirm a go.sum addition is legitimately graph-only.

## Inclusion cross-check (`internal/logclient/inclusioncheck.go`)

- **`VerifyInclusionEvidence(ctx, fetch TileFetcher, ev InclusionEvidence)` is the first
  production-shaped caller of `InclusionProofFromTiles` and the second external oracle (the
  hub-computed proof).** It guards `LeafIndex < TreeSize` BEFORE fetching (out-of-range never touches
  the fetcher), recomputes the proof from mirrored tiles, base64-**Std**-decodes each `ev.InclusionProof[i]`,
  then length-checks and `bytes.Equal`s each hash; a mismatch wraps `ErrInclusionMismatch` (a sentinel
  for the future `PollHub` `errors.Is`), a tile fault rides the `%w`-wrap so `os.ErrNotExist` survives
  and is NOT misreported as a mismatch (a dedicated negative test pins both). Imports are exactly
  `bytes/context/encoding/base64/encoding/json/errors/fmt` (no `net`/`os`/`sqlite`); reviewer verified.
- **The `InclusionEvidence` struct + guards are oracle-exact against `iscc_hub`.** Reviewer diffed the
  Go struct against `cauldron/iscc-hub/iscc_hub/log_tree.py inclusion_evidence` (the dict shape) and
  `schema.py Evidence` (the field constraints): `{type Literal "IsccLogInclusionProof", checkpoint,
  treeSize ge=1, leafIndex ge=0, inclusionProof list[str]}` — `ParseInclusionEvidence`'s wrong-`Type`
  and `TreeSize == 0` rejections mirror the `Literal` + `ge=1` floor exactly. Confirmed Python's
  `base64.b64encode` == Go's `base64.StdEncoding` (standard `+/` alphabet, `=` padding) by re-running
  both — so the hub's encoding and the monitor's decode are byte-compatible, not assumed.
- **Oracle gate APPLIES (RFC-6962 inclusion crypto), reviewer-reproduced mutation.** The golden plays
  the hub's role with `testonly.Tree.InclusionProof(index, 300)` (prover), the builder
  `InclusionProofFromTiles` (independent path), and a base64+`bytes.Equal` compare — three paths, not a
  tautology, across the 256-leaf boundary `{0,5,255,256,299}`. Reviewer neutered the length+`bytes.Equal`
  compares (short-circuited to `return nil`, kept `bytes` referenced) → BOTH `…CorruptedProof` and
  `…WrongLeafIndex` FAIL; reverted → green. A green-but-wrong check that ignored proof bytes cannot ship.
  Wrong-leaf is the sharp negative: a *valid* proof for leaf 5 re-labelled leaf 6 still fails, because
  the monitor recomputes leaf 6's (different) proof. `notecheck`/`derive_vkey.py` correctly N/A (no
  signature/did:web path — the bundled `checkpoint` is decoded but NEVER re-parsed; that stays
  `AcceptCheckpoint`'s job). Unwired export seam (`go vet` clean); first caller is the `PollHub`/
  `iscc_index` slice that resolves `iscc_id → leafIndex` and passes `SQLiteFetcher.ReadTile` straight in.

## Entry-bundle leaf hasher (`internal/logclient/leafhasher.go`)

- **`LeafHashes(bundle []byte) ([][]byte, error)` is a verbatim-in-shape port of `runfsck`'s
  `leafHasher`** (`cauldron/iscc-hub/conformance/runfsck/main.go:24-35`) — `api.EntryBundle{}.
  UnmarshalText` then `rfc6962.DefaultHasher.HashLeaf(e)` per entry, taking `h[:]`, pre-sized `out`.
  The ONLY deviation from the reference is the error wrap: `%w` + qualified prefix (`logclient.
  LeafHashes: unmarshal entry bundle: %w`) instead of runfsck's `%v` — strictly better (preserves the
  `errors.Is/As` chain) and exactly what `next.md` asked. It is the hasher `fsck.New(...)` takes, so
  the signature must stay `func([]byte) ([][]byte, error)` for the deferred `fsck`-rebuild slice.
- **The truncated/empty edge cases map straight onto `EntryBundle.UnmarshalText`'s framing loop**
  (`tessera@v1.0.2/api/state.go:74-92`, reviewer read source): `len(raw)==0` → loop body never runs →
  zero entries + nil err (nil and `{}` both); a 2-byte prefix promising `size` data bytes where
  `dataIndex+size > len(raw)` returns "require N bytes" — so `{0x00,0x05,0x01}` (claims 5, has 1) errs;
  a zero-length entry (`size==0`) hashes an empty slice fine. The test exercises all three.
- **Oracle gate APPLIES (RFC-6962 leaf-hash crypto) and is satisfied by independent ground truth.** The
  test's `encodeBundle` (manual `binary.BigEndian.PutUint16` framing) is a THIRD code path distinct from
  the decode (`UnmarshalText`) and hash (`HashLeaf`) paths under test; the cross-check is non-circular.
  Reviewer mutation-proved it: `h[0] ^= 0xff` before append → all 3 records FAIL the golden (then
  reverted) — a green-but-wrong hasher cannot ship. `notecheck`/`derive_vkey.py`/`fsck` ground-truth
  oracles are correctly N/A for THIS slice (no signature/did:web/tile-rebuild path introduced); they
  re-arm at the real `fsck.New(...).Check(...)` slice. `LeafHashes` is an intentional unused-until-wired
  export seam (like the consistency triggers, `IsFull`, `LookupHubKey`) — `go vet` clean, not dead code.
  go.mod/go.sum byte-identical (both `tessera/api` + `merkle/rfc6962` already in the closure via
  `proofbuilder.go`); file-level WASM purity holds (`GOOS=js GOARCH=wasm go build ./internal/logclient`
  exits 0, imports are exactly `fmt`+those two).

## fsck root-rebuild wiring (`internal/logclient/fsck.go`)

- **`RunFsck(ctx, vkey, origin string, f fsck.Fetcher) error` is thin glue: `note.NewVerifier(vkey)`
  then `fsck.New(origin, v, f, LeafHashes, fsck.Opts{N:1}).Check(ctx)`, both `%w`-wrapped.** It takes
  the `fsck.Fetcher` *interface* (NOT concrete `store.SQLiteFetcher`), so production `logclient` gains
  no `store` import edge; the test supplies the concrete fetcher from external `package logclient_test`.
  Imports are exactly `context`+`fmt`+`tessera/fsck`+`x/mod/sumdb/note`. No production caller yet —
  intentional unused-until-wired seam (like the consistency triggers / `LeafHashes` / `IsFull`), `go
  vet` clean. Its first caller needs the live tile-ingestion writer (M2, not yet built).
- **Oracle gate APPLIES (RFC-6962 root-rebuild crypto) and is satisfied + independently mutation-proven
  by the reviewer.** Forcing `RunFsck` to `return nil` (drop the `Check` call, reverted) makes BOTH
  corruption subtests FAIL (`RejectsCorruptedTile`/`RejectsCorruptedBundle`) — so the rebuild genuinely
  compares the re-derived root against the signed checkpoint root; a green-but-wrong fsck cannot ship.
  Green case klog: "Successfully fsck'd log with size 5 and root 00d21829…". The prover (`testonly.Tree`
  leaf+node hashes) and verifier (`fsck`+`LeafHashes`) are independent of the fetcher; the bundle
  encoder (manual uint16 framing) is a third path distinct from `EntryBundle.UnmarshalText`. The
  fully-independent `notecheck` oracle is the deferred CI companion (still the sole open `normal` issue).
- **The handoff's "logclient no longer builds for WASM" claim is OVERSTATED but harmless.** `GOOS=js
  GOARCH=wasm go build ./internal/logclient` actually still exits 0 even with `fsck.go` added —
  `net/http`/`klog`/`otel` are all usable under js/wasm (the js/wasm `net/http` is fetch-backed). The
  load-bearing purity invariant is NOT this package; it rides on `internal/didweb` (the WASM-shared
  verifier seam — `internal/proof` doesn't exist yet), which builds clean. Verify with the didweb WASM
  build, never by asserting logclient fails to build for WASM.
- **The `tessera/fsck` require-graph is legitimate, additive, and tidy-idempotent (verified).** `go mod
  why -m` traces `stretchr/testify`/`otel`/`klog/v2`/`formats` through `internal/logclient →
  tessera/fsck` (genuine compile-graph now that `fsck.go` imports it), so klog/otel/formats enter
  go.mod's indirect block and the testify-family (testify/go-spew/difflib/yaml.v3) enter go.sum only as
  transitive test-deps. `go mod tidy && git diff --exit-code -- go.mod go.sum` exits 0, `go mod verify`
  → all modules verified, directive stays `go 1.24.0` (no `toolchain` line). No existing entry rewritten.

## Entry-bundle projection fold (`internal/logclient/projection.go`)

- **`BundleProjections(bundle []byte, baseSeq uint64) ([]Projection, error)` is the pure, schema-agnostic
  iscc_index fold — sibling of `LeafHashes`, same `api.EntryBundle{}.UnmarshalText` decode + import
  posture.** It reads the TOP-LEVEL `iscc_id` and the INNER `note.$schema` (verified against iscc-log.md
  §5.1: envelope is `{$schema, iscc_id, note}`, deletion discriminator is `note.$schema`) and the
  record-content `sha256.Sum256(e)`. The content hash is NOT the RFC-6962 leaf hash — **no `0x00`
  prefix** (that is `LeafHashes`' job); a `0x00`-prefix mutation FAILS the golden (reviewer reproduced).
  Imports are exactly `crypto/sha256`+`encoding/json`+`fmt`+`tessera/api`; file-level WASM build green
  even though the package pulls `net/http`/`net/url`/`os` via `didresolve.go` (pre-existing, not this
  file). go.mod/go.sum/schema.sql byte-identical, store/database/sql un-imported by logclient.
- **`Seq == baseSeq + i` is the absolute leaf index = the spec sequence number** (iscc-log.md §5.2: the
  record committed at tree size N gets seq N). The caller supplies `baseSeq = bundleIndex*256`; this fold
  never computes it. Schema-agnostic per ADR-0008: empty `iscc_id` / unmodeled `note.$schema` indexed
  verbatim, never rejected; only a JSON *parse* failure (or a malformed/truncated bundle frame) is a
  wrapped error naming the absolute seq. Intentional unwired-until-M2 export seam (the store writer +
  `iscc_id→leafIndex` lookup are later slices) — `go vet` clean, not dead code.
- **The golden is non-vacuous, mutation-proven two ways (reverted).** It pins BOTH a declaration and a
  deletion record so the deletion-vs-declaration assertion forces reading the *inner* schema: a mutation
  reading a constant/outer schema FAILS both that test and the schema-agnostic case. Oracle gate
  correctly N/A — pure JSON + content-SHA-256 fold, no signature/RFC-6962/Merkle/did:web/fsck path;
  `notecheck`/`derive_vkey.py` untouched, re-arm at the fsck/inclusion-cross-check wiring slice.
- **`Projection.Timestamp` reads the verbatim optional inner `note.timestamp` (RFC-3339 string, "" when
  absent) — the §6 `· at` source.** Read from `recordEnvelope.Note.Timestamp`, NOT the ISCC-ID-embedded
  `body>>12` time, because a deletion carries the declaration's id (same embedded time) — only the inner
  `note.timestamp` distinguishes the two §6 rows. Read raw, never parsed (ADR-0008): the file-level
  import set stays `crypto/sha256`+`encoding/json`+`fmt`+`tessera/api` (NO `time`), so WASM purity holds
  (`GOOS=js GOARCH=wasm go build ./internal/logclient` exit 0). The golden pins present + absent→"" on
  the two records; constant-Timestamp mutation FAILS both. Follower copies it field-by-field into
  `store.ProjectionRecord.NoteTimestamp` (store never imports logclient).

## Self-consistency verdict (`internal/logclient/checkconsistency.go`)

- **`CheckConsistency(ctx, fetch TileFetcher, prevSize uint64, prevRoot [rootBytes]byte, prevFound
  bool, info CheckpointInfo) (violated, kind, err)` is the single pure home for the shrink→fork→
  equivocation verdict (ADR-0006), composing the three triggers in `consistency.go` + the proof source
  in `proofbuilder.go`.** It is a byte-faithful port of the old `follower.checkConsistency` *minus* the
  `st.CheckpointAt` store read: same branch order, same `prevFound &&` fork guard, same
  `!prevFound || info.TreeSize <= prevSize` equivocation guard, same `(false,"",nil)` missing-tile
  swallow, same `eerr`-wraps path. Imports are exactly `context`+`fmt`; `go list -deps
  ./internal/logclient | grep internal/store` is empty (dependency direction stays follower →
  logclient; the follower passes `SQLiteFetcher.ReadTile` in). The follower's `checkConsistency` keeps
  ONLY the prior-evidence store lookup + a `prevSize==0` early return (wasteful-read skip; duplicated
  by design with the pure guard so the function is total on its own).
- **The unconditional `prevRaw` return from the follower delegate is invisible to `PollHub` — verified.**
  The old follower returned `prevRaw` only on a true verdict and `nil` on clean/missing-tile paths;
  the refactored follower returns `prevRaw` from the store lookup regardless. But `PollHub`
  (follower.go:180-194) checks `err` before `violated` and consumes `prevRaw` ONLY inside `if violated`,
  so the discarded-on-clean-path bytes never reach `freeze`. Semantically identical, no contract change.
- **Oracle gate APPLIES (composes RFC-6962 consistency-proof verification) and is reviewer-mutation-
  proven non-vacuous two ways (reverted):** (1) suppress the `eq → (true, ViolationEquivocation)` return
  → the growing-split-view case FAILS (`violated=false`/`kind=""`); (2) `fork := false` → the fork case
  FAILS. The table builds merkle ground truth via `testonly.Tree` served through `tileFetcherFor` (the
  same fixture as `proofbuilder_test.go`/`inclusioncheck_test.go`); prover (`ConsistencyProof`) and
  composed verifier (`CheckConsistency`→`CheckEquivocation`→`VerifyConsistency`) are independent paths,
  so the cross-check is not a tautology. `notecheck` (in `mise run check`'s `cmd/notecheck`) green;
  `derive_vkey.py` reproduces both vectors (`40b74463`/`22b08f3e`) — N/A here (no signature/did:web
  path touched) but confirmed unmoved. go.mod/go.sum byte-identical; didweb WASM seam still builds.
