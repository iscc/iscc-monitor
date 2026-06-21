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

- **§1 SUBJECT gates on the accepted-tree cap (`seqs[0] < hub.LastSize`).** `LastSize` rides out of
  the one `ListHubs` scan via `followedHub` (no second round-trip). Honest declines: `LastSize==0`
  → "no accepted checkpoint yet"; `seqs[0] >= LastSize` → "not in accepted tree" (an unaccepted
  `iscc_index` projection ABOVE the accepted checkpoint — the http-surface trap). A size-N tree has
  leaves 0..N-1, so `>=` is the right boundary; matches `serveInclusion`/`serveEntries`. A frozen
  hub's `LastSize` is its last ACCEPTED size (ADR-0006), so the same cap caps it — no frozen branch.
  Mutation: neutering the cap fails `TestCertificateUnacceptedLeaf`.

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

- **§2 CHECKPOINT reads the accepted root back via `CheckpointAt(hub.HubID, hub.LastSize)`**
  (follow_state does NOT persist the root — store.md). Root is base64-**Std**, byte-identical to
  the log browser + verify-for-me. A DB `err` → 500 (buffer-then-200); `!found` leaves
  `HasClause2=false` (honest absence, NOT a 500 — deliberately divergent from verify-for-me which
  500s once it has committed to serving a proof). `found` is realistically always true
  (`AdvanceAccepted` records the checkpoint at the `tree_size` it advances to). Oracle gate N/A
  here (pure store read); it RE-ENGAGES at §3.

- **Interim Hub-List wiring lives in `cmd/iscc-monitor` (`hubListFromEntries`), not
  in config/registry.** Production has no real Hub-List document path yet; the
  binary builds `*registry.HubList` from realm.txt order (slot i = entry i, matching
  the testnet fixture sb0=0/sb1=1). KISS interim; documented TODO. `registry.go` and
  `internal/config` stay untouched. The `Hub` literal needs `*uint16` HubIDs.

- **§3 INCLUSION PROOF is gated on a fail-closed re-VERIFICATION, not a status flag.**
  `buildData` ports proofserve's `serveVerify` path verbatim over a `store.SQLiteFetcher`:
  `InclusionProofFromTiles` → `ReadEntryBundle` → `RecordBytesFromBundle` → `HashLeaf` →
  `proof.VerifyInclusion(hasher, Position, LastSize, leafHash, builtProof, root) == nil`, setting
  `HasClause3` ONLY on the nil verdict (siblings base64-Std; never hand-roll Merkle). `root` is §2's
  `CheckpointAt` `[]byte`, reused inside the `HasClause2` guard. Error split: `os.ErrNotExist` +
  `ErrLeafOutOfBundle` are honest gaps → §3 omitted; a non-nil `VerifyInclusion` is a SILENT decline
  (proof didn't rebuild the root), never a 500; any other fault → 500 (buffered before any 200).
  Replaced the `!hub.Frozen` gate — closes steady-state frozen-after-fork AND the fork-poll TOCTOU
  (no flag read); the re-verify rule is promoted to the index.
  - **`fixtureStoreTiled` must seed entry bundles, not just hash tiles** (copy `encodeBundle` from
    `logclient/fsck_test.go`; write `leaf-i` preimages so `HashLeaf(record)==tree.LeafHash(seq)`),
    or `RecordBytesFromBundle` misses and §3 never gets a leaf hash.
  - Mutation: `proof.VerifyInclusion(...) == nil` → `... == nil || true` fails
    `TestCertificateInclusionProofContradictory`, clean test still PASS. (`if true` won't compile —
    unused `proof`/`leafHash`; `|| true` is the same logical mutation.)
  - settled: §3 gate evolved unconditional → `!hub.Frozen` → re-verification (git history).

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

- **§6 RECORD HISTORY is a pure store-read clause — renders unconditionally for a
  certifiable id, no crypto/cache gate.** `buildData` reuses the `seqs` already in hand
  from `SeqsForISCCID` (ascending), one `RecordAt(hubID, seq)` per row for its verbatim
  `note.$schema`, mapped to a label by a LOCAL `recordKind` (the two FULL wire URIs +
  catch-all `kindUnknown`; `proofserve`'s constants are unexported, so they are copied —
  minor DRY debt, two pure 6-line switches). Two honesty disciplines: (1) cap rows to the
  accepted tree (`if seq >= hub.LastSize { continue }`, same boundary as §1; §1 already
  proved `seqs[0] < LastSize`, so the list is non-empty), so a deletion indexed ABOVE the
  accepted checkpoint is dropped, never implied vouched-for; (2) a `RecordAt` MISS
  (`found == false`, a projection gap) lists the seq with the empty→`kindUnknown` label,
  NOT a 500 — only a real `RecordAt` DB fault 500s (buffer-then-200, like every clause).
  `HasDeletion` ORs the per-row `isDeletion` for the conditional deletion note. Mutation-
  proven non-vacuous (review reproduced both): `HasClause6 = false` kills the whole clause;
  `if isDeletion` → `if false` suppresses the note + deletion-row label — each fails
  `TestCertificateRecordHistory`. The existing `fixtureStore` seeds the BARE
  `iscc-note-0.8.0.json` short form (not the wire URI), so the declaration-only test lands
  the `kindUnknown` path for free; `fixtureStoreHistory` seeds the FULL wire URIs.
- **Mockup §6 row carries a `· at` timestamp the projection has no column for.** The
  `.dc.html` §6 row is `label` + `seq N · at`; `RecordRow` (Seq/IsccID/NoteSchema) holds
  no per-record time, so the impl renders `label · seq N` only. Adding the timestamp needs
  a store schema change (out of scope) — filed `normal` visual-delta. The primary §6
  affordance (kind + seq + deletion note) is complete; the missing time is cosmetic.

## Downloadable proof bundle (`GET /inclusion/{iscc_id}.bundle`)

- **`.bundle` is dispatched inside `Handler` by `CutSuffix` BEFORE `index.Decode`** (the suffix
  is not part of the id), so `/inclusion/<id>` and `/inclusion/<id>.bundle` share the one
  decode→resolve→build chain; `main.go` stays untouched (the mount is already the subtree). The
  bundle reuses `buildData`'s §3 crypto path verbatim — `buildData` now returns
  `(certData, bundleArtifacts, int)`, the HTML path ignores the middle value, the bundle path reads
  it — so the §3 re-verification is the SINGLE gate for both the page ✓ and the bundle (`HasBundle =
  HasClause3`). `serveBundle` writes a fixed-shape `proofBundle` (base64-Std binary fields,
  verbatim signed-note `checkpoint` text matching `InclusionEvidence.Checkpoint`) with the
  drop-the-write-after-200 posture; the §3-declined / non-certifiable path is an honest 200
  `{iscc_id,error}` with NO attachment header and NO `IsccLogInclusionProof` member.
  Mutation-proven (review reproduced both): `proof.VerifyInclusion(...) == nil || true` fails the
  contradictory bundle+page tests; an unconditional `data.HasBundle = true` serves a fabricated
  bundle (empty proof/record — the artifacts are themselves gate-populated) and fails the gap test.
- **A user-supplied id in a `url`/`href` context MUST be path-rooted, never let `ISCC:` lead.**
  `html/template`'s URL-context escaper reads a leading `ISCC:` as an unknown scheme and filters the
  whole attribute to the `#ZgotmplZ` sentinel — a dead link. The download href is built canonically in
  `buildData` as `data.BundleHref = PathPrefix + strings.TrimPrefix(rawID, "ISCC:") + bundleSuffix`
  (path-rooted + prefix-free; the `.bundle` handler decodes the bare form identically). Set it inside
  the certifiable branch so a non-certifiable id leaves it empty (read only under `{{if .HasBundle}}`).
  Mutation: reverting `href="{{.BundleHref}}"` → `href="{{.IsccID}}.bundle"` reproduces
  `href="#ZgotmplZ.bundle"` and fails the `ISCC:`-prefixed sub-case of
  `TestCertificateProofBundleLinkRendered` (review-reproduced). The guard tests BOTH id forms.
  settled: the `#ZgotmplZ` critical (bare form worked, prefixed form broke) is closed by this href.
- **`bundle.Hub.DID` reuses §4's `"did:web:"+Domain` and inherits the `host:port` bug**
  (the same already-filed `normal` issue, now carried on a second surface). Fix both DID-building
  sites together when next touched; `%3A`-encode the port (reuse the resolver's encoding).
