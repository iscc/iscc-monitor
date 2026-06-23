## 2026-06-23 — Hub dossier increment 1 — numbered trust-document layout + §1–§4

**Done:** Rebuilt the served hub dossier (`GET /<domain>`) from the flat 5-row key/value `.ledger`
card into the mockup's numbered trust document: trust-document head (eyebrow "Hub dossier", `<h1>` hub
name, domain, `md` HubStatusBadge, "Compiled by <instance> · <time>"), a 2×2 numbered grid for
§1 Identity / §2 Coverage / §3 Latest checkpoint / §4 Bitcoin anchor, an honest §5 observation-log
placeholder, the two action links (`/` and `/{{.Origin}}/`), the `fork → "split view"` vocabulary map
in the soft-caution copy, and the frozen Exhibit kept verbatim ABOVE the sections. Every §3/§4 value is
honesty-gated against real store data (no fabricated timestamp/height/anchor state).

**Files changed:**
- `internal/store/hubs.go`: added two NULL-safe correlated subselects to `ListHubs` + two
  `HubSummary` fields — `CheckpointObserved` (newest checkpoint's `observed_at`, `ORDER BY tree_size
  DESC, id DESC LIMIT 1`) and `AnchorHeight` (the confirmed anchor's `btc_height`, scoped to
  `status = OTSStatusConfirmed`). Additive only; the `Anchor`/coverage/`LastSize` reads are unchanged.
- `internal/dossier/handler.go`: expanded `dossierData` + `buildData` with §3 observed time (honest ""
  when absent), §4 anchor label + decorative dot + block height (gated on confirmed AND non-zero height),
  the derived "N days observed" (`coverageDays`, only when coverage is set), the `fork → "split view"`
  `statusNote`, and `ShowCaution`. Ported `anchorLabel` verbatim from dashboard (`store.OTSStatus*`-keyed);
  added `observedTime`/`coverageDays`/`statusNote` helpers; added `fmt` import.
- `internal/dossier/dossier.html`: replaced the `.ledger`/`.row-*`/`.surface` card markup + CSS with the
  trust-document head, the numbered §1–§4 grid, the §5 placeholder, a soft-caution panel (distinct from
  the Exhibit), and the two action buttons. Masthead chrome, `← Realm index` back-link, and the frozen
  `{{if .Frozen}}` Exhibit are kept verbatim (Exhibit now sits inside the document, above the sections).
- `internal/store/hubs_test.go` (test): added `TestListHubsCheckpointAndAnchorHeight` (populated +
  zero-value, mutation-proven).
- `internal/dossier/handler_test.go` (test): updated `TestDossierRendersCoveredHub` to the §2 format +
  added trust-document-head / §1–§4 / action-link / §5 assertions; added `TestDossierConfirmedAnchorRendersHeight`,
  `TestDossierPendingAnchorHonest`, `TestDossierCautionForUnverified`.

**Verification:** `mise run check` → green (all 28 pkgs `ok`; `gofmt -l .` empty). Per-criterion:
- [x] `GET /<domain>` → 200 text/html, no `jsdelivr`/`cdn.`/`unpkg`/`googleapis`/`http://`, no-JS complete.
- [x] Trust-document head landmarks present (eyebrow, `<h1 class="doc-hub-name">`, domain, `md` badge
  markup `class="hub-status-badge"` + silhouette, "Compiled by … ·"); `← Realm index` (`href="/"`) intact.
- [x] §1–§4 each render label + value (`§1`/`did:web:sb0.iscc.id`; `§2`/since+size+"days observed";
  `§3`/`42 entries`+observed time; `§4`/anchor dot+label).
- [x] Confirmed-anchor fixture renders the §4 `block 869440`; pending/never-stamped renders honest
  "pending"/"not anchored", no 5xx, no error styling, no fabricated height.
- [x] Action links: `href="/">Prove an ISCC-ID…`, `href="/sb0.iscc.id/log/">Browse the log…`.
- [x] Frozen fixture still renders the non-dismissable Exhibit (existing `TestDossierFrozenExhibit`
  green); unverified fixture renders the soft caution (`class="caution"`, "split-view" copy), distinct
  from the Exhibit.
- [x] §5 renders heading + "no entries yet" placeholder, no fabricated observation lines.
- [x] `TestListHubsCheckpointAndAnchorHeight` green (populated + NULL-safe zero values; existing
  `TestListHubsAnchorStatus` unchanged).
- [x] Mutation checks non-vacuous: breaking the `AnchorHeight` subselect-assignment FAILs the store test;
  dropping the §4 height binding FAILs `TestDossierConfirmedAnchorRendersHeight`.

**Next:** Increment 2 (the sibling `critical`, GATED on this) — the §5 observation log + the richer
frozen Exhibit ("size before → presented" + evidence ref). It will add a `ListCheckpoints`-style store
read (and parse `Violation.RawA/RawB`), turning the §5 placeholder into real per-poll lines and filling
the Exhibit grid. The §5 markup hook (`.obs` / `.obs-empty`) and the `violationRow` struct are the seams
it extends.

**Notes:**
- Oracle/conformance gate is N/A: pure HTML render of persisted store rows + an in-memory status overlay;
  touches no signature/RFC-6962/Merkle/did:web/fsck/proof path. `go.mod`/`go.sum`/schema byte-identical.
- The §2 "N days observed" is a live wall-clock derivation from `Coverage.Since` (`time.Since`), so the
  rendered count grows with real time — the test asserts the literal "days observed" suffix, not a fixed
  number, to stay deterministic. A future-dated `Since` (clock skew) clamps to "0 days observed".
- The §4 height subselect is scoped to `status = OTSStatusConfirmed` (the dossier-specific gate); the
  pre-existing `Anchor` status subselect remains newest-stamped-regardless-of-status, matching the
  realm-index per-hub anchoring-activity semantics. These are two distinct subselects by design — the
  dossier §4 height MUST come from a confirmed row, never a newer pending one.
- Kept the dossier-local `overlayStatus`/`hubStatus`/`resolveIdentity`/fallback-const copies as-is
  (consolidation into `internal/badge`/a shared leaf is the tracked `low`, explicitly out of scope here).
  `anchorLabel` is now a 2nd copy (dashboard + dossier) — same consolidation pressure; left local per
  next.md's "do NOT export dashboard internals for a 4th-file edit".
- Mockup parity: the `md`-size badge is the badge package's intrinsic SSR size (the Go partial has no
  Size field; size is a CSS/mockup concern), so `{{template "hubStatusBadge" .}}` renders it directly.
