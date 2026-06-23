<!-- area: internal/dossier (handler.go, dossier.html) + store.ListHubs §3/§4 subselects -->
<!-- indexed-as: dossier.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# Hub dossier — server-rendered per-hub trust document at `GET /<domain>`

Read this when a step touches the area above. Durable cross-cutting rules live in
the index (`.claude/context/learnings.md`); the dashboard masthead/overlay mechanics
this surface shares live in `learnings/dashboard.md` (read both).

## Numbered trust-document layout (§1–§5)

- **The dossier is a numbered "trust document" matching `.claude/design/ISCC Monitor - Hub Dossier.dc.html`,
  NOT a flat key/value ledger.** Structure: trust-document head (eyebrow "Hub dossier", `<h1>` hub name,
  `Origin` domain, `md` `hubStatusBadge`, heavy `2×border-width` bottom rule, "Compiled by {{.Instance}} ·
  <observed-time>"), then the frozen Exhibit OR the soft caution, then the 2×2 §1–§4 grid, the §5
  observation-log placeholder, and the two action links. The `md` badge size is intrinsic to the partial
  (no Size field) — `{{template "hubStatusBadge" .}}` renders it directly. Visual pass (reviewer, confirmed
  vs mockup): full design-parity match incl. confirmed-anchor green dot + "block N".
- **Every §1–§4 value is honesty-gated against real store data (the recurring SSR-honesty trap).** §2's
  `CoverageDays` ("N days observed", live `time.Since(c.Since)`, clamps to 0 on future skew) is "" unless
  coverage is set; §3's `ObservedTime` is "" → "observed time unknown" when the checkpoint time is NULL;
  §4's `AnchorHeight` renders ONLY when `Anchor == OTSStatusConfirmed AND AnchorHeight > 0` (`HasAnchorHeight`)
  so a NULL/zero never reads as "block 0". `anchorLabel` is a verbatim port of dashboard's (switches on
  `store.OTSStatus*` consts, never literals; label is grayscale-safe, dot decorative — ADR-0010 inv.4).
- **The §3 observed-time subselect is now `AND c.tree_size = f.last_size ORDER BY c.id DESC LIMIT 1` (advance
  `820a831`) — tied to the accepted size, which CLOSES the higher-size/equivocation decouple but NOT the
  same-size FORK case.** Violation kinds are size-partitioned (`logclient/checkconsistency.go:35-37`): shrink
  `next<prev`, fork `next==prev`, equivocation `next>prev`. For an equivocation the rejected checkpoint is
  LARGER, so `tree_size = last_size` no longer matches it → §3 reads the accepted row (mutation-proven by
  `TestListHubsFrozenObservedTracksAcceptedSize`). For a FORK, `follower.freeze` records the contradictory
  checkpoint at `tree_size == last_size` (same size, later `id`), so the subselect matches BOTH rows and
  `id DESC LIMIT 1` STILL picks the rejected fork row — §3 pairs the accepted size with the fork's
  `observed_at` (reviewer-reproduced). The durable fix is `ORDER BY c.id ASC` (the accepted row at that size
  is the EARLIEST — `store.CheckpointAt` already uses `ORDER BY rowid`; a re-observed accepted checkpoint is
  deduped by `ON CONFLICT(hub_id,tree_size,root)`, so two rows at `last_size` means a fork), OR key by the
  accepted root. Open `normal` (the fork remainder). [Codex P2, reviewer-confirmed by reproduction.]
- **§1 Identity is static-derived ("Key resolved from did:web:<domain>"), rendered UNCONDITIONALLY (no
  network/store read) — so it contradicts the `unresolvable` overlay copy.** The mockup + next.md specify
  the static phrasing (§1 is about WHERE the key comes from = domain ownership, not a per-request verdict),
  but on the `unresolvable` path the caution says "signing key is unresolved" while §1 still asserts "Key
  resolved from". A design-rooted wording-honesty nit (the fix is neutral "Key source:" wording or gating
  "resolved" off `unresolvable`); do NOT silently rewrite the mockup-specified copy without a design pass.
  Open `normal`. [Codex P2, reviewer-confirmed.]

## Seams shared with dashboard (do not re-derive)

- **Masthead + overlay are byte-identical ports of dashboard's** (the dossier imports `dashboard.Identity`
  and reuses its `instanceFallback`/`operatorFallback`/`resolveIdentity`/`overlayStatus`/`hubStatus` shape
  as local copies — 3× now, a tracked `low` consolidation). The "Browse the log →" link is `/{{.Origin}}/`
  → `/<domain>/log/` because `Origin` = `<domain>/log`; "Prove an ISCC-ID in this hub →" links to `/` (the
  realm-index claim hero, NOT a fabricated per-hub form). Edit any masthead → mirror all three SSR HTML files.
- **§5 observation log + richer frozen Exhibit BOTH LANDED (2a `96e9600`, 2b `872ab8b`).** §5 is derived
  in the view layer from `store.ListCheckpoints` (newest-first by `observed_at DESC`) — a size-transition
  line per consecutive **strictly-increasing** pair, the oldest checkpoint as a singleton, an "anchored ·
  block N" line ONLY when `Anchor == OTSStatusConfirmed && AnchorHeight > 0`, and a "froze hub
  (<kind→split view>)" pointer per recorded violation (off the already-fetched `violations` slice). It
  NEVER synthesizes a per-poll "consistent" line. The Exhibit now reads each contradictory checkpoint's
  tree size back via `logclient.CheckpointSizeFromRaw(Violation.RawA/RawB)` (UNVERIFIED — the stored raws
  are already signature-verified evidence) and renders "tree size <before> → then presented <presented>"
  + a content-derived `evidenceRef` (`sha256(RawA‖RawB)[:6]` hex). **`RawA`=prior accepted, `RawB`=presented**
  (`follower.freeze`), so on a fork the "presented" size can be SMALLER than "before" (e.g. 10183 → 61) —
  that is honest evidence, not a bug; §5 separately suppresses it as a non-transition.
- **TRAP (RESOLVED 2b) — §5 must skip NON-INCREASING consecutive pairs, and gate the oldest singleton on a
  real transition (`transitioned bool`), never `len(checkpoints)>1`.** `follower.freeze` `RecordCheckpoint`s
  the contradictory checkpoint (same-size/different-root OR shrunk) at a LATER `observed_at` without
  advancing `last_size`, so a naive loop renders a phantom `size N → N` / `larger → smaller`. The fix
  (`newer.TreeSize <= older.TreeSize` → `continue`) is mutation-pinned by
  `TestDossierObservationLogFrozenNoPseudoTransition`. Keep this skip whenever editing the §5 loop.
- **settled:** the Exhibit's unverified size read MUST fail closed — both raws parse (`HasSizes`) or the row
  degrades to "tree sizes unavailable" (no half-size, no fabricated `0`); the evidence ref is still emitted
  (a stable handle) when sizes are unparseable, empty only when both raws are empty.
