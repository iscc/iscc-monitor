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
  - settled: cap + prefixed-lookup pinned by `TestCertificateUnacceptedLeaf`/`…KnownID`/`…PrefixedLookup`.

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

- **Masthead identity is config-driven via `Handler(hubList, st, statuses, id dashboard.Identity)`** —
  the SAME value `/` + dossier render (byte-identical chrome; dashboard.md). Resolve ONCE at the top of
  `Handler` and set `data.Instance/Operator` on the value `buildData` RETURNS — NOT inside `buildData`,
  whose `certData{}` literal early-returns bypass it — on BOTH the HTML AND `.bundle` branch. NO Realm
  slot. Fail-safe in the handler (not main.go) → seam-testable. **Trap:** a bare `Contains(body, "monitor
  instance")` is VACUOUS (the footer already carries "issued by this monitor **instance**").
  - settled: cert-local `resolveIdentity`+`instanceFallback`/`operatorFallback` consts (the 3rd byte-
    identical masthead copy — DRY debt filed `low`); pinned by `TestCertificateRendersInstanceIdentity`
    on the `chrome-instance">monitor instance` element, mutation-proven (git history).

- **§3 INCLUSION PROOF is gated on a fail-closed re-VERIFICATION, not a status flag.**
  `buildData` ports `serveVerify` over a `store.SQLiteFetcher`: `InclusionProofFromTiles` →
  `ReadEntryBundle` → `RecordBytesFromBundle` → `HashLeaf` → `proof.VerifyInclusion(... root) == nil`,
  setting `HasClause3` ONLY on nil (siblings base64-Std; `root` is §2's `CheckpointAt` `[]byte`, reused
  inside `HasClause2`). Error split: `os.ErrNotExist`/`ErrLeafOutOfBundle` → §3 omitted; a non-nil
  `VerifyInclusion` is a SILENT decline (never a 500); any other fault → 500 (buffered before any 200).
  `arts.record`/`arts.builtProof` are captured ONLY inside this `if ok` block — they are the SINGLE gate
  shared by the page ✓, the bundle, AND the tier-2 caller's `RecordB64`.
  - settled: §3 gate evolved to re-verification (closed the fork-poll TOCTOU; rule promoted to index);
    `fixtureStoreTiled` seeds `leaf-i` bundles so `HashLeaf==tree.LeafHash` (git history).

- **Tier-2 in-browser verifier (the certificate is the first SSR WASM caller — mechanics in
  `cmd-wasm.md`).** `RecordB64 = base64.StdEncoding.EncodeToString(record)` is set inside the §3 `if ok`
  block (same gate as `arts.record`/`HasClause3`), so it is empty on every honest decline; the template
  reads it ONLY under `{{if .HasBundle}}`. `cert.html` ships a `<script type="application/json">` data
  island (`record`/`root`/`proof[]`/`index`/`size`) + an end-of-body `/_ds/wasm_exec.js`+`/_ds/verify.wasm`
  loader calling `globalThis.isccVerifyInclusion`. Two independent re-verifications (server §3 + browser
  tier-2) must AGREE — gate both on the same re-verify, never a flag. Verified live end-to-end on the
  testnet (`verified` rendered). Mutation: blanking `RecordB64` fails `TestCertificateRendersWasmVerifier`.
  - **No-JS honesty (two regions, one rule): the static `HasBundle` honesty HEADER (`:484`) may only
    OFFER paths — the always-true offline bundle path unconditionally + a CONDITIONAL ("with JavaScript
    enabled") browser re-check; it must NEVER assert a present-tense verdict, because tier-2 is
    progressive enhancement that does not run with JS disabled. The `#tier2-result` panel (`:501`,
    "...the proof ABOVE...") stays the SOLE asserter of an actual browser verdict. Header says "below",
    panel says "above" — keep them distinct so a header assertion is non-vacuous. Pinned by the no-JS
    block in `TestCertificateRendersWasmVerifier` (negative: the old "This browser re-verifies the proof
    below" is gone; positive: the conditional header phrasing renders); reverting the copy FAILs it.

- **§4 SIGNING KEY derives the key id from the accepted checkpoint's OWN raw bytes, not synthetically.**
  `buildData` captures `CheckpointAt`'s `raw`, recovers the key id via `logclient.KeyIDFromCheckpoint(raw)`
  (pure stdlib+sumdb/note; reads only the BE-uint32 keyhash, does NOT verify the sig), then `store.LookupHubKey`.
  `HasClause4` set ONLY on a cache hit — a `KeyIDFromCheckpoint` error or `!found4` miss is an honest decline
  (no §4, no 500); only a `LookupHubKey` DB fault 500s. Key id is oracle-grounded (`0x40b74463` pin in
  `logclient/checkpointkey_test.go`). Since `KeyIDFromCheckpoint` ignores the sig, the §4-happy test threads a
  real signed `Raw` while §3 callers keep `[]byte("raw")`.
  - settled: `found4` cache-hit gate pinned by `TestCertificateSigningKeyUncached` (git history).
- **Any surface building a DID from a domain MUST `%3A`-encode the port** (else did:web reads `8443` as a
  path segment, naming a different did.json). The §4 `SigningKeyDID` AND the proof-bundle `Hub.DID` route
  through the local `didWeb(domain)` helper (`handler.go:116`, `strings.Replace(domain, ":", "%3A", 1)`).
  - settled: both DID sites CLOSED + pinned (`…SigningKeyDIDPortEncoded`/`…ProofBundleDIDPortEncoded`/
    `…DIDCleanDomain`), mutation-proven (git history).
- **`html/template` entity-escapes base64 `+`/`/` in text nodes (`+`→`&#43;`)** — only the
  execution-path contextual escaper, not `html.EscapeString`. Any test asserting on rendered base64
  chips must `html.UnescapeString(body)` first (the §3/§5 tests do); the on-page entity escaping is
  correct/harmless rendering.

- **Every rendered timestamp MUST be `.UTC().Format(time.RFC3339)`, never a bare `.Format`.** The store
  reads coverage/anchor/key instants back via `time.Unix` (`store/hubs.go`, `ots.go`), which re-wraps
  them in `time.Local` — so a bare `.Format(time.RFC3339)` emits the HOST's local offset (`…+01:00` on a
  CET box) and the cert chips fail on every non-UTC host (CI is UTC, so it stays green there and only bites
  a local dev box / non-UTC runner). All four cert chips are now `.UTC()`-normalized: §4 `SigningKeyRevoked`
  + bundle key `Revoked` (`:962`/`:653`, untested — no revoked-key fixture yet), §5 `BTCConfirmedAt`
  (`:1013`), Comparison-Anchor `CoverageSince` (`:1040`). Matches the rest of the federation
  (dashboard/dossier/log-browser already render `Z`); this is the always-loaded Coverage-honesty rule's
  byte-identical-output corollary AND the cross-platform quality bar. Test design (the non-vacuity trap):
  the store round-trip strips a seeded `FixedZone` location, so the non-UTC offset can only be forced by
  swapping `time.Local` for the test (scoped, `t.Cleanup`-restored; the cert suite has NO `t.Parallel`).
  `TestCertificateRendersTimestampsInUTC` pins the §5 + coverage chips to `…Z` and bans `+01:00`,
  mutation-proven on a UTC host (reverting either `.UTC()` re-introduces the offset and FAILS) — unlike the
  two pre-existing TZ-sensitive tests, which only fail off-UTC. Future timestamp chips inherit this rule.
- **§6 RECORD HISTORY is a pure store-read clause — renders unconditionally for a certifiable id.**
  Reuses the `seqs` from `SeqsForISCCID` (ascending), one `RecordAt(hubID, seq)` per row for its verbatim
  `note.$schema`, labelled by a LOCAL `recordKind` (the two FULL wire URIs + `kindUnknown`; `proofserve`'s
  constants are unexported so they are copied — minor DRY debt). Two honesty rules: cap rows to the
  accepted tree (`if seq >= hub.LastSize { continue }`, same boundary as §1) so a deletion above the
  accepted checkpoint is dropped; a `RecordAt` MISS lists the seq with the `kindUnknown` label, not a 500
  (only a DB fault 500s). `HasDeletion` ORs the per-row `isDeletion` for the deletion note.
  - settled: `HasClause6`/`isDeletion` + the `· <ts>` timestamp pinned by `TestCertificateRecordHistory`
    (git history). REMAINING (visual polish, not filed): the mockup humanizes to `2026-02-14 18:40 UTC` —
    a deliberate `time`-parse + format-policy step, not a free follow-up.

- **§5 BITCOIN ANCHOR reads the mirrored OTS row of §2's root and classifies via `ots.ConfirmedFor`
  (DIGEST-BOUND, not the digest-agnostic `ots.Confirmed`).** Inside `HasClause2`,
  `st.OTSForRoot(ctx, hub.HubID, hub.LastSize, root)` keys on §2's RAW `[]byte` root (the same
  `(hub,size,root)` key the `.ots` route + stamp loop use); the classifier fail-closes unless the proof's
  committed `File.Digest` equals that root, so §5 vouches "block N" only for a proof provably committing to
  §2's accepted root (the re-VERIFICATION-not-a-flag index rule). Four fail-closed states (ADR-0001/0004):
  miss OR empty-`OTSBytes` sentinel → §5 OMITTED; unparseable OR digest-mismatch → SILENT decline (never
  500); parseable+digest-bound → `HasClause5` (confirmed: `block <height>`+`UpgradedAt`; pending: "awaiting
  Bitcoin confirmation"). `internal/ots` is NOT WASM-pure but certificate is server-side only.
  - settled: digest-binding CLOSED + five states pinned (`TestCertificateBitcoinAnchor*`, oracle height
    358391 — git history).

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
  `{{else}}` "coverage just started" branch is dead for a rendered panel (`AdvanceAccepted` always sets
  `monitored_since_size` with `last_size`) — kept as a defensive fail-safe. Mockup omits this panel; target.md mandates it
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
