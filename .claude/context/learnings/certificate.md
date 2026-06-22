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

- **§1 SUBJECT — accepted-tree cap `seqs[0] < hub.LastSize`; lookup key is the stored `ISCC:`-prefixed
  form.** A size-N tree has leaves 0..N-1 so `>=` is the right "not in accepted tree" boundary (matches
  `serveInclusion`/`serveEntries`); a frozen hub's `LastSize` is its last ACCEPTED size (ADR-0006) so the
  same cap caps it (no frozen branch). The store keys `iscc_id` VERBATIM+PREFIXED and `SeqsForISCCID` is
  exact-bytes, so canonicalize to `"ISCC:" + strings.TrimPrefix(rawID, "ISCC:")` — any seam keyed on
  `iscc_id` must use the prefixed ground truth, never the bare decode input.
  - settled: the cap + prefixed-lookup mutations are pinned by `TestCertificateUnacceptedLeaf` /
    `…KnownID` / `…PrefixedLookup` (git history).

- **§2 CHECKPOINT reads the accepted root back via `CheckpointAt(hub.HubID, hub.LastSize)`** (follow_state
  does NOT persist the root — store.md); root is base64-**Std**, byte-identical to the log browser +
  verify-for-me. `!found` leaves `HasClause2=false` (honest absence, NOT a 500 — deliberately divergent
  from verify-for-me which 500s once committed to serving a proof); only a DB `err` 500s (buffer-then-200).
  **`HasClause2` is the gate the §5 OTS read AND the COMPARISON ANCHOR both sit inside** — both reuse §2's
  already-loaded `(size, root)`, so neither adds a read or a fault path.

- **Interim Hub-List wiring lives in `cmd/iscc-monitor` (`hubListFromEntries`), not
  in config/registry.** Production has no real Hub-List document path yet; the
  binary builds `*registry.HubList` from realm.txt order (slot i = entry i, matching
  the testnet fixture sb0=0/sb1=1). KISS interim; documented TODO. `registry.go` and
  `internal/config` stay untouched. The `Hub` literal needs `*uint16` HubIDs.

- **§3 INCLUSION PROOF is gated on a fail-closed re-VERIFICATION, not a status flag.**
  `buildData` ports `serveVerify` over a `store.SQLiteFetcher`: `InclusionProofFromTiles` →
  `ReadEntryBundle` → `RecordBytesFromBundle` → `HashLeaf` → `proof.VerifyInclusion(... root) == nil`,
  setting `HasClause3` ONLY on nil (siblings base64-Std; `root` is §2's `CheckpointAt` `[]byte`, reused
  inside `HasClause2`). Error split: `os.ErrNotExist`/`ErrLeafOutOfBundle` → §3 omitted; a non-nil
  `VerifyInclusion` is a SILENT decline (never a 500); any other fault → 500 (buffered before any 200).
  `arts.record`/`arts.builtProof` are captured ONLY inside this `if ok` block — they are the SINGLE gate
  shared by the page ✓, the bundle, AND the tier-2 caller's `RecordB64`.
  - settled: §3 gate evolved unconditional → `!hub.Frozen` → re-verification, closing the fork-poll
    TOCTOU; the re-verify rule is promoted to the index; `fixtureStoreTiled` seeds `leaf-i` bundles so
    `HashLeaf==tree.LeafHash` (git history).

- **Tier-2 in-browser verifier (the certificate is the first SSR WASM caller — mechanics in
  `cmd-wasm.md`).** `RecordB64 = base64.StdEncoding.EncodeToString(record)` is set inside the §3 `if ok`
  block (same gate as `arts.record`/`HasClause3`), so it is empty on every honest decline; the template
  reads it ONLY under `{{if .HasBundle}}`. `cert.html` ships a `<script type="application/json">` data
  island (`record`/`root`/`proof[]`/`index`/`size`) + an end-of-body `/_ds/wasm_exec.js`+`/_ds/verify.wasm`
  loader calling `globalThis.isccVerifyInclusion`. Two independent re-verifications (server §3 + browser
  tier-2) must AGREE — gate both on the same re-verify, never a flag. Verified live end-to-end on the
  testnet (`verified` rendered). Mutation: blanking `RecordB64` fails `TestCertificateRendersWasmVerifier`.

- **§4 SIGNING KEY derives the key id from the accepted checkpoint's OWN raw bytes, not synthetically.**
  `buildData` captures `CheckpointAt`'s `raw`, recovers the key id via `logclient.KeyIDFromCheckpoint(raw)`
  (pure stdlib+sumdb/note; reads only the BE-uint32 keyhash, does NOT verify the sig), and reads the cached
  resolution via `store.LookupHubKey(hubID, keyID)`. `HasClause4` is set ONLY on a cache hit — a
  `KeyIDFromCheckpoint` error or a `!found4` miss is an honest decline (no §4, no 500, no fabricated key);
  only a real `LookupHubKey` DB fault 500s. Key id is oracle-grounded (equals the `0x40b74463` pin in
  `logclient/checkpointkey_test.go`). Because `KeyIDFromCheckpoint` ignores the sig, the §4-happy-path test
  threads a real signed note `Raw` while §3 callers keep `[]byte("raw")`.
  - settled: `found4` cache-hit gate pinned by `TestCertificateSigningKeyUncached` (git history).
- **Any surface building a DID from a domain MUST `%3A`-encode the port** — a bare `host:port`
  colon makes did:web read `8443` as a path segment, naming a different did.json than the key
  resolved from. The §4 `SigningKeyDID` AND the proof-bundle `Hub.DID` both route through the local
  `didWeb(domain)` helper (`handler.go:116`, `strings.Replace(domain, ":", "%3A", 1)`, the resolver's
  idiom); a no-port domain round-trips byte-identical.
  - settled: the `host:port` DID bug (both sites) is CLOSED, pinned by
    `TestCertificateSigningKeyDIDPortEncoded` (§4) + `TestCertificateProofBundleDIDPortEncoded`
    (bundle), both mutation-proven, with `…DIDCleanDomain` as the no-port regression (git history).
- **`html/template` entity-escapes base64 `+`/`/` in text nodes (`+`→`&#43;`)** — only the
  execution-path contextual escaper, not `html.EscapeString`. Any test asserting on rendered base64
  chips must `html.UnescapeString(body)` first (the §3/§5 tests do); the on-page entity escaping is
  correct/harmless rendering.

- **§6 RECORD HISTORY is a pure store-read clause — renders unconditionally for a certifiable id.**
  Reuses the `seqs` from `SeqsForISCCID` (ascending), one `RecordAt(hubID, seq)` per row for its verbatim
  `note.$schema`, labelled by a LOCAL `recordKind` (the two FULL wire URIs + `kindUnknown`; `proofserve`'s
  constants are unexported so they are copied — minor DRY debt). Two honesty rules: cap rows to the
  accepted tree (`if seq >= hub.LastSize { continue }`, same boundary as §1) so a deletion above the
  accepted checkpoint is dropped; a `RecordAt` MISS lists the seq with the `kindUnknown` label, not a 500
  (only a DB fault 500s). `HasDeletion` ORs the per-row `isDeletion` for the deletion note.
  - settled: `HasClause6`/`isDeletion` mutations pinned by `TestCertificateRecordHistory`; KNOWN GAP
    (filed `normal`): the mockup §6 row carries a `· at` timestamp `RecordRow` has no column for —
    needs a store schema change, the impl renders `label · seq N` only (git history).

- **§5 BITCOIN ANCHOR reads the mirrored OTS row of §2's root and classifies via `ots.Confirmed`.**
  Inside `HasClause2`, `st.OTSForRoot(ctx, hub.HubID, hub.LastSize, root)` keys on §2's RAW `[]byte` root
  (NOT the base64 `CheckpointRoot`), the same `(hub,size,root)` key the `.ots` route + stamp loop use.
  Three fail-closed states (ADR-0001/0004): a miss OR the empty-`OTSBytes` sentinel → §5 OMITTED; an
  unparseable proof → SILENT decline (never 500); a parseable proof → `HasClause5=true` (confirmed shows
  `block <height>` + `UpgradedAt` RFC-3339; pending shows "awaiting Bitcoin confirmation"). `internal/ots`
  is NOT WASM-pure but certificate is server-side only.
  - settled: four state mutations pinned (height tied to oracle literal 358391); fixtures byte-identical
    from `internal/ots/testdata`. KNOWN GAP (filed `normal`, see issues.md): §5 does NOT bind the proof's
    `File.Digest` to §2's root, so a mis-stamped row would falsely render "block N" — fix `bytes.Equal`.

- **COMPARISON ANCHOR is §2's `(size, root)` reframed as the monitor's own observation — a SEPARATE,
  distinctly-labelled element from §5, NOT Bitcoin.** Set `data.HasComparisonAnchor = true` inside the
  `HasClause2` guard (reuses `data.CheckpointSize`/`CheckpointRoot`, no re-read/re-encode), plus the
  coverage window from `followedHub`'s `hub.Coverage` (`HubSummary`, no second store round-trip). It does
  NOT depend on the OTS row (the `IndependentOfOTS` test: §5 absent, panel present) — that decoupling is
  the load-bearing "separate, distinctly-labelled elements" Verify criterion. Copy stays glossary-clean:
  "Comparison anchor"/"detect a split view", NEVER "witness" (deferred M7) or any "anchoring"/Bitcoin
  lexicon (a panel-slice test bans `Bitcoin`/`anchoring`/`OpenTimestamps`/`BITCOIN ANCHOR`/`ots verify`
  inside the sliced panel). Coverage honesty (ADR-0001): `Coverage.Set`→state the window (`since size N`
  `· <RFC-3339>` only when the time is non-zero, mirroring `SigningKeyRevoked`'s zero-guard); the
  `{{else}}` "coverage just started" branch is effectively dead for a rendered panel — `AdvanceAccepted`
  always sets `monitored_since_size` in the same tx that advances `last_size`, so any §2-rendering hub has
  `Coverage.Set==true` (kept as defensive fail-safe, fine). Mockup omits this panel; target.md mandates it
  (design-parity: constraint > mockup), flagged in the docstrings. Mutations (review): `HasComparisonAnchor
  = false` AND `CoverageSize = 0` each fail the three new tests.

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
- **`bundle.Hub.DID` shares §4's `didWeb(domain)` helper** (one encoder, two surfaces) so a
  `host:port` hub's bundle DID is `did:web:host%3Aport` while `bundle.Hub.Domain` stays the verbatim
  `host:port`. settled: pinned by `TestCertificateProofBundleDIDPortEncoded` (git history).
