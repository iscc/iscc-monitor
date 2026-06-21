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

- **§3 INCLUSION PROOF recomputes the RFC-6962 proof from the mirror — a proof BUILT
  is not a proof VERIFIED, and the `!hub.Frozen` gate is necessary but NOT sufficient.**
  `buildData` calls `logclient.InclusionProofFromTiles(f.ReadTile, data.Position,
  hub.LastSize)` over a `store.SQLiteFetcher` and base64-Std encodes each sibling,
  exactly as proofserve's `serveInclusion` does (the same oracle-gated builder — never
  hand-roll Merkle math). `os.ErrNotExist` → honest §3-omitted gap (the page keeps
  §1/§2); any other err → 500. `InclusionProofFromTiles` is a pure *builder* — it folds
  whatever tile bytes the fetcher returns and NEVER checks the proof rebuilds the
  accepted root. `a687f5e` gated the §3 RENDER on `data.HasClause2 && !hub.Frozen`
  (handler.go:369), closing the STEADY-STATE frozen-after-fork case (mutation-proven by
  `TestCertificateInclusionProofFrozen`: mirror tree A, accept tree B's root, freeze →
  §1+§2 render, §3 absent; reverting the guard FAILS). **Still open (issues.md,
  critical):** the gate reads the `ListHubs` `hub.Frozen` flag, but the HTTP server runs
  CONCURRENTLY with the follower; in a fork poll `ingestTiles` (follower.go:174)
  overwrites same-size tiles in per-tile transactions BEFORE `freeze`→`st.Freeze`
  (follower.go:194, its last write; the freeze records evidence, never advances, so
  `CheckpointAt(LastSize)` keeps the old root). A request landing in that TOCTOU window
  reads `Frozen==false`, builds §3 from the fork's tiles, and renders them under the old
  root's `✓` — the same self-contradictory cert via a race. The durable fix is the
  fail-closed one the plan deferred: before `HasClause3 = true`, verify the built proof
  rebuilds `data.CheckpointRoot` via `proof.VerifyInclusion` (subsumes the frozen gate;
  catches race + steady state alike). §1/§2 are immune (they read the irreplaceable
  accepted-checkpoint *record* via `CheckpointAt`; only §3 reads the corruptible
  *mirror*). **Durable rule: on a self-verifiable surface, gate a rendered Merkle/`✓`
  assertion on a re-VERIFICATION against the accepted root, not on a status FLAG read
  from a separate, racily-updated row.** `TestCertificateInclusionProof` is
  mutation-proven non-vacuous against `testonly.Tree.InclusionProof` but only exercises
  the CLEAN tree; the fail-closed fix's test must assert §3 absent on a NON-frozen
  contradictory-tile fixture.

- **`html/template` entity-escapes base64 `+`/`/` in text nodes (`+`→`&#43;`).** Only
  the execution-path contextual escaper does this — `html.EscapeString` does not — so
  §2's `cm9vdA==` fixture root (no `+`) hid it. Any test asserting on rendered base64
  chips must `html.UnescapeString(body)` first (the §3 test does); the view-model
  `ProofHashes` strings stay byte-identical to `writeEvidence`'s, the on-page entity
  escaping is correct/harmless rendering.
