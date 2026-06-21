<!-- area: internal/certificate (realm-wide Certificate of Inclusion, GET /inclusion/{iscc_id}) -->
<!-- indexed-as: certificate.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# `internal/certificate` — realm-wide Certificate of Inclusion

Read this when a step touches the area above. Durable cross-cutting rules live in
the index (`.claude/context/learnings.md`); the package-local mechanics are here.

## Realm-wide certificate skeleton (`GET /inclusion/{iscc_id}`)

- **First real caller of the decode→resolve chain.** The handler runs
  `index.Decode(rawID)` → `registry.HubList.Resolve(id.HubID)` (host of the slot's
  url, scheme stripped) → `followedHub` (linear scan of `ListHubs` matching
  `Domain`) → `SeqsForISCCID(hubID, rawID)`. Every miss at each step is an honest
  **200** "cannot certify" verdict (ADR-0001 fail-closed); only a `ListHubs` /
  `SeqsForISCCID` DB error is a 500, detected via buffer-then-200. Mounts at the
  exact subtree `PathPrefix = "/inclusion/"` in `buildMux`, disjoint from the
  per-hub `/<domain>/log/inclusion` JSON route. `html/template`, `template.Must` at
  init, post-200 write-drop — same shell pattern as dossier. Non-GET → 405. The
  skeleton's tests are non-vacuous (reviewer reproduced: `Resolve(id.HubID+1)` →
  known-id FAILS; swallow the `Decode` error → malformed FAILS).

- **§1 SUBJECT now gates on the accepted-tree cap (`seqs[0] < hub.LastSize`).**
  `buildData` carries `LastSize` out of the one `ListHubs` scan via `followedHub`
  (returns `store.HubSummary`, no second store round-trip) and renders the honest
  cannot-certify states: `LastSize == 0` → "no accepted checkpoint yet";
  `seqs[0] >= LastSize` → "not in accepted tree" (a frozen/failed poll left an
  unaccepted `iscc_index` projection ABOVE the accepted checkpoint — the documented
  http-surface trap). Only `len(seqs) > 0 && seqs[0] < LastSize` certifies, matching
  `serveInclusion`/`serveEntries`/`serveRecord`. A tree of size N has leaves 0..N-1,
  so seq `LastSize-1` is the last certifiable leaf — `>=` is the correct boundary.
  A frozen hub's `LastSize` is its last ACCEPTED size (freeze stops advance,
  ADR-0006), so the same cap caps it at its accepted window with no frozen branch.
  Mutation-proven non-vacuous (review): neutering the cap → `TestCertificateUnacceptedLeaf`
  FAILS.

- **Lookup key is canonicalized to the stored `ISCC:`-prefixed form.** `buildData`
  builds `lookupID := "ISCC:" + strings.TrimPrefix(rawID, "ISCC:")` after a
  successful `index.Decode`, accepting either `/inclusion/MAIG…` or
  `/inclusion/ISCC:MAIG…` and never double-prefixing; `index.iscPrefix` is
  unexported so the literal `"ISCC:"` is used. Production stores `iscc_id` VERBATIM
  and PREFIXED (`projection.go:31-32`), and `SeqsForISCCID` is an exact-bytes match,
  so any store seam keyed on `iscc_id` must use the prefixed ground truth, never the
  bare decode input. Fixtures are re-grounded to the prefixed form + an accepted
  checkpoint (`AdvanceAccepted`, which sets `last_size = TreeSize`). Mutation-proven
  (review): reverting to bare `rawID` → `TestCertificateKnownID` +
  `TestCertificatePrefixedLookup` FAIL ("not found in log").

- **§2 CHECKPOINT reads the accepted root back via `CheckpointAt(hub.HubID, hub.LastSize)`.**
  follow_state does NOT persist the accepted root (store.md), so `buildData`'s certifiable
  branch reads it: size is `hub.LastSize` (the cap already proved `> 0`, no second read), root
  is `base64.StdEncoding.EncodeToString(root)` — base64-**Std**, byte-identical to the log
  browser + verify-for-me (`proofserve` `browserData.Root`/`VerifyVerdict.Root`). A DB `err` →
  500 (buffer-then-200 in place); `found == false` leaves `HasClause2 = false` (no fabricated
  root — honest absence, NOT a 500, deliberately divergent from proofserve verify-for-me which
  500s on `!found` because it has already committed to serving a proof). `found` is realistically
  always true (`AdvanceAccepted` records the checkpoint at the same `tree_size` it advances
  `last_size` to). The §2 test asserts the REAL committed root (`EncodeToString([]byte("root"))`,
  `cm9vdA==`), not a literal — mutation-proven (review): corrupt the rendered root or neuter
  `HasClause2` → `TestCertificateKnownID` FAILS. Oracle gate still N/A (pure store read + render);
  it RE-ENGAGES at §3 (inclusion proof must be non-vacuous vs `IsccLogInclusionProof`/`notecheck`).

- **Interim Hub-List wiring lives in `cmd/iscc-monitor` (`hubListFromEntries`), not
  in config/registry.** Production has no real Hub-List document path yet; the
  binary builds `*registry.HubList` from realm.txt order (slot i = entry i, matching
  the testnet fixture sb0=0/sb1=1). KISS interim; documented TODO. `registry.go` and
  `internal/config` stay untouched. The `Hub` literal needs `*uint16` HubIDs.

- **§3 INCLUSION PROOF is gated on a fail-closed re-VERIFICATION, not a status flag.**
  `buildData` calls `logclient.InclusionProofFromTiles(f.ReadTile, data.Position,
  hub.LastSize)` over a `store.SQLiteFetcher`, then ports proofserve's `serveVerify`
  path verbatim: `bundleIndex/offset/p` → `f.ReadEntryBundle` →
  `logclient.RecordBytesFromBundle(bundle, offset)` → `rfc6962.DefaultHasher.HashLeaf`
  → `proof.VerifyInclusion(hasher, data.Position, hub.LastSize, leafHash, builtProof,
  root) == nil`, and sets `HasClause3` ONLY on the nil verdict (siblings base64-Std
  encoded — never hand-roll Merkle math). `root` is the `[]byte` from §2's
  `CheckpointAt`, reused in place (meaningful only inside the `data.HasClause2` branch
  guard, so no re-decode). Fail-closed error split: `os.ErrNotExist` (missing
  tile/bundle) and `logclient.ErrLeafOutOfBundle` are honest gaps → §3 omitted, page
  keeps §1/§2; a non-nil `VerifyInclusion` result is a SILENT §3 decline (the proof did
  not rebuild the accepted root), never a 500; any other read/build fault → 500
  (buffered before any 200). This subsumes and REPLACED the `!hub.Frozen` gate: it fails
  closed against the steady-state frozen-after-fork case AND the fork-poll TOCTOU race
  (no flag read), closing the trust-root critical. The durable cross-cutting rule (re-
  verify, don't trust a flag) is promoted to the index.
  - **`fixtureStoreTiled` must seed entry bundles, not just hash tiles**, or
    `RecordBytesFromBundle` misses and §3 never gets a leaf hash. Copy `encodeBundle`
    (manual `binary.BigEndian.PutUint16` framing) from `logclient/fsck_test.go`; for each
    `tiles.BundleCoords(size)` write the `leaf-i` preimages so
    `HashLeaf(record) == tree.LeafHash(seq)` (a <256-leaf tree is one partial bundle at
    index 0). That equality is what makes the clean §3 test pass THROUGH the verification.
  - **Mutation-proven non-vacuous (reviewer reproduced):** replacing the
    `proof.VerifyInclusion(...) == nil` guard with `... == nil || true` makes
    `TestCertificateInclusionProofContradictory` (mirror tree A, accept tree B's root,
    `freeze=false`) FAIL while the clean `TestCertificateInclusionProof` stays PASS. (The
    bare `if true` from next.md won't compile — unused `proof`/`leafHash`; the `|| true`
    variant is the same logical mutation and keeps the build valid.)
  - settled: §3 history — `d95bea8` built the proof but rendered it unconditionally
    (self-contradictory for frozen-after-fork); `a687f5e` added `!hub.Frozen` (steady-
    state only, TOCTOU race remained); `681a2c6` replaced it with this re-verification.
    git history keeps the detail.

- **§4 SIGNING KEY derives the key id from the accepted checkpoint's own raw bytes, not synthetically.**
  `buildData` now captures the `raw` return of `CheckpointAt` (was `_`), recovers the key id via
  `logclient.KeyIDFromCheckpoint(raw)` (pure stdlib+sumdb/note; reads only the BE-uint32 keyhash, does
  NOT verify the sig), and reads the cached resolution back via `store.LookupHubKey(hubID, keyID)`.
  `HasClause4` is set ONLY on a cache hit — a `KeyIDFromCheckpoint` error (the cheap `[]byte("raw")`
  fixtures) or a `!found4` miss is an honest decline (no §4, no 500, no fabricated key); only a real
  `LookupHubKey` DB fault is a 500 (buffer-then-200). The derived key id is grounded in the oracle: it
  equals the `0x40b74463` pin in `logclient/checkpointkey_test.go` for the live sb0 checkpoint.
  Mutation-proven (review reproduced): `if found4` → `if found4 || true` makes
  `TestCertificateSigningKeyUncached` FAIL. `KeyIDFromCheckpoint` ignores the signature, so the live sb0
  note seeds an sb1-indexed fixture fine — the test threads a real signed note `Raw` only on the §4
  happy path; §3 callers keep `[]byte("raw")`.
- **`did:web:` + `data.Domain` is WRONG for a `host:port` hub (latent, Codex-confirmed).** §4 builds
  the DID as `"did:web:" + data.Domain`, but `internal/registry` explicitly supports `host:port`
  domains and `didweb.DocumentURL` requires the port colon `%3A`-encoded — so a `host:port` hub renders
  `did:web:localhost:8443` (which did:web reads as host `localhost`, path `8443`), naming a different
  DID than the key resolved from. Not currently exploitable (the testnet realm uses clean
  `sb0.iscc.id`/`sb1.amlet.id`); filed as a `normal` issue. Same fail-quietly-on-clean-fixtures class as
  the `hubDomain` ForceQuery gap. Any surface building a DID from a domain must `%3A`-encode the port.
- **`html/template` entity-escapes base64 `+`/`/` in text nodes (`+`→`&#43;`).** Only
  the execution-path contextual escaper does this — `html.EscapeString` does not — so
  §2's `cm9vdA==` fixture root (no `+`) hid it. Any test asserting on rendered base64
  chips must `html.UnescapeString(body)` first (the §3 test does); the view-model
  `ProofHashes` strings stay byte-identical to `writeEvidence`'s, the on-page entity
  escaping is correct/harmless rendering.
