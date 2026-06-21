---
status: accepted
---

# Evidence-Ledger frontend direction and ISCC Design System adoption

The v1 web surfaces (dashboard, log browser, verifier app) were specified
functionally (PRD §8, ADR-0003) but with **no chosen visual direction**. The UX
brief (`.claude/design/0001-ux-brief-v1.md`) asked for three alternative
directions along three axes. The UX designer has now decided and produced a
developer handoff: **`.claude/design/ISCC Monitor - Developer Handoff.dc.html`**,
with six standalone screen mockups and a design-system bundle
(`.claude/design/_ds/iscc-design-system-v2-…/`).

**Decision.** Adopt **Direction 2 — "Evidence Ledger"** (the forensic / document
axis: foreground durable per-claim evidence; document-like, citation-friendly,
screenshot-ready for legal/audit contexts) as the v1 frontend, built on the
**ISCC Design System v2** tokens. The Developer Handoff is the design source of
truth for build (subordinate to the ADRs/PRD; where they disagree, the ADR/PRD
wins — flag it). This is delivered as a distinct v1 milestone (**M-UI** in
`target.md`), sequenced after M3's functional surfaces and before the WASM
upgrade, so the server-rendered pages exist first and WASM re-verification layers
on top (ADR-0003).

**Scope — what the direction commits us to:**

- **Three surfaces, six screens.** Surface A (trust dashboard): *Realm index*
  (`GET /`, claim-lookup foregrounded above the hub register) + *Hub dossier*
  (a numbered trust document incl. the **frozen Exhibit**) + *Certificate of
  inclusion* (the proof-result page, proof-bundle download primary). Surface B
  (log browser): *record list* (newest-first, link-based pagination) + *single
  record*. Surface C (verifier app): *Independent verification* — the WASM
  milestone (ADR-0003), hosted at `monitor.iscc.codes`.
- **Server-rendered, no-JS baseline (hard rule).** Surfaces A & B must be useful
  and complete as plain SSR HTML/CSS with JavaScript disabled; WASM re-verification
  is additive progressive enhancement, never a gate. No SPA framework for A & B.
  Implemented with Go `html/template` + `go:embed` (consistent with the existing
  `internal/dashboard` leaf).
- **Design-system assets are self-hosted, embedded via `go:embed`.** The DS tokens
  (`tokens/*.css`) and the Readex Pro + JetBrains Mono woff2 fonts are vendored
  into the binary and served from the instance — **no third-party CDN at runtime**
  (the handoff's `fonts.css` points at jsDelivr; we override that). Rationale: a
  trust/evidence tool must render deterministically, offline, and from one static
  binary (`CGO_ENABLED=0`, ADR-0003), with no external availability or
  supply-chain dependency in the page-load path. Fonts/tokens are licensed for
  self-hosting (OFL / open).
- **`HubStatusBadge` → a server-rendered template partial.** The handoff's
  `HubStatusBadge.dc.html` React component becomes a Go template partial emitting
  **inline SVG** with the five distinct silhouettes (check-circle · cloud-? ·
  triangle · octagon-x · pause-circle). Status is conveyed by **icon + label +
  silhouette**, never hue alone — legible in grayscale and colorblind-safe, since
  these screens are screenshotted into legal/audit contexts (handoff invariant 4).
- **Full five-status taxonomy must become store-provable.** The current dashboard
  renders only the store-provable subset (`frozen` / `verified` / `inactive`); the
  Evidence Ledger requires all five, adding `unresolvable` and `unverified` with
  distinct silhouettes. This milestone must persist/expose the full taxonomy so the
  badge renders it honestly at the HTTP seam (today these richer statuses live only
  in the metrics registry, not in a store-provable form).
- **The handoff's ten invariants are conformance constraints** (two trust tiers
  distinct; honest coverage; five statuses distinct + frozen unmistakable &
  non-dismissable; color never the only signal; verifier independence felt; the two
  anchors never conflated; vocabulary discipline per `CLAUDE.md`/`CONTEXT.md`;
  SSR baseline; instance identity legible; truthful empties/pending). They are the
  acceptance bar for M-UI and the WASM milestone.

**Out of scope (unchanged, restated for the designer):** cosigning / cross-monitor
compare UI; resolution/projection views (current owner, gateway URL); auth / login
/ settings / write / config screens; operator deployment screens; in-page
authoritative Bitcoin verification (the page shows calendar-asserted anchor status
and offers the `.ots`; the user runs `ots verify`).

## Consequences

- **A new v1-blocking milestone (M-UI).** v1 "Done When" now includes the Evidence
  Ledger build-out; the functional M3 surfaces remain the baseline it dresses and
  extends. The CID loop advances M-UI after M3, before WASM.
- **New screens are real backend work, not just CSS.** The *record list* needs a
  paginated read over `iscc_index`; *single record* needs per-seq record + raw
  bytes + `note.$schema`; the *certificate* needs the **proof-bundle assembler**
  (`{checkpoint, inclusion/consistency proof, record bytes, hub key, ots?}`) the
  WASM verifier also shares; the five-status badge needs the status taxonomy made
  store-provable. These dependencies land inside M-UI.
- **The inclusion certificate is a realm-wide endpoint (decided).** An **ISCC-IDv1**
  is fully self-describing (we support v1 only): an 80-bit code = 16-bit header +
  64-bit body, where the header **SubType nibble is the realm** (0 = test/sandbox,
  1 = operational) and the body is a big-endian `uint64` splitting into
  `timestamp = body >> 12` (52 bits, µs since epoch) and `hub_id = body & 0xFFF`
  (low 12 bits, slot 0–4095). So the certificate is keyed by the id alone: a single
  realm-wide `/inclusion/{iscc_id}` (the holder has only an ISCC-ID, not a hub). The
  realm index, dossier, and log browser still use the established `/<domain>/log/…`
  per-hub subtree + exact-path mounting (`cmd/iscc-monitor/main.go`); only the
  certificate is realm-wide.
  *Implementation note:* decode `(realm, hub_id)` from the id — port the codec from
  `iscc/iscc-core` `iscc_id.py` (mirroring the hub side `iscc_hub/iscc_id.py`) into
  the not-yet-built `internal/index` `index/iscc.go` — then resolve the 12-bit
  `hub_id` slot to a hub via the **authoritative registry**: `iscc-hub`'s
  `hubs/<network>.yaml`, mapping `hub_id → url` per network (realm 0 →
  `testnet.yaml`: hub 0 = sb0.iscc.id, hub 1 = sb1.amlet.id; realm 1 →
  `mainnet.yaml`: hub 1 = iscc.id, hub 2 = amlet.id; the decoded realm also keys the
  per-network DB, ADR-0007). Resolving the slot is new machinery: today
  `hubs.hub_id` is a local surrogate from `UpsertHub`'s `last insert id` (**not** the
  embedded 12-bit field), `internal/store/iscc_index.go` stores the id *"without a
  codec"*, and `internal/registry` currently reads a **domains-only** `realm.txt`
  carrying no hub_id — so the monitor must consume the `hubs/<network>.yaml`
  `hub_id ↔ url ↔ active` map (an M1/registry-format change; see consequences). Its
  `pubkey` field is **deprecated and ignored** — the key source stays did:web
  (ADR-0009). Once the hub is resolved, reuse `SeqsForISCCID(hubID, id)`; a
  cross-hub fan-out is the trivial fallback. Concrete path picked during `advance`.
- **Hub-id resolution adopts the iscc-hub Hub-List (M1/registry update).** The
  authoritative `hub_id ↔ url` resolver is `iscc-hub`'s `hubs/<network>.yaml`
  (`{version, network, hubs:[{hub_id, url, active, pubkey?}]}`), owned by iscc-hub —
  the monitor only **consumes** it. M-UI's realm-wide certificate needs the
  `hub_id → url` map, so `internal/registry` moves from the domains-only `realm.txt`
  to this schema, capturing each hub's real **12-bit `hub_id`** (distinct from the
  surrogate `hubs.hub_id` PK) + `active`. Its `pubkey` is **deprecated and ignored**
  — keys come from did:web (ADR-0009); the superseded `valid_from` proposal
  (`.claude/hublist-schema-proposal.md`) confirms this. This is M1/registry work
  M-UI depends on; whether to persist `hub_id` in the `hubs` schema or resolve it
  in-registry is settled during `advance`.
- **Anchor states degrade honestly before OTS exists.** Building the dossier and
  certificate before the OTS milestone is fine: the Bitcoin-anchor panel renders
  the normal "pending / not yet anchored" state (handoff §6, invariant 10), and the
  comparison-anchor panel is a separate concept entirely (invariant 6).
- **The status palette extends the DS tokens.** The five bespoke status hues come
  from the handoff §5/§8 (`verified #5f8f1f` · `unresolvable #b07f00` · `unverified
  #d2691e` · `frozen #e0353f` · `inactive #9aa1ab`), layered on the DS `tokens/`;
  grayscale-distinguishability is carried by icon+label+silhouette regardless of
  hue, so the palette choice is non-load-bearing for accessibility.
- **Verifier app independence is unchanged (ADR-0003).** Surface C stays a separate
  static origin (`monitor.iscc.codes`), reproducible + hash-pinned; M-UI only adds
  the instance-side "verify independently →" affordance and the tier-1/tier-2
  visual split that the WASM result elevates.
