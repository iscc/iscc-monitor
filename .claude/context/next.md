# Next Work Package

## Step: Hub dossier increment 1 — numbered trust-document layout + §1–§4

## Advances
Closes the FIRST of the two `critical` Hub-Dossier increments (the immediate, code-closable DONE
blocker) — the steer-filed issue *"Hub dossier renders a flat key/value ledger, not the design's
numbered 'trust document' — increment 1 of 2 (layout + §1–§4)"*. It advances the **M-UI** milestone's
design-parity bar, the dossier named-region row of target.md:

> **hub dossier** — `ISCC Monitor - Hub Dossier.dc.html`: `← Realm index` back-link; the trust-document
> head (eyebrow "Hub dossier", hub name + domain, `md` `HubStatusBadge`, "Compiled by <instance> ·
> <time>"); the numbered sections **§1 Identity** (did:web), **§2 Coverage** (since + size + "N
> observed"), **§3 Latest checkpoint** (size + observed time), **§4 Bitcoin anchor** (dot + label +
> Bitcoin block height when confirmed + "run `ots verify`"); … the two actions ("Prove an ISCC-ID in
> this hub →" certificate, "Browse the log →").

This is a `critical` issue and preempts all `normal`/`low` work. Increment 2 (§5 observation log +
richer Exhibit) is **GATED on this** and is explicitly out of scope here (see Not In Scope).

## Goal
Rebuild the served hub dossier from a flat 5-row key/value card into the mockup's numbered trust
document: trust-document head + a 2×2 numbered grid for §1 Identity / §2 Coverage / §3 Latest
checkpoint / §4 Bitcoin anchor, the two action links, the `fork → "split view"` vocabulary map, and
§5 as an honest minimal placeholder — without fabricating any timestamp, size, or anchor state.

## Scope
- **Create**: (none)
- **Modify** (≤3 non-test/doc files):
  - `internal/store/hubs.go` — add two NULL-safe correlated subselects to `HubSummary` + `ListHubs`,
    mirroring the existing `Anchor` subselect: (1) the latest checkpoint's `observed_at` (§3 time, from
    `checkpoints.observed_at` for this hub) and (2) the confirmed anchor's `btc_height` (§4 height, from
    `ots.btc_height` where `status = OTSStatusConfirmed`). Additive only — existing `dashboard` /
    `proofserve` readers are untouched. Store stays a leaf (no `net/http`/web import).
  - `internal/dossier/handler.go` — expand `dossierData` + `buildData` with: §3 observed time (RFC-3339,
    honest "" when absent), §4 anchor label + dot-state + block height (port the dashboard's `anchorLabel`
    mapping — see Implementation Notes), the derived "N days observed" string from `Coverage.Since`, the
    `fork → "split view"` render map for status/caution copy, and the two action hrefs. No new store
    round-trip beyond `ListHubs`.
  - `internal/dossier/dossier.html` — replace the flat `.ledger` definition-row card with the
    trust-document head (eyebrow + `<h1>` hub name + domain + `md` `HubStatusBadge` + "Compiled by
    <instance> · <time>") and the numbered §1–§4 grid; render §5 as an honest minimal placeholder
    section; add the two action links. KEEP the existing masthead chrome, `← Realm index` back-link, and
    the frozen Exhibit `{{if .Frozen}}` section verbatim (already built + tested).
- **Tests (not counted): update** `internal/store/hubs_test.go` + `internal/dossier/handler_test.go`
  with the new store + HTTP-seam assertions (see Verification). No CLAUDE.md/README change — the dossier
  route is already documented there and its URL-level contract (one hub's badge + coverage + log link) is
  unchanged.
- **Reference** (read before implementing):
  - `.claude/design/ISCC Monitor - Hub Dossier.dc.html` — the authoritative mockup (lines 47–104: head,
    Exhibit, the 2×2 §1–§4 grid at 87–92, the §5 grid at 94–99, the two action buttons at 101–104).
  - `.claude/context/learnings/dashboard.md` — the `anchorLabel`/`anchorDot` precedent (label is the
    grayscale-safe load-bearing signal, dot is decorative), the additive correlated-subselect pattern
    proven on `Anchor`, and the masthead-identity duplication convention dossier↔dashboard already uses.
  - `internal/dashboard/handler.go:299-315` (`anchorLabel`) — port this exact `store.OTSStatus*`-keyed
    mapping locally into the dossier (it is unexported in dashboard; copy it, mirroring the existing
    dossier-local `resolveIdentity` copy — do NOT export dashboard internals for a 4th-file edit).
  - `internal/store/hubs.go:46-94` (`ListHubs` + the `Anchor` subselect to mirror) and
    `internal/store/schema.sql` (`checkpoints.observed_at`, `ots.btc_height`, `ots.status` columns).

## Not In Scope
- **The §5 observation log and the richer frozen Exhibit ("size before → presented" + evidence ref)** —
  that is increment 2 (the sibling `critical`), GATED on this one. Render §5 here as an honest minimal
  placeholder ONLY; do NOT add a `ListCheckpoints`-style store read or parse `Violation.RawA/RawB`.
- **Consolidating the 3× `overlayStatus`/`hubStatus` precedence or the masthead-identity fallback consts
  into `internal/badge`/a shared leaf** — both are tracked `low`s; leave the dossier-local copies.
- **A fabricated per-hub "Prove an ISCC-ID" form.** The dossier has no single subject id, so "Prove an
  ISCC-ID in this hub →" links to the `/` claim-lookup hero (the realm index already foregrounds it),
  NOT a synthesized per-hub lookup form. "Browse the log →" → `/{{.Origin}}/`.
- **The realm-index Anchor honesty `normal` / the per-checkpoint-vs-per-hub anchor question** — untouched.

## Implementation Notes
- **Honesty is the recurring trap here (learnings always-loaded: "gate a rendered claim on real data,
  never fabricate an un-recorded value").** Every §1–§4 value comes from the store row or config: §3
  observed time renders "" → an honest "—"/"observed time unknown" when the checkpoint `observed_at` is
  NULL; §4 renders the normal "pending" / "not anchored yet" state for an unstamped/pending root, NEVER
  an error and NEVER a fabricated "confirmed" (ADR-0001 / CLAUDE.md "Bitcoin anchoring"). The §4 block
  height shows ONLY when the anchor is `confirmed` AND `btc_height` is non-NULL; absent → omit the height,
  never render `0`.
- **§1 Identity is static-derived, not a store/network read:** render `did:web:{{.Domain}}` + the "Domain
  ownership is identity" sub-copy from the mockup (line 88). No did:web fetch — the dossier stays a pure
  store/config render (oracle gate N/A; touches no signature/Merkle/proof path).
- **§4 reuse:** port `anchorLabel(status) (label, dot string)` from `internal/dashboard/handler.go`
  verbatim into the dossier package — it switches on `store.OTSStatusConfirmed`/`OTSStatusPending`, never
  hand-typed literals, so the status strings never drift from the store. The dot is decorative; the LABEL
  carries the meaning grayscale-safe (ADR-0010 inv.4). Add the "OpenTimestamps. Authoritative check: run
  `ots verify`." sub-copy from the mockup.
- **Vocabulary map (BINDING — CLAUDE.md Language):** the store emits `Kind="fork"`, on the avoid-list.
  Where the dossier surfaces a violation kind or status copy OUTSIDE the Exhibit, map `fork → "split
  view"` at render. (The frozen Exhibit's per-violation `.Kind` list is increment-2 territory — leave it
  as-is; the map applies only to any §-level caution/status copy this increment adds.)
- **"N days observed" (§2):** derive from `Coverage.Since` (e.g. whole days since `c.Since`) ONLY when
  coverage is set; render nothing (or "—") when `!Coverage.Set`. Do not invent a count for an uncovered
  hub. Format the §2 line as the mockup's "since <RFC3339> @ size N · N days observed" — reuse the
  existing `coverageTime` helper for the timestamp half.
- **Store subselects:** mirror the `Anchor` subselect's NULL-safe `sql.Null*` + scan-then-assign idiom
  exactly (see `ListHubs` lines 62–88). The §3 observed-time subselect selects the newest checkpoint's
  `observed_at` for the hub (`ORDER BY tree_size DESC, id DESC LIMIT 1`); the §4 height subselect selects
  `btc_height` from the confirmed `ots` row for that hub. Scan into `sql.NullInt64`; expose zero values
  on NULL (`time.Time` / `uint64`).
- **Keep the buffer-then-200 + post-200-write-drop pattern** in `Handler` unchanged (render into a
  `bytes.Buffer`, 500 before any 200). The frozen-only `ListViolations` read stays as-is.
- **`html/template` auto-escapes** the hub domain/origin — keep using it (NOT `text/template`).

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -run TestDossier ./internal/dossier` passes, including new/updated cases asserting at the
  HTTP seam (golden, fixture store, observable no-JS HTML):
  - `GET /<domain>` → `200 text/html`, no `jsdelivr`/`cdn.`/`unpkg`/`googleapis`/`http://` in the body,
    complete with JS disabled.
  - Trust-document head landmarks present: the eyebrow "Hub dossier", the hub name in an `<h1>`, the
    domain, the `md` `HubStatusBadge` markup (`class="hub-status-badge"` + the silhouette marker), and a
    "Compiled by … ·" line; the `← Realm index` back-link (`href="/"`) still present.
  - **§1–§4 each render label + value:** `§1`/`did:web:sb0.iscc.id`; `§2`/coverage since + size + "days
    observed"; `§3`/`<size> entries` + observed time; `§4`/the anchor dot + label.
  - A `confirmed`-anchor fixture renders the §4 block height; a `pending`/never-stamped fixture renders
    the honest "pending"/"not anchored" copy, NO 5xx and no error styling.
  - The two action links resolve: "Prove an ISCC-ID in this hub →" → `href="/"`; "Browse the log →" →
    `href="/sb0.iscc.id/log/"` (`/{{.Origin}}/`).
  - A frozen fixture still renders the non-dismissable Exhibit ABOVE the sections (existing
    `TestDossierFrozenExhibit` stays green); an unresolvable/unverified fixture renders the soft caution
    note, visibly distinct from the Exhibit.
  - §5 renders as an honest minimal placeholder (a `§5` heading, no fabricated observation lines).
- `go test -run TestListHubs ./internal/store` passes — a fixture hub with a checkpoint + a confirmed OTS
  row reports the new §3 observed-time and §4 `btc_height` fields; a hub with neither reports their zero
  values (NULL-safe), and the existing `Anchor` assertions are unchanged.
- **Mutation check (non-vacuous):** reverting the dossier template to the flat `.ledger` rows (or dropping
  a §-section binding) makes a `TestDossier` assertion FAIL; dropping a store subselect makes a
  `TestListHubs` field assertion FAIL.

## Done When
`mise run check` is green and every Verification assertion passes — the served `GET /<domain>` renders the
numbered §1–§4 trust-document layout with honest (never fabricated) §3/§4 data, the two action links, the
`fork → "split view"` map, and an honest §5 placeholder, with the frozen Exhibit and masthead chrome intact.
