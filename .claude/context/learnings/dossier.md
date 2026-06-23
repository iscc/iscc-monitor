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
- **The §3 subselect (`checkpoints` newest by `tree_size DESC, id DESC`) is DECOUPLED from §3's rendered
  size (`.LastSize` = accepted `follow_state.last_size`).** For a verified hub they agree. For a FROZEN hub
  they can diverge: `follower.freeze` does `RecordCheckpoint` of the contradictory (often higher-tree-size)
  checkpoint WITHOUT advancing `last_size`, so the `tree_size DESC` subselect can pick the rejected
  checkpoint's `observed_at` while §3 shows the accepted size — an accepted size paired with a rejected
  timestamp. Confined to the frozen edge state (the loud Exhibit already says "do not trust new state"), and
  a `shrink` violation cannot trigger it (rejected size is smaller). Open `normal`; the durable fix is to
  select `observed_at` for the row whose `tree_size = f.last_size` (size + time from ONE row). [Codex P2,
  reviewer-confirmed against `follower.freeze`.]
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
- **§5 observation log LANDED (increment 2a, advance `96e9600`): it is derived in the view layer from
  `store.ListCheckpoints` (newest-first by `observed_at DESC`) — a size-transition line per consecutive
  pair, the oldest checkpoint as a singleton (only when ≥2 checkpoints), an "anchored · block N" line ONLY
  when `Anchor == OTSStatusConfirmed && AnchorHeight > 0`, and a "froze hub (<kind→split view>)" pointer
  per recorded violation on the frozen path (off the already-fetched `violations` slice, no second read).**
  It NEVER synthesizes a per-poll "consistent" line (no recorded per-poll verdict exists; emitting one
  asserts an un-run check). A NULL `observed_at` renders the size without a time. The richer frozen Exhibit
  ("size before → presented" + evidence ref) is still increment 2b (needs an unverified tree-size parse of
  `Violation.RawA/RawB`; `parseCheckpointBody` is unexported + verifies first).
- **TRAP — the §5 transition loop assumes monotonically-GROWING size, which the frozen path breaks.** It
  orders strictly by `observed_at DESC` and renders `size <older.TreeSize> → <newer.TreeSize>` for each
  consecutive pair. But `follower.freeze` `RecordCheckpoint`s the CONTRADICTORY checkpoint (the `checkpoints`
  UNIQUE is `(hub_id,tree_size,root)`, so a same-size/different-root row persists) at a LATER `observed_at`,
  WITHOUT advancing `last_size`. So a fork/equivocation (same size) renders a literal `size N → N` and a
  shrink renders `size <larger> → <smaller>` — neither is a real transition. Confined to the frozen edge (the
  loud Exhibit dominates above §5; the freeze pointer line records the real event), so it does NOT fabricate
  a "consistent" verdict — but it is misleading. Open `normal` (folded into 2b): skip non-increasing pairs in
  the loop and only emit the singleton when a real increasing transition was emitted. [Codex P2, reviewer-
  confirmed by a fork reproduction.]
